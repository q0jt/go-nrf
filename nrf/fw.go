package nrf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"

	"github.com/q0jt/go-nrf/nrf/config"
	"github.com/q0jt/go-nrf/nrf/config/arch"
)

var invalidCrc = errors.New("mismatch　crc32")

type Firmware struct {
	r    io.ReaderAt
	Attr *DfuSettingAttrs
	arch arch.Arch
	mem  *config.MemoryLayout
}

func OpenFirmware(name string) (*Firmware, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(b)
	attr, a, err := readSettingAttrs(r)
	if err != nil {
		return nil, err
	}
	fw := &Firmware{r: r, Attr: attr, arch: a}
	if err := fw.validArch(); err != nil {
		return nil, err
	}
	return fw, nil
}

func (f *Firmware) validArch() error {
	mem, err := getMemConfWithArch(f.arch)
	if err != nil {
		return err
	}
	addr := int64(mem.BootLoaderSettAddr)
	if _, err := f.extractApp(addr); err != nil {
		if !errors.Is(err, invalidCrc) {
			return err
		}
		if err := f.searchArch(addr); err != nil {
			return err
		}
	}
	return nil
}

func (f *Firmware) searchArch(off int64) error {
	chip, err := findAppAddrByAddr(f.arch, off)
	if err != nil {
		return err
	}
	for _, a := range chip {
		mem, err := getMemConfWithArch(a)
		if err != nil {
			return err
		}
		addr := int64(mem.BootLoaderSettAddr)
		if _, err := f.extractApp(addr); err != nil {
			if !errors.Is(err, invalidCrc) {
				return err
			}
			f.arch = a
			f.mem = mem
			return nil
		}
	}
	return errors.New("not found")
}

func (f *Firmware) Arch() string {
	return string(f.arch)
}

func (f *Firmware) ExtractApp() ([]byte, error) {
	addr := int64(f.mem.AppAreaAddr)
	return f.extractApp(addr)
}

func (f *Firmware) extractApp(off int64) ([]byte, error) {
	// The first 0x200 of the data contain CRC data.
	return f.Attr.extractApp(f.r, off)
}

func (f *Firmware) ExtractBootloader() ([]byte, error) {
	addr := int64(f.mem.BootLoaderAddr)
	out := make([]byte, 0x6000)
	if _, err := f.r.ReadAt(out, addr); err != nil {
		return nil, err
	}
	return out, nil
}

func readSettingAttrs(r io.ReaderAt) (*DfuSettingAttrs, arch.Arch, error) {
	conf, err := loadMemConfig()
	if err != nil {
		return nil, "", err
	}
	out := make([]byte, 0x5c)
	blank := bytes.Repeat([]byte{0xff}, 0x5c)
	for chip, layout := range conf.Layouts {
		addr := int64(layout.BootLoaderSettAddr)
		if _, err := r.ReadAt(out, addr); err != nil {
			return nil, "", err
		}
		if bytes.Equal(out, blank) {
			continue
		}
		attr, err := marshalAttr(out)
		if err != nil {
			continue
		}
		return attr, chip, nil
	}
	return nil, "", errors.New("no settings found")
}

func marshalAttr(b []byte) (*DfuSettingAttrs, error) {
	st, err := getDfuSetting(b)
	if err != nil {
		return nil, err
	}
	if err := st.checkCrc(b); err != nil {
		return nil, err
	}
	return st, nil
}

type BankImage struct {
	Size uint32
	Crc  uint32
	Code uint32
}

type DfuSettingAttrs struct {
	Crc         uint32
	Version     uint32
	AppVersion  uint32
	BlVersion   uint32
	BankLayout  uint32
	BankCurrent uint32
	Bank0Img    BankImage
	Bank1Img    BankImage
	WriteOffset uint32
	// soft device size
	SdSize  uint32
	Reserve [32]uint8
}

func getDfuSetting(b []byte) (*DfuSettingAttrs, error) {
	var attrs DfuSettingAttrs
	r := bytes.NewReader(b)
	err := binary.Read(r, binary.LittleEndian, &attrs)
	if err != nil {
		return nil, err
	}
	return &attrs, nil
}

func (a *DfuSettingAttrs) checkCrc(b []byte) error {
	if len(b) != 0x5c {
		return errors.New("invalid DFU settings size")
	}
	if a.Version != 1 && a.Version != 2 {
		return errors.New("invalid settings version")
	}
	// skip crc32 data
	crc := crc32.ChecksumIEEE(b[4:])
	if a.Crc != crc {
		return invalidCrc
	}
	return nil
}

func (a *DfuSettingAttrs) extractApp(r io.ReaderAt, off int64) ([]byte, error) {
	b := make([]byte, a.Bank0Img.Size)
	if _, err := r.ReadAt(b, off); err != nil {
		return nil, err
	}
	crc := crc32.ChecksumIEEE(b)
	if crc != a.Bank0Img.Crc {
		return nil, invalidCrc
	}
	return b, nil
}
