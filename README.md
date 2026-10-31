# poet

Turns a public post of a Mastodon instance into an image to share: **poet.bolha.us**.

Paste the address of a post, pick a theme, a background, what the footer shows and the margin, and download a PNG or a PDF. It also gives you alt text to go with it. The options live in the address, so a preview can be shared as a link: `/?url=<address of the post>`. Only posts of the instance work, including posts from other servers at their local address. To offer it on your instance, run your own.

It's a rewrite in Go of [Mastopoet](https://github.com/raikasdev/mastopoet) by Roni Äikäs, see `NOTICE`.

## Install

You need Docker. There is no database and nothing to register.

    cp .env.example .env   # instance and address of this app
    docker compose up -d

Then point your reverse proxy at `127.0.0.1:8080`. Run a single replica: media links are signed with a key made at startup.

### Prebuilt image

Every release is published, so you don't need to build it. In `docker-compose.yml`, replace the `build` and `image` lines with:

```yaml
image: registry.gutocarvalho.net/open/poet:latest
```

For a fixed version, use a tag such as `0.1.2-798`.

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
