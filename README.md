# RotMG Resource Extractor (Go)

Automatically extracts and publishes RotMG (Realm of the Mad God) build assets,
and announces new builds. This is a Go rewrite of the original Python tool.

It watches for new builds, extracts the Unity game data (objects, ground, items,
projectiles, …), consolidates it into combined XML files, diffs each build
against the last, writes a changelog, and can post a Discord notification.

> **Status:** local extraction is fully working. Live/remote downloading is
> shelved until the game's app-init endpoints are re-discovered (they now 404).
> il2cpp dumping is not yet ported. See [Roadmap](#roadmap).

---

## Quick start

```sh
go build -o extractor ./cmd/extractor
./extractor -once          # run a single pass
./extractor                # run continuously, polling on an interval
```

With no `extractor.yml` present, sensible defaults are used (local mode, native
backend, auto-discovered install). Flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `extractor.yml` | Path to the config file (optional) |
| `-once` | `false` | Run a single pass instead of looping |

## How it works

For the installed build, each pass:

1. **Discovers** the install (`internal/localsrc`) — macOS `.app` bundle or a
   Windows/Linux folder containing `*_Data`.
2. **Detects new builds** via a content hash of `global-metadata.dat`, compared
   against the last published build. Unchanged → nothing happens.
3. **Extracts** Unity TextAssets (`internal/unityassets`, the default native
   backend) or all assets (`internal/assetripper`).
4. **Merges** the many XML files into `object.xml` / `ground.xml` / `misc.xml`
   by top-level tag (`internal/mergexml`).
5. **Diffs** vs. the previous build — coarse file/line counts
   (`internal/builddiff`) and a semantic per-`id` game-data diff
   (`internal/gamediff`) rendered to a `changelog.md`.
6. **Publishes** the output to a versioned directory and refreshes
   `publish/.../current/`.
7. **Notifies** (optional Discord webhook, `internal/notify`).

## Output layout

```
<output.dir>/
  temp/                              cleared at startup (work-in-progress)
  publish/<env>/<build>/
    <version>-<hash>/                immutable archive of each build
    current/                         the latest build
      build_hash.txt                 build identity (drives new-build detection)
      exalt_version.txt
      timestamp.txt
      changelog.md                   semantic diff vs. the previous build
      extracted_assets/              raw extracted assets
      merged/{object,ground,misc}.xml
```

## Extraction backends

| Backend | Output | Needs a binary? | Notes |
|---------|--------|-----------------|-------|
| `native` (default) | Unity **TextAssets** as `.xml`/`.json`/`.txt` | No — pure Go | Cross-platform; produces the game data the merge/diff/changelog run on. |
| `assetripper` | **All** assets (textures, sprites, meshes, …) as a Unity project | Yes — bundled [AssetRipper](https://github.com/AssetRipper/AssetRipper) | Heavier; emits `.bytes` for text, so the XML merge step no-ops. See [`tools/assetripper/README.md`](tools/assetripper/README.md). |

## Configuration reference

All keys are optional; defaults are shown. See [`extractor.yml`](extractor.yml).

```yaml
source:
  mode: local            # "local" | "remote" (remote is shelved)
  local_path: ""         # install root; blank = auto-discover per OS
  snapshot: false        # also copy the full build files (slow, 100s of MB)

build:
  version_override: "6.11.0.0.0"   # used when the version can't be auto-detected

extraction:
  backend: native        # "native" | "assetripper"

assetripper:
  dir: tools/assetripper # directory holding the AssetRipper binary
  port: 50111
  export: primary        # "primary" (assets) | "project" (full Unity project)

notify:
  discord:
    enabled: false
    webhook_url: ""
    role_id: ""          # optional role to ping

poll:
  client_check_delay_minutes: 5
  launcher_check_delay_minutes: 30

output:
  dir: ./output

logging:
  level: debug           # debug | info | warn | error
  console: true
  colors: true
  file: true
```

## Platform notes

The `native` backend and the whole pipeline are pure Go and cross-compile for
macOS, Windows and Linux. Only the optional `assetripper` backend needs a
platform-specific binary (resolved automatically as `AssetRipper.GUI.Free` or
`.exe`). The original Python paths were macOS-specific; install-path
auto-discovery now covers all three platforms (Windows/Linux defaults are
best-effort — override with `source.local_path` if needed).

## Development

```sh
go test ./...                       # run tests
go vet ./...
GOOS=windows GOARCH=amd64 go build ./...   # cross-compile check
```

## Roadmap

- **il2cpp dump** — port the binary dump. See
  [`docs/il2cpp-reference.md`](docs/il2cpp-reference.md) (original used
  Il2CppInspector; Unity 6 likely needs Cpp2IL).
- **Remote download** — revive once the live app-init endpoints are known
  (`source.mode: remote` is stubbed and waiting).
- **Additional delivery targets** — the original config declared SSH, FTP and
  Redis pub-sub outputs (never implemented) alongside Discord. Add as
  `notify`/`publish` targets if needed.
