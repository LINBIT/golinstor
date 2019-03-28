package linstor

import "context"

// copy & paste from generated code

type Node struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	Flags []string `json:"flags,omitempty"`
	// A string to string property map.
	Props         map[string]string `json:"props,omitempty"`
	NetInterfaces []NetInterface    `json:"net_interfaces,omitempty"`
	// Enum describing the current connection status.
	ConnectionStatus string `json:"connection_status,omitempty"`
}

type NetInterface struct {
	Name                    string `json:"name"`
	Address                 string `json:"address"`
	SatellitePort           int32  `json:"satellite_port,omitempty"`
	SatelliteEncryptionType string `json:"satellite_encryption_type,omitempty"`
}

type StoragePool struct {
	StoragePoolName string       `json:"storage_pool_name"`
	NodeName        string       `json:"node_name,omitempty"`
	ProviderKind    ProviderKind `json:"provider_kind"`
	// A string to string property map.
	Props map[string]string `json:"props,omitempty"`
	// read only map of static storage pool traits
	StaticTraits map[string]string `json:"static_traits,omitempty"`
	// Kibi - read only
	FreeCapacity int64 `json:"free_capacity,omitempty"`
	// Kibi - read only
	TotalCapacity int64 `json:"total_capacity,omitempty"`
	// read only
	FreeSpaceMgrName string `json:"free_space_mgr_name,omitempty"`
}

type ProviderKind string

// List of ProviderKind
const (
	DISKLESS            ProviderKind = "DISKLESS"
	LVM                 ProviderKind = "LVM"
	LVM_THIN            ProviderKind = "LVM_THIN"
	ZFS                 ProviderKind = "ZFS"
	ZFS_THIN            ProviderKind = "ZFS_THIN"
	SWORDFISH_TARGET    ProviderKind = "SWORDFISH_TARGET"
	SWORDFISH_INITIATOR ProviderKind = "SWORDFISH_INITIATOR"
)

// custom code

type NodeService struct {
	client *Client
}

func (n *NodeService) ListAll(ctx context.Context, opts *ListOpts) ([]Node, error) {
	var nodes []Node
	_, err := n.client.GET(ctx, "/v1/nodes", opts, &nodes)
	return nodes, err
}

func (n *NodeService) List(ctx context.Context, opts *ListOpts, nodeName string) (Node, error) {
	var node Node
	_, err := n.client.GET(ctx, "/v1/nodes/"+nodeName, nil, &node)
	return node, err
}

func (n *NodeService) Create(ctx context.Context, node Node) error {
	_, err := n.client.POST(ctx, "/v1/nodes", nil, node)
	return err
}

func (n *NodeService) Modify(ctx context.Context, nodeName string, props PropsModify) error {
	_, err := n.client.PUT(ctx, "/v1/nodes/"+nodeName, nil, props)
	return err
}

func (n *NodeService) Delete(ctx context.Context, nodeName string) error {
	_, err := n.client.DELETE(ctx, "/v1/nodes/"+nodeName, nil, nil)
	return err
}

func (n *NodeService) Lost(ctx context.Context, nodeName string) error {
	_, err := n.client.DELETE(ctx, "/v1/nodes/"+nodeName+"/lost", nil, nil)
	return err
}

func (n *NodeService) Reconnect(ctx context.Context, nodeName string) error {
	_, err := n.client.PUT(ctx, "/v1/nodes/"+nodeName+"/reconnect", nil, nil)
	return err
}

func (n *NodeService) ListNetInterfaces(ctx context.Context, opts *ListOpts, nodeName string) ([]NetInterface, error) {
	var nifs []NetInterface
	_, err := n.client.GET(ctx, "/v1/nodes/"+nodeName+"/net-interfaces", opts, &nifs)
	return nifs, err
}

func (n *NodeService) ListNetInterface(ctx context.Context, opts *ListOpts, nodeName, nifName string) (NetInterface, error) {
	var nif NetInterface
	_, err := n.client.GET(ctx, "/v1/nodes/"+nodeName+"/net-interfaces/"+nifName, opts, nif)
	return nif, err
}

func (n *NodeService) CreateNetInterface(ctx context.Context, nodeName string, nif NetInterface) error {
	_, err := n.client.POST(ctx, "/v1/nodes/"+nodeName+"/net-interfaces", nil, nif)
	return err
}

func (n *NodeService) ModifyNetInterface(ctx context.Context, nodeName, nifName string, nif NetInterface) error {
	_, err := n.client.PUT(ctx, "/v1/nodes/"+nodeName+"/net-interfaces/"+nifName, nil, nif)
	return err
}

func (n *NodeService) DeleteNetinterface(ctx context.Context, nodeName, nifName string) error {
	_, err := n.client.DELETE(ctx, "/v1/nodes/"+nodeName+"/net-interfaces/"+nifName, nil, nil)
	return err
}

func (n *NodeService) ListStoragePools(ctx context.Context, opts *ListOpts, nodeName string) ([]StoragePool, error) {
	var sps []StoragePool
	_, err := n.client.GET(ctx, "/v1/nodes/"+nodeName+"/storage-pools", opts, &sps)
	return sps, err
}

func (n *NodeService) ListStoragePool(ctx context.Context, opts *ListOpts, nodeName, spName string) (StoragePool, error) {
	var sp StoragePool
	_, err := n.client.GET(ctx, "/v1/nodes/"+nodeName+"/storage-pools/"+spName, opts, sp)
	return sp, err
}

func (n *NodeService) CreateStoragePool(ctx context.Context, nodeName string, sp StoragePool) error {
	_, err := n.client.POST(ctx, "/v1/nodes/"+nodeName+"/storage-pools", nil, sp)
	return err
}

func (n *NodeService) ModifyStoragePool(ctx context.Context, nodeName, spName string, sp StoragePool) error {
	_, err := n.client.POST(ctx, "/v1/nodes/"+nodeName+"/storage-pools/"+spName, nil, sp)
	return err
}

func (n *NodeService) DeleteStoragePool(ctx context.Context, nodeName, spName string) error {
	_, err := n.client.DELETE(ctx, "/v1/nodes/"+nodeName+"/storage-pools/"+spName, nil, nil)
	return err
}
