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

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

type ResourceDefinitionService struct {
	client *Client
}

// copy & paste from generated code

type ResourceDefinition struct {
	Name string `json:"name,omitempty"`
	// External name can be used to have native resource names. If you need to store a non Linstor compatible resource name use this field and Linstor will generate a compatible name.
	ExternalName string `json:"external_name,omitempty"`
	// A string to string property map.
	Props     map[string]string         `json:"props,omitempty"`
	Flags     []string                  `json:"flags,omitempty"`
	LayerData []ResourceDefinitionLayer `json:"layer_data,omitempty"`
}

type ResourceDefinitionCreate struct {
	// drbd port for resources
	DrbdPort int32 `json:"drbd_port,omitempty"`
	// drbd resource secret
	DrbdSecret         string             `json:"drbd_secret,omitempty"`
	DrbdTransportType  string             `json:"drbd_transport_type,omitempty"`
	ResourceDefinition ResourceDefinition `json:"resource_definition"`
}

type ResourceDefinitionLayer struct {
	Type LayerType                        `json:"type,omitempty"`
	Data OneOfDrbdResourceDefinitionLayer `json:"data,omitempty"`
}

type DrbdResourceDefinitionLayer struct {
	ResourceNameSuffix string `json:"resource_name_suffix,omitempty"`
	PeerSlots          int32  `json:"peer_slots,omitempty"`
	AlStripes          int64  `json:"al_stripes,omitempty"`
	// used drbd port for this resource
	Port          int32  `json:"port,omitempty"`
	TransportType string `json:"transport_type,omitempty"`
	// drbd resource secret
	Secret string `json:"secret,omitempty"`
	Down   bool   `json:"down,omitempty"`
}

type LayerType string

// List of LayerType
const (
	DRBD    LayerType = "DRBD"
	LUKS    LayerType = "LUKS"
	STORAGE LayerType = "STORAGE"
)

type VolumeDefinitionCreate struct {
	VolumeDefinition VolumeDefinition `json:"volume_definition"`
	DrbdMinorNumber  int32            `json:"drbd_minor_number,omitempty"`
}

type VolumeDefinition struct {
	VolumeNumber int32 `json:"volume_number,omitempty"`
	// Size of the volume in Kibi.
	SizeKib uint64 `json:"size_kib"`
	// A string to string property map.
	Props     map[string]string       `json:"props,omitempty"`
	Flags     []string                `json:"flags,omitempty"`
	LayerData []VolumeDefinitionLayer `json:"layer_data,omitempty"`
}

type VolumeDefinitionLayer struct {
	Type LayerType                 `json:"type"`
	Data OneOfDrbdVolumeDefinition `json:"data,omitempty"`
}

type DrbdVolumeDefinition struct {
	ResourceNameSuffix string `json:"resource_name_suffix,omitempty"`
	VolumeNumber       int32  `json:"volume_number,omitempty"`
	MinorNumber        int32  `json:"minor_number,omitempty"`
}

// custom code
type resourceDefinitionLayerIn struct {
	Type LayerType       `json:"type,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

func (rd *ResourceDefinitionLayer) UnmarshalJSON(b []byte) error {
	var rdIn resourceDefinitionLayerIn
	if err := json.Unmarshal(b, &rdIn); err != nil {
		return err
	}

	rd.Type = rdIn.Type
	switch rd.Type {
	case DRBD:
		dst := new(DrbdResourceDefinitionLayer)
		if err := json.Unmarshal(rdIn.Data, &dst); err != nil {
			return err
		}
		rd.Data = dst
	case LUKS, STORAGE: // valid types, but do not set data
	default:
		return fmt.Errorf("'%+v' is not a valid type to Unmarshal", rd.Type)
	}

	return nil
}

type OneOfDrbdResourceDefinitionLayer interface {
	isOneOfDrbdResourceDefinitionLayer()
}

func (d *DrbdResourceDefinitionLayer) isOneOfDrbdResourceDefinitionLayer() {}

type volumeDefinitionLayerIn struct {
	Type LayerType       `json:"type,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

func (vd *VolumeDefinitionLayer) UnmarshalJSON(b []byte) error {
	var vdIn volumeDefinitionLayerIn
	if err := json.Unmarshal(b, &vdIn); err != nil {
		return err
	}

	vd.Type = vdIn.Type
	switch vd.Type {
	case DRBD:
		dst := new(DrbdVolumeDefinition)
		if err := json.Unmarshal(vdIn.Data, &dst); err != nil {
			return err
		}
		vd.Data = dst
	case LUKS, STORAGE: // valid types, but do not set data
	default:
		return fmt.Errorf("'%+v' is not a valid type to Unmarshal", vd.Type)
	}

	return nil
}

type OneOfDrbdVolumeDefinition interface {
	isOneOfDrbdVolumeDefinition()
}

func (d *DrbdVolumeDefinition) isOneOfDrbdVolumeDefinition() {}

func (n *ResourceDefinitionService) GetAll(ctx context.Context, opts ...*ListOpts) ([]ResourceDefinition, error) {
	var resDefs []ResourceDefinition
	_, err := n.client.doGET(ctx, "/v1/resource-definitions", &resDefs, opts...)
	return resDefs, err
}

func (n *ResourceDefinitionService) Get(ctx context.Context, resDefName string, opts ...*ListOpts) (ResourceDefinition, error) {
	var resDef ResourceDefinition
	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resDefName, &resDef, opts...)
	return resDef, err
}

func (n *ResourceDefinitionService) Create(ctx context.Context, resDef ResourceDefinitionCreate) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions", resDef)
	return err
}

func (n *ResourceDefinitionService) Modify(ctx context.Context, resDefName string, props PropsModify) error {
	_, err := n.client.doPUT(ctx, "/v1/resource-definitions/"+resDefName, props)
	return err
}

func (n *ResourceDefinitionService) Delete(ctx context.Context, resDefName string) error {
	_, err := n.client.doDELETE(ctx, "/v1/resource-definitions/"+resDefName, nil)
	return err
}

func (n *ResourceDefinitionService) GetVolumeDefinitions(ctx context.Context, resDefName string, opts ...*ListOpts) ([]VolumeDefinition, error) {
	var volDefs []VolumeDefinition
	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resDefName+"/volume-definitions", &volDefs, opts...)
	return volDefs, err
}

func (n *ResourceDefinitionService) GetVolumeDefinition(ctx context.Context, resDefName string, opts ...*ListOpts) (VolumeDefinition, error) {
	var volDef VolumeDefinition
	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resDefName+"/volume-definitions", &volDef, opts...)
	return volDef, err
}

// only size required
func (n *ResourceDefinitionService) CreateVolumeDefinition(ctx context.Context, resDefName string, volDef VolumeDefinitionCreate) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions/"+resDefName+"/volume-definitions", volDef)
	return err
}

func (n *ResourceDefinitionService) ModifyVolumeDefinition(ctx context.Context, resDefName string, volNr int, props PropsModify) error {
	_, err := n.client.doPUT(ctx, "/v1/resource-definitions/"+resDefName+"/volume-definitions/"+strconv.Itoa(volNr), props)
	return err
}

func (n *ResourceDefinitionService) DeleteVolumeDefinition(ctx context.Context, resDefName string, volNr int) error {
	_, err := n.client.doDELETE(ctx, "/v1/resource-definitions/"+resDefName+"/volume-definitions/"+strconv.Itoa(volNr), nil)
	return err
}
