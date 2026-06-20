# il2cpp dump — reference (for roadmap item #4)

Notes preserved from the original Python tool before its source was deleted, to
guide porting the il2cpp dump. The Go implementation will live in
`internal/il2cpp` and follow the same bundled-binary pattern as
[`tools/assetripper`](../tools/assetripper/README.md).

## Inputs

- **GameAssembly** binary — `GameAssembly.dylib` (macOS), `GameAssembly.dll`
  (Windows), `GameAssembly.so` (Linux). Already located by `localsrc.Build.GameAssembly`.
- **`global-metadata.dat`** — already located by `localsrc.Build.Metadata`.

## Original Python invocation (Il2CppInspector)

```
Il2CppInspector \
  --bin <GameAssembly> \
  --metadata <global-metadata.dat> \
  --layout class \
  --select-outputs \
  --py-out   <out>/il2cpp.py \
  --json-out <out>/metadata.json \
  --cs-out   <out>/types \
  --cpp-out  <out>/cpp
```

Output directory (publish as `il2cpp_dump/`): `il2cpp.py`, `metadata.json`,
`types/` (C# stubs), `cpp/`.

## Tooling notes

- The old repo bundled **Il2CppInspector** binaries (`Il2CppInspector-linux`,
  `Il2CppInspector-cli-win.exe`, + plugins) and an `unpacker-*` for the
  launcher. There was **no macOS binary**, so they are unusable on the current
  dev machine (mac/arm64). They were removed with `src/` and remain recoverable
  from git history if needed.
- This build is **Unity 6 (6000.0.58f2)**; Il2CppInspector's support for that
  metadata version is uncertain. **Cpp2IL** (cross-platform, self-contained,
  active) is the likely choice — bundle it under `tools/` and resolve the
  per-OS binary like AssetRipper.

## Namespaces of interest

From the old `NamespaceTypes` — useful for filtering the dump down to the game
types that matter (e.g. extracting packet definitions):

| Purpose | Namespace |
|---------|-----------|
| Incoming packets | `Net.SocketServer.Messages.Incoming` |
| Outgoing packets | `Net.SocketServer.Messages.Outgoing` |
| Pool managers | `Managers.Pool` |
| Debug tools | `DebugTools` |
