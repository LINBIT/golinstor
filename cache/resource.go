package cache

import (
	"context"
	"time"

	"github.com/LINBIT/golinstor/client"
)

// ResourceCache caches responses from a client.ResourceProvider.
type ResourceCache struct {
	// Timeout for the cached responses.
	Timeout time.Duration

	resourceCache cache
	snapshotCache cache
}

// backupShim hooks into the backup provider and invalidates the resource cache on certain operations.
type backupShim struct {
	client.BackupProvider
	resourceCache *cache
	snapshotCache *cache
}

func (r *ResourceCache) apply(c *client.Client) {
	c.Resources = &resourceCacheProvider{
		cl:    c.Resources,
		cache: r,
	}
	c.Backup = backupShim{
		BackupProvider: c.Backup,
		resourceCache:  &r.resourceCache,
		snapshotCache:  &r.snapshotCache,
	}
}

type resourceCacheProvider struct {
	cl    client.ResourceProvider
	cache *ResourceCache
}

var _ client.ResourceProvider = &resourceCacheProvider{}
var _ client.BackupProvider = backupShim{}

func (r *resourceCacheProvider) GetResourceView(ctx context.Context, opts ...*client.ListOpts) ([]client.ResourceWithVolumes, error) {
	result, err := r.cache.resourceCache.Get(r.cache.Timeout, func() (any, error) {
		return r.cl.GetResourceView(ctx, cacheOpt)
	})
	if err != nil {
		return nil, err
	}

	return filterNodeAndPoolOpts(result.([]client.ResourceWithVolumes), opts...), nil
}

func (r *resourceCacheProvider) GetAll(ctx context.Context, resName string, opts ...*client.ListOpts) ([]client.Resource, error) {
	ress, err := r.GetResourceView(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var result []client.Resource

	for i := range ress {
		if ress[i].Name == resName {
			result = append(result, ress[i].Resource)
		}
	}

	return result, nil
}

func (r *resourceCacheProvider) Get(ctx context.Context, resName, nodeName string, opts ...*client.ListOpts) (client.Resource, error) {
	ress, err := r.GetAll(ctx, resName, opts...)
	if err != nil {
		return client.Resource{}, err
	}

	for i := range ress {
		if ress[i].NodeName == nodeName {
			return ress[i], nil
		}
	}
	return client.Resource{}, client.NotFoundError
}

func (r *resourceCacheProvider) GetVolumes(ctx context.Context, resName, nodeName string, opts ...*client.ListOpts) ([]client.Volume, error) {
	ress, err := r.GetResourceView(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var result []client.Volume
	for i := range ress {
		if ress[i].NodeName != nodeName || ress[i].Name != resName {
			continue
		}

		for j := range ress[i].Volumes {
			result = append(result, ress[i].Volumes[j])
		}
	}

	return result, nil
}

func (r *resourceCacheProvider) GetVolume(ctx context.Context, resName, nodeName string, volNr int, opts ...*client.ListOpts) (client.Volume, error) {
	volumes, err := r.GetVolumes(ctx, resName, nodeName, opts...)
	if err != nil {
		return client.Volume{}, err
	}

	for i := range volumes {
		if int(volumes[i].VolumeNumber) == volNr {
			return volumes[i], nil
		}
	}

	return client.Volume{}, client.NotFoundError
}

func (r *resourceCacheProvider) Create(ctx context.Context, res client.ResourceCreate) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Create(ctx, res)
}

func (r *resourceCacheProvider) Modify(ctx context.Context, resName, nodeName string, props client.GenericPropsModify) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Modify(ctx, resName, nodeName, props)
}

func (r *resourceCacheProvider) Delete(ctx context.Context, resName, nodeName string) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Delete(ctx, resName, nodeName)
}

func (r *resourceCacheProvider) ModifyVolume(ctx context.Context, resName, nodeName string, volNr int, props client.GenericPropsModify) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.ModifyVolume(ctx, resName, nodeName, volNr, props)
}

func (r *resourceCacheProvider) Diskless(ctx context.Context, resName, nodeName, disklessPoolName string) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Diskless(ctx, resName, nodeName, disklessPoolName)
}

func (r *resourceCacheProvider) Diskful(ctx context.Context, resName, nodeName, storagePoolName string, props *client.ToggleDiskDiskfulProps) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Diskful(ctx, resName, nodeName, storagePoolName, props)
}

func (r *resourceCacheProvider) Migrate(ctx context.Context, resName, fromNodeName, toNodeName, storagePoolName string) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Migrate(ctx, resName, fromNodeName, toNodeName, storagePoolName)
}

func (r *resourceCacheProvider) Autoplace(ctx context.Context, resName string, apr client.AutoPlaceRequest) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Autoplace(ctx, resName, apr)
}

func (r *resourceCacheProvider) Activate(ctx context.Context, resName string, nodeName string) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Activate(ctx, resName, nodeName)
}

func (r *resourceCacheProvider) Deactivate(ctx context.Context, resName string, nodeName string) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.Deactivate(ctx, resName, nodeName)
}

func (r *resourceCacheProvider) MakeAvailable(ctx context.Context, resName, nodeName string, makeAvailable client.ResourceMakeAvailable) error {
	r.cache.resourceCache.Invalidate()
	return r.cl.MakeAvailable(ctx, resName, nodeName, makeAvailable)
}

func (r *resourceCacheProvider) GetSnapshotView(ctx context.Context, opts ...*client.ListOpts) ([]client.Snapshot, error) {
	result, err := r.cache.snapshotCache.Get(r.cache.Timeout, func() (any, error) {
		return r.cl.GetSnapshotView(ctx, cacheOpt)
	})
	if err != nil {
		return nil, err
	}

	return filterNodeAndPoolOpts(result.([]client.Snapshot), opts...), nil
}

func (r *resourceCacheProvider) GetSnapshots(ctx context.Context, resName string, opts ...*client.ListOpts) ([]client.Snapshot, error) {
	snaps, err := r.GetSnapshotView(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var result []client.Snapshot
	for i := range snaps {
		if snaps[i].ResourceName == resName {
			result = append(result, snaps[i])
		}
	}

	return result, nil
}

func (r *resourceCacheProvider) GetSnapshot(ctx context.Context, resName, snapName string, opts ...*client.ListOpts) (client.Snapshot, error) {
	snaps, err := r.GetSnapshots(ctx, resName, opts...)
	if err != nil {
		return client.Snapshot{}, err
	}

	for i := range snaps {
		if snaps[i].Name == snapName {
			return snaps[i], nil
		}
	}

	return client.Snapshot{}, client.NotFoundError
}

func (r *resourceCacheProvider) CreateSnapshot(ctx context.Context, snapshot client.Snapshot) error {
	r.cache.snapshotCache.Invalidate()
	return r.cl.CreateSnapshot(ctx, snapshot)
}

func (r *resourceCacheProvider) DeleteSnapshot(ctx context.Context, resName, snapName string, nodes ...string) error {
	r.cache.snapshotCache.Invalidate()
	return r.cl.DeleteSnapshot(ctx, resName, snapName, nodes...)
}

func (r *resourceCacheProvider) RestoreSnapshot(ctx context.Context, origResName, snapName string, snapRestoreConf client.SnapshotRestore) error {
	// This will create new resources, not touch snapshots
	r.cache.resourceCache.Invalidate()
	return r.cl.RestoreSnapshot(ctx, origResName, snapName, snapRestoreConf)
}

func (r *resourceCacheProvider) RestoreVolumeDefinitionSnapshot(ctx context.Context, origResName, snapName string, snapRestoreConf client.SnapshotRestore) error {
	// This will create new resources, not touch snapshots
	r.cache.resourceCache.Invalidate()
	return r.cl.RestoreVolumeDefinitionSnapshot(ctx, origResName, snapName, snapRestoreConf)
}

func (r *resourceCacheProvider) RollbackSnapshot(ctx context.Context, resName, snapName string) error {
	return r.cl.RollbackSnapshot(ctx, resName, snapName)
}

func (r *resourceCacheProvider) GetConnections(ctx context.Context, resName, nodeAName, nodeBName string, opts ...*client.ListOpts) ([]client.ResourceConnection, error) {
	return r.cl.GetConnections(ctx, resName, nodeAName, nodeBName, opts...)
}

func (r *resourceCacheProvider) ModifyConnection(ctx context.Context, resName, nodeAName, nodeBName string, props client.GenericPropsModify) error {
	return r.cl.ModifyConnection(ctx, resName, nodeAName, nodeBName, props)
}

func (r *resourceCacheProvider) EnableSnapshotShipping(ctx context.Context, resName string, ship client.SnapshotShipping) error {
	return r.cl.EnableSnapshotShipping(ctx, resName, ship)
}

func (r *resourceCacheProvider) ModifyDRBDProxy(ctx context.Context, resName string, props client.DrbdProxyModify) error {
	return r.cl.ModifyDRBDProxy(ctx, resName, props)
}

func (r *resourceCacheProvider) EnableDRBDProxy(ctx context.Context, resName, nodeAName, nodeBName string) error {
	return r.cl.EnableDRBDProxy(ctx, resName, nodeAName, nodeBName)
}

func (r *resourceCacheProvider) DisableDRBDProxy(ctx context.Context, resName, nodeAName, nodeBName string) error {
	return r.cl.DisableDRBDProxy(ctx, resName, nodeAName, nodeBName)
}

func (r *resourceCacheProvider) QueryMaxVolumeSize(ctx context.Context, filter client.AutoSelectFilter) (client.MaxVolumeSizes, error) {
	return r.cl.QueryMaxVolumeSize(ctx, filter)
}

func (r *resourceCacheProvider) GetSnapshotShippings(ctx context.Context, opts ...*client.ListOpts) ([]client.SnapshotShippingStatus, error) {
	return r.cl.GetSnapshotShippings(ctx, opts...)
}

func (r *resourceCacheProvider) GetPropsInfos(ctx context.Context, resName string, opts ...*client.ListOpts) ([]client.PropsInfo, error) {
	return r.cl.GetPropsInfos(ctx, resName, opts...)
}

func (r *resourceCacheProvider) GetVolumeDefinitionPropsInfos(ctx context.Context, resName string, opts ...*client.ListOpts) ([]client.PropsInfo, error) {
	return r.cl.GetVolumeDefinitionPropsInfos(ctx, resName, opts...)
}

func (r *resourceCacheProvider) GetVolumePropsInfos(ctx context.Context, resName, nodeName string, opts ...*client.ListOpts) ([]client.PropsInfo, error) {
	return r.cl.GetVolumePropsInfos(ctx, resName, nodeName, opts...)
}

func (r *resourceCacheProvider) GetConnectionPropsInfos(ctx context.Context, resName string, opts ...*client.ListOpts) ([]client.PropsInfo, error) {
	return r.cl.GetConnectionPropsInfos(ctx, resName, opts...)
}

func (b backupShim) Restore(ctx context.Context, remoteName string, request client.BackupRestoreRequest) error {
	b.snapshotCache.Invalidate()
	b.resourceCache.Invalidate()
	return b.BackupProvider.Restore(ctx, remoteName, request)
}

func (b backupShim) Create(ctx context.Context, remoteName string, request client.BackupCreate) (string, error) {
	b.snapshotCache.Invalidate()
	return b.BackupProvider.Create(ctx, remoteName, request)
}

func (b backupShim) Ship(ctx context.Context, remoteName string, request client.BackupShipRequest) (string, error) {
	b.snapshotCache.Invalidate()
	return b.BackupProvider.Ship(ctx, remoteName, request)
}
