package nrf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

type MCUBootImgHeader struct {
	Magic            uint32
	LoadAddr         uint32
	Size             uint16
	ProtectedTLVSize uint16
	ImgSize          uint32
	Flags            uint32
	Ver              ImgVersion
	Pad              uint32
}

type ImgVersion struct {
	Major    uint8
	Minor    uint8
	Revision uint16
	BuildNum uint32
}

// ImageTLVInfo is information about TLV(tag-length-value).
// Magic and total size of TLV area.
type ImageTLVInfo struct {
	Magic     uint16
	TotalSize uint16
}

type ImageTLV struct {
	ItType uint8
	Pad    uint8
	ItSize uint16
}

type TLVType int

const (
	ImageTLVKeyHash       TLVType = 0x01 // hash of the public key
	ImageTLVSHA256        TLVType = 0x10 // SHA256 of image hdr and body
	ImageTLVRsa2048PSS    TLVType = 0x20 // RSA2048 of hash output
	ImageTLVEcdsa224      TLVType = 0x21 // ECDSA of hash output - Not supported anymore
	ImageTLVEcdsaSig      TLVType = 0x22 // ECDSA of hash output
	ImageTLVRsa3072PSS    TLVType = 0x23 // RSA3072 of hash output
	ImageTLVED25519       TLVType = 0x24 // ED25519 of hash output
	ImageTLVEncRsa2048    TLVType = 0x30 // Key encrypted with RSA-OAEP-2048
	ImageTLVEncKW         TLVType = 0x31 // Key encrypted with AES-KW-128 or 256
	ImageTLVEncEC256      TLVType = 0x32 // Key encrypted with ECIES-P256
	ImageTLVEncX25519     TLVType = 0x33 // Key encrypted with ECIES-X25519
	ImageTLVEncDependency TLVType = 0x40 // Image depends on other image
	ImageTLVEncSecCnt     TLVType = 0x50 // security counter
)

type MCUBoot struct {
	r      io.ReaderAt
	base   int64
	offset int64

	header *MCUBootImgHeader
}

func DetectMCUBoot(name string) (*MCUBoot, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return detectMCUBoot(b)
}

func detectMCUBoot(b []byte) (*MCUBoot, error) {
	offsets := findMCUBootImageMagic(b)
	var header MCUBootImgHeader
	var off int64
	for _, offset := range offsets {
		if !isMCUBootImageHeader(b, offset) {
			continue
		}
		r := bytes.NewReader(b[offset : offset+0x20])
		err := binary.Read(r, binary.LittleEndian, &header)
		if err != nil {
			return nil, err
		}
		off = offset
		break
	}
	r := bytes.NewReader(b)
	boot := &MCUBoot{r: r, base: off, header: &header}
	return boot, nil
}

func findMCUBootImageMagic(b []byte) []int64 {
	magic := []byte{0x3D, 0xB8, 0xF3, 0x96}
	r := make([]int64, 0)
	idx := 0
	for {
		offset := bytes.Index(b[idx:], magic)
		if offset == -1 {
			break
		}
		r = append(r, int64(idx+offset))
		idx += offset + 1
	}
	return r
}

func isMCUBootImageHeader(b []byte, offset int64) bool {
	s := offset + 0x20
	data := b[s : s+0x1e0]
	if !bytes.Equal(data, bytes.Repeat([]byte{0xFF}, 0x1e0)) {
		return false
	}
	return true
}

func (b *MCUBoot) Header() *MCUBootImgHeader {
	return b.header
}

func (b *MCUBoot) ExtractImage() ([]byte, error) {
	out := make([]byte, b.header.ImgSize)
	_, err := b.r.ReadAt(out, b.base+int64(b.header.Size))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (b *MCUBoot) seek(x int64) {
	b.offset += x
}

type TLVArea struct {
	ImageHash []byte
	KeyHash   []byte
	Signature []byte
}

func (b *MCUBoot) ReadTLVArea() (*TLVArea, error) {
	offset := b.base + int64(b.header.Size) + int64(b.header.ImgSize)
	b.offset = offset
	ti := make([]byte, 4)
	_, err := b.r.ReadAt(ti, b.offset)
	if err != nil {
		return nil, err
	}
	var tlvInfo ImageTLVInfo
	r := bytes.NewReader(ti)
	err = binary.Read(r, binary.LittleEndian, &tlvInfo)
	if err != nil {
		return nil, err
	}
	if tlvInfo.Magic != 0x6907 && tlvInfo.Magic != 0x6908 {
		return nil, errors.New("mcu-boot: invalid tlv image magic")
	}
	if b.header.ProtectedTLVSize != 0 && tlvInfo.Magic == 0x6908 {
		// TODO: parse protectedTLVs
		fmt.Println("Protected!!!")
	}
	b.seek(4)
	h, t, err := b.readTLVAreaInfo()
	if t != ImageTLVSHA256 {
		return nil, errors.New("mcu-boot: invalid tlv hash")
	}
	kh, t, err := b.readTLVAreaInfo()
	if err != nil {
		return nil, err
	}
	if t != ImageTLVKeyHash {
		return nil, errors.New("mcu-boot: invalid tlv key hash")
	}
	sig, _, err := b.readTLVAreaInfo()
	if err != nil {
		return nil, err
	}
	area := &TLVArea{ImageHash: h, KeyHash: kh, Signature: sig}
	return area, nil
}

func (b *MCUBoot) readTLVAreaInfo() ([]byte, TLVType, error) {
	it := make([]byte, 4)
	if _, err := b.r.ReadAt(it, b.offset); err != nil {
		return nil, -1, err
	}
	var tlv ImageTLV
	r := bytes.NewReader(it)
	err := binary.Read(r, binary.LittleEndian, &tlv)
	if err != nil {
		return nil, -1, err
	}
	b.seek(4)
	out := make([]byte, tlv.ItSize)
	if _, err := b.r.ReadAt(out, b.offset); err != nil {
		return nil, -1, err
	}
	b.seek(int64(tlv.ItSize))
	return out, TLVType(tlv.ItType), nil
}
