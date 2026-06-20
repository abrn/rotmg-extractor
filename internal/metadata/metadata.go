// Package metadata decrypts RotMG's obfuscated il2cpp global-metadata.dat into a
// standard, dumpable il2cpp metadata file.
//
// RotMG ships the metadata with an XXTEA-encrypted custom header, two XOR-masked
// sections (the string heap and a second table), and the on-disk payload shifted
// by a fixed offset. The macOS build, however, ships an already-valid metadata
// (it begins with the il2cpp magic), so decryption is skipped there — see
// IsDecrypted / Prepare.
//
// The decryption constants (XXTEA key, magic offsets) are reverse-engineered
// from a specific GameAssembly build and rotate between releases; see
// docs/il2cpp-reference.md for how to refresh them.
package metadata

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
)

// Magic is the standard il2cpp global-metadata.dat sanity magic (0xFAB11BAF).
const Magic uint32 = 0xFAB11BAF

// DefaultVersion is the il2cpp metadata version written into the decrypted
// output header.
const DefaultVersion uint32 = 29

const (
	delta      uint32 = 0x9E3779B9 // XXTEA round constant
	shift             = 0x1E4      // on-disk payload is shifted by this many bytes
	lenOffBase        = 0x2F1AF    // base for locating the length seed
	teaLenAdd         = 0x621CF    // added to the length seed to get the XXTEA block length
)

// The XXTEA key is stored in GameAssembly XOR-obfuscated with xorKey64 (an
// 8-byte repeating little-endian key). Both are build-specific and rotate
// between releases (see docs/il2cpp-reference.md to refresh).
const (
	xorKey64       uint64 = 0x89D75571BBA92FD7
	xxteaKeyEncHex        = "e117cb894531b5bfe31c9c8b4560e3bf" +
		"b24ecbd8126cb5b8e74a9e8f1060eeb9" +
		"e017a9"
)

// keyText de-obfuscates the XXTEA key (GameAssembly XOR literal): each byte is
// XORed with the repeating 8-byte xorKey64.
func keyText() []byte {
	enc, err := hex.DecodeString(xxteaKeyEncHex)
	if err != nil {
		panic("metadata: invalid xxteaKeyEncHex: " + err.Error())
	}
	var k [8]byte
	binary.LittleEndian.PutUint64(k[:], xorKey64)
	out := make([]byte, len(enc))
	for i, c := range enc {
		out[i] = c ^ k[i&7]
	}
	return out
}

// IsDecrypted reports whether data already begins with the il2cpp magic, i.e. it
// is a standard metadata file that needs no decryption (the macOS case).
func IsDecrypted(data []byte) bool {
	return len(data) >= 4 && binary.LittleEndian.Uint32(data) == Magic
}

// keyWords packs the key bytes into the four XXTEA key words. k0 intentionally
// uses byte 6 as its high byte, matching GameAssembly's key packing.
func keyWords(key []byte) [4]uint32 {
	return [4]uint32{
		binary.LittleEndian.Uint32([]byte{key[0], key[1], key[2], key[6]}),
		binary.LittleEndian.Uint32(key[4:8]),
		binary.LittleEndian.Uint32(key[8:12]),
		binary.LittleEndian.Uint32(key[12:16]),
	}
}

// xxteaDecryptWithLength decrypts an XXTEA block whose final word holds the
// plaintext length.
func xxteaDecryptWithLength(data []byte, key [4]uint32) ([]byte, error) {
	n := (len(data) + 3) / 4
	if n == 0 {
		return nil, nil
	}
	v := make([]uint32, n)
	for i, c := range data {
		v[i/4] |= uint32(c) << uint((i&3)*8)
	}

	if n > 1 {
		rounds := 6 + 52/n
		s := uint32(rounds) * delta
		y := v[0]
		for s != 0 {
			e := int((s >> 2) & 3)
			for p := n - 1; p > 0; p-- {
				z := v[p-1]
				mx := (((z >> 5) ^ (y << 2)) + ((z << 4) ^ (y >> 3))) ^ ((s ^ y) + (key[(p&3)^e] ^ z))
				v[p] -= mx
				y = v[p]
			}
			z := v[n-1]
			mx := (((z >> 5) ^ (y << 2)) + ((z << 4) ^ (y >> 3))) ^ ((s ^ y) + (key[e] ^ z))
			v[0] -= mx
			y = v[0]
			s -= delta
		}
	}

	// The final word holds the plaintext length, which must land within the last
	// XXTEA word (data_len-3 .. data_len).
	outLen := int64(v[n-1])
	dataLen := int64((n - 1) * 4)
	if outLen < dataLen-3 || outLen > dataLen {
		return nil, fmt.Errorf("bad XXTEA plaintext length %d", v[n-1])
	}
	out := make([]byte, outLen)
	for i := range out {
		out[i] = byte(v[i/4] >> uint((i&3)*8))
	}
	return out, nil
}

// Decrypt converts RotMG's obfuscated metadata bytes into a standard il2cpp
// metadata file with the given version written into the header.
func Decrypt(enc []byte, version uint32) ([]byte, error) {
	if len(enc) < 8 {
		return nil, fmt.Errorf("metadata too small (%d bytes)", len(enc))
	}

	// Locate the length seed: len_off = -0x2F1AF - *(int32*)enc - 4.
	enc0 := int32(binary.LittleEndian.Uint32(enc[0:]))
	lenOff := -lenOffBase - int(enc0) - 4
	if lenOff < 0 || lenOff+4 > len(enc) {
		return nil, fmt.Errorf("length offset %d out of range (size %d)", lenOff, len(enc))
	}
	lenSeed := int32(binary.LittleEndian.Uint32(enc[lenOff:]))
	teaLen := int(lenSeed) + teaLenAdd
	if teaLen < 0 || 4+teaLen > len(enc) {
		return nil, fmt.Errorf("XXTEA block length %d out of range (size %d)", teaLen, len(enc))
	}

	header, err := xxteaDecryptWithLength(enc[4:4+teaLen], keyWords(keyText()))
	if err != nil {
		return nil, err
	}
	if len(header) < 0x210 {
		return nil, fmt.Errorf("decrypted header too small (%d bytes)", len(header))
	}

	// Game post-processing applied after XXTEA.
	header[0], header[len(header)-1] = header[len(header)-1], header[0]
	header[9] ^= 0x27
	header[5] ^= 0x59

	// The on-disk data area is shifted by 0x1e4. Rebuild a standard file:
	// magic + version + decrypted header + shifted payload.
	if shift > len(enc) {
		return nil, fmt.Errorf("shift offset beyond file")
	}
	dec := make([]byte, len(enc)-shift)
	copy(dec, enc[shift:])
	if len(dec) < 8+len(header) {
		dec = append(dec, make([]byte, 8+len(header)-len(dec))...)
	}
	binary.LittleEndian.PutUint32(dec[0:], Magic)
	binary.LittleEndian.PutUint32(dec[4:], version)
	copy(dec[8:8+len(header)], header)

	// Table 1: XOR each byte with (0x0D - i).
	t1Off := binary.LittleEndian.Uint32(header[0xF4:])
	t1Size := binary.LittleEndian.Uint32(header[0x1C:])
	for i := uint32(0); i < t1Size; i++ {
		if int(t1Off+i) < len(dec) {
			dec[t1Off+i] ^= byte(0x0D - i)
		}
	}

	// Table 2 / string data: XOR each byte with (i + 0x5F).
	t2Off := binary.LittleEndian.Uint32(header[0x54:])
	t2Size := binary.LittleEndian.Uint32(header[0x20C:])
	for i := uint32(0); i < t2Size; i++ {
		if int(t2Off+i) < len(dec) {
			dec[t2Off+i] ^= byte(i + 0x5F)
		}
	}

	return dec, nil
}

// Prepare ensures a standard il2cpp metadata file exists at dstPath, given the
// (possibly obfuscated) source at srcPath.
//
// If the source is already valid (begins with the il2cpp magic — the macOS
// case), it is copied through unchanged and decrypted=false is returned, so the
// macOS pipeline never runs the build-specific decryption. Otherwise the source
// is decrypted.
func Prepare(srcPath, dstPath string, version uint32) (decrypted bool, err error) {
	enc, err := os.ReadFile(srcPath)
	if err != nil {
		return false, err
	}
	if IsDecrypted(enc) {
		return false, os.WriteFile(dstPath, enc, 0o644)
	}
	dec, err := Decrypt(enc, version)
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(dstPath, dec, 0o644)
}
