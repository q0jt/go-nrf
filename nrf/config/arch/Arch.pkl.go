// Code generated from Pkl module `MemoryConfig`. DO NOT EDIT.
package arch

import (
	"encoding"
	"fmt"
)

type Arch string

const (
	NRF52805 Arch = "nRF52805"
	NRF52810 Arch = "nRF52810"
	NRF52811 Arch = "nRF52811"
	NRF52820 Arch = "nRF52820"
	NRF52832 Arch = "nRF52832"
	NRF52833 Arch = "nRF52833"
	NRF52840 Arch = "nRF52840"
)

// String returns the string representation of Arch
func (rcv Arch) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(Arch)

// UnmarshalBinary implements encoding.BinaryUnmarshaler for Arch.
func (rcv *Arch) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "nRF52805":
		*rcv = NRF52805
	case "nRF52810":
		*rcv = NRF52810
	case "nRF52811":
		*rcv = NRF52811
	case "nRF52820":
		*rcv = NRF52820
	case "nRF52832":
		*rcv = NRF52832
	case "nRF52833":
		*rcv = NRF52833
	case "nRF52840":
		*rcv = NRF52840
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid Arch`, str)
	}
	return nil
}
