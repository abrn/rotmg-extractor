# Il2CppDumper (bundled)

This directory holds the [Il2CppDumper](https://github.com/Perfare/Il2CppDumper)
binary used by the `il2cppdumper` IL2CPP backend (`il2cpp.backend: il2cppdumper`).

The binaries are **git-ignored** (everything in this directory except this
README). Drop the build for your platform directly into this directory. The
extractor resolves the executable per-OS — `Il2CppDumper.exe` on Windows,
`Il2CppDumper`/`Il2CppDumper-{linux,mac}` elsewhere — or falls back to the
cross-platform `Il2CppDumper.dll`, which is run via `dotnet` (requires the .NET
runtime).

**Windows / cross-platform:** download the latest release zip from the
[releases page](https://github.com/Perfare/Il2CppDumper/releases) and extract its
contents (including `config.json`) into this directory. The framework-dependent
build ships `Il2CppDumper.dll`; run it with an installed .NET runtime.

Override the location with `il2cpp.il2cppdumper.dir` or point
`il2cpp.il2cppdumper.binary` straight at the executable/`.dll`.

The default `native` Unity-asset backend and the `cpp2il` IL2CPP backend need
none of this.
