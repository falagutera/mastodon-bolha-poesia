# poet

Turns a public post of a Mastodon instance into an image to share: **poesia.bolha.us** (Poesia; the old address poet.bolha.us redirects there).

Paste the address of a post, pick a theme, a background, what the footer shows and the margin, and download a PNG or a PDF. It also gives you alt text to go with it. The options live in the address, so a preview can be shared as a link: `/?url=<address of the post>`. Only posts of the instance work, including posts from other servers at their local address. To offer it on your instance, run your own.

It's a rewrite in Go of [Mastopoet](https://github.com/raikasdev/mastopoet) by Roni Äikäs, see `NOTICE`.

## Contents

- [Screenshot](#screenshot)
- [Testing and security](#testing-and-security)
- [Architecture](#architecture)
- [Install](#install)
- [How it works](#how-it-works)
- [Updating html-to-image](#updating-html-to-image)
- [Development](#development)

## Screenshot

[![poet turning a post into a shareable image](docs/screenshot.png)](https://poesia.bolha.us)

*A live instance at [poesia.bolha.us](https://poesia.bolha.us): the card options and the generated image, PT/EN.*

## Testing and security

poet takes an address from anyone, fetches a post and its images, and puts the post's HTML on a page. So it is tested against what would hurt: turning a non-public post into an image, being used to fetch other addresses (including internal ones) through its media proxy, and post HTML that could inject markup or script. What is done today (2026-11-08):

**Every pipeline.** GitLab CI ([`.gitlab-ci.yml`](.gitlab-ci.yml), plus the shared security stage from `open/gcn/ci-templates`, file `security.yml`). The test job runs on every pipeline; the security stage runs on every merge request and every push to `main`; the image is built and published only from `main`.

- Blocking (the pipeline fails and no image is published):
  - `go vet`, a `gofmt` check, and the full test suite (no services needed: the tests use a fake instance);
  - `govulncheck`: known vulnerabilities in Go code paths poet actually reaches (in the test job and in the security stage);
  - `gitleaks`: secrets committed to the repository;
  - `trivy`: critical vulnerabilities that have a fix, secrets and misconfigurations in the repository.
- Informative (reported, not blocking): `semgrep`, `gosec`, `hadolint` (Dockerfile), `yamllint`, and `poutine` (security of the CI pipeline itself). The `gosec` findings that are false positives are annotated in the code with the reason.

**The test suite.** 16 tests (`go test -list`), running the real HTTP handlers against a fake Mastodon instance. They cover:

- only public posts of the instance: posts are read without a token, non-public ones are "not found", a boost shows the original; accepted addresses are `https` on the instance's exact host, and look-alike hosts, `user@host` tricks, `http` and `..` are refused;
- the media proxy: it serves only addresses this process signed, only image types (no SVG), and refuses private, loopback, link-local and CGNAT addresses, also after redirects; no remote image reaches the page directly;
- untrusted HTML: sanitizing, custom emoji replaced only in text, never inside links, the alt text as plain text;
- options: unknown values fall back to the default, each value changes the page;
- both languages, required settings, and the release notes.

**Reviews.** A [Strix](https://github.com/usestrix/strix) review found that a custom emoji shortcode inside a link could break out of the link's markup; fixed in 0.1.5 (emoji are now replaced only in text) with a regression test. In 0.1.7 the findings of the updated linters (`gosec`, `semgrep`, `hadolint`) were fixed. These are what we checked; they are not an outside audit and don't certify anything.

**Reporting a vulnerability.** Write to contato@gutocarvalho.net. Please don't open a public issue for it.

## Architecture

**Why it exists.** People share posts as images (on other networks, in slides, in print), and screenshots come out cropped, with the interface around them and no alt text. poet makes a clean card of one public post, with alt text to go with it, from the instance's own address.

**In plain words.**

1. the person pastes the address of a post of the instance;
2. poet reads the post from the instance's public API as an anonymous visitor, sanitizes its HTML, and draws the card on the server as an HTML page, with the chosen options;
3. images on the card (avatar, pictures, custom emoji) point to poet's own `/media`, which fetches them from the original server;
4. on "download", a small script in the browser turns the card into a PNG (or a one-page PDF holding that image). The browser does this step because it renders emoji and every writing system correctly.

**Components.**

```
          browser  (poet.js + vendored html-to-image: PNG/PDF of the card)
             |
       poet (this app, one replica)
         |              \
         |               +--> /media proxy --> image servers (https, public addresses only,
         |                    signed URLs only        image types only, 16 MiB at most)
         v
   the Mastodon instance (one only)
     - public API, no token: the post
```

- Nothing is stored: no database, no sign-in, no disk. Posts are cached in memory for a minute (at most 256).
- The media proxy signs each image address with a key made at startup, so only addresses that came from the API are fetched. That key lives only in memory, which is why poet runs as a single replica.
- The page's CSP lets images load only from poet itself, so the browser never fetches images from other servers.

## Install

You need Docker. There is no database and nothing to register, and the image is ready to pull.

1. Create a folder with these two files.

   `.env`:

   ```ini
   POET_INSTANCE=https://example.social
   POET_BASE_URL=https://poet.example.social
   ```

   `docker-compose.yml`:

   ```yaml
   services:
     poet:
       image: registry.gutocarvalho.net/images/poet:latest
       container_name: poet
       restart: unless-stopped
       env_file: .env
       read_only: true
       cap_drop: [ALL]
       security_opt: ["no-new-privileges:true"]
       mem_limit: 128m
       ports:
         - "127.0.0.1:8080:8080"
   ```

2. Start it: `docker compose up -d`.

3. Point your reverse proxy at `127.0.0.1:8080`. Run a single replica: media links are signed with a key made at startup.

To update: `docker compose pull && docker compose up -d`. For a fixed version, swap `latest` for a tag such as `0.1.3-849`.

To build it yourself instead, clone this repository, copy `.env.example` to `.env` and run `docker compose up -d --build`.

### Versions

Versions look like `X.Y.Z-BUILD`. `X.Y.Z` is in the `VERSION` file and follows SemVer, and `BUILD` is the GitLab pipeline number. A local build shows `X.Y.Z-dev`. The page footer shows the version and links to the [container registry](https://gitlab.gutocarvalho.net/images/poet/container_registry).

What changed in each version is in [CHANGELOG.md](CHANGELOG.md), which the app also serves as its release notes page (`/release-notes`, linked at the top of the page) and as `/release-notes.json`. Add an entry when you bump `VERSION`: a test fails without it.

| Image tag | Moves? |
|---|---|
| `X.Y.Z-BUILD` | Never. Pin this one in production. |
| `X.Y.Z` | On each build of that version. |
| `latest` | On each build. |

### Settings

| Variable | Required | Default | What it is |
|---|---|---|---|
| `POET_INSTANCE` | yes |  | Your instance. Only its posts can be turned into images. |
| `POET_BASE_URL` | yes |  | Where this app is served. Its name goes in the corner of the image. |
| `POET_LANG` | no | `en` | `en` or `pt`, when the browser asks for neither. |
| `POET_LISTEN` | no | `:8080` | Address inside the container. |

## How it works

- Posts come from the instance's public API, as an anonymous visitor would see them, so nothing private can be turned into an image. The HTML is sanitized again here, and the page has a strict CSP.
- Go draws the preview. Only the last step runs in the browser, which is the one thing that gets emoji and every writing system right: `assets/poet.js` asks [html-to-image](https://github.com/bubkoo/html-to-image) for a PNG of the preview. The PDF is one page holding that image, so its text is part of the picture. The page works without JavaScript, except for those buttons and the copy one.
- Avatars and images go through `/media`, on this same address, so the browser can read them to build the PNG. It only serves addresses signed here, which only ever come from the API, and only image types.
- Posts are cached in memory for a minute. Nothing is written to disk.
- English and Brazilian Portuguese. It follows the browser, with a selector at the top to switch (kept in a cookie).

## Updating html-to-image

`assets/html-to-image.js` is the unmodified file from the npm package. To update it, replace the file, check it against the package's integrity hash and update the version and SHA-256 in `NOTICE`.

## Development

    go test ./...

Changing the code, with or without a coding agent: [`AGENTS.md`](AGENTS.md) has the commands,
a map of the files and the rules a change must keep.

AGPL-3.0, see `LICENSE` and `NOTICE`.
