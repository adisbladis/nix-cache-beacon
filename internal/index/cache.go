package index

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"io"

	"github.com/nix-community/go-nix/pkg/narinfo"

	cache_info "github.com/adisbladis/nix-cache-beacon/internal/cache_info"
)

type BinaryCache struct {
	URL      string
	Priority int
}

func NewBinaryCache(URL string, priority int) *BinaryCache {
	return &BinaryCache{
		URL:      URL,
		Priority: priority,
	}
}

func (c *BinaryCache) GetCacheInfo(ctx context.Context, client *http.Client) (*cache_info.CacheInfo, error) {
	u, err := url.JoinPath(c.URL, cache_info.URLSuffix)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	cacheInfo := cache_info.NewCacheInfo()
	return cacheInfo, cache_info.Unmarshal(data, cacheInfo)
}

func (c *BinaryCache) GetNarInfo(ctx context.Context, path string, client *http.Client) (*narinfo.NarInfo, error) {
	target := c.URL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	ninfo, err := narinfo.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	ninfo.URL = c.URL + "/" + ninfo.URL

	return ninfo, nil
}
