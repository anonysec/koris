#!/usr/bin/env node
/**
 * Koris SPA smoke test — runs headless Chromium against the three embedded SPAs
 * (landing / admin / portal) and fails on any console error or pageerror.
 *
 * Self-contained: resolves `playwright` and a cached Chromium build without
 * requiring them as project dependencies (handy for quick local/CI checks).
 *
 * Usage:
 *   node web/e2e/smoke.cjs                 # defaults to https://localhost:2096
 *   BASE_URL=https://panel.example.com node web/e2e/smoke.cjs
 *
 * Cookie jars for the authenticated apps can be produced with curl, e.g.:
 *   curl -k -c /tmp/cj.txt  -X POST .../api/auth/admin  -d '{"username":..,"password":..}'
 *   curl -k -c /tmp/wscj.txt -X POST .../api/auth/customer -d '{"username":..,"password":..}'
 * (Paths are overridable via ADMIN_JAR / PORTAL_JAR env vars.)
 *
 * For a permanent CI setup, prefer `npm i -D playwright && npx playwright install chromium`
 * and set PLAYWRIGHT_MODULE / CHROMIUM_PATH accordingly.
 */
const fs = require('fs')
const path = require('path')
const { execSync } = require('child_process')

const BASE = (process.env.BASE_URL || 'https://localhost:2096').replace(/\/$/, '')

// ── Resolve playwright module ────────────────────────────────────────────────
function resolvePlaywright() {
  if (process.env.PLAYWRIGHT_MODULE) return require(process.env.PLAYWRIGHT_MODULE)
  try { return require('playwright') } catch { /* not on PATH */ }
  // Fall back to an npx cache install if present.
  try {
    const glob = execSync('ls -d /home/dev/.npm/_npx/*/node_modules/playwright 2>/dev/null')
      .toString().trim().split('\n')[0]
    if (glob) return require(glob)
  } catch { /* ignore */ }
  throw new Error('playwright not found — install it or set PLAYWRIGHT_MODULE')
}

// ── Resolve a Chromium executable ─────────────────────────────────────────────
function resolveChromium() {
  if (process.env.CHROMIUM_PATH && fs.existsSync(process.env.CHROMIUM_PATH)) return process.env.CHROMIUM_PATH
  try {
    const found = execSync('ls -d /home/dev/.cache/ms-playwright/chromium-*/chrome-linux64/chrome 2>/dev/null')
      .toString().trim().split('\n').filter(Boolean)
    if (found.length) return found[found.length - 1]
  } catch { /* ignore */ }
  return undefined // let playwright pick its bundled browser
}

// ── Parse a Netscape cookie jar into playwright cookie objects ────────────────
function jarToCookies(jarPath, domain) {
  if (!jarPath || !fs.existsSync(jarPath)) return []
  return fs.readFileSync(jarPath, 'utf8')
    .split('\n')
    // Netscape jars mark HttpOnly cookies with a "#HttpOnly_" prefix on the
    // domain field — those are real cookies, NOT comments. Strip the prefix,
    // then drop genuine comment lines (those still starting with '#').
    .map((l) => (l.startsWith('#HttpOnly_') ? l.slice('#HttpOnly_'.length) : l))
    .filter((l) => l && !l.startsWith('#'))
    .map((l) => l.split('\t'))
    .filter((f) => f.length >= 7)
    .map((f) => ({ name: f[5], value: f[6], domain, path: f[2] || '/', secure: true }))
}

const { chromium } = resolvePlaywright()
const executablePath = resolveChromium()

// Endpoints that may 4xx in some environments by design (e.g. the admin
// update-check returns 400 "updates_disabled" when PANEL_RELEASE_URL is unset,
// which is the normal state for a dev/offline box). These are not app bugs.
function benign(url) {
  return url.includes('/api/admin/update/check')
}

async function checkPage(browser, target) {
  const context = await browser.newContext({
    ignoreHTTPSErrors: true,
    baseURL: BASE,
    ...(target.cookies ? { storageState: { cookies: target.cookies, origins: [] } } : {}),
  })
  const page = await context.newPage()
  const pageErrors = []
  const badResponses = []
  page.on('pageerror', (e) => pageErrors.push('PAGEERR: ' + e.message))
  page.on('response', (r) => {
    if (r.status() >= 400 && !benign(r.url())) badResponses.push(`${r.status()} ${r.url()}`)
  })

  try {
    // Use 'load' not 'networkidle': the admin SPA holds a persistent realtime
    // WebSocket open, so the network never goes fully idle.
    await page.goto(target.url, { waitUntil: 'load', timeout: 20000 })
    await page.waitForTimeout(2500)
    // Scroll to trigger any lazy sections / infinite lists.
    for (let y = 0; y < 8000; y += 600) {
      await page.evaluate((v) => window.scrollTo(0, v), y)
      await page.waitForTimeout(120)
    }
    await page.waitForTimeout(800)
    const text = await page.evaluate(() => document.body.innerText.replace(/\s+/g, ' ').trim())

    const missing = (target.expect || []).filter((s) => !text.includes(s))
    const errors = [...pageErrors, ...badResponses]
    const ok = errors.length === 0 && text.length > 50 && missing.length === 0
    return {
      name: target.name,
      ok,
      textLen: text.length,
      errors: errors.slice(0, 8),
      missing,
    }
  } catch (e) {
    return { name: target.name, ok: false, errors: ['NAV: ' + e.message], missing: [] }
  } finally {
    await context.close()
  }
}

async function main() {
  const domain = new URL(BASE).hostname
  const targets = [
    { name: 'landing', url: BASE + '/', expect: ['Pricing', 'Features'] },
    {
      name: 'admin',
      url: BASE + '/admin/',
      cookies: jarToCookies(process.env.ADMIN_JAR || '/tmp/cj.txt', domain),
      expect: ['Dashboard'],
    },
    {
      name: 'portal',
      url: BASE + '/account/',
      cookies: jarToCookies(process.env.PORTAL_JAR || '/tmp/wscj.txt', domain),
    },
  ]

  const browser = await chromium.launch({
    ...(executablePath ? { executablePath } : {}),
    args: ['--no-sandbox'],
  })

  const results = []
  for (const t of targets) results.push(await checkPage(browser, t))
  await browser.close()

  console.log('\n=== Koris SPA smoke results ===')
  let failed = 0
  for (const r of results) {
    if (r.ok) {
      console.log(`  PASS  ${r.name}  (text ${r.textLen} chars)`)
    } else {
      failed++
      console.log(`  FAIL  ${r.name}`)
      if (r.missing.length) console.log(`        missing expected text: ${r.missing.join(', ')}`)
      if (r.errors.length) console.log(`        errors: ${JSON.stringify(r.errors)}`)
    }
  }
  console.log(`\n${results.length - failed}/${results.length} passed`)
  process.exit(failed ? 1 : 0)
}

main().catch((e) => { console.error(e); process.exit(2) })
