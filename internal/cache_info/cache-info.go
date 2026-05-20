package cache_info

import (
	"bytes"
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/nix-community/go-nix/pkg/storepath"
)

const (
	URLSuffix = "nix-cache-info"
	DefaultPriority = 50
)

type CacheInfo struct {
	StoreDir      string
	WantMassQuery int
	Priority      int
}

func NewCacheInfo() *CacheInfo {
	return &CacheInfo{
		StoreDir: storepath.StoreDir,
		WantMassQuery: 1,
		Priority: DefaultPriority,
	}
}

func (ci *CacheInfo) String() string {
	return fmt.Sprintf("StoreDir: %s\nWantMassQuery: %d\nPriority: %d\n", ci.StoreDir, ci.WantMassQuery, ci.Priority)
}

func Unmarshal(input []byte, info *CacheInfo) error {
	scanner := bufio.NewScanner(bytes.NewReader(input))

	for scanner.Scan() {
		line := scanner.Text()

		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}

		switch key {
		case "StoreDir":
			info.StoreDir = value
		case "WantMassQuery":
			n, err := strconv.Atoi(value)
			if err != nil {
				return err
			}

			info.WantMassQuery = n
		case "Priority":
			n, err := strconv.Atoi(value)
			if err != nil {
				return err
			}

			info.Priority = n
		}
	}

	return nil
}
