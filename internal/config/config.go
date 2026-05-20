package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nix-community/go-nix/pkg/narinfo/signature"
	"github.com/nix-community/go-nix/pkg/storepath"

	cache_info "github.com/adisbladis/nix-cache-beacon/internal/cache_info"
)

const (
	DefaultPriority       = 20
	DefaultRequestTimeout = 2.5
)

type Config struct {
	RequestTimeout time.Duration
	Keys           map[string]signature.PublicKey
	CacheInfo      *cache_info.CacheInfo
}

func NewConfig() *Config {
	return &Config{
		RequestTimeout: time.Duration(DefaultRequestTimeout*1000) * time.Millisecond,
		Keys:           make(map[string]signature.PublicKey),
		CacheInfo: &cache_info.CacheInfo{
			StoreDir:      storepath.StoreDir,
			WantMassQuery: 1,
			Priority:      DefaultPriority,
		},
	}
}

type configJSON struct {
	// Request timeout in seconds
	Timeout *float32 `json:"timeout"`
	// Nix cache public keys
	Keys *[]string `json:"keys"`
	// Information served at cache-info endpoint
	CacheInfo *cache_info.CacheInfo `json:"cacheInfo"`
}

func Unmarshal(data []byte, cfg *Config) error {
	c := &configJSON{}

	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	if c.Timeout != nil {
		cfg.RequestTimeout = time.Duration(*c.Timeout*1000) * time.Millisecond
	}

	if c.Keys != nil {
		for _, spec := range *c.Keys {
			pub, err := signature.ParsePublicKey(spec)
			if err != nil {
				return err
			}

			_, ok := cfg.Keys[pub.Name]
			if ok {
				return fmt.Errorf("duplicate keys for name '%s'", pub.Name)
			}

			cfg.Keys[pub.Name] = pub
		}
	}

	if c.CacheInfo != nil {
		cfg.CacheInfo = c.CacheInfo
	}

	return nil
}
