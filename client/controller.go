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
	"net/url"
	"strconv"
	"time"
)

// copy & paste from generated code

// ControllerVersion represents version information of the LINSTOR controller
type ControllerVersion struct {
	Version        string `json:"version,omitempty"`
	GitHash        string `json:"git_hash,omitempty"`
	BuildTime      string `json:"build_time,omitempty"`
	RestApiVersion string `json:"rest_api_version,omitempty"`
}

// ErrorReport struct for ErrorReport
type ErrorReport struct {
	NodeName  string `json:"node_name,omitempty"`
	ErrorTime int64  `json:"error_time"`
	// Filename of the error report on the server.  Format is: ```ErrorReport-{instanceid}-{nodeid}-{sequencenumber}.log```
	Filename string `json:"filename,omitempty"`
	// Contains the full text of the error report file.
	Text string `json:"text,omitempty"`
	// Which module this error occurred.
	Module string `json:"module,omitempty"`
	// Linstor version this error report was created.
	Version string `json:"version,omitempty"`
	// Peer client that was involved.
	Peer string `json:"peer,omitempty"`
	// Exception that occurred
	Exception string `json:"exception,omitempty"`
	// Exception message
	ExceptionMessage string `json:"exception_message,omitempty"`
	// Origin file of the exception
	OriginFile string `json:"origin_file,omitempty"`
	// Origin method where the exception occurred
	OriginMethod string `json:"origin_method,omitempty"`
	// Origin line number
	OriginLine int32 `json:"origin_line,omitempty"`
}

type ErrorReportDelete struct {
	// timestamp in millis start date to delete
	Since int64 `json:"since,omitempty"`
	// timestamp in millis for the end date to delete
	To int64 `json:"to,omitempty"`
	// on which nodes to delete error-reports, if empty/null all nodes
	Nodes []string `json:"nodes,omitempty"`
	// delete all error reports with the given exception
	Exception string `json:"exception,omitempty"`
	// delete all error reports from the given version
	Version string `json:"version,omitempty"`
	// error report ids to delete
	Ids []string `json:"ids,omitempty"`
}

type PropsInfo struct {
	Info     string `json:"info,omitempty"`
	PropType string `json:"prop_type,omitempty"`
	Value    string `json:"value,omitempty"`
	Dflt     string `json:"dflt,omitempty"`
	Unit     string `json:"unit,omitempty"`
}

// ExternalFile is an external file which can be configured to be deployed by Linstor
type ExternalFile struct {
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
}

// custom code

// ControllerProvider acts as an abstraction for a ControllerService. It can be
// swapped out for another ControllerService implementation, for example for
// testing.
type ControllerProvider interface {
	// GetVersion queries version information for the controller.
	GetVersion(ctx context.Context, opts ...*ListOpts) (ControllerVersion, error)
	// GetConfig queries the configuration of a controller
	GetConfig(ctx context.Context, opts ...*ListOpts) (ControllerConfig, error)
	// Modify modifies the controller node and sets/deletes the given properties.
	Modify(ctx context.Context, props GenericPropsModify) error
	// GetProps gets all properties of a controller
	GetProps(ctx context.Context, opts ...*ListOpts) (ControllerProps, error)
	// DeleteProp deletes the given property/key from the controller object.
	DeleteProp(ctx context.Context, prop string) error
	// GetErrorReports returns all error reports. The Text field is not populated,
	// use GetErrorReport to get the text of an error report.
	GetErrorReports(ctx context.Context, opts ...*ListOpts) ([]ErrorReport, error)
	// DeleteErrorReports deletes error reports as specified by the ErrorReportDelete struct.
	DeleteErrorReports(ctx context.Context, del ErrorReportDelete) error
	// GetErrorReportsSince returns all error reports created after a certain point in time.
	GetErrorReportsSince(ctx context.Context, since time.Time, opts ...*ListOpts) ([]ErrorReport, error)
	// GetErrorReport returns a specific error report, including its text.
	GetErrorReport(ctx context.Context, id string, opts ...*ListOpts) (ErrorReport, error)
	// CreateSOSReport creates an SOS report in the log directory of the controller
	CreateSOSReport(ctx context.Context, opts ...*ListOpts) error
	// DownloadSOSReport request sos report to download
	DownloadSOSReport(ctx context.Context, opts ...*ListOpts) error
	GetSatelliteConfig(ctx context.Context, node string) (SatelliteConfig, error)
	ModifySatelliteConfig(ctx context.Context, node string, cfg SatelliteConfig) error
	// GetPropsInfos gets meta information about the properties that can be
	// set on a controller.
	GetPropsInfos(ctx context.Context, opts ...*ListOpts) ([]PropsInfo, error)
	// GetPropsInfosAll gets meta information about all properties that can
	// be set on a controller and all entities it contains (nodes, resource
	// definitions, ...).
	GetPropsInfosAll(ctx context.Context, opts ...*ListOpts) ([]PropsInfo, error)
	// GetExternalFile get a list of previously registered external files.
	// Content is intentionally skipped, use GetExternalFile to get it.
	GetExternalFiles(ctx context.Context) ([]ExternalFile, error)
	// GetExternalFile gets the requested external file including its content
	GetExternalFile(ctx context.Context, name string) (ExternalFile, error)
	// ModifyExternalFile registers or modifies a previously registered
	// external file
	ModifyExternalFile(ctx context.Context, name string, file ExternalFile) error
	// DeleteExternalFile deletes the given external file. This effectively
	// also deletes the file on all satellites
	DeleteExternalFile(ctx context.Context, name string) error
}

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

// GetPropsInfos gets meta information about the properties that can be set on
// a controller.
func (s *ControllerService) GetPropsInfos(ctx context.Context, opts ...*ListOpts) ([]PropsInfo, error) {
	var infos []PropsInfo
	_, err := s.client.doGET(ctx, "/v1/controller/properties/info", &infos, opts...)
	return infos, err
}

// GetPropsInfosAll gets meta information about all properties that can be set
// on a controller and all entities it contains (nodes, resource definitions, ...).
func (s *ControllerService) GetPropsInfosAll(ctx context.Context, opts ...*ListOpts) ([]PropsInfo, error) {
	var infos []PropsInfo
	_, err := s.client.doGET(ctx, "/v1/controller/properties/info/all", &infos, opts...)
	return infos, err
}

// GetErrorReports returns all error reports. The Text field is not populated,
// use GetErrorReport to get the text of an error report.
func (s *ControllerService) GetErrorReports(ctx context.Context, opts ...*ListOpts) ([]ErrorReport, error) {
	var reports []ErrorReport
	_, err := s.client.doGET(ctx, "/v1/error-reports", &reports, opts...)
	return reports, err
}

// DeleteErrorReports deletes error reports as specified by the ErrorReportDelete struct.
func (s *ControllerService) DeleteErrorReports(ctx context.Context, del ErrorReportDelete) error {
	_, err := s.client.doDELETE(ctx, "/v1/error-reports", del)
	return err
}

// unixMilli returns t formatted as milliseconds since Unix epoch
func unixMilli(t time.Time) int64 {
	return t.UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// GetErrorReportsSince returns all error reports created after a certain point in time.
func (s *ControllerService) GetErrorReportsSince(ctx context.Context, since time.Time, opts ...*ListOpts) ([]ErrorReport, error) {
	var reports []ErrorReport
	v := url.Values{}
	v.Set("since", strconv.FormatInt(unixMilli(since), 10))
	_, err := s.client.doGET(ctx, "/v1/error-reports/?"+v.Encode(), &reports, opts...)
	return reports, err
}

// GetErrorReport returns a specific error report, including its text.
func (s *ControllerService) GetErrorReport(ctx context.Context, id string, opts ...*ListOpts) (ErrorReport, error) {
	var report []ErrorReport
	_, err := s.client.doGET(ctx, "/v1/error-reports/"+id, &report, opts...)
	return report[0], err
}

// CreateSOSReport creates an SOS report in the log directory of the controller
func (s *ControllerService) CreateSOSReport(ctx context.Context, opts ...*ListOpts) error {
	_, err := s.client.doGET(ctx, "/v1/sos-report", nil, opts...)
	return err
}

// DownloadSOSReport request sos report to download
func (s *ControllerService) DownloadSOSReport(ctx context.Context, opts ...*ListOpts) error {
	_, err := s.client.doGET(ctx, "/v1/sos-report/download", nil, opts...)
	return err
}

func (s *ControllerService) GetSatelliteConfig(ctx context.Context, node string) (SatelliteConfig, error) {
	var cfg SatelliteConfig
	_, err := s.client.doGET(ctx, "/v1/nodes/"+node+"/config", &cfg)
	return cfg, err
}

func (s *ControllerService) ModifySatelliteConfig(ctx context.Context, node string, cfg SatelliteConfig) error {
	_, err := s.client.doPUT(ctx, "/v1/nodes/"+node+"/config", &cfg)
	return err
}

// GetExternalFile get a list of previously registered external files.  Content
// is intentionally skipped, use GetExternalFile to get it.
func (s *ControllerService) GetExternalFiles(ctx context.Context) ([]ExternalFile, error) {
	var files []ExternalFile
	_, err := s.client.doGET(ctx, "/v1/files", &files)
	return files, err
}

// GetExternalFile gets the requested external file including its content
func (s *ControllerService) GetExternalFile(ctx context.Context, name string) (ExternalFile, error) {
	var file ExternalFile
	_, err := s.client.doGET(ctx, "/v1/files/"+name, &file)
	return file, err
}

// ModifyExternalFile registers or modifies a previously registered external
// file
func (s *ControllerService) ModifyExternalFile(ctx context.Context, name string, file ExternalFile) error {
	_, err := s.client.doPUT(ctx, "/v1/files/"+name, file)
	return err
}

// DeleteExternalFile deletes the given external file. This effectively also
// deletes the file on all satellites
func (s *ControllerService) DeleteExternalFile(ctx context.Context, name string) error {
	_, err := s.client.doDELETE(ctx, "/v1/files/"+name, nil)
	return err
}
