# AssetRipper (bundled)

This directory holds the [AssetRipper](https://github.com/AssetRipper/AssetRipper)
binary used by the `assetripper` extraction backend (full Unity asset export:
textures, sprites, meshes, text, etc.).

The binaries are **git-ignored** (everything in this directory except this
README) because they exceed GitHub's 100MB file limit. They live in the working
tree but are not committed. Extract the build for your platform directly into
this directory. The extractor resolves the executable name per-OS:
`AssetRipper.GUI.Free` on macOS/Linux, `AssetRipper.GUI.Free.exe` on Windows.

**macOS (arm64):**

```sh
curl -L -o ar.zip \
  https://github.com/AssetRipper/AssetRipper/releases/download/1.3.14/AssetRipper_mac_arm64.zip
unzip -o ar.zip -d .
rm ar.zip
xattr -d com.apple.quarantine AssetRipper.GUI.Free 2>/dev/null || true
chmod +x AssetRipper.GUI.Free
```

**Windows (x64):** download
`AssetRipper_win_x64.zip` from the
[releases page](https://github.com/AssetRipper/AssetRipper/releases) and extract
`AssetRipper.GUI.Free.exe` (plus its sidecar files) into this directory.

Other platform builds: `AssetRipper_{mac,linux,win}_{x64,arm64}.zip`.

To commit the binary instead of ignoring it, track it with
[git-lfs](https://git-lfs.com): `git lfs track "tools/assetripper/AssetRipper.GUI.Free"`.

The default `native` backend needs none of this — it parses Unity TextAssets in
pure Go with no external binary.
