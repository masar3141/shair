package local

import (
	"bytes"
	"encoding/binary"
	"os"
)

// header represents the metadata sent before a file transfer.
type header struct {
	headerSize  uint16   // holds the length []bytes for an encoded header.
	numFiles    uint16   // Number of files to transfer.
	nameLengths []uint8  // Lengths of each filename (max 255 characters). (count the bytes not the runes)
	names       []string // Actual filenames.
	fileSize    []int64  // Size of each file in bytes.
}

// newHeader creates a Header from a list of file information.
func newHeader(finfos ...os.FileInfo) *header {
	h := &header{
		numFiles:    uint16(len(finfos)),
		nameLengths: make([]uint8, len(finfos)),
		names:       make([]string, len(finfos)),
		fileSize:    make([]int64, len(finfos)),
	}

	for i, f := range finfos {
		name := f.Name()
		h.names[i] = name
		h.nameLengths[i] = uint8(len(name))
		h.fileSize[i] = int64(f.Size())
	}

	return h
}

// encode serializes the header into a byte slice.
func (h *header) encode() []byte {
	buf := bytes.NewBuffer(make([]byte, 4))

	// leave space for header size
	binary.BigEndian.PutUint16(buf.Bytes()[2:], h.numFiles)

	for _, nameLen := range h.nameLengths {
		buf.WriteByte(byte(nameLen))
	}

	for _, name := range h.names {
		buf.WriteString(name)
	}

	// Write file lengths using Uvarint.
	for _, fileLen := range h.fileSize {
		tmp := make([]byte, binary.MaxVarintLen64) // Worst case size
		n := binary.PutVarint(tmp, fileLen)
		buf.Write(tmp[:n])
	}

	// update header size in the header
	h.headerSize = uint16(buf.Len())

	// Update the header size at the beginning.
	binary.BigEndian.PutUint16(buf.Bytes()[:2], uint16(len(buf.Bytes())))

	return buf.Bytes()
}

// decodeHeader parses a byte slice into a Header.
func decodeHeader(p []byte) *header {
	offset := 0

	hdrSize := binary.BigEndian.Uint16(p[offset:])
	offset += 2

	numFiles := binary.BigEndian.Uint16(p[offset:])
	offset += 2

	h := &header{
		headerSize:  hdrSize,
		numFiles:    numFiles,
		nameLengths: make([]uint8, numFiles),
		names:       make([]string, numFiles),
		fileSize:    make([]int64, numFiles),
	}

	for i := 0; i < int(numFiles); i++ {
		h.nameLengths[i] = p[offset]
		offset++
	}

	for i := 0; i < int(numFiles); i++ {
		nameSize := int(h.nameLengths[i])
		h.names[i] = string(p[offset : offset+nameSize])
		offset += nameSize
	}

	for i := 0; i < int(numFiles); i++ {
		fileLen, n := binary.Varint(p[offset:])
		h.fileSize[i] = int64(fileLen)
		offset += n
	}

	return h
}
