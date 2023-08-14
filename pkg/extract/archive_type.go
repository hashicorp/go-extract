package extract

import "fmt"

type ArchiveType struct {
	Algorithm  string
	MagicBytes []byte
	Name       string
	Offset     int
	// Extract Pointer
}

var KnownTypes = []ArchiveType{
	{
		Algorithm:  "Lempel-Ziv-Welch",
		Name:       "zip",
		Offset:     0,
		MagicBytes: []byte{0x1F, 0x9D},
	},
}

func (at *ArchiveType) FileHeaderSize() int {
	return at.Offset + len(at.MagicBytes)
}

func MaxArchiveHeaderLength() int {
	bufferSize := 0
	for _, at := range KnownTypes {
		signatureLen := (at.Offset + len(at.MagicBytes))
		if signatureLen > bufferSize {
			bufferSize = signatureLen
		}
	}
	return bufferSize
}

func DetermineArchiveType(inputArchive []byte) (*ArchiveType, error) {
	return nil, fmt.Errorf("unknown filetype")
}
