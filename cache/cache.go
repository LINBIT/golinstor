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
	"fmt"
	"strings"
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
func filterNodeAndPoolOpts[T Filterable](items []T, opts ...*client.ListOpts) []T {
	filterNodes := make(map[string]struct{})
	filterPools := make(map[string]struct{})
	var filterProps []string

	for _, o := range opts {
		for _, n := range o.Node {
			filterNodes[n] = struct{}{}
		}

		for _, sp := range o.StoragePool {
			filterPools[sp] = struct{}{}
		}

		filterProps = append(filterProps, o.Prop...)
	}

	var result []T

outer:
	for i := range items {
		if len(filterNodes) > 0 {
			if !anyMatches(filterNodes, nodes(&items[i])) {
				continue
			}
		}

		if len(filterPools) > 0 {
			if !anyMatches(filterPools, pools(&items[i])) {
				continue
			}
		}

		itemProps := props(&items[i])
		for _, filterProp := range filterProps {
			key, val, found := strings.Cut(filterProp, "=")
			itemVal, ok := itemProps[key]
			if !ok {
				continue outer
			}

			if found && val != itemVal {
				continue outer
			}
		}

		result = append(result, items[i])
	}

	return result
}

type Filterable interface {
	client.Node | client.StoragePool | client.ResourceWithVolumes | client.Snapshot | client.PhysicalStorageViewItem
}

func anyMatches(haystack map[string]struct{}, items []string) bool {
	for _, item := range items {
		if _, ok := haystack[item]; ok {
			return true
		}
	}

	return false
}

func nodes(item any) []string {
	switch item.(type) {
	case *client.Node:
		return []string{item.(*client.Node).Name}
	case *client.StoragePool:
		return []string{item.(*client.StoragePool).NodeName}
	case *client.ResourceWithVolumes:
		return []string{item.(*client.ResourceWithVolumes).NodeName}
	case *client.Snapshot:
		return item.(*client.Snapshot).Nodes
	case *client.PhysicalStorageViewItem:
		var result []string
		for k := range item.(*client.PhysicalStorageViewItem).Nodes {
			result = append(result, k)
		}
		return result
	default:
		panic(fmt.Sprintf("unsupported item type: %T", item))
	}
}

func pools(item any) []string {
	switch item.(type) {
	case *client.Node:
		return nil
	case *client.StoragePool:
		return []string{item.(*client.StoragePool).StoragePoolName}
	case *client.ResourceWithVolumes:
		var result []string
		for _, vol := range item.(*client.ResourceWithVolumes).Volumes {
			result = append(result, vol.StoragePoolName)
		}
		return result
	case *client.Snapshot:
		return nil
	case *client.PhysicalStorageViewItem:
		return nil

	default:
		panic(fmt.Sprintf("unsupported item type: %T", item))

	}
}

func props(item any) map[string]string {
	switch item.(type) {
	case *client.Node:
		return item.(*client.Node).Props
	case *client.StoragePool:
		return item.(*client.StoragePool).Props
	case *client.ResourceWithVolumes:
		return item.(*client.ResourceWithVolumes).Props
	case *client.Snapshot:
		return item.(*client.Snapshot).Props
	case *client.PhysicalStorageViewItem:
		return nil
	default:
		panic(fmt.Sprintf("unsupported item type: %T", item))
	}
}
