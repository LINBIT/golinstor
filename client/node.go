/*
* A REST client to interact with LINSTOR's REST API
* Copyright © 2019 LINBIT HA-Solutions GmbH
* Author: Roland Kammerer <roland.kammerer@linbit.com>
*
* This program is free software; you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation; either version 2 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program; if not, see <http://www.gnu.org/licenses/>.
 */

package client

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

func (n *NodeService) GetAll(ctx context.Context, opts ...*ListOpts) ([]Node, error) {
	var nodes []Node
	_, err := n.client.doGET(ctx, "/v1/nodes", &nodes, opts...)
	return nodes, err
}

func (n *NodeService) Get(ctx context.Context, nodeName string, opts ...*ListOpts) (Node, error) {
	var node Node
	_, err := n.client.doGET(ctx, "/v1/nodes/"+nodeName, &node, opts...)
	return node, err
}

func (n *NodeService) Create(ctx context.Context, node Node) error {
	_, err := n.client.doPOST(ctx, "/v1/nodes", node)
	return err
}

func (n *NodeService) Modify(ctx context.Context, nodeName string, props PropsModify) error {
	_, err := n.client.doPUT(ctx, "/v1/nodes/"+nodeName, props)
	return err
}

func (n *NodeService) Delete(ctx context.Context, nodeName string) error {
	_, err := n.client.doDELETE(ctx, "/v1/nodes/"+nodeName, nil)
	return err
}

func (n *NodeService) Lost(ctx context.Context, nodeName string) error {
	_, err := n.client.doDELETE(ctx, "/v1/nodes/"+nodeName+"/lost", nil)
	return err
}

func (n *NodeService) Reconnect(ctx context.Context, nodeName string) error {
	_, err := n.client.doPUT(ctx, "/v1/nodes/"+nodeName+"/reconnect", nil)
	return err
}

func (n *NodeService) GetNetInterfaces(ctx context.Context, nodeName string, opts ...*ListOpts) ([]NetInterface, error) {
	var nifs []NetInterface
	_, err := n.client.doGET(ctx, "/v1/nodes/"+nodeName+"/net-interfaces", &nifs, opts...)
	return nifs, err
}

func (n *NodeService) GetNetInterface(ctx context.Context, nodeName, nifName string, opts ...*ListOpts) (NetInterface, error) {
	var nif NetInterface
	_, err := n.client.doGET(ctx, "/v1/nodes/"+nodeName+"/net-interfaces/"+nifName, nif, opts...)
	return nif, err
}

func (n *NodeService) CreateNetInterface(ctx context.Context, nodeName string, nif NetInterface) error {
	_, err := n.client.doPOST(ctx, "/v1/nodes/"+nodeName+"/net-interfaces", nif)
	return err
}

func (n *NodeService) ModifyNetInterface(ctx context.Context, nodeName, nifName string, nif NetInterface) error {
	_, err := n.client.doPUT(ctx, "/v1/nodes/"+nodeName+"/net-interfaces/"+nifName, nif)
	return err
}

func (n *NodeService) DeleteNetinterface(ctx context.Context, nodeName, nifName string) error {
	_, err := n.client.doDELETE(ctx, "/v1/nodes/"+nodeName+"/net-interfaces/"+nifName, nil)
	return err
}

func (n *NodeService) GetStoragePools(ctx context.Context, nodeName string, opts ...*ListOpts) ([]StoragePool, error) {
	var sps []StoragePool
	_, err := n.client.doGET(ctx, "/v1/nodes/"+nodeName+"/storage-pools", &sps, opts...)
	return sps, err
}

func (n *NodeService) GetStoragePool(ctx context.Context, nodeName, spName string, opts ...*ListOpts) (StoragePool, error) {
	var sp StoragePool
	_, err := n.client.doGET(ctx, "/v1/nodes/"+nodeName+"/storage-pools/"+spName, &sp, opts...)
	return sp, err
}

func (n *NodeService) CreateStoragePool(ctx context.Context, nodeName string, sp StoragePool) error {
	_, err := n.client.doPOST(ctx, "/v1/nodes/"+nodeName+"/storage-pools", sp)
	return err
}

func (n *NodeService) ModifyStoragePool(ctx context.Context, nodeName, spName string, sp StoragePool) error {
	_, err := n.client.doPOST(ctx, "/v1/nodes/"+nodeName+"/storage-pools/"+spName, sp)
	return err
}

func (n *NodeService) DeleteStoragePool(ctx context.Context, nodeName, spName string) error {
	_, err := n.client.doDELETE(ctx, "/v1/nodes/"+nodeName+"/storage-pools/"+spName, nil)
	return err
}
