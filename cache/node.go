package cache

import (
	"context"
	"time"

	"github.com/LINBIT/golinstor/client"
)

// NodeCache caches respones from a client.NodeProvider.
type NodeCache struct {
	// Timeout for the cached responses.
	Timeout time.Duration

	nodeCache            cache
	storagePoolCache     cache
	physicalStorageCache cache
}

func (n *NodeCache) apply(c *client.Client) {
	c.Nodes = &nodeCacheProvider{
		cl:    c.Nodes,
		cache: n,
	}
}

type nodeCacheProvider struct {
	cl    client.NodeProvider
	cache *NodeCache
}

var _ client.NodeProvider = &nodeCacheProvider{}

func (n *nodeCacheProvider) GetAll(ctx context.Context, opts ...*client.ListOpts) ([]client.Node, error) {
	c, err := n.cache.nodeCache.Get(n.cache.Timeout, func() (any, error) {
		return n.cl.GetAll(ctx, cacheOpt)
	})
	if err != nil {
		return nil, err
	}

	return filterNodeAndPoolOpts(c.([]client.Node), opts...), nil
}

func (n *nodeCacheProvider) Get(ctx context.Context, nodeName string, opts ...*client.ListOpts) (client.Node, error) {
	nodes, err := n.GetAll(ctx, opts...)
	if err != nil {
		return client.Node{}, err
	}

	for i := range nodes {
		if nodes[i].Name == nodeName {
			return nodes[i], nil
		}
	}

	return client.Node{}, client.NotFoundError
}

func (n *nodeCacheProvider) Create(ctx context.Context, node client.Node) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.Create(ctx, node)
}

func (n *nodeCacheProvider) CreateEbsNode(ctx context.Context, name string, remoteName string) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.CreateEbsNode(ctx, name, remoteName)
}

func (n *nodeCacheProvider) Modify(ctx context.Context, nodeName string, props client.NodeModify) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.Modify(ctx, nodeName, props)
}

func (n *nodeCacheProvider) Delete(ctx context.Context, nodeName string) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.Delete(ctx, nodeName)
}

func (n *nodeCacheProvider) Lost(ctx context.Context, nodeName string) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.Lost(ctx, nodeName)
}

func (n *nodeCacheProvider) Reconnect(ctx context.Context, nodeName string) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.Reconnect(ctx, nodeName)
}

func (n *nodeCacheProvider) GetNetInterfaces(ctx context.Context, nodeName string, opts ...*client.ListOpts) ([]client.NetInterface, error) {
	node, err := n.Get(ctx, nodeName, opts...)
	if err != nil {
		return nil, err
	}

	return node.NetInterfaces, nil
}

func (n *nodeCacheProvider) GetNetInterface(ctx context.Context, nodeName, nifName string, opts ...*client.ListOpts) (client.NetInterface, error) {
	interfaces, err := n.GetNetInterfaces(ctx, nodeName, opts...)
	if err != nil {
		return client.NetInterface{}, err
	}

	for i := range interfaces {
		if interfaces[i].Name == nifName {
			return interfaces[i], nil
		}
	}

	return client.NetInterface{}, client.NotFoundError
}

func (n *nodeCacheProvider) CreateNetInterface(ctx context.Context, nodeName string, nif client.NetInterface) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.CreateNetInterface(ctx, nodeName, nif)
}

func (n *nodeCacheProvider) ModifyNetInterface(ctx context.Context, nodeName, nifName string, nif client.NetInterface) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.ModifyNetInterface(ctx, nodeName, nifName, nif)
}

func (n *nodeCacheProvider) DeleteNetinterface(ctx context.Context, nodeName, nifName string) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.DeleteNetinterface(ctx, nodeName, nifName)
}

func (n *nodeCacheProvider) Evict(ctx context.Context, nodeName string) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.Evict(ctx, nodeName)
}

func (n *nodeCacheProvider) Restore(ctx context.Context, nodeName string, restore client.NodeRestore) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.Restore(ctx, nodeName, restore)
}

func (n *nodeCacheProvider) Evacuate(ctx context.Context, nodeName string, evacuate *client.NodeEvacuate) error {
	defer n.cache.nodeCache.Invalidate()
	return n.cl.Evacuate(ctx, nodeName, evacuate)
}

func (n *nodeCacheProvider) GetStoragePoolView(ctx context.Context, opts ...*client.ListOpts) ([]client.StoragePool, error) {
	result, err := n.cache.storagePoolCache.Get(n.cache.Timeout, func() (any, error) {
		return n.cl.GetStoragePoolView(ctx, cacheOpt)
	})
	if err != nil {
		return nil, err
	}

	return filterNodeAndPoolOpts(result.([]client.StoragePool), opts...), nil
}

func (n *nodeCacheProvider) GetStoragePools(ctx context.Context, nodeName string, opts ...*client.ListOpts) ([]client.StoragePool, error) {
	allPools, err := n.GetStoragePoolView(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var pools []client.StoragePool

	for i := range allPools {
		if allPools[i].NodeName == nodeName {
			pools = append(pools, allPools[i])
		}
	}

	return pools, nil
}

func (n *nodeCacheProvider) GetStoragePool(ctx context.Context, nodeName, spName string, opts ...*client.ListOpts) (client.StoragePool, error) {
	pools, err := n.GetStoragePools(ctx, nodeName, opts...)
	if err != nil {
		return client.StoragePool{}, err
	}

	for i := range pools {
		if pools[i].StoragePoolName == spName {
			return pools[i], nil
		}
	}

	return client.StoragePool{}, client.NotFoundError
}

func (n *nodeCacheProvider) CreateStoragePool(ctx context.Context, nodeName string, sp client.StoragePool) error {
	defer n.cache.storagePoolCache.Invalidate()
	return n.cl.CreateStoragePool(ctx, nodeName, sp)
}

func (n *nodeCacheProvider) ModifyStoragePool(ctx context.Context, nodeName, spName string, genericProps client.GenericPropsModify) error {
	defer n.cache.storagePoolCache.Invalidate()
	return n.cl.ModifyStoragePool(ctx, nodeName, spName, genericProps)
}

func (n *nodeCacheProvider) DeleteStoragePool(ctx context.Context, nodeName, spName string) error {
	defer n.cache.storagePoolCache.Invalidate()
	return n.cl.DeleteStoragePool(ctx, nodeName, spName)
}

func (n *nodeCacheProvider) GetPhysicalStorageView(ctx context.Context, opts ...*client.ListOpts) ([]client.PhysicalStorageViewItem, error) {
	result, err := n.cache.physicalStorageCache.Get(n.cache.Timeout, func() (any, error) {
		return n.cl.GetPhysicalStorageView(ctx, cacheOpt)
	})
	if err != nil {
		return nil, err
	}

	return filterNodeAndPoolOpts(result.([]client.PhysicalStorageViewItem), opts...), nil
}

func (n *nodeCacheProvider) GetPhysicalStorage(ctx context.Context, nodeName string) ([]client.PhysicalStorageNode, error) {
	view, err := n.GetPhysicalStorageView(ctx)
	if err != nil {
		return nil, err
	}

	var result []client.PhysicalStorageNode
	for i := range view {
		for j := range view[i].Nodes[nodeName] {
			result = append(result, client.PhysicalStorageNode{
				Rotational:            view[i].Rotational,
				Size:                  view[i].Size,
				PhysicalStorageDevice: view[i].Nodes[nodeName][j],
			})
		}
	}

	return result, nil
}

func (n *nodeCacheProvider) CreateDevicePool(ctx context.Context, nodeName string, psc client.PhysicalStorageCreate) error {
	defer n.cache.physicalStorageCache.Invalidate()
	return n.cl.CreateDevicePool(ctx, nodeName, psc)
}

func (n *nodeCacheProvider) GetStoragePoolPropsInfos(ctx context.Context, nodeName string, opts ...*client.ListOpts) ([]client.PropsInfo, error) {
	return n.cl.GetStoragePoolPropsInfos(ctx, nodeName, opts...)
}

func (n *nodeCacheProvider) GetPropsInfos(ctx context.Context, opts ...*client.ListOpts) ([]client.PropsInfo, error) {
	return n.cl.GetPropsInfos(ctx, opts...)
}
