# Build fixtures and golden files

`internal/build/golden_test.go` builds each fixture in this directory and
diffs the rendered output against a committed `golden/` tree.
The fixtures cover the four regression classes that have shipped in past
releases:

- nav class names (`--nested` vs `--section`)
- relative vs absolute asset paths under sub-path GitHub Pages
- `.md` → `.html` rewrite in rendered content
- presence of override stylesheets

## Layout

```
testdata/
├── README.md                    ← this file
├── flat-nav/
│   ├── mkdocs.yml               ← single root-base site, two flat pages
│   ├── docs/                    ← markdown source
│   └── golden/                  ← expected rendered output (committed)
│       ├── _assets.txt          ← manifest of asset filenames (presence-only)
│       ├── index.html           ← byte-exact expected HTML
│       ├── about.html
│       └── search/search_index.json
├── nested-subpath/              ← multi-section nav, non-root site_url,
│   ├── ...                        nested directories — exercises base-path
│   └── golden/                    computation and `--nested` class
└── callouts-tabs/               ← `> [!NOTE]` callouts and `=== "Tab"` blocks
    ├── ...                        — exercises markdown preprocessors
    └── golden/
```

What's in `golden/`:

- HTML files and `search/search_index.json` are compared **byte-exact**.
- `_assets.txt` is a sorted manifest of every file under `_assets/`.
  Asset **contents** are not compared — the upstream Material CSS/JS
  blobs are large and minified; checking presence-by-filename is enough to
  catch the regression class we care about.

## Updating goldens

When an intentional change shifts rendered output:

```sh
go test ./internal/build/... -run TestGolden -update
```

That regenerates `testdata/<fixture>/golden/` for every fixture.
Inspect the result with `git diff testdata/` and commit if the changes are
expected.

## Adding a fixture

1. Create `testdata/<name>/` with `mkdocs.yml` and a `docs/` source tree.
   Keep the surface area minimal — every file in the fixture becomes a
   golden file that future PRs need to keep green.
2. Run `go test ./internal/build/... -run TestGolden -update` to populate
   `golden/`.
3. Run the tests again without `-update` to confirm they pass.
4. Commit the fixture and its goldens together.

The test harness discovers fixtures automatically by listing
`testdata/*/`; no Go code changes are needed when adding a fixture.
