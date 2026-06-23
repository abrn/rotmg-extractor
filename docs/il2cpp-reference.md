# il2cpp dump — reference (for roadmap item #4)

Notes preserved from the original Python tool before its source was deleted, to
guide the il2cpp dump implementation. The Go implementation lives in
`internal/il2cpp` and follows the same bundled-binary pattern as
[`tools/assetripper`](../tools/assetripper/README.md). Cpp2IL support is
implemented first; Il2CppDumper is still planned as a separate backend.

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
| `xxteaKeyEncHex` ⊕ `xorKey64` | XXTEA key, stored XOR-obfuscated (8-byte repeating `xorKey64`); de-obfuscated by `keyText()`. Key packing from `GameAssembly!FUN_180227460` |
| `lenOffBase` (`0x2F1AF`) | base for `len_off = -0x2F1AF - *(int32*)meta - 4` |
| `teaLenAdd` (`0x621CF`) | added to the length seed → XXTEA block length |
| `shift` (`0x1E4`) | on-disk payload shift |
| post-XXTEA fixups | swap `header[0]`/`header[-1]`; `header[9]^=0x27`; `header[5]^=0x59` |
| table 1 XOR | `dec[hdr[0xF4]+i] ^= 0x0D-i` (size `hdr[0x1C]`) |
| table 2 / string data XOR | `dec[hdr[0x54]+i] ^= i+0x5F` (size `hdr[0x20C]`) |

The two XOR sections are applied by `GameAssembly!FUN_1802224a0`. To **refresh**
the constants for a new build, reverse-engineer that function (XOR offsets) and
`FUN_180227460` (XXTEA key packing). A wrong key surfaces as
`bad XXTEA plaintext length`. Verify a refresh by checking the output starts with
`0xFAB11BAF` and has plaintext identifier strings (`MonoBehaviour`, `mscorlib`).
Verified working on the current live Windows build (output is byte-identical to
the reference Python).

## Current Go invocation (Cpp2IL)

The pipeline prepares `game_files/global-metadata.decrypted.dat`, stages a
minimal Cpp2IL game folder, then runs Cpp2IL into `il2cpp_dump/`.

With `il2cpp.cpp2il.full_dump: true`, it first runs:

```
Cpp2IL --list-output-formats
```

Then each listed format is run separately:

```
Cpp2IL \
  --game-path=<staged-game-dir> \
  --exe-name=RotMGExalt \
  --output-to=<out>/il2cpp_dump/cpp2il/<format> \
  --output-as=<format>
```

Logs are written to `il2cpp_dump/logs/`, and `manifest.json` records selected
formats, command arguments, durations, input hashes, and errors.

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
  metadata version is uncertain. **Cpp2IL** is the primary supported backend —
  bundle it under `tools/il2cpp/cpp2il` and resolve the per-OS binary like
  AssetRipper.

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

## Il2CppDumper backend (roadmap item #3)

[Perfare/Il2CppDumper](https://github.com/Perfare/Il2CppDumper) is wired in as a
selectable backend (`il2cpp.backend: il2cppdumper`), implemented in
`internal/il2cpp/il2cppdumper.go`.

Invocation: `Il2CppDumper <GameAssembly> <global-metadata.dat> <output-dir>`
(the managed `Il2CppDumper.dll` is run as `dotnet Il2CppDumper.dll ...`). It reads
`config.json` from the binary's directory; the backend ensures
`RequireAnyKey: false` so it never blocks on a prompt, and sets
`ForceIl2CppVersion`/`ForceVersion` when `force_version` is configured.

Outputs: `DummyDll/`, `dump.cs`, `il2cpp.h`, `script.json`, `stringliteral.json`
(plus IDA/Ghidra/BinaryNinja scripts) under `il2cpp_dump/`.

Rationale: Perfare's documented range is "Unity 5.3 – 2022.2", which covers the
IL2CPP metadata v29 era. The live build reports metadata v29.1, where Cpp2IL
currently fails with `EndOfStreamException`, so Il2CppDumper is a candidate path
to a working dump. Confirm against the current build; use `force_version: "29"`
if auto-detection misfires.
