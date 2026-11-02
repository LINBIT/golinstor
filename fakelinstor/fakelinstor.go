// Package fakelinstor provides an in-memory HTTP test double for the LINSTOR
// REST API. It implements a subset of the API against which a client.Client can
// be pointed, so applications can exercise their LINSTOR interactions in tests
// without a running LINSTOR controller.
//
// All state is kept in memory. The following client calls are supported:
//
//   - Controller: GetVersion (reports a static fake version)
//   - Nodes: Create, GetAll, Get, Delete, Evacuate, Restore, Lost
//   - Node storage pools: GetStoragePools, GetStoragePool, CreateStoragePool, ModifyStoragePool, DeleteStoragePool
//   - Node physical storage: CreateDevicePool
//   - ResourceGroups: Create, GetAll
//   - ResourceDefinitions: Create, GetAll (optionally with volume definitions), Delete, Truncate,
//     CreateVolumeDefinition, GetVolumeDefinitions, ModifyVolumeDefinition,
//     Clone (completes synchronously), CloneStatus
//   - Resources: Create, Get, GetAll, GetResourceView
//
// Any endpoint not listed above responds with 501 Not Implemented rather than
// 404, so an unimplemented call is not mistaken for a genuine
// [github.com/LINBIT/golinstor/client.NotFoundError].
//
// [FakeLinstor] additionally exposes the test-only helpers StoragePools and
// DevicePoolCreations for assertions, and SetConnectionStatus to simulate node
// connectivity changes.
package fakelinstor

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"

	linstor "github.com/LINBIT/golinstor"
	lapi "github.com/LINBIT/golinstor/client"
	"github.com/LINBIT/golinstor/clonestatus"
)

func New() *FakeLinstor {
	f := &FakeLinstor{
		volumeDefinitions: map[string][]lapi.VolumeDefinition{},
		clones:            map[string]clonestatus.CloneStatus{},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /v1/controller/version", wrapHandler(f.getVersion))
	mux.Handle("POST /v1/nodes", wrapHandler(f.createNode))
	mux.Handle("GET /v1/nodes", wrapHandler(f.getNodes))
	mux.Handle("GET /v1/nodes/{node}", wrapHandler(f.getNode))
	mux.Handle("DELETE /v1/nodes/{node}", wrapHandler(f.deleteNode))
	mux.Handle("PUT /v1/nodes/{node}/evacuate", wrapHandler(f.evacuateNode))
	mux.Handle("PUT /v1/nodes/{node}/restore", wrapHandler(f.restoreNode))
	mux.Handle("DELETE /v1/nodes/{node}/lost", wrapHandler(f.lostNode))
	mux.Handle("POST /v1/resource-groups", wrapHandler(f.createResourceGroup))
	mux.Handle("GET /v1/resource-groups", wrapHandler(f.getResourceGroups))
	mux.Handle("POST /v1/resource-definitions", wrapHandler(f.createResourceDefinition))
	mux.Handle("GET /v1/resource-definitions", wrapHandler(f.getResourceDefinitions))
	mux.Handle("DELETE /v1/resource-definitions/{rd}", wrapHandler(f.deleteResourceDefinition))
	mux.Handle("POST /v1/resource-definitions/{rd}/clone", wrapHandler(f.cloneResourceDefinition))
	mux.Handle("GET /v1/resource-definitions/{rd}/clone/{clone}", wrapHandler(f.cloneStatus))
	mux.Handle("POST /v1/resource-definitions/{rd}/volume-definitions", wrapHandler(f.createVolumeDefinition))
	mux.Handle("GET /v1/resource-definitions/{rd}/volume-definitions", wrapHandler(f.getVolumeDefinitions))
	mux.Handle("PUT /v1/resource-definitions/{rd}/volume-definitions/{volNr}", wrapHandler(f.modifyVolumeDefinition))
	mux.Handle("GET /v1/resource-definitions/{rd}/resources", wrapHandler(f.getResources))
	mux.Handle("DELETE /v1/resource-definitions/{rd}/resources", wrapHandler(f.truncateResourceDefinition))
	mux.Handle("GET /v1/resource-definitions/{rd}/resources/{node}", wrapHandler(f.getResource))
	mux.Handle("POST /v1/resource-definitions/{rd}/resources/{node}", wrapHandler(f.createResource))
	mux.Handle("GET /v1/view/resources", wrapHandler(f.viewResources))
	mux.Handle("GET /v1/nodes/{node}/storage-pools", wrapHandler(f.getStoragePools))
	mux.Handle("GET /v1/nodes/{node}/storage-pools/{sp}", wrapHandler(f.getStoragePool))
	mux.Handle("POST /v1/nodes/{node}/storage-pools", wrapHandler(f.createStoragePool))
	mux.Handle("PUT /v1/nodes/{node}/storage-pools/{sp}", wrapHandler(f.modifyStoragePool))
	mux.Handle("DELETE /v1/nodes/{node}/storage-pools/{sp}", wrapHandler(f.deleteStoragePool))
	mux.Handle("POST /v1/physical-storage/{node}", wrapHandler(f.createDevicePool))
	mux.Handle("/", http.HandlerFunc(notImplemented))

	f.Server = httptest.NewServer(mux)
	return f
}

// notImplemented responds to any endpoint the fake does not handle. It uses
// 501 Not Implemented rather than 404 so callers do not mistake an
// unimplemented endpoint for a genuine lapi.NotFoundError.
func notImplemented(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(lapi.ApiCallError{{
		RetCode: 1,
		Message: fmt.Sprintf("endpoint not implemented in fakelinstor: %s %s", r.Method, r.URL.Path),
	}})
}

type FakeLinstor struct {
	Server *httptest.Server

	mu                  sync.Mutex
	nodes               []lapi.Node
	resources           []lapi.ResourceWithVolumes
	resourceDefinitions []lapi.ResourceDefinition
	volumeDefinitions   map[string][]lapi.VolumeDefinition
	clones              map[string]clonestatus.CloneStatus // keyed by cloneKey(source, clone)
	resourceGroups      []lapi.ResourceGroup
	storagePools        []lapi.StoragePool
	devicePoolCreations int
}

func wrapHandler(f func(r *http.Request) (any, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := f(r)
		if err != nil {
			if errors.Is(err, lapi.NotFoundError) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(lapi.ApiCallError{{RetCode: 1, Message: err.Error()}})
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(lapi.ApiCallError{{RetCode: 1, Message: err.Error()}})
			}
		} else {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(data)
		}
	})
}

func (f *FakeLinstor) getVersion(r *http.Request) (any, error) {
	return lapi.ControllerVersion{
		Version:        "1.0.0-fake",
		RestApiVersion: "fake-rest-api",
		GitHash:        "fake-git-hash",
		BuildTime:      "fake-build-time",
	}, nil
}

func (f *FakeLinstor) createNode(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var node lapi.Node
	err := json.NewDecoder(r.Body).Decode(&node)
	if err != nil {
		return nil, fmt.Errorf("error decoding node: %w", err)
	}

	if len(node.NetInterfaces) > 0 && node.Type == linstor.ValNodeTypeStlt {
		node.ConnectionStatus = "ONLINE"
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.nodes {
		if f.nodes[i].Name == node.Name {
			return nil, fmt.Errorf("node '%s' already exists", node.Name)
		}
	}
	f.nodes = append(f.nodes, node)

	return nil, nil
}

func (f *FakeLinstor) getNode(r *http.Request) (any, error) {
	node := r.PathValue("node")

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.nodes {
		if f.nodes[i].Name == node {
			return f.nodes[i], nil
		}
	}

	return nil, lapi.NotFoundError
}

func (f *FakeLinstor) getNodes(r *http.Request) (any, error) {
	q := r.URL.Query()
	names := q["nodes"]
	propFilters := q["props"]

	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]lapi.Node, 0, len(f.nodes))
	for i := range f.nodes {
		if len(names) > 0 && !slices.Contains(names, f.nodes[i].Name) {
			continue
		}
		if !matchesPropFilters(f.nodes[i].Props, propFilters) {
			continue
		}

		out = append(out, f.nodes[i])
	}

	return out, nil
}

func (f *FakeLinstor) deleteNode(r *http.Request) (any, error) {
	node := r.PathValue("node")

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.resources {
		if f.resources[i].NodeName == node {
			return nil, fmt.Errorf("cannot delete node '%s', it still contains resource '%s'", node, f.resources[i].Name)
		}
	}

	f.nodes = slices.DeleteFunc(f.nodes, func(n lapi.Node) bool {
		return n.Name == node
	})

	return nil, nil
}

func (f *FakeLinstor) createResourceGroup(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var rg lapi.ResourceGroup
	err := json.NewDecoder(r.Body).Decode(&rg)
	if err != nil {
		return nil, fmt.Errorf("error decoding resource group: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if slices.ContainsFunc(f.resourceGroups, func(exist lapi.ResourceGroup) bool {
		return exist.Name == rg.Name
	}) {
		return nil, fmt.Errorf("resource group '%s' already exists", rg.Name)
	}

	f.resourceGroups = append(f.resourceGroups, rg)
	return nil, nil
}

func (f *FakeLinstor) getResourceGroups(r *http.Request) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.resourceGroups, nil
}

func (f *FakeLinstor) createResourceDefinition(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var rd lapi.ResourceDefinitionCreate
	err := json.NewDecoder(r.Body).Decode(&rd)
	if err != nil {
		return nil, fmt.Errorf("error decoding resource definition: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if slices.ContainsFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
		return exist.Name == rd.ResourceDefinition.Name
	}) {
		return nil, fmt.Errorf("resource definition '%s' already exists", rd.ResourceDefinition.Name)
	}

	f.resourceDefinitions = append(f.resourceDefinitions, rd.ResourceDefinition)
	return nil, nil
}

func (f *FakeLinstor) getResourceDefinitions(r *http.Request) (any, error) {
	q := r.URL.Query()
	names := q["resource_definitions"]
	propFilters := q["props"]
	withVolumeDefinitions := q.Get("with_volume_definitions") == "true"

	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]lapi.ResourceDefinitionWithVolumeDefinition, 0, len(f.resourceDefinitions))
	for i := range f.resourceDefinitions {
		rd := f.resourceDefinitions[i]

		if len(names) > 0 && !slices.Contains(names, rd.Name) {
			continue
		}
		if !matchesPropFilters(rd.Props, propFilters) {
			continue
		}

		entry := lapi.ResourceDefinitionWithVolumeDefinition{ResourceDefinition: rd}
		if withVolumeDefinitions {
			entry.VolumeDefinitions = slices.Clone(f.volumeDefinitions[rd.Name])
		}

		out = append(out, entry)
	}

	return out, nil
}

// matchesPropFilters reports whether props satisfies every "key=value" filter.
func matchesPropFilters(props map[string]string, filters []string) bool {
	for _, filter := range filters {
		key, value, _ := strings.Cut(filter, "=")
		if props[key] != value {
			return false
		}
	}

	return true
}

func (f *FakeLinstor) getResource(r *http.Request) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	resource := r.PathValue("rd")
	node := r.PathValue("node")

	for i := range f.resources {
		if f.resources[i].NodeName == node && f.resources[i].Name == resource {
			return f.resources[i], nil
		}
	}

	return nil, lapi.NotFoundError
}

func (f *FakeLinstor) getResources(r *http.Request) (any, error) {
	rd := r.PathValue("rd")
	nodes := r.URL.Query()["nodes"]

	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]lapi.ResourceWithVolumes, 0)
	for i := range f.resources {
		if f.resources[i].Name != rd {
			continue
		}
		if len(nodes) > 0 && !slices.Contains(nodes, f.resources[i].NodeName) {
			continue
		}

		out = append(out, f.resources[i])
	}

	return out, nil
}

func (f *FakeLinstor) createResource(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var rc lapi.ResourceCreate
	err := json.NewDecoder(r.Body).Decode(&rc)
	if err != nil {
		return nil, fmt.Errorf("error decoding resource create: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	rd := r.PathValue("rd")
	node := r.PathValue("node")
	rc.Resource.Name = rd
	rc.Resource.NodeName = node

	if !slices.ContainsFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
		return exist.Name == rd
	}) {
		return nil, lapi.NotFoundError
	}

	if !slices.ContainsFunc(f.nodes, func(exist lapi.Node) bool {
		return exist.Name == node
	}) {
		return nil, lapi.NotFoundError
	}

	if !slices.ContainsFunc(f.resources, func(exist lapi.ResourceWithVolumes) bool {
		return exist.Name == rd && exist.NodeName == node
	}) {
		f.resources = append(f.resources, lapi.ResourceWithVolumes{
			Resource: rc.Resource,
		})
	}

	return nil, nil
}

func (f *FakeLinstor) deleteResourceDefinition(r *http.Request) (any, error) {
	rd := r.PathValue("rd")

	f.mu.Lock()
	defer f.mu.Unlock()

	f.resources = slices.DeleteFunc(f.resources, func(exist lapi.ResourceWithVolumes) bool {
		return exist.Name == rd
	})
	f.resourceDefinitions = slices.DeleteFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
		return exist.Name == rd
	})
	delete(f.volumeDefinitions, rd)
	for key := range f.clones {
		if strings.HasSuffix(key, "/"+rd) {
			delete(f.clones, key)
		}
	}

	return nil, nil
}

func (f *FakeLinstor) truncateResourceDefinition(r *http.Request) (any, error) {
	rd := r.PathValue("rd")
	deleteEmpty, _ := strconv.ParseBool(r.URL.Query().Get("delete_empty_resource_definition"))

	f.mu.Lock()
	defer f.mu.Unlock()

	if !slices.ContainsFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
		return exist.Name == rd
	}) {
		return nil, lapi.NotFoundError
	}

	f.resources = slices.DeleteFunc(f.resources, func(exist lapi.ResourceWithVolumes) bool {
		return exist.Name == rd
	})

	// The fake keeps no snapshots, so after the truncate the
	// resource-definition is always empty.
	if deleteEmpty {
		f.resourceDefinitions = slices.DeleteFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
			return exist.Name == rd
		})
		delete(f.volumeDefinitions, rd)
	}

	return nil, nil
}

// cloneKey identifies a clone operation by its source and target resource
// definition, mirroring the status endpoint's path.
func cloneKey(src, clone string) string {
	return src + "/" + clone
}

// cloneResourceDefinition copies a resource definition with its volume
// definitions and resources. Unlike a real controller, the fake completes the
// clone synchronously, so CloneStatus reports COMPLETE right away.
func (f *FakeLinstor) cloneResourceDefinition(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var req lapi.ResourceDefinitionCloneRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("error decoding resource definition clone request: %w", err)
	}

	src := r.PathValue("rd")

	f.mu.Lock()
	defer f.mu.Unlock()

	srcIdx := slices.IndexFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
		return exist.Name == src
	})
	if srcIdx < 0 {
		return nil, lapi.NotFoundError
	}
	srcRd := f.resourceDefinitions[srcIdx]

	if req.Name == "" {
		return nil, errors.New("clone request requires a name")
	}

	if slices.ContainsFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
		return exist.Name == req.Name
	}) {
		return nil, fmt.Errorf("resource definition '%s' already exists", req.Name)
	}

	// Grow the cloned volume definitions as requested. The sizes are applied
	// per volume number, 0 keeps the source size and shrinking is rejected.
	vds := slices.Clone(f.volumeDefinitions[src])
	for volNr, size := range req.VolumeSizes {
		if size == 0 {
			continue
		}
		if size < 0 {
			return nil, fmt.Errorf("invalid size %d for volume %d", size, volNr)
		}

		i := slices.IndexFunc(vds, func(vd lapi.VolumeDefinition) bool {
			return vd.VolumeNumber != nil && int(*vd.VolumeNumber) == volNr
		})
		if i < 0 {
			return nil, fmt.Errorf("resource definition '%s' has no volume %d", src, volNr)
		}
		if uint64(size) < vds[i].SizeKib {
			return nil, fmt.Errorf("cannot shrink volume %d from %d KiB to %d KiB during clone", volNr, vds[i].SizeKib, size)
		}

		vds[i].SizeKib = uint64(size)
	}

	clone := srcRd
	clone.Name = req.Name
	clone.ExternalName = req.ExternalName
	clone.Uuid = ""
	if req.ResourceGroup != "" {
		clone.ResourceGroupName = req.ResourceGroup
	}

	clone.Props = make(map[string]string, len(srcRd.Props))
	for k, v := range srcRd.Props {
		clone.Props[k] = v
	}
	for k, v := range req.OverrideProps {
		clone.Props[k] = v
	}
	for _, k := range req.DeleteProps {
		delete(clone.Props, k)
	}

	f.resourceDefinitions = append(f.resourceDefinitions, clone)
	f.volumeDefinitions[clone.Name] = vds

	for _, res := range f.resources {
		if res.Name != src {
			continue
		}

		res.Name = clone.Name
		res.Uuid = ""
		f.resources = append(f.resources, res)
	}

	f.clones[cloneKey(src, clone.Name)] = clonestatus.Complete

	return lapi.ResourceDefinitionCloneStarted{
		Location:   "/v1/resource-definitions/" + src + "/clone/" + clone.Name,
		SourceName: src,
		CloneName:  clone.Name,
	}, nil
}

func (f *FakeLinstor) cloneStatus(r *http.Request) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	status, ok := f.clones[cloneKey(r.PathValue("rd"), r.PathValue("clone"))]
	if !ok {
		return nil, lapi.NotFoundError
	}

	return lapi.ResourceDefinitionCloneStatus{Status: status}, nil
}

func (f *FakeLinstor) createVolumeDefinition(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var vdc lapi.VolumeDefinitionCreate
	err := json.NewDecoder(r.Body).Decode(&vdc)
	if err != nil {
		return nil, fmt.Errorf("error decoding volume definition create: %w", err)
	}

	rd := r.PathValue("rd")

	f.mu.Lock()
	defer f.mu.Unlock()

	if !slices.ContainsFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
		return exist.Name == rd
	}) {
		return nil, lapi.NotFoundError
	}

	vd := vdc.VolumeDefinition

	// LINSTOR assigns the next free volume number when none is requested.
	if vd.VolumeNumber == nil {
		next := int32(len(f.volumeDefinitions[rd]))
		vd.VolumeNumber = &next
	}

	if slices.ContainsFunc(f.volumeDefinitions[rd], func(exist lapi.VolumeDefinition) bool {
		return exist.VolumeNumber != nil && *exist.VolumeNumber == *vd.VolumeNumber
	}) {
		return nil, fmt.Errorf("volume definition '%s/%d' already exists", rd, *vd.VolumeNumber)
	}

	f.volumeDefinitions[rd] = append(f.volumeDefinitions[rd], vd)

	return nil, nil
}

func (f *FakeLinstor) getVolumeDefinitions(r *http.Request) (any, error) {
	rd := r.PathValue("rd")

	f.mu.Lock()
	defer f.mu.Unlock()

	if !slices.ContainsFunc(f.resourceDefinitions, func(exist lapi.ResourceDefinition) bool {
		return exist.Name == rd
	}) {
		return nil, lapi.NotFoundError
	}

	return slices.Clone(f.volumeDefinitions[rd]), nil
}

func (f *FakeLinstor) modifyVolumeDefinition(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var mod lapi.VolumeDefinitionModify
	err := json.NewDecoder(r.Body).Decode(&mod)
	if err != nil {
		return nil, fmt.Errorf("error decoding volume definition modification: %w", err)
	}

	rd := r.PathValue("rd")

	volNr, err := strconv.Atoi(r.PathValue("volNr"))
	if err != nil {
		return nil, fmt.Errorf("invalid volume number '%s': %w", r.PathValue("volNr"), err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	vds := f.volumeDefinitions[rd]
	for i := range vds {
		if vds[i].VolumeNumber == nil || int(*vds[i].VolumeNumber) != volNr {
			continue
		}

		if vds[i].Props == nil {
			vds[i].Props = make(map[string]string)
		}
		for k, v := range mod.OverrideProps {
			vds[i].Props[k] = v
		}
		for _, k := range mod.DeleteProps {
			delete(vds[i].Props, k)
		}

		return nil, nil
	}

	return nil, lapi.NotFoundError
}

func (f *FakeLinstor) evacuateNode(r *http.Request) (any, error) {
	node := r.PathValue("node")

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.nodes {
		if f.nodes[i].Name == node {
			if !slices.Contains(f.nodes[i].Flags, linstor.FlagEvacuate) {
				f.nodes[i].Flags = append(f.nodes[i].Flags, linstor.FlagEvacuate)
			}
			return nil, nil
		}
	}

	return nil, lapi.NotFoundError
}

func (f *FakeLinstor) restoreNode(r *http.Request) (any, error) {
	node := r.PathValue("node")

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.nodes {
		if f.nodes[i].Name == node {
			f.nodes[i].Flags = slices.DeleteFunc(f.nodes[i].Flags, func(s string) bool {
				return s == linstor.FlagEvacuate || s == linstor.FlagEvicted
			})
			return nil, nil
		}
	}

	return nil, lapi.NotFoundError
}

func (f *FakeLinstor) lostNode(r *http.Request) (any, error) {
	node := r.PathValue("node")

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.nodes {
		if f.nodes[i].Name == node {
			if f.nodes[i].ConnectionStatus != "OFFLINE" {
				return nil, fmt.Errorf("node '%s' is not 'OFFLINE'", node)
			}
		}
	}

	f.nodes = slices.DeleteFunc(f.nodes, func(n lapi.Node) bool {
		return n.Name == node
	})
	f.resources = slices.DeleteFunc(f.resources, func(n lapi.ResourceWithVolumes) bool {
		return n.NodeName == node
	})

	return nil, nil
}

func (f *FakeLinstor) viewResources(r *http.Request) (any, error) {
	r.URL.Query()

	resources := slices.Clone(f.resources)
	if len(r.URL.Query()["nodes"]) > 0 {
		resources = slices.DeleteFunc(resources, func(exist lapi.ResourceWithVolumes) bool {
			return !slices.Contains(r.URL.Query()["nodes"], exist.NodeName)
		})
	}

	return resources, nil
}

func (f *FakeLinstor) getStoragePools(r *http.Request) (any, error) {
	node := r.PathValue("node")

	f.mu.Lock()
	defer f.mu.Unlock()

	pools := make([]lapi.StoragePool, 0)
	for i := range f.storagePools {
		if f.storagePools[i].NodeName == node {
			pools = append(pools, f.storagePools[i])
		}
	}

	return pools, nil
}

func (f *FakeLinstor) getStoragePool(r *http.Request) (any, error) {
	node := r.PathValue("node")
	sp := r.PathValue("sp")

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.storagePools {
		if f.storagePools[i].NodeName == node && f.storagePools[i].StoragePoolName == sp {
			return f.storagePools[i], nil
		}
	}

	return nil, lapi.NotFoundError
}

func (f *FakeLinstor) createStoragePool(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var sp lapi.StoragePool
	err := json.NewDecoder(r.Body).Decode(&sp)
	if err != nil {
		return nil, fmt.Errorf("error decoding storage pool: %w", err)
	}

	sp.NodeName = r.PathValue("node")

	f.mu.Lock()
	defer f.mu.Unlock()

	if slices.ContainsFunc(f.storagePools, func(exist lapi.StoragePool) bool {
		return exist.NodeName == sp.NodeName && exist.StoragePoolName == sp.StoragePoolName
	}) {
		return nil, fmt.Errorf("storage pool '%s' already exists on node '%s'", sp.StoragePoolName, sp.NodeName)
	}

	f.storagePools = append(f.storagePools, sp)
	return nil, nil
}

func (f *FakeLinstor) modifyStoragePool(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var mod lapi.GenericPropsModify
	err := json.NewDecoder(r.Body).Decode(&mod)
	if err != nil {
		return nil, fmt.Errorf("error decoding storage pool modification: %w", err)
	}

	node := r.PathValue("node")
	sp := r.PathValue("sp")

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.storagePools {
		if f.storagePools[i].NodeName == node && f.storagePools[i].StoragePoolName == sp {
			if f.storagePools[i].Props == nil {
				f.storagePools[i].Props = make(map[string]string)
			}
			for k, v := range mod.OverrideProps {
				f.storagePools[i].Props[k] = v
			}
			for _, k := range mod.DeleteProps {
				delete(f.storagePools[i].Props, k)
			}
			return nil, nil
		}
	}

	return nil, lapi.NotFoundError
}

func (f *FakeLinstor) deleteStoragePool(r *http.Request) (any, error) {
	node := r.PathValue("node")
	sp := r.PathValue("sp")

	f.mu.Lock()
	defer f.mu.Unlock()

	f.storagePools = slices.DeleteFunc(f.storagePools, func(exist lapi.StoragePool) bool {
		return exist.NodeName == node && exist.StoragePoolName == sp
	})

	return nil, nil
}

func (f *FakeLinstor) createDevicePool(r *http.Request) (any, error) {
	defer r.Body.Close() //nolint:errcheck

	var psc lapi.PhysicalStorageCreate
	err := json.NewDecoder(r.Body).Decode(&psc)
	if err != nil {
		return nil, fmt.Errorf("error decoding physical storage create: %w", err)
	}

	node := r.PathValue("node")

	f.mu.Lock()
	defer f.mu.Unlock()

	f.devicePoolCreations++

	// CreateDevicePool also registers a storage pool when WithStoragePool is set.
	if psc.WithStoragePool.Name != "" {
		if slices.ContainsFunc(f.storagePools, func(exist lapi.StoragePool) bool {
			return exist.NodeName == node && exist.StoragePoolName == psc.WithStoragePool.Name
		}) {
			return nil, fmt.Errorf("storage pool '%s' already exists on node '%s'", psc.WithStoragePool.Name, node)
		}

		f.storagePools = append(f.storagePools, lapi.StoragePool{
			StoragePoolName: psc.WithStoragePool.Name,
			NodeName:        node,
			ProviderKind:    psc.ProviderKind,
			Props:           psc.WithStoragePool.Props,
		})
	}

	return nil, nil
}

// StoragePools returns a copy of the registered storage pools, for assertions in tests.
func (f *FakeLinstor) StoragePools() []lapi.StoragePool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.storagePools)
}

// DevicePoolCreations returns how many times CreateDevicePool was called, for assertions in tests.
func (f *FakeLinstor) DevicePoolCreations() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.devicePoolCreations
}

func (f *FakeLinstor) SetConnectionStatus(name string, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.nodes {
		if f.nodes[i].Name == name {
			f.nodes[i].ConnectionStatus = status
		}
	}
}
