package index

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"
)

const (
	evictDebounce    = 500 * time.Millisecond
	retryBaseDelay   = 500 * time.Millisecond
	retryMaxAttempts = 5
)

var ErrNotFound = errors.New("cache not found")

type CacheIndex struct {
	caches map[int][]*BinaryCache
	mux    sync.RWMutex
	prios  []int // priorities in order

	// Keep track of evictions
	evictMux    sync.Mutex
	evictTimers map[string]*time.Timer
}

func NewCacheIndex() *CacheIndex {
	return &CacheIndex{
		caches:      make(map[int][]*BinaryCache),
		evictTimers: make(map[string]*time.Timer),
	}
}

func (cm *CacheIndex) Close() {
	cm.evictMux.Lock()
	defer cm.evictMux.Unlock()

	for _, timer := range cm.evictTimers {
		timer.Stop()
	}
}

func (cm *CacheIndex) Iter() iter.Seq[[]*BinaryCache] {
	return func(yield func([]*BinaryCache) bool) {
		cm.mux.RLock()
		defer cm.mux.RUnlock()

		for _, prio := range cm.prios {
			caches := cm.caches[prio]
			if !yield(caches) {
				return
			}
		}
	}
}

func (cm *CacheIndex) updateKeys() {
	// Store keys in sorted order so it doesn't need to be done again on every request
	cm.prios = slices.Sorted(maps.Keys(cm.caches))
}

func (cm *CacheIndex) Add(cache *BinaryCache) {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	updated := false

	caches := cm.caches[cache.Priority]
	if slices.IndexFunc(caches, func(other *BinaryCache) bool {
		return other.URL == cache.URL
	}) == -1 {
		caches = append(caches, cache)
		cm.caches[cache.Priority] = caches
		updated = true
	}

	if updated {
		cm.updateKeys()
	}
}

func (cm *CacheIndex) Remove(URL string) *BinaryCache {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	updated := false

	var cache *BinaryCache
	for prio, caches := range cm.caches {
		i := slices.IndexFunc(caches, func(other *BinaryCache) bool {
			return other.URL == URL
		})

		if i > -1 {
			cache = caches[i]
			caches = slices.Delete(caches, i, i+1)
			if len(caches) == 0 {
				delete(cm.caches, prio)
			} else {
				cm.caches[prio] = caches
			}
			updated = true
		}
	}

	if updated {
		cm.updateKeys()
	}

	return cache
}

func (cm *CacheIndex) Get(URL string) (*BinaryCache, error) {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	for _, caches := range cm.caches {
		i := slices.IndexFunc(caches, func(other *BinaryCache) bool {
			return other.URL == URL
		})

		if i > -1 {
			return caches[i], nil
		}
	}

	return nil, ErrNotFound
}

func (cm *CacheIndex) Evict(URL string, client *http.Client) {
	// First remove from cache
	cache := cm.Remove(URL)
	if cache == nil {
		return
	}

	cm.evictMux.Lock()
	defer cm.evictMux.Unlock()

	// Debounce: if evict is called multiple times within a short period only run the recovery routine once.
	_, alreadyPending := cm.evictTimers[URL]

	// If the host is alive add back
	if !alreadyPending {
		cm.evictTimers[URL] = time.AfterFunc(evictDebounce, func() {
			cm.evictMux.Lock()
			delete(cm.evictTimers, URL)
			cm.evictMux.Unlock()

			go func() {
				success := retryWithBackoff(func() error {
					_, err := cache.GetCacheInfo(context.Background(), client)
					return err
				}, retryBaseDelay, retryMaxAttempts)
				if success {
					slog.Info("reviving", "URL", cache.URL, "priority", cache.Priority)
					cm.Add(cache)
				}
			}()
		})
	}
}

func retryWithBackoff(fn func() error, baseDelay time.Duration, maxAttempts int) bool {
	delay := baseDelay
	for range maxAttempts {
		if fn() == nil {
			return true
		}

		time.Sleep(delay)
		delay *= 2
	}

	return false
}
