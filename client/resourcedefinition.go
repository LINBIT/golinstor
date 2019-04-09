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
	default:
		return fmt.Errorf("'%+v' is not a valid type to Unmarshal", vd.Type)
	}

	return nil
}

type OneOfDrbdVolumeDefinition interface {
	isOneOfDrbdVolumeDefinition()
}

func (d *DrbdVolumeDefinition) isOneOfDrbdVolumeDefinition() {}

func (n *ResourceDefinitionService) ListAll(ctx context.Context, opts ...*ListOpts) ([]ResourceDefinition, error) {
	var resDefs []ResourceDefinition
	_, err := n.client.doGET(ctx, "/v1/resource-definitions", &resDefs, opts...)
	return resDefs, err
}

func (n *ResourceDefinitionService) List(ctx context.Context, resDefName string, opts ...*ListOpts) (ResourceDefinition, error) {
	var resDef ResourceDefinition
	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resDefName, &resDef, opts...)
	return resDef, err
}

func (n *ResourceDefinitionService) Create(ctx context.Context, resDef ResourceDefinition) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-definitions/", resDef)
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

func (n *ResourceDefinitionService) ListVolumeDefinitions(ctx context.Context, resDefName string, opts ...*ListOpts) ([]VolumeDefinition, error) {
	var volDefs []VolumeDefinition
	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resDefName+"/volume-definitions", &volDefs, opts...)
	return volDefs, err
}

func (n *ResourceDefinitionService) ListVolumeDefinition(ctx context.Context, resDefName string, opts ...*ListOpts) (VolumeDefinition, error) {
	var volDef VolumeDefinition
	_, err := n.client.doGET(ctx, "/v1/resource-definitions/"+resDefName+"/volume-definitions", &volDef, opts...)
	return volDef, err
}

// only size required
func (n *ResourceDefinitionService) CreateVolumeDefinition(ctx context.Context, resDefName string, volDef VolumeDefinition) error {
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
