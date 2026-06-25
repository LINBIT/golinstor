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
	"fmt"
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
)

// ListOpts is a struct primarily used to define parameters used for pagination. It is also used for filtering (e.g., the /view/ calls)
type ListOpts struct {
	// Number of items to skip. Only used if Limit is a positive value
	Offset int `url:"offset"`
	// Maximum number of items to retrieve
	Limit int `url:"limit"`
	// Some responses can be cached controller side, such as snapshot lists
	Cached *bool `url:"cached,omitempty"`

	StoragePool []string `url:"storage_pools"`
	Resource    []string `url:"resources"`
	Node        []string `url:"nodes"`
	Prop        []string `url:"props"`
	Snapshots   []string `url:"snapshots"`
	Status      string   `url:"status,omitempty"`

	// Content is used in the files API. If true, fetching files will include the content.
	Content bool `url:"content,omitempty"`
}

func Optional[T any](objs ...*T) (*T, error) {
	switch len(objs) {
	case 0:
		return nil, nil
	case 1:
		return objs[0], nil
	default:
		// Safe to dereference 0 here, we are in the len(objs) > 1 case.
		return nil, fmt.Errorf("expected exactly zero or one arguments %T, got %d", objs[0], len(objs))
	}
}

func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	vs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = vs.Encode()
	return u.String(), nil
}
