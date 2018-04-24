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
	"reflect"
	"testing"
)

func TestDoResExists(t *testing.T) {

	testOut1 := []byte("[\n  {\n    \"resources\": [\n      {\n        \"vlms\": [\n          {\n            \"vlm_dfn_uuid\": \"8000558f-9061-4256-b0fe-ec8d3b7fc051\", \n            \"stor_pool_uuid\": \"2d840084-f071-4173-a509-9a56d695a606\", \n            \"vlm_uuid\": \"ff382153-d47c-4a9f-a466-4ef4b3268656\", \n            \"vlm_nr\": 0, \n            \"stor_pool_name\": \"drbdpool\"\n          }, \n          {\n            \"vlm_dfn_uuid\": \"f763311c-3eb2-4f0e-ad55-361d4d026346\", \n            \"stor_pool_uuid\": \"2d840084-f071-4173-a509-9a56d695a606\", \n            \"vlm_uuid\": \"02a24f94-03a8-4918-8173-506b4cca369c\", \n            \"vlm_nr\": 1, \n            \"stor_pool_name\": \"drbdpool\"\n          }\n        ], \n        \"node_uuid\": \"1efa3fdf-129a-4018-a3f2-3d0f46571c5b\", \n        \"uuid\": \"d34e324a-54d8-4c79-adf5-3da33eca3371\", \n        \"node_name\": \"kubelet-a\", \n        \"props\": [\n          {\n            \"value\": \"drbdpool\", \n            \"key\": \"StorPoolName\"\n          }\n        ], \n        \"rsc_dfn_uuid\": \"83c4ce34-1bb5-48a7-a153-457f056223e1\", \n        \"name\": \"r00\"\n      }, \n      {\n        \"rsc_flags\": [\n          \"DELETE\"\n        ], \n        \"vlms\": [\n          {\n            \"vlm_nr\": 0, \n            \"stor_pool_name\": \"drbdpool\", \n            \"stor_pool_uuid\": \"ea3ddc0b-bc4f-4860-ae38-744c65cb3d18\", \n            \"vlm_uuid\": \"aebe934e-cda3-4b67-8d08-48c9444d12eb\", \n            \"vlm_dfn_uuid\": \"8000558f-9061-4256-b0fe-ec8d3b7fc051\", \n            \"vlm_flags\": [\n              \"DELETE\"\n            ]\n          }, \n          {\n            \"vlm_nr\": 1, \n            \"stor_pool_name\": \"drbdpool\", \n            \"stor_pool_uuid\": \"ea3ddc0b-bc4f-4860-ae38-744c65cb3d18\", \n            \"vlm_uuid\": \"36e679a3-d59a-4147-8f1e-299ae689c1d1\", \n            \"vlm_dfn_uuid\": \"f763311c-3eb2-4f0e-ad55-361d4d026346\", \n            \"vlm_flags\": [\n              \"DELETE\"\n            ]\n          }\n        ], \n        \"node_uuid\": \"c361253c-4e75-48d5-b00a-8a3df5eb3a69\", \n        \"uuid\": \"12a54b0c-5d76-4766-b973-26cfb1beaae4\", \n        \"node_name\": \"kubelet-b\", \n        \"props\": [\n          {\n            \"value\": \"drbdpool\", \n            \"key\": \"StorPoolName\"\n          }\n        ], \n        \"rsc_dfn_uuid\": \"83c4ce34-1bb5-48a7-a153-457f056223e1\", \n        \"name\": \"r00\"\n      }\n    ]\n  }\n]")
	var resInfoTests = []struct {
		resource string
		resInfo  []byte
		out      bool
	}{
		{"r00", testOut1, true},
		{"wrong-o", testOut1, false},
	}

	for _, tt := range resInfoTests {
		ok, _ := doResExists(tt.resource, tt.resInfo)
		if ok != tt.out {
			t.Errorf("Called: doResExists(%q, %q), Expected: %v, Got: %v", tt.resource, tt.resInfo, tt.out, ok)
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
