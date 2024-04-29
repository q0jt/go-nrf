// Code generated from Pkl module `MemoryConfig`. DO NOT EDIT.
package config

type MemoryLayout struct {
	// Bootloader start address
	BootLoaderAddr uint32 `pkl:"bootLoaderAddr"`

	// Bootloader Settings start address
	BootLoaderSettAddr uint32 `pkl:"bootLoaderSettAddr"`

	// Application Area start address
	// Includes free space
	AppAreaAddr uint32 `pkl:"appAreaAddr"`
}
