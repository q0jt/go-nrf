package nrf

import (
	"bytes"
	"io"

	"github.com/marcinbor85/gohex"
)

func HexFileToBinary(b []byte) ([]byte, error) {
	r := bytes.NewReader(b)
	return intelHexToBinary(r, true)
}

func intelHexToBinary(r io.Reader, full bool) ([]byte, error) {
	mem := gohex.NewMemory()
	if err := mem.ParseIntelHex(r); err != nil {
		return nil, err
	}
	var size uint32
	for _, segment := range mem.GetDataSegments() {
		addr := segment.Address
		if !full && (addr == 0x1000) {
			return segment.Data, nil
		}
		size = addr + uint32(len(segment.Data))
	}
	b := mem.ToBinary(0, size, 0xFF)
	return b, nil
}

func readSoftDevice(b []byte) ([]byte, error) {
	r := bytes.NewReader(b)
	return intelHexToBinary(r, false)
}
