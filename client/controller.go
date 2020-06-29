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

import "context"

// copy & paste from generated code

// ControllerVersion represents version information of the LINSTOR controller
type ControllerVersion struct {
	Version        string `json:"version,omitempty"`
	GitHash        string `json:"git_hash,omitempty"`
	BuildTime      string `json:"build_time,omitempty"`
	RestApiVersion string `json:"rest_api_version,omitempty"`
}

// custom code

// ControllerService is the service that deals with the LINSTOR controller.
type ControllerService struct {
	client *Client
}

// GetVersion queries version information for the controller.
func (s *ControllerService) GetVersion(ctx context.Context, opts ...*ListOpts) (ControllerVersion, error) {
	var vers ControllerVersion
	_, err := s.client.doGET(ctx, "/v1/controller/version", &vers, opts...)
	return vers, err
}

// GetConfig queries the configuration of a controller
func (s *ControllerService) GetConfig(ctx context.Context, opts ...*ListOpts) (ControllerConfig, error) {
	var cfg ControllerConfig
	_, err := s.client.doGET(ctx, "/v1/controller/config", &cfg, opts...)
	return cfg, err
}

// Modify modifies the controller node and sets/deletes the given properties.
func (s *ControllerService) Modify(ctx context.Context, props GenericPropsModify) error {
	_, err := s.client.doPOST(ctx, "/v1/controller/properties", props)
	return err
}

type ControllerProps map[string]string

// GetProps gets all properties of a controller
func (s *ControllerService) GetProps(ctx context.Context, opts ...*ListOpts) (ControllerProps, error) {
	var props ControllerProps
	_, err := s.client.doGET(ctx, "/v1/controller/properties", &props, opts...)
	return props, err
}

// DeleteProp deletes the given property/key from the controller object.
func (s *ControllerService) DeleteProp(ctx context.Context, prop string) error {
	_, err := s.client.doDELETE(ctx, "/v1/controller/properties/"+prop, nil)
	return err
}
