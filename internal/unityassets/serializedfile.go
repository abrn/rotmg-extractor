// Package unityassets parses Unity SerializedFile assets in pure Go, focused on
// extracting TextAssets (the RotMG game data: objects.xml, equip.xml, ...).
//
// It targets the format shipped by the game's Unity 6 (6000.0.58f2) il2cpp
// build: SerializedFile version 22, little-endian, with the type tree disabled.
// That layout lets us walk the type and object tables and read TextAsset
// (classID 49) objects positionally, without a generic type-tree interpreter.
//
// It deliberately does NOT handle textures, sprites, audio or meshes — those
// require decoders and remain the domain of the AssetRipper backend.
package unityassets

import (
	"encoding/binary"
	"fmt"
	"os"
)

// classIDTextAsset is the Unity class ID for TextAsset.
const classIDTextAsset = 49

// minSupportedVersion is the SerializedFile format version this parser targets.
// Earlier versions have different header/object layouts and are not handled.
const minSupportedVersion = 22

// TextAsset is a single extracted TextAsset object.
type TextAsset struct {
	Name   string
	Script []byte
}

// objectInfo is one entry in the SerializedFile object table.
type objectInfo struct {
	pathID    int64
	byteStart int64 // absolute offset into the file
	byteSize  uint32
	typeIndex int32
}

// SerializedFile is a parsed Unity asset file. Only the pieces needed to locate
// and read TextAssets are retained.
type SerializedFile struct {
	path       string
	order      binary.ByteOrder
	dataOffset int64
	classIDs   []int32 // per type-table entry
	objects    []objectInfo
}

// reader is a tiny cursor over an in-memory metadata buffer.
type reader struct {
	b     []byte
	pos   int
	order binary.ByteOrder
}

func (r *reader) u8() uint8      { v := r.b[r.pos]; r.pos++; return v }
func (r *reader) i16() int16     { v := r.order.Uint16(r.b[r.pos:]); r.pos += 2; return int16(v) }
func (r *reader) i32() int32     { v := r.order.Uint32(r.b[r.pos:]); r.pos += 4; return int32(v) }
func (r *reader) u32() uint32    { v := r.order.Uint32(r.b[r.pos:]); r.pos += 4; return v }
func (r *reader) i64() int64     { v := r.order.Uint64(r.b[r.pos:]); r.pos += 8; return int64(v) }
func (r *reader) align4()        { r.pos = (r.pos + 3) &^ 3 }
func (r *reader) skip(n int)     { r.pos += n }
func (r *reader) remaining() int { return len(r.b) - r.pos }

func (r *reader) cstring() string {
	start := r.pos
	for r.pos < len(r.b) && r.b[r.pos] != 0 {
		r.pos++
	}
	s := string(r.b[start:r.pos])
	r.pos++ // skip null
	return s
}

// OpenSerializedFile parses the metadata of a Unity SerializedFile. It returns a
// non-nil error if the file is not a supported SerializedFile (callers may treat
// that as "not an asset file" and skip it).
func OpenSerializedFile(path string) (*SerializedFile, error) {
	// The metadata lives at the start of the file (before dataOffset). We read a
	// generous prefix to parse it; object payloads are read lazily by seeking.
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read the fixed header (always big-endian) first.
	var hdr [0x30]byte
	if _, err := f.ReadAt(hdr[:], 0); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	version := binary.BigEndian.Uint32(hdr[0x08:])
	if version < minSupportedVersion {
		return nil, fmt.Errorf("unsupported SerializedFile version %d (need >= %d)", version, minSupportedVersion)
	}

	endianFlag := hdr[0x10]
	metadataSize := binary.BigEndian.Uint32(hdr[0x14:])
	dataOffset := int64(binary.BigEndian.Uint64(hdr[0x20:]))

	order := binary.ByteOrder(binary.LittleEndian)
	if endianFlag != 0 {
		order = binary.BigEndian
	}

	// Metadata begins at 0x30 and runs to (roughly) dataOffset. Read up to
	// dataOffset so the object table is fully covered.
	metaEnd := dataOffset
	if metaEnd <= 0x30 || metaEnd > int64(metadataSize)+0x40 {
		// metadataSize is measured from 0x30; bound the read sensibly.
		metaEnd = int64(metadataSize) + 0x30
	}
	buf := make([]byte, metaEnd-0x30)
	if _, err := f.ReadAt(buf, 0x30); err != nil {
		return nil, fmt.Errorf("reading metadata: %w", err)
	}

	sf := &SerializedFile{path: path, order: order, dataOffset: dataOffset}
	if err := sf.parseMetadata(buf, order); err != nil {
		return nil, err
	}
	return sf, nil
}

func (sf *SerializedFile) parseMetadata(buf []byte, order binary.ByteOrder) (err error) {
	defer func() {
		// The positional parse trusts the format; guard against malformed files
		// (e.g. a non-asset file that slipped through) rather than panicking.
		if r := recover(); r != nil {
			err = fmt.Errorf("malformed metadata in %s: %v", sf.path, r)
		}
	}()

	r := &reader{b: buf, order: order}

	_ = r.cstring() // unity version string
	_ = r.i32()     // target platform
	enableTypeTree := r.u8()
	if enableTypeTree != 0 {
		return fmt.Errorf("type tree enabled in %s; not supported by the native parser", sf.path)
	}

	// Type table.
	typeCount := int(r.i32())
	sf.classIDs = make([]int32, typeCount)
	for i := 0; i < typeCount; i++ {
		classID := r.i32()
		_ = r.u8()          // isStrippedType
		_ = r.i16()         // scriptTypeIndex
		if classID == 114 { // MonoBehaviour carries a 16-byte script id
			r.skip(16)
		}
		r.skip(16) // old type hash
		sf.classIDs[i] = classID
	}

	// Object table.
	objectCount := int(r.i32())
	sf.objects = make([]objectInfo, 0, objectCount)
	for i := 0; i < objectCount; i++ {
		r.align4()
		pathID := r.i64()
		byteStart := r.i64() // version >= 22: 64-bit
		byteSize := r.u32()
		typeIndex := r.i32()
		sf.objects = append(sf.objects, objectInfo{
			pathID:    pathID,
			byteStart: sf.dataOffset + byteStart,
			byteSize:  byteSize,
			typeIndex: typeIndex,
		})
	}
	return nil
}

// classID returns the Unity class ID for an object's type index.
func (sf *SerializedFile) classID(typeIndex int32) int32 {
	if typeIndex < 0 || int(typeIndex) >= len(sf.classIDs) {
		return -1
	}
	return sf.classIDs[typeIndex]
}
