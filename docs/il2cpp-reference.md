# il2cpp dump — reference (for roadmap item #4)

Notes preserved from the original Python tool before its source was deleted, to
guide porting the il2cpp dump. The Go implementation will live in
`internal/il2cpp` and follow the same bundled-binary pattern as
[`tools/assetripper`](../tools/assetripper/README.md).

## Inputs

- **GameAssembly** binary — `GameAssembly.dylib` (macOS), `GameAssembly.dll`
  (Windows), `GameAssembly.so` (Linux). Already located by `localsrc.Build.GameAssembly`.
- **`global-metadata.dat`** — already located by `localsrc.Build.Metadata`.

## Metadata decryption (`internal/metadata`)

RotMG obfuscates `global-metadata.dat` on **Windows** (XXTEA-encrypted custom
header, two XOR-masked sections, payload shifted by `0x1E4`). The **macOS** build
ships an already-valid metadata (begins with magic `0xFAB11BAF`), so decryption
is detected and skipped (`metadata.IsDecrypted`). The pipeline runs
`prepareMetadata` automatically (config `extraction.decrypt_metadata`, default
on), writing `game_files/global-metadata.decrypted.dat` only when decryption is
actually performed.

**The decryption constants are build-specific and rotate.** They live at the top
of `internal/metadata/metadata.go`:

| Constant | Meaning |
|----------|---------|
| `xxteaKeyText` | XXTEA key bytes (from `GameAssembly!FUN_180223b50` key packing) |
| `lenOffBase` (`0x2F1AF`) | base for `len_off = -0x2F1AF - *(int32*)meta - 4` |
| `teaLenAdd` (`0x621CF`) | added to the length seed → XXTEA block length |
| `shift` (`0x1E4`) | on-disk payload shift |
| post-XXTEA fixups | swap `header[0]`/`header[-1]`; `header[9]^=0x27`; `header[5]^=0x59` |
| string heap XOR | `dec[hdr[0x2C]+i] ^= i+0x5F` (size `hdr[0x0C]`) |
| second table XOR | `dec[hdr[0x58]+i] ^= 0x0D-i` (size `hdr[0x23C]`) |

To **refresh** them for a new build, reverse-engineer the current
`GameAssembly.dll`: the metadata loader (`FUN_18021eb70` in the RE notes) holds
the XOR offsets/constants, and `FUN_180223b50` the XXTEA key packing. A wrong key
surfaces as `bad XXTEA plaintext length` (current live build is stale — needs a
refresh). Verify a refresh by checking the output starts with `0xFAB11BAF` and
has plaintext identifier strings.

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
| Data packets | `Net.SocketServer.Messages.Data` |
| Pool managers | `Managers.Pool` |
| Debug tools | `DebugTools` |
