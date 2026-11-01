# poet

Turns a public post of a Mastodon instance into an image to share: **poet.bolha.us**.

Paste the address of a post, pick a theme, a background, what the footer shows and the margin, and download a PNG or a PDF. It also gives you alt text to go with it. The options live in the address, so a preview can be shared as a link: `/?url=<address of the post>`. Only posts of the instance work, including posts from other servers at their local address. To offer it on your instance, run your own.

It's a rewrite in Go of [Mastopoet](https://github.com/raikasdev/mastopoet) by Roni Äikäs, see `NOTICE`.

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

Versions look like `X.Y.Z-BUILD`. `X.Y.Z` is in the `VERSION` file and follows SemVer, and `BUILD` is the GitLab pipeline number. A local build shows `X.Y.Z-dev`. The page footer shows the version and links to the [container registry](https://gitlab.gutocarvalho.net/open/poet/container_registry).

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

AGPL-3.0, see `LICENSE` and `NOTICE`.
