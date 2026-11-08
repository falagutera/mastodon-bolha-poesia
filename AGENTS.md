# AGENTS.md

Notes for coding agents (and people) changing poet. The README says what the app does and how to
install it; this file says how to change it without breaking the promises the README makes.

## What it is

One Go binary, no database, no sign-in, no disk. `/?url=<post>` reads a post of the instance from
its **public API as an anonymous visitor**, and Go renders a card. Only the last step runs in the
browser: `assets/poet.js` asks the vendored html-to-image for a PNG (and wraps it in a PDF).
Images on the card go through `/media`, a proxy that serves only addresses this process signed.

## Commands

```sh
gofmt -l .            # must print nothing
go vet ./...
go test ./...         # everything runs against a fake instance (fakeInstance in poet_test.go)
```

There is no database and no test variable. To look at a rendered card:
`D=$(mktemp -d) && POET_DUMP=$D go test -run TestCard ./... && echo $D` writes
`card.html` and `assets/` there (pictures point at `/media` and stay blank; the export buttons need
the real app). To run it for real: `POET_INSTANCE=https://<instance> POET_BASE_URL=http://localhost:8080 go run .`
and open `http://localhost:8080/?url=<address of a public post>`.

## Map

| File | What |
|---|---|
| `main.go` | config from `POET_*` env, the per-process media key, HTTP server timeouts, `-healthcheck` |
| `server.go` | routes, `securityHeaders` (CSP), `options` (theme, bg, footer, mark, pad) and their allowlists, `/media` proxy, the post cache |
| `mastodon.go` | `parsePostURL` (which addresses are accepted) and the public API client (`api.post`) |
| `render.go` | `postPolicy` (bluemonday), `sanitize`, emoji in text nodes only, `plainText` (alt text), `newCard` |
| `fetch.go` | `mediaClient`: the only client for remote URLs (public addresses only) |
| `i18n.go` | every text, in `en` and `pt`, including the option labels and month names |
| `templates/index.html` | the whole page: form, options, the card (`#stage`), alt text |
| `assets/` | `style.css` (the page's look, a copy from open/gcn/ui: never edit it here), `poet.css` (the page's own rules), `card.css` (the exported card: its own fonts and colours, one class per option value; keep the image identical), `poet.js`, `html-to-image.js` (vendored, unmodified), fonts (the card's only). `check.sh --card-fonts` |
| `internal/relnotes` | copy of the shared release-notes package (synced from its own repo, never edited here) |

Tests: `poet_test.go` (`fakeInstance` serves post 1 with CW, poll, emoji, media and an insecure
preview; post 2 is a boost; `/ok.png` and `/bad.svg` for the proxy; `testServer`, `get`, `page`),
`emoji_test.go`, `i18n_test.go`, `changelog_test.go`, `dump_test.go` (the `POET_DUMP` helper).

## Rules that must hold

1. **Only public posts.** `api.post` calls the instance **without a token**, so followers-only
   and direct posts answer 404/401 and become "not found"; never add authentication to it. A
   boost shows the original. `TestErrors`, `TestBoostShowsTheOriginal`.
2. **Only posts of the instance.** `parsePostURL` wants `https`, the instance's host exactly, and
   a post path (`/@name/id`, `/@name@server/id`, `/users/name/statuses/id`); the id is all that
   is sent to the API. `TestParsePostURL` (look-alike hosts, `user@host` tricks, `http`, `..`).
3. **`/media` fetches only what this process signed.** `proxy` signs the URL with the key made in
   `main.go` (so run one replica), `handleMedia` checks the HMAC, wants `https`, fetches through
   `mediaClient` (refuses private, loopback, link-local and CGNAT addresses at dial time, also
   after redirects), serves only `imageTypes` (no SVG), at most 16 MiB (`maxMedia`) and 16 at once
   (`slots`). `TestMediaOnlyServesSignedImages`, `TestMediaClientRefusesPrivateAddresses`.
4. **Every image on the card goes through `proxy`:** avatar, pictures, custom emoji. CSP
   `img-src 'self' data: blob:` enforces it; `TestCard` checks no remote host reaches the page.
5. **Post HTML is untrusted.** `postPolicy` keeps text markup and http/https links only; custom
   emoji become `<img>` only inside text nodes, with an https URL and a plain shortcode; the
   display name is escaped first. `TestSanitize`, `TestEmojiOnlyInTextNodes`, `TestCard`.
6. **Options are allowlisted.** `parseOptions` maps anything unknown to the default (the first
   value of each list) and `pad` to `0..maxPad`; the values become CSS class names, so nothing
   else may reach the template. `TestOptionsFallBack`, `TestOptionsChangeThePage`.
7. **Little JavaScript, all of it ours or vendored.** Only `poet.js` and `html-to-image.js`
   (unmodified; version and SHA-256 in `NOTICE`, update steps in the README). No inline script or
   style; the page must work without JavaScript except the download and copy buttons. CSP in
   `securityHeaders`; `base-uri 'self'`, `data:` and `blob:` exist for html-to-image.
8. **Both languages.** Every key in `translations` exists in `en` and `pt`, with 12 month names
   and a `<key>_<value>` label for every option value (`TestTextsInBothLanguages`).
9. **Bounded work.** Posts cached for a minute (`cacheTTL`, at most `cacheLimit` 256, errors
   included), 4 MiB per API answer, 15 s client timeouts.

## Adding an option (the usual change)

0. **Check it is not there already.** The lists in `server.go`: `themes`, `backgrounds`,
   `footers`, `marks`, and `pad` (0..`maxPad`).
1. Append the value to its list (the first one is the default: do not reorder).
2. The CSS class in `assets/poet.css` (`.bg-<value>`, `.card.<theme>`, ...), and template markup
   if it changes what the card shows. A dark background also needs the light watermark: add
   `.bg-<value> .mark` to the `.bg-night .mark` selector, or the mark is unreadable.
3. The label `<key>_<value>` in both languages in `i18n.go`.
4. A test for the new value with its own request (`TestOptionsChangeThePage` checks one fixed
   query): the class is on the page, and what must disappear does.
5. `gofmt`, `go vet`, `go test`, and look at the dump.

## Adding a card detail

1. The field in `status` (`mastodon.go`), then `card` in `newCard` (`render.go`), then the
   template; HTML only through `sanitize`, images only through `proxy`.
2. Texts in both languages; the data in `fakeInstance` (post 1) and a check in `TestCard`,
   including what must not reach the page.
3. `gofmt`, `go vet`, `go test`, and look at the dump.

## Releasing

- `VERSION` holds `X.Y.Z`; CI adds the build number (`X.Y.Z-BUILD`). Which part to bump: a fix or a
  small visual tweak is a patch; a new feature, option or detail is a minor; a change that needs a
  new grant or setting at install is a minor whose changelog entry starts with that step in **bold**;
  a breaking change is a major; docs, tests or CI only need no bump. Work not released yet can sit
  under `## Unreleased` in `CHANGELOG.md` (`internal/relnotes` skips that section).
- Every version needs its `CHANGELOG.md` entry (`TestChangelogHasVersion`); it is also the public
  `/release-notes` page and `/release-notes.json`, so write it for the people who use the site.
- Code, comments, commits and the README in English. Keep commits small with a message that says why.

## Security checks

CI runs `go vet`, `gofmt -l`, the tests, govulncheck and the shared security stage: gitleaks, govulncheck, trivy, gosec,
semgrep, poutine, hadolint, shellcheck, yamllint. Before a release that touches `/media`,
sanitizing or the accepted addresses, also review where a request can go and what a post can put
on the page — the part scanners do not see.
