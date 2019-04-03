package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

type ResourceService struct {
	client *Client
}

// copy & paste from generated code

type Resource struct {
	Name     string `json:"name,omitempty"`
	NodeName string `json:"node_name,omitempty"`
	// A string to string property map.
	Props       map[string]string `json:"props,omitempty"`
	Flags       []string          `json:"flags,omitempty"`
	LayerObject ResourceLayer     `json:"layer_object,omitempty"`
	State       ResourceState     `json:"state,omitempty"`
}

type ResourceLayer struct {
	Children           []ResourceLayer `json:"children,omitempty"`
	ResourceNameSuffix string          `json:"resource_name_suffix,omitempty"`
	Type               LayerType       `json:"type,omitempty"`
	Drbd               DrbdResource    `json:"drbd,omitempty"`
	Luks               LuksResource    `json:"luks,omitempty"`
	Storage            StorageResource `json:"storage,omitempty"`
}

type DrbdResource struct {
	DrbdResourceDefinition DrbdResourceDefinitionLayer `json:"drbd_resource_definition,omitempty"`
	NodeId                 uint32                      `json:"node_id,omitempty"`
	PeerSlots              int32                       `json:"peer_slots,omitempty"`
	AlStripes              int32                       `json:"al_stripes,omitempty"`
	AlSize                 uint32                      `json:"al_size,omitempty"`
	Flags                  []string                    `json:"flags,omitempty"`
	DrbdVolumes            DrbdVolume                  `json:"drbd_volumes,omitempty"`
}

type DrbdVolume struct {
	DrbdVolumeDefinition DrbdVolumeDefinition `json:"drbd_volume_definition,omitempty"`
	// drbd device path e.g. '/dev/drbd1000'
	DevicePath string `json:"device_path,omitempty"`
	// block device used by drbd
	BackingDevice    string `json:"backing_device,omitempty"`
	MetaDisk         string `json:"meta_disk,omitempty"`
	AllocatedSizeKib int32  `json:"allocated_size_kib,omitempty"`
	UsableSizeKib    int32  `json:"usable_size_kib,omitempty"`
	// String describing current volume state
	DiskState string `json:"disk_state,omitempty"`
}

type LuksResource struct {
	StorageVolumes []LuksVolume `json:"storage_volumes,omitempty"`
}

type LuksVolume struct {
	VolumeNumber int32 `json:"volume_number,omitempty"`
	// block device path
	DevicePath string `json:"device_path,omitempty"`
	// block device used by luks
	BackingDevice    string `json:"backing_device,omitempty"`
	AllocatedSizeKib int32  `json:"allocated_size_kib,omitempty"`
	UsableSizeKib    int32  `json:"usable_size_kib,omitempty"`
	// String describing current volume state
	DiskState string `json:"disk_state,omitempty"`
	Opened    bool   `json:"opened,omitempty"`
}

type StorageResource struct {
	StorageVolumes []StorageVolume `json:"storage_volumes,omitempty"`
}

type StorageVolume struct {
	VolumeNumber int32 `json:"volume_number,omitempty"`
	// block device path
	DevicePath       string `json:"device_path,omitempty"`
	AllocatedSizeKib int32  `json:"allocated_size_kib,omitempty"`
	UsableSizeKib    int32  `json:"usable_size_kib,omitempty"`
	// String describing current volume state
	DiskState string `json:"disk_state,omitempty"`
}

type ResourceState struct {
	InUse bool `json:"in_use,omitempty"`
}

type Volume struct {
	VolumeNumber     int32        `json:"volume_number,omitempty"`
	StoragePool      string       `json:"storage_pool,omitempty"`
	ProviderKind     ProviderKind `json:"provider_kind,omitempty"`
	DevicePath       string       `json:"device_path,omitempty"`
	AllocatedSizeKib int32        `json:"allocated_size_kib,omitempty"`
	UsableSizeKib    int32        `json:"usable_size_kib,omitempty"`
	// A string to string property map.
	Props         map[string]string `json:"props,omitempty"`
	Flags         []string          `json:"flags,omitempty"`
	State         VolumeState       `json:"state,omitempty"`
	LayerDataList []VolumeLayer     `json:"layer_data_list,omitempty"`
}

type VolumeLayer struct {
	Type LayerType                              `json:"type,omitempty"`
	Data OneOfDrbdVolumeLuksVolumeStorageVolume `json:"data,omitempty"`
}

type VolumeState struct {
	DiskState string `json:"disk_state,omitempty"`
}

type AutoPlaceRequest struct {
	DisklessOnRemaining bool             `json:"diskless_on_remaining,omitempty"`
	SelectFilter        AutoSelectFilter `json:"select_filter,omitempty"`
	LayerList           []LayerType      `json:"layer_list,omitempty"`
}

type AutoSelectFilter struct {
	PlaceCount           int32    `json:"place_count,omitempty"`
	StoragePool          string   `json:"storage_pool,omitempty"`
	NotPlaceWithRsc      []string `json:"not_place_with_rsc,omitempty"`
	NotPlaceWithRscRegex string   `json:"not_place_with_rsc_regex,omitempty"`
	ReplicasOnSame       []string `json:"replicas_on_same,omitempty"`
	ReplicasOnDifferent  []string `json:"replicas_on_different,omitempty"`
}

type ResourceConnection struct {
	// source node of the connection
	NodeA string `json:"node_a,omitempty"`
	// target node of the connection
	NodeB string `json:"node_b,omitempty"`
	// A string to string property map.
	Props map[string]string `json:"props,omitempty"`
	Flags []string          `json:"flags,omitempty"`
	Port  int32             `json:"port,omitempty"`
}

type Snapshot struct {
	Name         string   `json:"name,omitempty"`
	ResourceName string   `json:"resource_name,omitempty"`
	Nodes        []string `json:"nodes,omitempty"`
	// A string to string property map.
	Props             map[string]string        `json:"props,omitempty"`
	Flags             []string                 `json:"flags,omitempty"`
	VolumeDefinitions SnapshotVolumeDefinition `json:"volume_definitions,omitempty"`
}

type SnapshotVolumeDefinition struct {
	VolumeNumber int32 `json:"volume_number,omitempty"`
	// Volume size in KiB
	SizeKib int32 `json:"size_kib,omitempty"`
}

type SnapshotRestore struct {
	// Resource where to restore the snapshot
	ToResource string `json:"to_resource"`
	// List of nodes where to place the restored snapshot
	Nodes []string `json:"nodes,omitempty"`
}

// custom code
type volumeLayerIn struct {
	Type LayerType       `json:"type,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

func (v *VolumeLayer) UnmarshalJSON(b []byte) error {
	var vIn volumeLayerIn
	if err := json.Unmarshal(b, &vIn); err != nil {
		return err
	}

	v.Type = vIn.Type
	switch v.Type {
	case DRBD:
		dst := new(DrbdVolume)
		if err := json.Unmarshal(vIn.Data, &dst); err != nil {
			return err
		}
		v.Data = dst
	case LUKS:
		dst := new(LuksVolume)
		if err := json.Unmarshal(vIn.Data, &dst); err != nil {
			return err
		}
		v.Data = dst
	case STORAGE:
		dst := new(StorageVolume)
		if err := json.Unmarshal(vIn.Data, &dst); err != nil {
			return err
		}
		v.Data = dst
	default:
		return fmt.Errorf("'%+v' is not a valid type to Unmarshal", v.Type)
	}

	return nil
}

type OneOfDrbdVolumeLuksVolumeStorageVolume interface {
	isOneOfDrbdVolumeLuksVolumeStorageVolume()
}

func (d *DrbdVolume) isOneOfDrbdVolumeLuksVolumeStorageVolume()    {}
func (d *LuksVolume) isOneOfDrbdVolumeLuksVolumeStorageVolume()    {}
func (d *StorageVolume) isOneOfDrbdVolumeLuksVolumeStorageVolume() {}

func (n *ResourceService) ListAll(ctx context.Context, resName string, opts ...*ListOpts) ([]Resource, error) {
	var reses []Resource
	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resName+"/resources", &reses, opts...)
	return reses, err
}

func (n *ResourceService) List(ctx context.Context, resName, nodeName string, opts ...*ListOpts) (Resource, error) {
	var res Resource
	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resName+"/resources/"+nodeName, &res, opts...)
	return res, err
}

func (n *ResourceService) Create(ctx context.Context, res Resource) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions/"+res.Name+"/resources/"+res.NodeName, res)
	return err
}

func (n *ResourceService) Modify(ctx context.Context, resName, nodeName string, props PropsModify) error {
	_, err := n.client.doPUT(ctx, "/v1/resource-definitions/"+resName+"/resources/"+nodeName, props)
	return err
}

func (n *ResourceService) Delete(ctx context.Context, resName, nodeName string) error {
	_, err := n.client.doDELETE(ctx, "/v1/resource-definitions/"+resName+"/resources/"+nodeName, nil)
	return err
}

func (n *ResourceService) ListVolumes(ctx context.Context, resName, nodeName string, opts ...*ListOpts) ([]Volume, error) {
	var vols []Volume

	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resName+"/resources/"+nodeName+"/volumes", &vols, opts...)
	return vols, err
}

func (n *ResourceService) ListVolume(ctx context.Context, resName, nodeName string, volNr int, opts ...*ListOpts) (Volume, error) {
	var vol Volume

	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resName+"/resources/"+nodeName+"/volumes/"+strconv.Itoa(volNr), &vol, opts...)
	return vol, err
}

func (n *ResourceService) Diskless(ctx context.Context, resName, nodeName, disklessPoolName string) error {
	u := "/v1/resource-definitions/" + resName + "/resources/" + nodeName + "/toggle-disk/diskless"
	if disklessPoolName != "" {
		u += "/" + disklessPoolName
	}
	_, err := n.client.doPUT(ctx, u, nil)
	return err
}

func (n *ResourceService) Diskful(ctx context.Context, resName, nodeName, storagePoolName string) error {
	u := "/v1/resource-definitions/" + resName + "/resources/" + nodeName + "/toggle-disk/diskful"
	if storagePoolName != "" {
		u += "/" + storagePoolName
	}
	_, err := n.client.doPUT(ctx, u, nil)
	return err
}

func (n *ResourceService) Migrate(ctx context.Context, resName, fromNodeName, toNodeName, storagePoolName string) error {
	u := "/v1/resource-definitions/" + resName + "/resources/" + toNodeName + "/migrate-disk/" + toNodeName
	if storagePoolName != "" {
		u += "/" + storagePoolName
	}
	_, err := n.client.doPUT(ctx, u, nil)
	return err
}

func (n *ResourceService) Autoplace(ctx context.Context, resName string, apr AutoPlaceRequest) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions/"+resName+"/autoplace", apr)
	return err
}

func (n *ResourceService) ListConnections(ctx context.Context, resName, nodeAName, nodeBName string, opts ...*ListOpts) ([]ResourceConnection, error) {
	var resConns []ResourceConnection

	u := "/v1/resource-definitions/" + resName + "/resources-connections"
	if nodeAName != "" && nodeBName != "" {
		u += fmt.Sprintf("/%s/%s", nodeAName, nodeBName)
	}

	_, err := n.client.doGET(ctx, u, &resConns, opts...)
	return resConns, err
}

func (n *ResourceService) ModifyConnection(ctx context.Context, resName, nodeAName, nodeBName string, props PropsModify) error {
	u := fmt.Sprintf("/v1/resource-definitions/%s/resource-connections/%s/%s", resName, nodeAName, nodeBName)
	_, err := n.client.doPUT(ctx, u, props)
	return err
}

func (n *ResourceService) ListSnapshots(ctx context.Context, resName string, opts ...*ListOpts) ([]Snapshot, error) {
	var snaps []Snapshot

	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resName+"/snapshots", &snaps, opts...)
	return snaps, err
}

func (n *ResourceService) ListSnapshot(ctx context.Context, resName, snapName string, opts ...*ListOpts) (Snapshot, error) {
	var snap Snapshot

	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resName+"/snapshots/"+snapName, &snap, opts...)
	return snap, err
}

func (n *ResourceService) CreateSnapshot(ctx context.Context, snapshot Snapshot) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions/"+snapshot.ResourceName+"/snapshots/"+snapshot.Name, snapshot)
	return err
}

func (n *ResourceService) DeleteSnapshot(ctx context.Context, resName, snapName string) error {
	_, err := n.client.doDELETE(ctx, "/v1/resource-definitions/"+resName+"/snapshots/"+snapName, nil)
	return err
}

func (n *ResourceService) RestoreSnapshot(ctx context.Context, origResName, snapName string, snapRestoreConf SnapshotRestore) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions/"+origResName+"/snapshot-restore-resource/"+snapName, snapRestoreConf)
	return err
}

func (n *ResourceService) RestoreVolumeDefinitionSnapshot(ctx context.Context, origResName, snapName string, snapRestoreConf SnapshotRestore) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions/"+origResName+"/snapshot-restore-resource/"+snapName, snapRestoreConf)
	return err
}

func (n *ResourceService) RollbackSnapshot(ctx context.Context, resName, snapName string) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions/"+resName+"/snapshot-rollback/"+snapName, nil)
	return err
}

func (n *ResourceService) ModifyDRBDProxy(ctx context.Context, resName string, props PropsModify) error {
	_, err := n.client.doPUT(ctx, "/v1/resource-definitions/"+resName+"/drbd-proxy", props)
	return err
}

func (n *ResourceService) enableDisableDRBDProxy(ctx context.Context, what, resName, nodeAName, nodeBName string) error {
	u := fmt.Sprintf("/v1/resource-definitions/%s/drbd-proxy/%s/%s/%s", resName, what, nodeAName, nodeBName)
	_, err := n.client.doPOST(ctx, u, nil)
	return err
}

func (n *ResourceService) EnableDRBDProxy(ctx context.Context, resName, nodeAName, nodeBName string) error {
	return n.enableDisableDRBDProxy(ctx, "enable", resName, nodeAName, nodeBName)
}

func (n *ResourceService) DisableDRBDProxy(ctx context.Context, resName, nodeAName, nodeBName string) error {
	return n.enableDisableDRBDProxy(ctx, "disable", resName, nodeAName, nodeBName)
}
