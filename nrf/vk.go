package nrf

import (
	"bytes"
	"errors"
)

var keys = map[string][]byte{
	"rsa2048":        {0x30, 0x82, 0x01, 0x0a},
	"rsa3072":        {0x30, 0x82, 0x01, 0x8a},
	"ecdsaP256":      {0x30, 0x59, 0x30, 0x13},
	"ecdsaP384":      {0x30, 0x76, 0x30, 0x10},
	"ed25519/x25519": {0x30, 0x2a, 0x30, 0x05},
}

func DumpVerifyingKey(b []byte) ([]byte, error) {
	r := bytes.NewReader(b)
	for suite, v := range keys {
		offset := findVerifyingKey(b, v)
		if offset == -1 {
			continue
		}
		size := getVerifyingKeySize(suite)
		key := make([]byte, size)
		_, err := r.ReadAt(key, int64(offset))
		if err != nil {
			return nil, err
		}
		return key, nil
	}
	return nil, errors.New("verify key not found")
}

func getVerifyingKeySize(suite string) int {
	switch suite {
	case "rsa2048":
		return 0x10e
	case "rsa3072":
		return 0x18e
	case "ecdsaP384":
		return 0x78
	case "ecdsaP256":
		return 0x5b
	case "ed25519/x25519":
		return 0x2c
	}
	return 0
}

func findVerifyingKey(b, magic []byte) int {
	offset := bytes.Index(b, magic)
	if offset == -1 {
		return -1
	}
	return offset
}
