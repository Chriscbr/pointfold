# Pointfold

Turn nested Markdown bullets into a compact, expandable HTML outline. Points have copyable IDs, Markdown copy buttons, and cross-references; tables, images, and diagrams appear on expansion.

[Website and examples](https://rybicki.io/pointfold/) · [Format](FORMAT.md) · [Agent skill](skill/pointfold/SKILL.md)

## Install

Requires Go 1.22 or newer.

```sh
go install github.com/Chriscbr/pointfold@latest
pointfold render notes.md -o notes.html
pointfold check notes.md
pointfold refs notes.md
```

Open `notes.html` in a browser. `--depth 2` initially shows two levels; `--depth 0` expands everything. Stdin and stdout are supported with `-`.

## Write

```markdown
# Should we run a pilot?

- **Case for.** A pilot could test demand. {#pilot}
  - The estimate comes from a small sample. {#sample}
- The time cost should be weighed against the [sampling uncertainty](#sample).
```

Use one logical statement per point, optional bold titles, and `**Definitions**` for a definitions branch. Preserve stable `{#ids}` across edits; displayed numbers identify positions in the current version.

Indent tables, image paragraphs, or fenced `mermaid` / `pele` blocks under a point after a blank line. Local images are embedded and initially limited to 200px tall. Mermaid is bundled for offline use; Pele loads from a pinned CDN release when opened.

## Develop

```sh
go test ./...
go vet ./...
go build -o bin/pointfold .
bash scripts/build-site.sh
```

The Pages workflow builds `dist/` and publishes it on pushes to `main`.

Optional browser checks: `npm ci`, `npx playwright install chromium`, then `npm run test:browser`. Set `POINTFOLD_CHROME` to use an installed Chrome. Rebuild vendored Mermaid with `npm run build:mermaid`.

MIT licensed; dependency licenses are in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
