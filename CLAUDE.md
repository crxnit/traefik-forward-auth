# CLAUDE.md

Working notes for future Claude sessions on this repo.

## What this repo is

A security-patched, multi-arch container-publishing fork of
[thomseddon/traefik-forward-auth](https://github.com/thomseddon/traefik-forward-auth),
owned by **crxnit**. The downstream consumer is `crxnit/traefik-nginx-portal`
(a separate repo — do **not** modify from here).

The fork branched from upstream `v2.3.0` (the latest stable upstream tag,
published 2024-05-06) and has shipped one patched release line so far.
Upstream's `master` is only a couple of dep-bump commits past `v2.3.0` and
is effectively dormant; there is no upstream-published image for `v2.3.0`
(Docker Hub's `thomseddon/traefik-forward-auth:latest` is from 2021 and
carries ~70 HIGH+CRITICAL CVEs — not usable). Publishing our own image is
the whole point.

`AUDIT.md` at the repo root is the canonical record of findings and their
status. Keep it up to date when closing or reopening items.

## Posture (baseline — mirror on new forks)

- Ruleset-protected `main`: linear history, signed commits, required PR
  review, required status checks (`go-vet` / `staticcheck` / `gosec` /
  `govulncheck`), admin bypass on for the owner (`crxnit`).
- Signed commits via SSH (`tag.gpgsign=true`, `commit.gpgsign=true`,
  `user.signingkey=/Users/john/.ssh/id_rsa.pub`). If a commit fails with a
  signing error mid-session, ask the user to run `ssh-add ~/.ssh/id_rsa`
  rather than disabling signing.
- Pre-commit: gitleaks, trailing-whitespace, EOF newline, YAML parse,
  shebang-consistency, mixed-line-ending. Install with `pre-commit install`
  after clone.
- Dependabot weekly on `github-actions`, `gomod`, `docker`.

## CI layout

- `.github/workflows/ci.yml` — `go vet`, `staticcheck`, `gosec -severity
  high`, and a `govulncheck` wrapper. Runs on every push and PR.
- `.github/workflows/release.yml` — fires on `v*` tag push. Builds
  multi-arch (`linux/amd64,linux/arm64`) via buildx (`TARGETOS`/`TARGETARCH`
  consumed by the Dockerfile), publishes `ghcr.io/crxnit/traefik-forward-auth`
  with two tags: `:<version>` and `:<version>-<short-sha>`.
- `.github/workflows/codeql-analysis.yml` — CodeQL Go scan on push/PR to
  `main` and weekly Tue 10:00 UTC.

### The govulncheck allowlist

`ci.yml` has an inline accept-list for four unfixable
`github.com/traefik/traefik/v2` CVEs: `GO-2026-4880`, `GO-2026-4679`,
`GO-2026-4484`, `GO-2025-4205`. All four have `Fixed: N/A` upstream and
target provider packages we never import (Knative, K8s Gateway, Postgres
TCP, ingress-nginx). The wrapper's contract: run govulncheck normally; if
it reports findings, fail CI unless **every** reported ID is on the accept
list. Any new vulnerability — or any future fix for one of the four —
will fail CI. Review the list if you touch traefik imports.

### staticcheck.conf

Disables pure-style / simplification checks (`ST1000`, `ST1003`, `ST1005`,
`ST1008`, `ST1013`, `S1021`, `S1031`) and `SA5008` (false-positive for
`thomseddon/go-flags`'s idiomatic repeated `choice:` / `default:` struct
tags). Everything else runs. Revisit only if you do a deliberate
code-style cleanup pass.

## Versioning

Fork release tags follow **`v{FORK}-src.v{UPSTREAM}`** — e.g.
`v1.0.1-src.v2.3.0`. The fork owns its `v1.0.0`, `v1.0.1`, `v1.1.0`, …
numbering independent of upstream; the `-src.vX.Y.Z` suffix documents the
upstream source tag the build was cut from.

SemVer caveat: because `-src.` is a pre-release identifier, every tag
sorts below a bare `v{FORK}` would. Keep **every** release named with the
suffix — never publish a naked `v{FORK}` — and ordering within the fork
line stays consistent: `v1.0.0-src.v2.3.0` < `v1.0.1-src.v2.3.0` <
`v1.1.0-src.v2.3.0`.

### Cutting a release

From a clean `main`:

```bash
git tag -s -m "crxnit fork v1.X.Y, built from upstream v2.3.0

<what's in this release>" v1.X.Y-src.v2.3.0

git push origin v1.X.Y-src.v2.3.0
```

`-s` signs the tag (`tag.gpgsign=true` is already set). Push fires
`release.yml`. The multi-arch image lands on GHCR under both tags;
downstream consumers should pin the **index digest** (top-level manifest
list sha256) for immutable multi-arch references.

## Running scanners locally

```bash
go vet ./...
staticcheck ./...                                                     # respects staticcheck.conf
gosec -severity high -confidence low -fmt text ./...
govulncheck ./...                                                     # see allowlist note above
docker buildx build --platform linux/amd64,linux/arm64 -t test .      # multi-arch smoke build
trivy image --severity HIGH,CRITICAL ghcr.io/crxnit/traefik-forward-auth:<tag>
```

Scanners are installed via `go install` from their canonical modules
(`honnef.co/go/tools/cmd/staticcheck`, `github.com/securego/gosec/v2/cmd/gosec`,
`golang.org/x/vuln/cmd/govulncheck`). `trivy` is system-installed via
Homebrew.

## Known deferred chores

- `docker/setup-qemu-action@v3`, `docker/setup-buildx-action@v3`,
  `docker/login-action@v3`, `docker/build-push-action@v6` still run on
  Node 20. Default flip to Node 24 on the runner is 2026-06-02, Node 20
  removed 2026-09-16. Bump to Node-24 majors when they ship.
- `github/codeql-action@v3` deprecation announced for December 2026 —
  bump to `@v4` when it lands.

## Don't

- Don't reference or modify `crxnit/traefik-nginx-portal` from this repo.
  Coordination happens via the published GHCR image only.
- Don't use `--delete-branch` on `gh pr merge` when merging a stack of
  dependent PRs — GitHub auto-closes any PR whose base branch gets
  deleted. Retarget dependent PRs to `main` first, delete branches at the
  end.
- Don't use `context.WithTimeout(...)` with `oidc.NewProvider` — go-oidc v2
  stores that context and reuses it for later JWKS refreshes; once the
  `defer cancel()` fires on the setup function's return, every later
  `Verify` breaks with `context canceled`. Use
  `oidc.ClientContext(context.Background(), &http.Client{Timeout: ...})`
  instead — context lives forever, per-request HTTP timeout bounds every
  call.
