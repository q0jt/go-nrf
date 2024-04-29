package nrf

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func DetectSDKVersion(r io.ReaderAt) (string, error) {
	sig, err := generateSignature(r)
	if err != nil {
		return "", err
	}
	enc := hex.EncodeToString(sig)
	s, err := readJson("nrf/signature.json")
	if err != nil {
		return "", err
	}
	for _, signature := range s.Signatures {
		for _, hash := range signature.Hashes {
			if enc == hash.Signature {
				fmt.Println("found signature:", hash.Signature)
				return signature.SdkVersion, nil
			}
		}
	}
	return "", errors.New("no SDK version detected")
}

func generateSignature(r io.ReaderAt) ([]byte, error) {
	out := make([]byte, 0x2710)
	_, err := r.ReadAt(out, 0x1000)
	if err != nil {
		return nil, err
	}
	return sha256Sum(out), nil
}

type SDKSignatures struct {
	Signatures []Signatures `json:"signatures"`
}

type Hash struct {
	SoftDevice string `json:"softDevice"`
	Signature  string `json:"signature"`
}
type Signatures struct {
	SdkVersion string `json:"sdkVersion"`
	Hashes     []Hash `json:"hashes"`
}

func readJson(name string) (*SDKSignatures, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var s SDKSignatures
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func writeJson(version string, h []Hash) error {
	s, err := readJson("nrf/signature.json")
	if err != nil {
		return err
	}
	var ss Signatures
	ss.SdkVersion = version
	ss.Hashes = h
	s.Signatures = append(s.Signatures, ss)
	b, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("nrf/signature.json", b, 0644)
}

func GenSignatureFromSDK(name, version string) ([]Hash, error) {
	temp, err := os.MkdirTemp("", "github.com/q0jt/go-nrf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(temp)
	p := newSDKParser()
	if err := p.expandZipFile(name, temp); err != nil {
		return nil, err
	}
	var hashes []Hash
	for arch, hash := range p.signatures {
		hashes = append(hashes, Hash{
			SoftDevice: arch,
			Signature:  hash,
		})
	}
	if err := writeJson(version, hashes); err != nil {
		return nil, err
	}
	return hashes, nil
}

type sdkParser struct {
	signatures map[string]string
}

func newSDKParser() *sdkParser {
	return &sdkParser{
		signatures: map[string]string{},
	}
}

func (p *sdkParser) setSignatures(arch, hash string) {
	arch = strings.Split(arch, "_")[0]
	arch = strings.ToUpper(arch)
	p.signatures[arch] = hash
}

func (p *sdkParser) expandZipFile(name, temp string) error {
	r, err := zip.OpenReader(name)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, file := range r.File {
		fn := file.Name
		ext := filepath.Ext(fn)
		if ext != ".zip" {
			if isHexFile(ext, fn) {
				b, err := fs.ReadFile(r, fn)
				if err != nil {
					return err
				}
				bin, err := HexFileToBinary(b)
				if err != nil {
					return err
				}
				hash, err := genSignatureFromSDK(bin)
				if err != nil {
					return err
				}
				p.setSignatures(fn, hex.EncodeToString(hash))
			}
			continue
		}
		b, err := fs.ReadFile(r, fn)
		if err != nil {
			return err
		}
		if err := writeTmpFile(temp, fn, b); err != nil {
			return err
		}
		if err := p.expandZipFile(filepath.Join(temp, fn), temp); err != nil {
			return err
		}
	}
	return nil
}

func isHexFile(ext, name string) bool {
	isHex := ext == ".hex"
	return isHex && !strings.Contains(name, "nRF5_SDK")
}

func genSignatureFromSDK(b []byte) ([]byte, error) {
	r := bytes.NewReader(b)
	return generateSignature(r)
}
