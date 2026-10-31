# Stash website

The public site at <https://stash.pnguyen.au> — a marketing homepage plus the docs section.
Built with [Hugo](https://gohugo.io) and the [Hextra](https://imfing.github.io/hextra) theme.

## Local development

Hugo and Task are pinned in `mise.toml`, so from the repo root:

```sh
mise install
task site:run      # dev server with live reload
task site:build    # production build
```

The output lands in `site/public/` (gitignored). `site:build` uses the `baseURL` from
`hugo.yaml` unless `BASE_URL` is set in the environment (the deploy workflow sets it).

## Structure

| Path | Contents |
| --- | --- |
| `hugo.yaml` | Site config: title, `baseURL`, Hextra theme, nav menu. |
| `content/_index.md` | The marketing homepage. |
| `content/docs/` | The docs section (setup guides). |
| `static/CNAME` | The custom domain `stash.pnguyen.au`. |
| `go.mod` / `go.sum` | Pin the Hextra theme as a Hugo module. |

## Deploy

`.github/workflows/pages.yaml` builds `./site` and deploys to GitHub Pages on every push to
`main`. It uses `actions/configure-pages` → `actions/deploy-pages`, so the site source must be
set to **GitHub Actions** in the repository's Pages settings.

### Custom domain (one-time)

`stash.pnguyen.au` is a subdomain, so it needs a CNAME record, plus the domain configured in
GitHub:

1. At your DNS provider for `pnguyen.au`, add a CNAME record:
   - **Host:** `stash`
   - **Value:** `hpcsc.github.io`
2. In the repo **Settings → Pages**, set **Custom domain** to `stash.pnguyen.au`.
   GitHub verifies the CNAME and serves the site at that domain (HTTPS via a Let's Encrypt
   certificate, issued automatically).

Until the custom domain is configured, the site is available at
`https://hpcsc.github.io/stash/`.
