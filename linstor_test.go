/*
* A helpful library to interact with Linstor
* Copyright © 2018 LINBIT USA LCC
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

package linstor

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/satori/go.uuid"
)

func TestNewResourceDeployment(t *testing.T) {
	// Test totally unconfigured deployment.
	res := NewResourceDeployment(ResourceDeploymentConfig{})
	if _, err := uuid.FromString(res.Name); err != nil {
		t.Errorf("Expected a UUID got %s", res.Name)
	}
	if res.AutoPlace != 1 {
		t.Errorf("Expected autoplace to be %d, got %d", 1, res.AutoPlace)
	}
	if res.SizeKiB != 4096 {
		t.Errorf("Expected SizeKiB to be %d, got %d", 4096, res.SizeKiB)
	}
	if res.StoragePool != "DfltStorPool" {
		t.Errorf("Expected StoragePool to be %s, got %s", "DfltStorPool", res.StoragePool)
	}
	if res.DisklessStoragePool != "DfltDisklessStorPool" {
		t.Errorf("Expected DisklessStoragePool to be %s, got %s", "DfltDisklessStorPool", res.DisklessStoragePool)
	}
	if res.Controllers != "" {
		t.Errorf("Expected Controllers to be %s, got %s", "", res.Controllers)
	}

	// Test regularly configured deployment autoplace.
	name := "Agamemnon"
	res = NewResourceDeployment(
		ResourceDeploymentConfig{
			Name:        name,
			AutoPlace:   5,
			SizeKiB:     10000,
			Controllers: "192.168.100.100:9001,172.5.1.30:5605,192.168.5.1:8080",
		},
	)
	if res.Name != name {
		t.Errorf("Expected %s to equal %s", res.Name, name)
	}
	if res.AutoPlace != 5 {
		t.Errorf("Expected autoplace to be %d, got %d", 5, res.AutoPlace)
	}
	if res.SizeKiB != 10000 {
		t.Errorf("Expected SizeKiB to be %d, got %d", 10000, res.SizeKiB)
	}
	if res.Controllers != "192.168.100.100:9001,172.5.1.30:5605,192.168.5.1:8080" {
		t.Errorf("Expected Controllers to be %s, got %s", "192.168.100.100:9001,172.5.1.30:5605,192.168.5.1:8080", res.Controllers)
	}

	// Test regularly configured deployment manual.
	nodes := []string{"host1", "host2"}
	res = NewResourceDeployment(
		ResourceDeploymentConfig{
			NodeList: nodes,
			SizeKiB:  10000,
		})
	if res.AutoPlace != 0 {
		t.Errorf("Expected autoplace to be %d, got %d", 0, res.AutoPlace)
	}
	if !reflect.DeepEqual(res.NodeList, nodes) {
		t.Errorf("Expected: %v, Got: %v", nodes, res.NodeList)
	}
}

func TestPrependOpts(t *testing.T) {
	var optsTests = []struct {
		in  []string
		out []string
	}{
		{[]string{"resource", "list"},
			[]string{"-m", "resource", "list"}},
		{[]string{"fee", "fie", "fo", "fum"},
			[]string{"-m", "fee", "fie", "fo", "fum"}},
	}

	r1 := NewResourceDeployment(ResourceDeploymentConfig{})

	for _, tt := range optsTests {
		result := r1.prependOpts(tt.in...)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("Called: prependOpts(%v), Expected: %v, Got: %v. %+v", tt.in, tt.out, result, r1)
		}
	}

	r2 := NewResourceDeployment(
		ResourceDeploymentConfig{
			Controllers: "192.168.100.100:9001,172.5.1.30:5605",
		},
	)

	var optsTestsControllers = []struct {
		in  []string
		out []string
	}{
		{[]string{"resource", "list"},
			[]string{"-m", "--controllers", r2.Controllers, "resource", "list"}},
		{[]string{"fee", "fie", "fo", "fum"},
			[]string{"-m", "--controllers", r2.Controllers, "fee", "fie", "fo", "fum"}},
	}

	for _, tt := range optsTestsControllers {
		result := r2.prependOpts(tt.in...)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("Called: prependOpts(%v), Expected: %v, Got: %v. %+v", tt.in, tt.out, result, r2)
		}
	}
}

func TestUniq(t *testing.T) {

	var uniqTests = []struct {
		in  []string
		out []string
	}{
		{[]string{"foo", "bar", "foo", "baz", "baz"},
			[]string{"foo", "bar", "baz"}},
		{[]string{"fee", "fie", "fo", "fum"},
			[]string{"fee", "fie", "fo", "fum"}},
	}

	for _, tt := range uniqTests {
		result := uniq(tt.in)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("Called: uniq(%v), Expected: %v, Got: %v", tt.in, tt.out, result)
		}
	}
}

func TestSubtract(t *testing.T) {

	var subTests = []struct {
		s1  []string
		s2  []string
		out []string
	}{
		{[]string{"foo", "bar", "foo", "baz", "baz"},
			[]string{"foo", "bar", "baz"},
			[]string{}},
		{[]string{"foo", "bar", "foo", "baz", "baz"},
			[]string{"fee", "fie", "fo", "fum"},
			[]string{"fee", "fie", "fo", "fum"}},
		{[]string{"cat", "dog", "monkey"},
			[]string{"pineapple", "peach", "dog", "mango"},
			[]string{"pineapple", "peach", "mango"}},
	}

	for _, tt := range subTests {
		result := subtract(tt.s1, tt.s2)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("Called: uniq(%v, %v), Expected: %v, Got: %v", tt.s1, tt.s2, tt.out, result)
		}
	}
}

func TestDoResExists(t *testing.T) {

	testOut := []byte("[\n  {\n    \"resources\": [\n      {\n        \"vlms\": [\n          {\n            \"vlm_dfn_uuid\": \"8000558f-9061-4256-b0fe-ec8d3b7fc051\", \n            \"stor_pool_uuid\": \"2d840084-f071-4173-a509-9a56d695a606\", \n            \"vlm_uuid\": \"ff382153-d47c-4a9f-a466-4ef4b3268656\", \n            \"vlm_nr\": 0, \n            \"stor_pool_name\": \"drbdpool\"\n          }, \n          {\n            \"vlm_dfn_uuid\": \"f763311c-3eb2-4f0e-ad55-361d4d026346\", \n            \"stor_pool_uuid\": \"2d840084-f071-4173-a509-9a56d695a606\", \n            \"vlm_uuid\": \"02a24f94-03a8-4918-8173-506b4cca369c\", \n            \"vlm_nr\": 1, \n            \"stor_pool_name\": \"drbdpool\"\n          }\n        ], \n        \"node_uuid\": \"1efa3fdf-129a-4018-a3f2-3d0f46571c5b\", \n        \"uuid\": \"d34e324a-54d8-4c79-adf5-3da33eca3371\", \n        \"node_name\": \"kubelet-a\", \n        \"props\": [\n          {\n            \"value\": \"drbdpool\", \n            \"key\": \"StorPoolName\"\n          }\n        ], \n        \"rsc_dfn_uuid\": \"83c4ce34-1bb5-48a7-a153-457f056223e1\", \n        \"name\": \"r00\"\n      }, \n      {\n        \"rsc_flags\": [\n          \"DELETE\"\n        ], \n        \"vlms\": [\n          {\n            \"vlm_nr\": 0, \n            \"stor_pool_name\": \"drbdpool\", \n            \"stor_pool_uuid\": \"ea3ddc0b-bc4f-4860-ae38-744c65cb3d18\", \n            \"vlm_uuid\": \"aebe934e-cda3-4b67-8d08-48c9444d12eb\", \n            \"vlm_dfn_uuid\": \"8000558f-9061-4256-b0fe-ec8d3b7fc051\", \n            \"vlm_flags\": [\n              \"DELETE\"\n            ]\n          }, \n          {\n            \"vlm_nr\": 1, \n            \"stor_pool_name\": \"drbdpool\", \n            \"stor_pool_uuid\": \"ea3ddc0b-bc4f-4860-ae38-744c65cb3d18\", \n            \"vlm_uuid\": \"36e679a3-d59a-4147-8f1e-299ae689c1d1\", \n            \"vlm_dfn_uuid\": \"f763311c-3eb2-4f0e-ad55-361d4d026346\", \n            \"vlm_flags\": [\n              \"DELETE\"\n            ]\n          }\n        ], \n        \"node_uuid\": \"c361253c-4e75-48d5-b00a-8a3df5eb3a69\", \n        \"uuid\": \"12a54b0c-5d76-4766-b973-26cfb1beaae4\", \n        \"node_name\": \"kubelet-b\", \n        \"props\": [\n          {\n            \"value\": \"drbdpool\", \n            \"key\": \"StorPoolName\"\n          }\n        ], \n        \"rsc_dfn_uuid\": \"83c4ce34-1bb5-48a7-a153-457f056223e1\", \n        \"name\": \"r00\"\n      }\n    ]\n  }\n]")
	list := resList{}
	if !json.Valid(testOut) {
		t.Errorf("invalid json")
	}
	if err := json.Unmarshal(testOut, &list); err != nil {
		t.Errorf("couldn't Unmarshal '%s' :%v", testOut, err)
	}

	var resInfoTests = []struct {
		resource string
		resInfo  resList
		out      bool
	}{
		{"r00", list, true},
		{"wrong-o", list, false},
	}

	for _, tt := range resInfoTests {
		ok, _ := doResExists(tt.resource, tt.resInfo)
		if ok != tt.out {
			t.Errorf("Called: doResExists(%q, %v), Expected: %v, Got: %v", tt.resource, tt.resInfo, tt.out, ok)
		}
	}
}

func TestDoCheckFSType(t *testing.T) {
	var blkidOutputStringTests = []struct {
		in  string
		out string
	}{
		{"ID_FS_UUID=15336bdb-4584-4c30-9719-754f5c4744e1\nID_FS_UUID_ENC=15336bdb-4584-4c30-9719-754f5c4744e1\nID_FS_TYPE=ext4\n", "ext4"},
		{"ID_FS_UUID=15336bdb-4584-4c30-9719-754f5c4744e1\nID_FS_UUID_ENC=15336bdb-4584-4c30-9719-754f5c4744e1\nID_FS_TYPE=xfs\n", "xfs"},
		{"\n", ""},
	}

	for _, tt := range blkidOutputStringTests {
		FSType, _ := doCheckFSType(tt.in)
		if FSType != tt.out {
			t.Errorf("Called: doCheckFSType(%q), Expected: %q, Got: %q", tt.in, tt.out, FSType)
		}
	}
}

func TestPopulateArgs(t *testing.T) {
	var populateArgsTests = []struct {
		in  FSUtil
		out []string
	}{
		{FSUtil{
			FSType:    "xfs",
			BlockSize: 4096,
		}, []string{"-b", "size=4096"}},
		{FSUtil{
			FSType:    "xfs",
			BlockSize: 2048,
		}, []string{"-b", "size=2048"}},
		{FSUtil{
			FSType:    "ext4",
			BlockSize: 2048,
		}, []string{"-b", "2048"}},
		{FSUtil{
			FSType: "xfs",
			Force:  true,
		}, []string{"-f"}},
		{FSUtil{
			FSType: "ext4",
			Force:  true,
		}, []string{"-F"}},
		{FSUtil{
			FSType:    "xfs",
			XFSDataSU: "128k",
		}, []string{"-d", "su=128k"}},
		{FSUtil{
			FSType:    "xfs",
			XFSDataSW: 1,
		}, []string{"-d", "sw=1"}},
		{FSUtil{
			FSType:    "xfs",
			XFSDataSU: "128k",
			XFSDataSW: 1,
			Force:     true,
			// Sadly, the order here matters based on how this []string is built in
			// function. It's a little bit fragile, but probably not worth messing
			// with right now.
		}, []string{"-f", "-d", "su=128k", "-d", "sw=1"}},
		{FSUtil{
			FSType:    "xfs",
			XFSLogDev: "/dev/example",
		}, []string{"-l", "logdev=/dev/example"}},
	}

	for _, tt := range populateArgsTests {

		tt.in.populateArgs()

		if !reflect.DeepEqual(tt.in.args, tt.out) {
			t.Errorf("Expected: %v Got: %v", tt.out, tt.in.args)
		}

	}

}

func TestDoIsClient(t *testing.T) {

	out, err := ioutil.ReadFile("test_json/mixed_diskless.json")
	if err != nil {
		t.Error(err)
	}

	list := resList{}
	if err := json.Unmarshal(out, &list); err != nil {
	}

	var isClientTests = []struct {
		resource string
		node     string
		l        resList
		out      bool
	}{
		{"default-william", "kubelet-a", list, false},
		{"default-william", "kubelet-x", list, false},
		{"default-william", "kubelet-c", list, true},
	}

	for _, tt := range isClientTests {
		r := NewResourceDeployment(ResourceDeploymentConfig{Name: tt.resource})

		ok := r.doIsClient(tt.l, tt.node)

		if tt.out != ok {
			t.Errorf("Expected: %v on %s Got: %v", tt.out, tt.node, ok)
		}
	}
}

func TestDoResOnNode(t *testing.T) {

	out, err := ioutil.ReadFile("test_json/mixed_diskless.json")
	if err != nil {
		t.Error(err)
	}

	list := resList{}
	if err := json.Unmarshal(out, &list); err != nil {
	}

	var isClientTests = []struct {
		resource string
		node     string
		l        resList
		out      bool
	}{
		{"default-william", "kubelet-a", list, true},
		{"default-william", "kubelet-x", list, false},
		{"default-william", "kubelet-c", list, true},
	}

	for _, tt := range isClientTests {

		ok := doResOnNode(tt.l, tt.resource, tt.node)
		if tt.out != ok {
			t.Errorf("Expected: %v on %s Got: %v", tt.out, tt.node, ok)
		}
	}

}
