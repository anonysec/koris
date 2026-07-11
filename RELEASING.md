# Releasing Koris

Releases are cut by pushing a semver git tag. Everything else is automated.

## Cut a release

```bash
# From a clean main:
git tag v0.93.0                # or whatever the next version is
git push origin v0.93.0
```

That's it. GitHub Actions (`.github/workflows/release.yml`) will:

1. Build all three frontends (admin, portal, landing) once
2. Cross-compile the Go binary for `linux/amd64` and `linux/arm64`, in both
   `full` and `lite` editions — **4 binaries total**, all with embedded
   frontend assets via `//go:embed`
3. Package each binary as `koris-<edition>-<os>-<arch>.tar.gz` bundled with
   `migrations/`, `koris.sh`, and `deploy/`
4. Compute SHA256 checksums (per-file and combined `SHA256SUMS`)
5. Build a multi-arch Docker image and push to `ghcr.io/anonysec/koris`
   with tags `:v0.93.0`, `:0.93`, `:0`, and (for main-branch tags) `:latest`
6. Create a GitHub Release with all binary artifacts attached, using
   the matching `CHANGELOG.md` section as the release body

Total time: ~8 minutes end-to-end.

## Version numbering

We follow SemVer. Rough guidance:

- **Patch** (`v0.93.1`) — bug fix, no config change, safe to auto-update
- **Minor** (`v0.94.0`) — new feature, backward-compatible config change
- **Major** (`v1.0.0`) — breaking config change or DB migration that can't be rolled back

Pre-release suffixes (`-rc.1`, `-beta.2`) trigger a **pre-release** on GitHub
(not marked as latest, won't auto-install).

## Before pushing the tag

1. **Add a `## [0.93.0] – YYYY-MM-DD` section to `CHANGELOG.md`** listing
   the changes. The release workflow extracts the section between this
   heading and the next `## ` heading as the GitHub Release body.
2. Bump `VERSION` file to match: `echo 0.93.0 > VERSION`
3. Commit: `git commit -am "chore(release): 0.93.0"`
4. Tag: `git tag v0.93.0`
5. Push: `git push origin main && git push origin v0.93.0`

## Manual release (bypass tag trigger)

```
# From the Actions tab on GitHub, run the Release workflow with a custom tag.
# Useful for re-running a failed publish without moving the tag.
gh workflow run release.yml -f tag=v0.93.0
```

## What users get

**Installer default (Docker + pre-built image):**
```bash
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh)
```
→ pulls `ghcr.io/anonysec/koris:latest` (~5s startup)

**Specific version:**
```bash
bash <(curl -Ls https://raw.githubusercontent.com/anonysec/koris/main/install.sh) --version=v0.93.0
```

**Direct binary download** (for non-Docker environments):
```bash
curl -LO https://github.com/anonysec/koris/releases/download/v0.93.0/koris-full-linux-amd64.tar.gz
tar xzf koris-full-linux-amd64.tar.gz
./koris-full-linux-amd64
```

**Verify:**
```bash
curl -LO https://github.com/anonysec/koris/releases/download/v0.93.0/SHA256SUMS
sha256sum -c SHA256SUMS
```

## Yanking a bad release

GitHub Releases:
```
gh release delete v0.93.0 --yes
git tag -d v0.93.0
git push origin :refs/tags/v0.93.0
```

GHCR image:
```
# Manually via https://github.com/users/anonysec/packages/container/koris/settings
# or with gh api / OCI delete calls.
```

## First-time GHCR setup

The `Release` workflow uses `GITHUB_TOKEN` with `packages: write`, which is
already granted at the top of `release.yml`. No manual PAT is needed.

The first time an image is pushed, GHCR creates the package as **private** by
default. Make it public (so users can `docker pull` without login):

1. https://github.com/users/anonysec/packages/container/koris/settings
2. Scroll to **Danger Zone** → **Change visibility** → **Public**
