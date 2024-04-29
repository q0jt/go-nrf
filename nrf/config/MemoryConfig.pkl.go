// Code generated from Pkl module `MemoryConfig`. DO NOT EDIT.
package config

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"github.com/q0jt/go-nrf/nrf/config/arch"
)

// nRF Memory layout Data
type MemoryConfig struct {
	// nRF Architecture, MemoryLayout
	Layouts map[arch.Arch]*MemoryLayout `pkl:"layouts"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a MemoryConfig
func LoadFromPath(ctx context.Context, path string) (ret *MemoryConfig, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a MemoryConfig
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (*MemoryConfig, error) {
	var ret MemoryConfig
	if err := evaluator.EvaluateModule(ctx, source, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}
