package metadata

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestXXTEADecryptGolden verifies the Go XXTEA decrypt matches the reference
// Python implementation byte-for-byte. The ciphertext was produced by XXTEA-
// encrypting the plaintext (with the length word appended) in Python.
func TestXXTEADecryptGolden(t *testing.T) {
	key := [4]uint32{0x11223344, 0x55667788, 0x99aabbcc, 0xddeeff00}
	cipher, _ := hex.DecodeString("8848d4412f5bb0c3da73c0ec5bf0afc555a72fa40158ee99faf38e50ce4d0c4910b0cb12")
	want := []byte("objects.xml RotMG metadata test!")

	got, err := xxteaDecryptWithLength(cipher, key)
	if err != nil {
		t.Fatalf("xxteaDecryptWithLength: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("decrypt = %q, want %q", got, want)
	}
}

func TestKeyWords(t *testing.T) {
	// Packing of the real key, matching GameAssembly (k0 high byte is key[6]).
	got := keyWords(xxteaKeyText)
	want := [4]uint32{0x35313138, 0x38353132, 0x62363161, 0x66346536}
	if got != want {
		t.Errorf("keyWords = %#x, want %#x", got, want)
	}
}

func TestIsDecrypted(t *testing.T) {
	valid := make([]byte, 8)
	binary.LittleEndian.PutUint32(valid, Magic)
	if !IsDecrypted(valid) {
		t.Error("magic-prefixed data should be detected as decrypted")
	}
	if IsDecrypted([]byte{0x55, 0xd4, 0xf3, 0xfe}) {
		t.Error("encrypted data should not be detected as decrypted")
	}
	if IsDecrypted([]byte{0x01, 0x02}) {
		t.Error("short data should not be detected as decrypted")
	}
}

// TestRealMacMetadata checks the installed macOS metadata is recognised as
// already-decrypted (so the mac pipeline skips decryption). Skipped if absent.
func TestRealMacMetadata(t *testing.T) {
	path := "/Users/admin/.local/share/RealmOfTheMadGod/Production/RotMGExalt.app/Contents/Resources/Data/il2cpp_data/Metadata/global-metadata.dat"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("macOS metadata not installed; skipping")
	}
	if !IsDecrypted(data) {
		t.Fatal("installed macOS metadata should already be a valid il2cpp file")
	}
	dst := filepath.Join(t.TempDir(), "out.dat")
	decrypted, err := Prepare(path, dst, DefaultVersion)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if decrypted {
		t.Error("macOS metadata should pass through without decryption")
	}
}

func TestPreparePassthrough(t *testing.T) {
	// An already-valid metadata (macOS case) is copied through unchanged with
	// decrypted=false — no build-specific decryption attempted.
	dir := t.TempDir()
	src := filepath.Join(dir, "global-metadata.dat")
	dst := filepath.Join(dir, "out.dat")

	content := make([]byte, 32)
	binary.LittleEndian.PutUint32(content, Magic)
	copy(content[8:], []byte("identifier-strings-here"))
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}

	decrypted, err := Prepare(src, dst, DefaultVersion)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if decrypted {
		t.Error("already-valid metadata should not be decrypted")
	}
	got, _ := os.ReadFile(dst)
	if !bytes.Equal(got, content) {
		t.Error("passthrough output differs from input")
	}
}
