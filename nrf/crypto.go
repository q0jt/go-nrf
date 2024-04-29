package nrf

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

func sha256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

func setP256PublicKey(key []byte) (*ecdsa.PublicKey, error) {
	size := len(key)
	if size == 0x41 && key[0] == 0x04 {
		key = key[1:]
		size--
	}
	if size != 0x40 {
		return nil, errors.New("invalid ecdsa public key size")
	}
	pk := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     big.NewInt(0).SetBytes(key[:0x20]),
		Y:     big.NewInt(0).SetBytes(key[0x20:]),
	}
	return pk, nil
}

func verifySignatureP256(key *ecdsa.PublicKey, msg, sig []byte) bool {
	h := sha256Sum(msg)
	r := big.NewInt(0).SetBytes(sig[:0x20])
	s := big.NewInt(0).SetBytes(sig[0x20:])
	return ecdsa.Verify(key, h, r, s)
}

func verifySignatureEd25519(key ed25519.PublicKey, msg, sig []byte) bool {
	return ed25519.Verify(key, msg, sig)
}

// VerifySignature verifies firmware signatures using ECDSA P-256 or Ed25519 public keys.
// pem format is also supported.
func VerifySignature(vk, cmd, sig []byte) error {
	if len(sig) != 0x40 {
		return errors.New("invalid signature size")
	}
	if strings.Contains(string(vk), "PUBLIC KEY") {
		key, err := loadKeyFromPem(vk)
		if err != nil {
			return err
		}
		var ok bool
		switch key.(type) {
		case *ecdsa.PublicKey:
			pk := key.(*ecdsa.PublicKey)
			ok = verifySignatureP256(pk, cmd, sig)
		case ed25519.PublicKey:
			pk := key.(ed25519.PublicKey)
			ok = verifySignatureEd25519(pk, cmd, sig)
		}
		return validSignature(ok)
	}
	return verifySignature(vk, cmd, sig)
}

func validSignature(ok bool) error {
	if ok {
		return nil
	}
	return errors.New("failed to firmware verification")
}

func verifySignature(key, msg, sig []byte) error {
	var ok bool
	switch len(key) {
	case 0x20:
		pk := ed25519.PublicKey(key)
		ok = verifySignatureEd25519(pk, msg, sig)
	case 0x40, 0x41:
		pk, err := setP256PublicKey(key)
		if err != nil {
			return err
		}
		ok = verifySignatureP256(pk, msg, sig)
		fmt.Println(ok)
	}
	return validSignature(ok)
}

func loadKeyFromPem(b []byte) (any, error) {
	block, _ := pem.Decode(b)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid block or block type")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}
