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
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Resource contains all the information needed to query and assign/deploy
// a resource. If you're deploying a resource, Redundancy is required. If you're
// assigning a resource to a particular node, NodeName is required.
type Resource struct {
	Name                string
	NodeName            string
	Redundancy          string
	NodeList            []string
	ClientList          []string
	AutoPlace           string
	DoNotPlaceWithRegex string
	SizeKiB             uint64
	StoragePool         string
	Encryption          bool
}

type resList []struct {
	ResourceStates []struct {
		RequiresAdjust bool      `json:"requires_adjust"`
		RscName        string    `json:"rsc_name"`
		IsPrimary      bool      `json:"is_primary"`
		VlmStates      []volInfo `json:"vlm_states"`
		IsPresent      bool      `json:"is_present"`
		NodeName       string    `json:"node_name"`
	} `json:"resource_states"`
	Resources []resInfo `json:"resources"`
}
type resInfo struct {
	Vlms []struct {
		VlmNr        int    `json:"vlm_nr"`
		StorPoolName string `json:"stor_pool_name"`
		StorPoolUUID string `json:"stor_pool_uuid"`
		VlmMinorNr   int    `json:"vlm_minor_nr"`
		VlmUUID      string `json:"vlm_uuid"`
		VlmDfnUUID   string `json:"vlm_dfn_uuid"`
	} `json:"vlms"`
	NodeUUID string `json:"node_uuid"`
	UUID     string `json:"uuid"`
	NodeName string `json:"node_name"`
	Props    []struct {
		Value string `json:"value"`
		Key   string `json:"key"`
	} `json:"props"`
	RscDfnUUID string   `json:"rsc_dfn_uuid"`
	Name       string   `json:"name"`
	RscFlags   []string `json:"rsc_flags,omitempty"`
}

type volInfo struct {
	HasDisk       bool `json:"has_disk"`
	CheckMetaData bool `json:"check_meta_data"`
	HasMetaData   bool `json:"has_meta_data"`
	IsPresent     bool `json:"is_present"`
	DiskFailed    bool `json:"disk_failed"`
	NetSize       int  `json:"net_size"`
	VlmMinorNr    *int `json:"vlm_minor_nr"` // Allow nil checking.
	GrossSize     int  `json:"gross_size"`
	VlmNr         int  `json:"vlm_nr"`
}

type returnStatuses []struct {
	DetailsFormat string `json:"details_format"`
	MessageFormat string `json:"message_format"`
	CauseFormat   string `json:"cause_format,omitempty"`
	ObjRefs       []struct {
		Value string `json:"value"`
		Key   string `json:"key"`
	} `json:"obj_refs"`
	Variables []struct {
		Value string `json:"value"`
		Key   string `json:"key"`
	} `json:"variables"`
	RetCode uint64 `json:"ret_code"`
}

type resDefInfo []struct {
	RscDfns []struct {
		VlmDfns []struct {
			VlmDfnUUID string `json:"vlm_dfn_uuid"`
			VlmMinor   int    `json:"vlm_minor"`
			VlmNr      int    `json:"vlm_nr"`
			VlmSize    int    `json:"vlm_size"`
		} `json:"vlm_dfns,omitempty"`
		RscDfnSecret string `json:"rsc_dfn_secret"`
		RscDfnUUID   string `json:"rsc_dfn_uuid"`
		RscName      string `json:"rsc_name"`
		RscDfnPort   int    `json:"rsc_dfn_port"`
		RscDfnProps  []struct {
			Value string `json:"value"`
			Key   string `json:"key"`
		} `json:"rsc_dfn_props,omitempty"`
	} `json:"rsc_dfns"`
}

func (s returnStatuses) validate() error {
	for _, message := range s {
		if !linstorSuccess(message.RetCode) {
			msg, err := json.Marshal(s)
			if err != nil {
				return err
			}
			return fmt.Errorf("error status from one or more linstor operations: %s", msg)
		}
	}
	return nil
}

func linstorSuccess(retcode uint64) bool {
	const maskError = 0xC000000000000000 // includes warnings and info (i.e., everything != SUCCESS)
	return (retcode & maskError) == 0
}

// CreateAndAssign deploys the resource, created a new one if it doesn't exist.
func (r Resource) CreateAndAssign() error {
	if err := r.Create(); err != nil {
		return err
	}
	return r.Assign()
}

// Only use this for things that return the normal returnStatuses json.
func linstor(args ...string) error {
	args = append([]string{"-m"}, args...)
	out, err := exec.Command("linstor", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v : %s", err, out)
	}

	if !json.Valid(out) {
		return fmt.Errorf("not a valid json input: %s", out)
	}
	s := returnStatuses{}
	if err := json.Unmarshal(out, &s); err != nil {
		return fmt.Errorf("couldn't Unmarshal %s :%v", out, err)
	}

	return s.validate()
}

// Create reserves the resource name in Linstor.
func (r Resource) Create() error {
	defPresent, volZeroPresent, err := r.checkDefined()
	if err != nil {
		return err
	}

	if !defPresent {
		if err := linstor("create-resource-definition", r.Name); err != nil {
			return fmt.Errorf("unable to reserve resource name %s :%v", r.Name, err)
		}
	}

	if !volZeroPresent {

		args := []string{"create-volume-definition", r.Name, fmt.Sprintf("%dkib", r.SizeKiB)}
		if r.Encryption {
			args = append(args, "--encrypt")
		}

		if err := linstor(args...); err != nil {
			return fmt.Errorf("unable to reserve resource name %s :%v", r.Name, err)
		}
	}

	return nil
}

func (r Resource) checkDefined() (bool, bool, error) {
	out, err := exec.Command("linstor", "-m", "list-resource-definitions").CombinedOutput()
	if err != nil {
		return false, false, fmt.Errorf("%v: %s", err, out)
	}

	if !json.Valid(out) {
		return false, false, fmt.Errorf("not a valid json input: %s", out)
	}
	s := resDefInfo{}
	if err := json.Unmarshal(out, &s); err != nil {
		return false, false, fmt.Errorf("couldn't Unmarshal %s :%v", out, err)
	}

	var defPresent, volZeroPresent bool

	for _, def := range s[0].RscDfns {
		if def.RscName == r.Name {
			defPresent = true
			for _, vol := range def.VlmDfns {
				if vol.VlmNr == 0 {
					volZeroPresent = true
					break
				}
			}
			break
		}
	}

	return defPresent, volZeroPresent, nil
}

// Assign assigns a resource with diskfull storage to all nodes in its NodeList,
// then attaches the resource disklessly to all nodes in its ClientList.
func (r Resource) Assign() error {
	// Make sure the resource is defined before trying to assign it.
	//	ok, err := r.Exists()
	//	if err != nil {
	//		return fmt.Errorf("Unable to determine if resource %s is defined %v", r.Name, err)
	//	}
	//	if !ok {
	//		return fmt.Errorf("No resource definition for resource %s", r.Name)
	//	}
	//
	for _, node := range r.NodeList {
		present, err := r.OnNode(node)
		if err != nil {
			return fmt.Errorf("unable to assign resource %s failed to check if it was already present on node %s: %v", r.Name, node, err)
		}
		if !present {
			if err = linstor("create-resource", r.Name, node, "-s", r.StoragePool); err != nil {
				return err
			}
		}
	}

	for _, node := range r.ClientList {
		present, err := r.OnNode(node)
		if err != nil {
			return fmt.Errorf("unable to assign resource %s failed to check if it was already present on node %s: %v", r.Name, node, err)
		}
		if !present {
			if err = linstor("create-resource", r.Name, node, "--diskless"); err != nil {
				return err
			}
		}
	}

	if r.AutoPlace != "" {
		args := []string{"create-resource", r.Name, "--auto-place", r.AutoPlace}
		if r.DoNotPlaceWithRegex != "" {
			args = append(args, "--do-not-place-with-regex", r.DoNotPlaceWithRegex)
		}

		if err := linstor(args...); err != nil {
			return err
		}
	}

	return nil
}

// Unassign unassigns a resource from a particular node.
func (r Resource) Unassign(nodeName string) error {
	if err := linstor("delete-resource", r.Name, nodeName); err != nil {
		return fmt.Errorf("failed to unassign resource %s from node %s: %v", r.Name, nodeName, err)
	}
	return nil
}

// Delete removes a resource entirely from all nodes.
func (r Resource) Delete() error {
	defPresent, _, err := r.checkDefined()
	if err != nil {
		return fmt.Errorf("failed to delete resource %s: %v", r.Name, err)
	}

	// If the resource definition doesn't exist, then the resource is as deleted
	// as we can possibly make it.
	if !defPresent {
		return nil
	}

	if err := linstor("delete-resource-definition", r.Name); err != nil {
		return fmt.Errorf("failed to delete resource %s: %v", r.Name, err)
	}
	return nil
}

// Exists checks to see if a resource is defined in DRBD Manage.
func (r Resource) Exists() (bool, error) {
	out, err := exec.Command("linstor", "-m", "list-resources").CombinedOutput()
	if err != nil {
		return false, err
	}

	// Inject real implementations here, test through the internal function.
	return doResExists(r.Name, out)
}

func doResExists(resourceName string, resInfo []byte) (bool, error) {
	resources := resList{}

	if !json.Valid(resInfo) {
		return false, fmt.Errorf("not a valid json input: %s", resInfo)
	}
	err := json.Unmarshal(resInfo, &resources)
	if err != nil {
		return false, fmt.Errorf("couldn't Unmarshal %s :%v", resInfo, err)
	}

	for _, r := range resources[0].Resources {
		if r.Name == resourceName {
			return true, nil
		}
	}

	return false, nil
}

//OnNode determines if a resource is present on a particular node.
func (r Resource) OnNode(nodeName string) (bool, error) {
	out, err := exec.Command("linstor", "-m", "list-resources").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("%v: %s", err, out)
	}

	if !json.Valid(out) {
		return false, fmt.Errorf("not a valid json input: %s", out)
	}
	l := resList{}
	if err := json.Unmarshal(out, &l); err != nil {
		return false, fmt.Errorf("couldn't Unmarshal %s :%v", out, err)
	}

	return doResOnNode(l, r.Name, nodeName), nil
}

func doResOnNode(list resList, resName, nodeName string) bool {
	for _, res := range list[0].Resources {
		if res.Name == resName && res.NodeName == nodeName {
			return true
		}
	}
	return false
}

// IsClient determines if resource is running as a client on nodeName.
func (r Resource) IsClient(nodeName string) bool {
	out, _ := exec.Command("linstor", "-m", "list-resources").CombinedOutput()

	if !json.Valid(out) {
		return false
	}
	list := resList{}
	if err := json.Unmarshal(out, &list); err != nil {
		return false
	}

	return r.doIsClient(list, nodeName)
}

func (r Resource) doIsClient(list resList, nodeName string) bool {
	// Traverse all the volume states to find volume 0 of our resource on nodeName.
	// Assume volume 0 is the one we want.
	for _, res := range list[0].ResourceStates {
		if r.Name == res.RscName && nodeName == res.NodeName {
			for _, v := range res.VlmStates {
				if v.VlmNr == 0 {
					return !v.HasDisk
				}
			}
		}
	}

	return false
}

// EnoughFreeSpace checks to see if there's enough free space to create a new resource.
func EnoughFreeSpace(requestedKiB, replicas string) error {
	return nil
}

// FSUtil handles creating a filesystem and mounting resources.
type FSUtil struct {
	*Resource
	FSType    string
	BlockSize int64
	args      []string
}

// Mount the FSUtil's resource on the path.
func (f FSUtil) Mount(path string) error {
	device, err := WaitForDevPath(*f.Resource, 3)
	if err != nil {
		return fmt.Errorf("unable to mount device, couldn't find Resource device path: %v", err)
	}

	err = f.safeFormat(device)
	if err != nil {
		return fmt.Errorf("unable to mount device: %v", err)
	}

	out, err := exec.Command("mkdir", "-p", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to mount device, failed to make mount directory: %v: %s", err, out)
	}

	out, err = exec.Command("mount", device, path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to mount device: %v: %s", err, out)
	}

	return nil
}

// UnMount the FSUtil's resource from the path.
func (f FSUtil) UnMount(path string) error {
	// If the path isn't a directory, we're not mounted there.
	_, err := exec.Command("test", "-d", path).CombinedOutput()
	if err != nil {
		return nil
	}

	// If the path isn't mounted, then we're not mounted.
	_, err = exec.Command("findmnt", "-f", path).CombinedOutput()
	if err != nil {
		return nil
	}

	out, err := exec.Command("umount", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to unmount device: %q: %s", err, out)
	}

	return nil
}

func (f FSUtil) safeFormat(path string) error {
	deviceFS, err := checkFSType(path)
	if err != nil {
		return fmt.Errorf("unable to format filesystem for %q: %v", path, err)
	}

	// Device is formatted correctly already.
	if deviceFS == f.FSType {
		return nil
	}

	if deviceFS != "" && deviceFS != f.FSType {
		return fmt.Errorf("device %q already formatted with %q filesystem, refusing to overwrite with %q filesystem", path, deviceFS, f.FSType)
	}

	f.populateArgs()

	args := []string{"-t", f.FSType}
	args = append(args, f.args...)
	args = append(args, path)

	out, err := exec.Command("mkfs", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("couldn't create %s filesystem %v: %q", f.FSType, err, out)
	}

	return nil
}

func (f *FSUtil) populateArgs() error {

	if f.BlockSize != 0 {
		b := strconv.FormatInt(f.BlockSize, 10)

		if f.FSType == "xfs" {
			b = fmt.Sprintf("size=%s", b)
		}

		f.args = []string{"-b", b}
	}

	return nil
}

func checkFSType(dev string) (string, error) {
	// If there's no filesystem, then we'll have a nonzero exit code, but no output
	// doCheckFSType handles this case.
	out, _ := exec.Command("blkid", "-o", "udev", dev).CombinedOutput()

	FSType, err := doCheckFSType(string(out))
	if err != nil {
		return "", err
	}
	return FSType, nil
}

// Parse the filesystem from the output of `blkid -o udev`
func doCheckFSType(s string) (string, error) {
	f := strings.Fields(s)

	// blkid returns an empty string if there's no filesystem and so do we.
	if len(f) == 0 {
		return "", nil
	}

	blockAttrs := make(map[string]string)
	for _, pair := range f {
		p := strings.Split(pair, "=")
		if len(p) < 2 {
			return "", fmt.Errorf("couldn't parse filesystem data from %s", s)
		}
		blockAttrs[p[0]] = p[1]
	}

	FSKey := "ID_FS_TYPE"
	fs, ok := blockAttrs[FSKey]
	if !ok {
		return "", fmt.Errorf("couldn't find %s in %s", FSKey, blockAttrs)
	}
	return fs, nil
}

// WaitForDevPath polls until the resourse path appears on the system.
func WaitForDevPath(r Resource, maxRetries int) (string, error) {
	var path string
	var err error

	for i := 0; i < maxRetries; i++ {
		path, err = getDevPath(r)
		if path != "" {
			return path, err
		}
		time.Sleep(time.Second * 2)
	}
	return path, err
}

func getDevPath(r Resource) (string, error) {
	out, err := exec.Command("linstor", "-m", "list-resources").CombinedOutput()
	if err != nil {
		return "", err
	}

	if !json.Valid(out) {
		return "", fmt.Errorf("not a valid json input: %s", out)
	}
	list := resList{}
	if err := json.Unmarshal(out, &list); err != nil {
		return "", err
	}

	// Traverse all the volume states to find volume 0 of our resource.
	// Assume volume 0 is the one we want.
	vol := &volInfo{}
	for _, res := range list[0].ResourceStates {
		if r.Name == res.RscName {
			for _, v := range res.VlmStates {
				if v.VlmNr == 0 {
					vol = &v
				}
			}
		}
	}

	if vol.VlmMinorNr == nil {
		return "", fmt.Errorf("Unable to receive volume state information from Linstor satellite")
	}

	devicePath := doGetDevPath(*vol)

	if _, err := os.Lstat(devicePath); err != nil {
		return "", fmt.Errorf("Couldn't stat %s: %v", devicePath, err)
	}

	return devicePath, nil
}

func doGetDevPath(vol volInfo) string {
	return fmt.Sprintf("/dev/drbd%d", *vol.VlmMinorNr)
}
