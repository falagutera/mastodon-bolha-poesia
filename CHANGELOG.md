# Changelog

All notable changes to poet. Versions follow [SemVer](https://semver.org/); images are
published as `registry.gutocarvalho.net/images/poet:<version>-<build>`.

## 0.2.0 — 2026-11-08

- **The bolha.io look** for the page, from `open/gcn/ui` (`assets/style.css`, a synced copy, and `VISUAL.md`), like the other bolha apps: light and dark from the system, system fonts, a header with the bubble, "A post as **an image**" as the heading, the card options in a card, pill buttons and the footer "version · source code · part of bolha.io · based on Mastopoet".
- **The image you download is the same as before**: the card has its own stylesheet (`assets/card.css`) and keeps its IBM Plex fonts and colours, checked pixel by pixel against the previous version.

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
