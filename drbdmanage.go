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

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Resource contains all the information needed to query and assign/deploy
// a resource. If you're deploying a resource, Redundancy is required. If you're
// assigning a resource to a particular node, NodeName is required.
type Resource struct {
	Name       string
	NodeName   string
	Redundancy string
}

// Assign assigns a resource to it's NodeName.
func (r Resource) Assign() (bool, error) {
	// Make sure the resource is defined before trying to assign it.
	if ok, err := r.Exists(); err != nil || !ok {
		return ok, err
	}

	// If the resource is already assigned, we're done.
	if ok, err := resAssigned(r); err != nil || ok {
		return ok, err
	}

	rawOut, err := exec.Command("drbdmanage", "assign-resource", r.Name, r.NodeName, "--client").CombinedOutput()
	out := stripDMOutput(rawOut)
	if err != nil {
		return false, fmt.Errorf("unable to assign resource %q on node %q: %s", r.Name, r.NodeName, out)
	}
	return r.WaitForAssignment(5)
}

// Unassign unassigns a resource from it's NodeName.
func (r Resource) Unassign() error {
	rawOut, err := exec.Command("drbdmanage", "unassign-resource", r.Name, r.NodeName, "--quiet").CombinedOutput()
	out := stripDMOutput(rawOut)
	if err != nil {
		return fmt.Errorf("failed to unassign resource %q from node %q. Error: %s", r.Name, r.NodeName, out)
	}
	ok, err := r.WaitForUnassignment(3)
	if err != nil {
		return fmt.Errorf("failed to unassign resource %q from node %q. Error: %v", r.Name, r.NodeName, err)
	}
	if !ok {
		return fmt.Errorf("failed to unassign resource %q from node %q. Error: Resource still assigned", r.Name, r.NodeName)
	}
	return nil
}

// Exists checks to see if a resource is defined in DRBD Manage.
func (r Resource) Exists() (bool, error) {
	rawOut, err := exec.Command("drbdmanage", "list-resources", "--resources", r.Name, "--machine-readable").CombinedOutput()
	out := stripDMOutput(rawOut)
	if err != nil {
		return false, err
	}

	// Inject real implementations here, test through the internal function.
	return doResExists(r.Name, out)
}

func doResExists(resource, resInfo string) (bool, error) {
	if resInfo == "" {
		return false, nil
	}
	if strings.Split(resInfo, ",")[0] != resource {
		return false, fmt.Errorf("error retriving resource information from the following output: %q", resInfo)
	}

	return true, nil
}

// WaitForAssignment polls drbdmanage until resource assignment is complete.
func (r Resource) WaitForAssignment(maxRetries int) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		// If there are no errors and the resource is assigned, we can exit early.
		if ok, err := resAssigned(r); err == nil && ok {
			return ok, nil
		}
		time.Sleep(1 * time.Second)
		// See if we can recover from any errors or complete pending state changes.
		retryFailedActions(r)
	}
	// Return any errors that might have prevented resource assignment.
	return resAssigned(r)
}

// WaitForUnassignment Poll drbdmanage until resource unassignment is complete.
func (r Resource) WaitForUnassignment(maxRetries int) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		// If there are no errors and the resource is unassigned, we can exit early.
		if ok, err := resAssigned(r); err == nil && !ok {
			return !ok, nil
		}
		time.Sleep(1 * time.Second)
		// See if we can recover from any errors or complete pending state changes.
		retryFailedActions(r)
	}
	// Return any errors that might have prevented resource unassignment.
	ok, err := resAssigned(r)
	return !ok, err
}

func resAssigned(r Resource) (bool, error) {
	args := []string{"list-assignments", "--resources", r.Name, "--machine-readable"}

	if r.NodeName != "" {
		args = append(args, "--nodes", r.NodeName)
	}

	rawOut, err := exec.Command("drbdmanage", args...).CombinedOutput()
	out := stripDMOutput(rawOut)
	if err != nil {
		return false, fmt.Errorf("%s: %v", out, err)
	}

	// Loop through assignments if any aren't assigned, return false.
	for _, assignmentInfo := range strings.Split(out, "\n") {
		ok, err := doResAssigned(assignmentInfo)
		if !ok || err != nil {
			return ok, err
		}
	}

	return true, nil
}

func doResAssigned(assignmentInfo string) (bool, error) {
	if assignmentInfo == "" {
		return false, nil
	}

	fields := strings.Split(assignmentInfo, ",")
	if len(fields) != 5 {
		return false, fmt.Errorf("malformed assignmentInfo: %q", assignmentInfo)
	}

	// Target state differs from current state.
	// The assignment exists, but is in a transient state or unhealthy.
	currentState := strings.TrimSpace(fields[3])
	targetState := strings.TrimSpace(fields[4])
	if currentState != targetState {
		return false, fmt.Errorf("assignment targetState %q differs from currentState %q", targetState, currentState)
	}

	return true, nil
}

func retryFailedActions(r Resource) {
	exec.Command("drbdmanage", "resume-all").CombinedOutput()
	time.Sleep(time.Second * 2)
}

// IsClient determines is running as a client locally.
func (r Resource) IsClient() bool {
	rawOut, err := exec.Command("drbdmanage", "list-assignments", "--resources", r.Name, "--nodes", r.NodeName, "--machine-readable").CombinedOutput()
	out := stripDMOutput(rawOut)
	if err != nil {
		return false
	}
	return doIsClient(out)
}

func doIsClient(assignmentInfo string) bool {
	// No assignment for the resource on the host.
	if assignmentInfo == "" {
		return false
	}
	fields := strings.Split(assignmentInfo, ",")
	if len(fields) != 5 {
		return false
	}

	targetState := strings.TrimSpace(fields[4])

	if targetState != "connect|deploy|diskless" {
		return false
	}

	return true
}

// GetResourceNameFromDevice takes in a device path a returns the resource name
// (if any) associated with that device.
func GetResourceNameFromDevice(r Resource, device string) (string, error) {
	minor, err := getMinorFromDevice(device)
	if err != nil {
		return "", err
	}

	rawOut, err := exec.Command("drbdmanage", "list-volumes", "--machine-readable").CombinedOutput()
	out := stripDMOutput(rawOut)
	if err != nil {
		return "", fmt.Errorf("unable to get volume information: %s", out)
	}

	res, err := getResFromVolumes(out, minor)
	if err != nil {
		return "", err
	}

	return res, nil
}

func getMinorFromDevice(device string) (string, error) {
	if ok, _ := regexp.MatchString("/dev/drbd\\d+", device); !ok {
		return "", fmt.Errorf("tried to get minor from non-DRBD device: %q", device)
	}

	return device[9:], nil
}

func getResFromVolumes(volumes, minor string) (string, error) {
	vols := strings.Split(volumes, "\n")
	for _, v := range vols {
		fields := strings.Split(v, ",")

		// If we get badly formatted volume info, skip it: the next one might be ok.
		if len(fields) != 7 {
			continue
		}
		if fields[5] == minor {
			return fields[0], nil
		}
	}
	return "", nil
}

// EnoughFreeSpace checks to see if there's enough free space to create a new resource.
func EnoughFreeSpace(requestedKiB, replicas string) error {

	rawOut, err := exec.Command("drbdmanage", "list-free-space", "-m", replicas).CombinedOutput()
	out := stripDMOutput(rawOut)
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}

	if _, err := doEnoughFreeSpace(requestedKiB, out); err != nil {
		return err
	}

	return nil
}

func doEnoughFreeSpace(requestedKiB, dmListFreeSpaceOut string) (bool, error) {
	request, err := strconv.Atoi(requestedKiB)
	if err != nil || request < 1 {
		return false, fmt.Errorf("requsted storage must be a positive interger, got %s", requestedKiB)
	}

	if strings.HasPrefix(dmListFreeSpaceOut, "Error:") {
		return false, errors.New(dmListFreeSpaceOut)
	}

	free, err := strconv.Atoi(strings.Split(dmListFreeSpaceOut, ",")[0])
	if err != nil {
		return false, fmt.Errorf("unable to determine free space: %v", err)
	}

	if ok := request < free; !ok {
		return ok, fmt.Errorf("not enough space available to provision a new resource: want %dKiB have %dKiB",
			request, free)
	}

	return true, nil
}

func stripDMOutput(out []byte) string {
	// DM will output a message indicating contact to Dbus was succesfull
	// this can end up in front of, or behind the output we care about.
	extraOutput := "Operation completed successfully"
	s := strings.TrimSpace(string(out))
	return strings.TrimSpace(
		strings.TrimPrefix(strings.TrimSuffix(s, extraOutput), extraOutput))
}

// FSUtil handles creating a filesystem and mounting resources.
type FSUtil struct {
	*Resource
	FSType string
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

	out, err := exec.Command("mkfs", "-t", f.FSType, path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("couldn't create %s filesystem %v: %q", f.FSType, err, out)
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
	rawOut, err := exec.Command("drbdmanage", "list-volumes", "--resources", r.Name, "--machine-readable").CombinedOutput()
	out := stripDMOutput(rawOut)
	if err != nil {
		return "", fmt.Errorf("Unable to get volume information: %s", out)
	}

	devicePath, err := doGetDevPath(out)
	if err != nil {
		return "", err
	}

	if _, err := os.Lstat(devicePath); err != nil {
		return "", fmt.Errorf("Couldn't stat %s: %v", devicePath, err)
	}

	return devicePath, nil
}

func doGetDevPath(volInfo string) (string, error) {
	if volInfo == "" {
		return "", fmt.Errorf("Resource is not configured")
	}

	s := strings.Split(volInfo, ",")
	if len(s) != 7 {
		return "", fmt.Errorf("malformed volume string: %q", volInfo)
	}

	minor := s[5]
	ok, err := regexp.MatchString("\\d+", minor)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("bad device minor %q in volume string: %q", minor, volInfo)
	}

	return "/dev/drbd" + minor, nil
}
