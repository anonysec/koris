# SPA Smoke Test

Headless-Chromium smoke test for the three embedded SPAs (landing, admin, portal).
Fails the run on **any** console error or page error, and asserts expected copy is present.

## Run (local, using the cached Playwright + Chromium)

```bash
node web/e2e/smoke.cjs
# override target:
BASE_URL=https://panel.example.com node web/e2e/smoke.cjs
```

The script resolves `playwright` and a cached Chromium build automatically (npx cache /
`~/.cache/ms-playwright`). To pin locations: `PLAYWRIGHT_MODULE=... CHROMIUM_PATH=...`.

## Auth cookies

Landing needs no auth. Admin and portal need session cookies, produced once with curl:

```bash
curl -k -c /tmp/cj.txt   -X POST https://localhost:2096/api/auth/admin    -H 'Content-Type: application/json' -d '{"username":"...","password":"..."}'
curl -k -c /tmp/wscj.txt -X POST https://localhost:2096/api/auth/customer -H 'Content-Type: application/json' -d '{"username":"...","password":"..."}'
```

Override jar paths with `ADMIN_JAR` / `PORTAL_JAR`.

## Permanent / CI setup

```bash
npm i -D playwright && npx playwright install chromium
node web/e2e/smoke.cjs
```
