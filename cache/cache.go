// Package cache
//
// Implement client side caching for client.Client. This is useful for burst-happy applications that will try to query
// a lot of the same information in small chunks.
//
// For example, an application could try to check the state of nodes, but do so using one request per node. This is
// obviously not ideal in larger cluster, where it would be more efficient to request the state of all nodes at once.
// Depending on the application, this may not be possible, however.
//
// This package contains ready-to-use client side caches with configurable duration and automatic invalidation under the
// assumption that modifications are made from the same client.
package cache

import (
	"sync"
	"time"

	"github.com/LINBIT/golinstor/client"
)

var (
	yes      = true
	cacheOpt = &client.ListOpts{Cached: &yes}
)

type Cache interface {
	apply(c *client.Client)
}

// WithCaches sets up the given caches on the client.Client.
func WithCaches(caches ...Cache) client.Option {
	return func(cl *client.Client) error {
		for _, ca := range caches {
			ca.apply(cl)
		}

		return nil
	}
}

type cache struct {
	mu         sync.Mutex
	lastUpdate time.Time
	cache      any
}

// Invalidate forcefully resets the cache.
// The next call to Get will always invoke the provided function.
func (c *cache) Invalidate() {
	c.mu.Lock()
	c.lastUpdate = time.Time{}
	c.cache = nil
	c.mu.Unlock()
}

// Get returns a cached response or the result of the provided update function.
//
// If the cache is current, it will return the last successful cached response.
// If the cache is outdated, it will run the provided function to retrieve a result. A successful response
// is cached for later use.
func (c *cache) Get(timeout time.Duration, updateFunc func() (any, error)) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	if timeout != 0 && c.lastUpdate.Add(timeout).Before(now) {
		result, err := updateFunc()
		if err != nil {
			return nil, err
		}

		c.cache = result
		c.lastUpdate = now
	}

	return c.cache, nil
}

// filterNodeAndPoolOpts filters generic items based on the provided client.ListOpts
// This tries to mimic the behaviour of LINSTOR when using the node and storage pool query parameters.
func filterNodeAndPoolOpts[T any](items []T, getNodeAndPoolNames func(*T) ([]string, []string), opts ...*client.ListOpts) []T {
	filterNames := make(map[string]struct{})
	filterPools := make(map[string]struct{})

	for _, o := range opts {
		for _, n := range o.Node {
			filterNames[n] = struct{}{}
		}

		for _, sp := range o.StoragePool {
			filterPools[sp] = struct{}{}
		}
	}

	var result []T

outer:
	for i := range items {
		nodes, pools := getNodeAndPoolNames(&items[i])

		if len(filterNames) > 0 {
			for _, nodeName := range nodes {
				if _, ok := filterNames[nodeName]; !ok {
					continue outer
				}
			}
		}

		if len(filterPools) > 0 {
			for _, poolName := range pools {
				if _, ok := filterPools[poolName]; !ok {
					continue outer
				}
			}
		}

		result = append(result, items[i])
	}

	return result
}
