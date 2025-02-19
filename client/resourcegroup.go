// A REST client to interact with LINSTOR's REST API
// Copyright (C) LINBIT HA-Solutions GmbH
// All Rights Reserved.
// Author: Roland Kammerer <roland.kammerer@linbit.com>
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package client

import (
	"context"
	"net/http"
	"strconv"
)

type ResourceGroup struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// A string to string property map.
	Props        map[string]string `json:"props,omitempty"`
	SelectFilter AutoSelectFilter  `json:"select_filter,omitempty"`
	// unique object id
	Uuid string `json:"uuid,omitempty"`
}

type ResourceGroupModify struct {
	Description string `json:"description,omitempty"`
	// A string to string property map.
	OverrideProps    map[string]string `json:"override_props,omitempty"`
	DeleteProps      []string          `json:"delete_props,omitempty"`
	DeleteNamespaces []string          `json:"delete_namespaces,omitempty"`
	SelectFilter     AutoSelectFilter  `json:"select_filter,omitempty"`
}

type ResourceGroupSpawn struct {
	// name of the resulting resource-definition
	ResourceDefinitionName string `json:"resource_definition_name"`
	// External name can be used to have native resource names. If you need to store a non Linstor compatible resource name use this field and Linstor will generate a compatible name.
	ResourceDefinitionExternalName string `json:"resource_definition_external_name,omitempty"`
	// sizes (in kib) of the resulting volume-definitions
	VolumeSizes  []int64          `json:"volume_sizes,omitempty"`
	SelectFilter AutoSelectFilter `json:"select_filter,omitempty"`
	// If false, the length of the vlm_sizes has to match the number of volume-groups or an error is returned.  If true and there are more vlm_sizes than volume-groups, the additional volume-definitions will simply have no pre-set properties (i.e. \"empty\" volume-definitions) If true and there are less vlm_sizes than volume-groups, the additional volume-groups won't be used.  If the count of vlm_sizes matches the number of volume-groups, this \"partial\" parameter has no effect.
	Partial bool `json:"partial,omitempty"`
	// If true, the spawn command will only create the resource-definition with the volume-definitions but will not perform an auto-place, even if it is configured.
	DefinitionsOnly bool `json:"definitions_only,omitempty"`
}

type ResourceGroupAdjust struct {
	SelectFilter *AutoSelectFilter `json:"select_filter,omitempty"`
}

type VolumeGroup struct {
	VolumeNumber int32 `json:"volume_number,omitempty"`
	// A string to string property map.
	Props map[string]string `json:"props,omitempty"`
	// unique object id
	Uuid  string   `json:"uuid,omitempty"`
	Flags []string `json:"flags,omitempty"`
}

type VolumeGroupModify struct {
	// A string to string property map.
	OverrideProps map[string]string `json:"override_props,omitempty"`
	// To add a flag just specify the flag name, to remove a flag prepend it with a '-'.  Flags:   * GROSS_SIZE
	Flags            []string `json:"flags,omitempty"`
	DeleteProps      []string `json:"delete_props,omitempty"`
	DeleteNamespaces []string `json:"delete_namespaces,omitempty"`
}

// QuerySizeInfoResponseSpaceInfo contains information returned from the QuerySizeInfo API call
type QuerySizeInfoResponseSpaceInfo struct {
	MaxVlmSizeInKib                 int64                      `json:"max_vlm_size_in_kib"`
	AvailableSizeInKib              *int64                     `json:"available_size_in_kib,omitempty"`
	CapacityInKib                   *int64                     `json:"capacity_in_kib,omitempty"`
	DefaultMaxOversubscriptionRatio *float64                   `json:"default_max_oversubscription_ratio,omitempty"`
	NextSpawnResult                 []QuerySizeInfoSpawnResult `json:"next_spawn_result,omitempty"`
}

// QuerySizeInfoSpawnResult describes the result of a potential spawn operation
type QuerySizeInfoSpawnResult struct {
	NodeName     string `json:"node_name"`
	StorPoolName string `json:"stor_pool_name"`
}

// QuerySizeInfoRequest is the request object for the QuerySizeInfo API call
type QuerySizeInfoRequest struct {
	SelectFilter *AutoSelectFilter `json:"select_filter,omitempty"`
}

type QuerySizeInfoResponse struct {
	SpaceInfo *QuerySizeInfoResponseSpaceInfo `json:"space_info,omitempty"`
	Reports   []ApiCallRc                     `json:"reports,omitempty"`
}

// custom code

// ResourceGroupProvider acts as an abstraction for a
// ResourceGroupService. It can be swapped out for another
// ResourceGroupService implementation, for example for testing.
type ResourceGroupProvider interface {
	// GetAll lists all resource-groups
	GetAll(ctx context.Context, opts ...*ListOpts) ([]ResourceGroup, error)
	// Get return information about a resource-defintion
	Get(ctx context.Context, resGrpName string, opts ...*ListOpts) (ResourceGroup, error)
	// Create adds a new resource-group
	Create(ctx context.Context, resGrp ResourceGroup) error
	// Modify allows to modify a resource-group
	Modify(ctx context.Context, resGrpName string, props ResourceGroupModify) error
	// Delete deletes a resource-group
	Delete(ctx context.Context, resGrpName string) error
	// Spawn creates a new resource-definition and auto-deploys if configured to do so
	Spawn(ctx context.Context, resGrpName string, resGrpSpwn ResourceGroupSpawn) error
	// GetVolumeGroups lists all volume-groups for a resource-group
	GetVolumeGroups(ctx context.Context, resGrpName string, opts ...*ListOpts) ([]VolumeGroup, error)
	// GetVolumeGroup lists a volume-group for a resource-group
	GetVolumeGroup(ctx context.Context, resGrpName string, volNr int, opts ...*ListOpts) (VolumeGroup, error)
	// Create adds a new volume-group to a resource-group
	CreateVolumeGroup(ctx context.Context, resGrpName string, volGrp VolumeGroup) error
	// Modify allows to modify a volume-group of a resource-group
	ModifyVolumeGroup(ctx context.Context, resGrpName string, volNr int, props VolumeGroupModify) error
	DeleteVolumeGroup(ctx context.Context, resGrpName string, volNr int) error
	// GetPropsInfos gets meta information about the properties that can be
	// set on a resource group.
	GetPropsInfos(ctx context.Context, opts ...*ListOpts) ([]PropsInfo, error)
	// GetVolumeGroupPropsInfos gets meta information about the properties
	// that can be set on a resource group.
	GetVolumeGroupPropsInfos(ctx context.Context, resGrpName string, opts ...*ListOpts) ([]PropsInfo, error)
	// Adjust all resource-definitions (calls autoplace for) of the given resource-group
	Adjust(ctx context.Context, resGrpName string, adjust ResourceGroupAdjust) error
	// AdjustAll adjusts all resource-definitions (calls autoplace) according to their associated resource group.
	AdjustAll(ctx context.Context, adjust ResourceGroupAdjust) error
	// QuerySizeInfo returns information about the space available in a resource group
	QuerySizeInfo(ctx context.Context, resGrpName string, req QuerySizeInfoRequest) (QuerySizeInfoResponse, error)
}

var _ ResourceGroupProvider = &ResourceGroupService{}

// ResourceGroupService is the service that deals with resource group related tasks.
type ResourceGroupService struct {
	client *Client
}

// GetAll lists all resource-groups
func (n *ResourceGroupService) GetAll(ctx context.Context, opts ...*ListOpts) ([]ResourceGroup, error) {
	var resGrps []ResourceGroup
	_, err := n.client.doGET(ctx, "/v1/resource-groups", &resGrps, opts...)
	return resGrps, err
}

// Get return information about a resource-defintion
func (n *ResourceGroupService) Get(ctx context.Context, resGrpName string, opts ...*ListOpts) (ResourceGroup, error) {
	var resGrp ResourceGroup
	_, err := n.client.doGET(ctx, "/v1/resource-groups/"+resGrpName, &resGrp, opts...)
	return resGrp, err
}

// Create adds a new resource-group
func (n *ResourceGroupService) Create(ctx context.Context, resGrp ResourceGroup) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-groups", resGrp)
	return err
}

// Modify allows to modify a resource-group
func (n *ResourceGroupService) Modify(ctx context.Context, resGrpName string, props ResourceGroupModify) error {
	_, err := n.client.doPUT(ctx, "/v1/resource-groups/"+resGrpName, props)
	return err
}

// Delete deletes a resource-group
func (n *ResourceGroupService) Delete(ctx context.Context, resGrpName string) error {
	_, err := n.client.doDELETE(ctx, "/v1/resource-groups/"+resGrpName, nil)
	return err
}

// Spawn creates a new resource-definition and auto-deploys if configured to do so
func (n *ResourceGroupService) Spawn(ctx context.Context, resGrpName string, resGrpSpwn ResourceGroupSpawn) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-groups/"+resGrpName+"/spawn", resGrpSpwn)
	return err
}

// GetVolumeGroups lists all volume-groups for a resource-group
func (n *ResourceGroupService) GetVolumeGroups(ctx context.Context, resGrpName string, opts ...*ListOpts) ([]VolumeGroup, error) {
	var volGrps []VolumeGroup
	_, err := n.client.doGET(ctx, "/v1/resource-groups/"+resGrpName+"/volume-groups", &volGrps, opts...)
	return volGrps, err
}

// GetVolumeGroup lists a volume-group for a resource-group
func (n *ResourceGroupService) GetVolumeGroup(ctx context.Context, resGrpName string, volNr int, opts ...*ListOpts) (VolumeGroup, error) {
	var volGrp VolumeGroup
	_, err := n.client.doGET(ctx, "/v1/resource-groups/"+resGrpName+"/volume-groups/"+strconv.Itoa(volNr), &volGrp, opts...)
	return volGrp, err
}

// Create adds a new volume-group to a resource-group
func (n *ResourceGroupService) CreateVolumeGroup(ctx context.Context, resGrpName string, volGrp VolumeGroup) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-groups/"+resGrpName+"/volume-groups", volGrp)
	return err
}

// Modify allows to modify a volume-group of a resource-group
func (n *ResourceGroupService) ModifyVolumeGroup(ctx context.Context, resGrpName string, volNr int, props VolumeGroupModify) error {
	_, err := n.client.doPUT(ctx, "/v1/resource-groups/"+resGrpName+"/volume-groups/"+strconv.Itoa(volNr), props)
	return err
}

func (n *ResourceGroupService) DeleteVolumeGroup(ctx context.Context, resGrpName string, volNr int) error {
	_, err := n.client.doDELETE(ctx, "/v1/resource-groups/"+resGrpName+"/volume-groups/"+strconv.Itoa(volNr), nil)
	return err
}

// GetPropsInfos gets meta information about the properties that can be set on
// a resource group.
func (n *ResourceGroupService) GetPropsInfos(ctx context.Context, opts ...*ListOpts) ([]PropsInfo, error) {
	var infos []PropsInfo
	_, err := n.client.doGET(ctx, "/v1/resource-groups/properties/info", &infos, opts...)
	return infos, err
}

// GetVolumeGroupPropsInfos gets meta information about the properties that can
// be set on a resource group.
func (n *ResourceGroupService) GetVolumeGroupPropsInfos(ctx context.Context, resGrpName string, opts ...*ListOpts) ([]PropsInfo, error) {
	var infos []PropsInfo
	_, err := n.client.doGET(ctx, "/v1/resource-groups/"+resGrpName+"/volume-groups/properties/info", &infos, opts...)
	return infos, err
}

// Adjust all resource-definitions (calls autoplace for) of the given resource-group
func (n *ResourceGroupService) Adjust(ctx context.Context, resGrpName string, adjust ResourceGroupAdjust) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-groups/"+resGrpName+"/adjust", adjust)
	return err
}

// AdjustAll adjusts all resource-definitions (calls autoplace) according to their associated resource group.
func (n *ResourceGroupService) AdjustAll(ctx context.Context, adjust ResourceGroupAdjust) error {
	_, err := n.client.doPOST(ctx, "/v1/resource-groups/adjustall", adjust)
	return err
}

// QuerySizeInfo returns information about the space available in a resource group
func (n *ResourceGroupService) QuerySizeInfo(ctx context.Context, resGrpName string, req QuerySizeInfoRequest) (QuerySizeInfoResponse, error) {
	var resp QuerySizeInfoResponse
	httpReq, err := n.client.newRequest(http.MethodPost, "/v1/resource-groups/"+resGrpName+"/query-size-info", req)
	if err != nil {
		return resp, err
	}
	_, err = n.client.doJSON(ctx, httpReq, &resp)
	return resp, err
}
