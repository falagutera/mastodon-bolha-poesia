# Changelog

All notable changes to poet. Versions follow [SemVer](https://semver.org/); images are
published as `registry.gutocarvalho.net/images/poet:<version>-<build>`.

## 0.1.7 — 2026-11-07

Hardening from the updated linters.

- The post id sent to the instance is path-escaped.
- The image declares its non-root user (65532) explicitly.

## 0.1.6 — 2026-11-06

- Release notes page, linked from the top of the page, and the same data at `/release-notes.json`.

## 0.1.5 — 2026-11-05

- Custom emoji are replaced only in the text of a post, never inside links or markup.

## 0.1.4 — 2026-10-31

First tagged version on poet.bolha.us.

- Turn a public post into a PNG or PDF to share, keeping its links.
- Media fetched only from public addresses, through a signed proxy.
- Version in the footer.
