package nrf

import (
	"context"
	"errors"

	"github.com/q0jt/go-nrf/nrf/config"
	"github.com/q0jt/go-nrf/nrf/config/arch"
)

func loadMemConfig() (*config.MemoryConfig, error) {
	ctx := context.Background()
	mem, err := config.LoadFromPath(ctx, "./pkl/config.pkl")
	if err != nil {
		return nil, err
	}
	return mem, nil
}

func findAppAddrByAddr(origin arch.Arch, addr int64) ([]arch.Arch, error) {
	mem, err := loadMemConfig()
	if err != nil {
		return nil, err
	}
	var arches []arch.Arch
	for a, layout := range mem.Layouts {
		if a == origin {
			continue
		}
		if layout.BootLoaderSettAddr == uint32(addr) {
			arches = append(arches, a)
		}
	}
	return arches, nil
}

func getMemConfWithArch(arch arch.Arch) (*config.MemoryLayout, error) {
	mem, err := loadMemConfig()
	if err != nil {
		return nil, err
	}
	for chip, layout := range mem.Layouts {
		if chip != arch {
			continue
		}
		return layout, nil
	}
	return nil, errors.New("arch is not registered")
}
