/*
* A helpful library to interact with DRBD Manage
* Copyright © 2017 LINBIT USA LCC
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

package drbdmanage

import "testing"

func TestDoGetDevPath(t *testing.T) {
	var volumeStringTests = []struct {
		in  string
		out string
	}{
		{"test0,,0,102400,7001,130,\n", "/dev/drbd130"},
		{"test1,,0,102400,7002,131,\n", "/dev/drbd131"},
		{"test2,,0,102400,2003,132,\ntest3,,0,102400,2004,133,\n", ""},
	}

	for _, tt := range volumeStringTests {
		dev, _ := doGetDevPath(tt.in)
		if dev != tt.out {
			t.Errorf("Called: doGetDevPath(%q), Expected: %q, Got: %q", tt.in, tt.out, dev)
		}
	}
}

func TestDoResExists(t *testing.T) {
	var resInfoTests = []struct {
		resource string
		resInfo  string
		out      bool
	}{
		{"test0", "test0,7001,\n", true},
		{"test1", "test1,7002,\n", true},
		{"test1", "test1,7002,\n", true},
		{"vc-28d98b6b-e053-11e7-8146-663764653533", "vc-28d98b6b-e053-11e7-8146-663764653533,", true},
		{"test2", "test9,7003,\n", false},
		{"test3", "", false},
	}

	for _, tt := range resInfoTests {
		ok, _ := doResExists(tt.resource, tt.resInfo)
		if ok != tt.out {
			t.Errorf("Called: doResExists(%q, %q), Expected: %v, Got: %v", tt.resource, tt.resInfo, tt.out, ok)
		}
	}
}

func TestDoResAssigned(t *testing.T) {
	var resAssignmentTests = []struct {
		assignmentInfo string
		out            bool
	}{
		{"node0,test0,0,connect|deploy,connect|deploy\n", true},
		{"node1,test1,0,connect|deploy|diskless,connect|deploy|diskless\n", true},
		{"", false},
		{"node0,test0,0,,connect|deploy\n", false},
	}

	for _, tt := range resAssignmentTests {
		ok, _ := doResAssigned(tt.assignmentInfo)
		if ok != tt.out {
			t.Errorf("Called: doResAssigned(%q), Expected: %v, Got: %v", tt.assignmentInfo, tt.out, ok)
		}
	}
}

func TestDoIsClient(t *testing.T) {
	var isClientTests = []struct {
		assignmentInfo string
		out            bool
	}{
		{"node0,test0,1,connect|deploy|diskless,connect|deploy|diskless\n", true},
		{"bielefeld,test1,1,connect|deploy,connect|deploy\n", false},
		{"", false},
	}

	for _, tt := range isClientTests {
		ok := doIsClient(tt.assignmentInfo)
		if ok != tt.out {
			t.Errorf("Called: doIsClient(%q), Expected: %v, Got: %v", tt.assignmentInfo, tt.out, ok)
		}
	}
}

func TestDoGetMinorFromDevice(t *testing.T) {
	var getMinorTests = []struct {
		device string
		out    string
	}{
		{"/dev/drbd100", "100"},
		{"/dev/drbd5123", "5123"},
		{"/dev/drbd0", "0"},
		{"/dev/sda1", ""},
	}

	for _, tt := range getMinorTests {
		minor, _ := getMinorFromDevice(tt.device)
		if minor != tt.out {
			t.Errorf("Called: getMinorFromDevice(%q), Expected: %v, Got: %v", tt.device, tt.out, minor)
		}
	}
}

func TestGetResFromVolumes(t *testing.T) {
	var getResFromVolumesTests = []struct {
		volumes string
		minor   string
		out     string
	}{
		{"test0,,0,102400,7000,100,\n", "100", "test0"},
		{"test0,,0,102400,7000,100,\ntest1,,0,102400,7001,101,\ntest2,,0,2097152,7002,102,\n", "101", "test1"},
		{"test0,,0,102400,7000,100,\ntest1,,0,102400,7001,101,\ntest2,,0,2097152,7002,102,\n", "7001", ""},
		{"", "104", ""},
	}

	for _, tt := range getResFromVolumesTests {
		res, _ := getResFromVolumes(tt.volumes, tt.minor)
		if res != tt.out {
			t.Errorf("Called: getResFromVolumes(%q, %q) Expected: %v, Got: %v", tt.volumes, tt.minor, tt.out, res)
		}
	}
}

func TestDoEnoughFreeSpace(t *testing.T) {

	var freeSpaceTests = []struct {
		requestedKiB       string
		dmListFreeSpaceOut string
		out                bool
	}{
		{"5000", "3136828,16760832\n", true},
		{"50000000", "3136828,16760832\n", false},
		{"38967", "Error: Deployment node count exceeds the number of nodes in the cluster\n", false},
		{"banana", "3136828,16760832\n", false},
		{"-40", "3136828,16760832\n", false},
		{"3.14", "3136828,16760832\n", false},
	}

	for _, tt := range freeSpaceTests {
		ok, err := doEnoughFreeSpace(tt.requestedKiB, tt.dmListFreeSpaceOut)
		if ok != tt.out {
			t.Fatalf("TestDoEnoughFreeSpace(%q, %q) should be %v, got %v :%v ",
				tt.requestedKiB, tt.dmListFreeSpaceOut, tt.out, ok, err)
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
