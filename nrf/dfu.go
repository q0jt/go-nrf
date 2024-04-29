package nrf

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/q0jt/go-nrf/nrf/dfu"
	"google.golang.org/protobuf/proto"
)

const manifestFileName = "manifest.json"

func OpenDfuFile(name string) (*DfuInfo, error) {
	temp, err := os.MkdirTemp("", "github.com/q0jt/go-nrf/nrf")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)
	if err := openDfuFile(name, temp); err != nil {
		return nil, err
	}
	df := os.DirFS(temp)
	f, err := df.Open(manifestFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := parseManifest(f)
	if err != nil {
		return nil, err
	}
	pf, err := df.Open(m.Manifest.App.DatFile)
	if err != nil {
		panic(err)
	}
	packet, err := readPacket(pf)
	if err != nil {
		return nil, err
	}
	return parsePacket(packet)
}

func openDfuFile(name, dir string) error {
	r, err := zip.OpenReader(name)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		fn := filepath.Base(f.Name)
		if strings.HasPrefix(fn, ".") || f.FileInfo().IsDir() {
			continue
		}
		b, err := fs.ReadFile(r, f.Name)
		if err != nil {
			return err
		}
		if err := writeTmpFile(dir, fn, b); err != nil {
			return err
		}
	}
	return nil
}

func writeTmpFile(dir, name string, b []byte) error {
	path := filepath.Join(dir, name)
	return os.WriteFile(path, b, 0666)
}

type DfuContents struct {
	Manifest Manifest `json:"manifest"`
}

type Manifest struct {
	App Application `json:"application"`
}

type Application struct {
	BinFile string `json:"bin_file"`
	DatFile string `json:"dat_file"`
}

func parseManifest(f fs.File) (*DfuContents, error) {
	var contents DfuContents
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&contents); err != nil {
		return nil, err
	}
	return &contents, nil
}

func readPacket(f fs.File) (*dfu.Packet, error) {
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return unmarshalPacket(b)
}

// UnmarshalDfuFile parses the dat file and returns an init packet.
func UnmarshalDfuFile(name string) (*dfu.Packet, error) {
	if filepath.Ext(name) != ".dat" {
		return nil, errors.New("invalid dfu file")
	}
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return unmarshalPacket(b)
}

func unmarshalPacket(b []byte) (*dfu.Packet, error) {
	var packet dfu.Packet
	if err := proto.Unmarshal(b, &packet); err != nil {
		return nil, err
	}
	return &packet, nil
}

type DfuInfo struct {
	appHash []byte
	sig     []byte
	cmd     []byte
}

var NoFirmwareSigned = errors.New("firmware is not signed")

func parsePacket(packet *dfu.Packet) (*DfuInfo, error) {
	if packet == nil {
		return nil, errors.New("invalid packet data")
	}
	sc := packet.SignedCommand
	if sc == nil {
		return nil, NoFirmwareSigned
	}
	cmd := sc.Command.Init
	v, err := proto.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	hash := cmd.Hash
	if hash.HashType.String() != "SHA256" {
		return nil, errors.New("no impl hash type")
	}
	slices.Reverse(hash.Hash)
	sig := sc.Signature
	//　Public keys are used in little-endian and need to be converted back to big-endian.
	slices.Reverse(sig[:0x20])
	slices.Reverse(sig[0x20:])

	return &DfuInfo{
		sig: sig, appHash: hash.Hash, cmd: v}, nil
}

func (d *DfuInfo) String() string {
	return fmt.Sprintf("- appHash: %x\n- signature: %x\n- cmd: %x",
		d.appHash, d.sig, d.cmd)
}

// Signature returns the signature contained in the init packet
func (d *DfuInfo) Signature() []byte {
	return d.sig
}

// Verify verifies the signature of OTA dfu files
func (d *DfuInfo) Verify(key []byte) error {
	return VerifySignature(key, d.cmd, d.sig)
}
