package fakelinstor_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	linstor "github.com/LINBIT/golinstor"
	lapi "github.com/LINBIT/golinstor/client"
	"github.com/LINBIT/golinstor/fakelinstor"
)

// newClient returns a golinstor client pointed at the given fake controller.
func newClient(t *testing.T, f *fakelinstor.FakeLinstor) *lapi.Client {
	t.Helper()

	u, err := url.Parse(f.Server.URL)
	require.NoError(t, err)

	c, err := lapi.NewClient(lapi.BaseURL(u), lapi.HTTPClient(f.Server.Client()))
	require.NoError(t, err)

	return c
}

// satelliteNode is a satellite with a network interface, which the fake reports
// as ONLINE.
func satelliteNode(name string) lapi.Node {
	return lapi.Node{
		Name:          name,
		Type:          linstor.ValNodeTypeStlt,
		NetInterfaces: []lapi.NetInterface{{Name: "default"}},
	}
}

// volumeDefinition returns the volume definition with the given number, failing
// the test if it is absent.
func volumeDefinition(t *testing.T, vds []lapi.VolumeDefinition, volNr int32) lapi.VolumeDefinition {
	t.Helper()

	for _, vd := range vds {
		if vd.VolumeNumber != nil && *vd.VolumeNumber == volNr {
			return vd
		}
	}

	t.Fatalf("volume definition %d not found in %+v", volNr, vds)
	return lapi.VolumeDefinition{}
}

func TestUnimplementedEndpoint(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)

	// Reconnect (PUT /v1/nodes/{node}/reconnect) is not implemented by the fake.
	err := c.Nodes.Reconnect(context.Background(), "node-a")
	require.Error(t, err)
	assert.NotErrorIs(t, err, lapi.NotFoundError)
	assert.ErrorContains(t, err, "not implemented")
}

func TestControllerVersion(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)

	version, err := c.Controller.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "1.0.0-fake", version.Version)
	assert.Equal(t, "fake-rest-api", version.RestApiVersion)
}

func TestNodeLifecycle(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	_, err := c.Nodes.Get(ctx, "node-a")
	assert.ErrorIs(t, err, lapi.NotFoundError)

	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-a")))

	node, err := c.Nodes.Get(ctx, "node-a")
	require.NoError(t, err)
	assert.Equal(t, "node-a", node.Name)
	assert.Equal(t, "ONLINE", node.ConnectionStatus)

	err = c.Nodes.Create(ctx, satelliteNode("node-a"))
	assert.ErrorContains(t, err, "already exists")

	require.NoError(t, c.Nodes.Delete(ctx, "node-a"))

	_, err = c.Nodes.Get(ctx, "node-a")
	assert.ErrorIs(t, err, lapi.NotFoundError)
}

func TestNodeEvacuateAndRestore(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-a")))

	require.NoError(t, c.Nodes.Evacuate(ctx, "node-a", lapi.NodeEvacuate{}))
	node, err := c.Nodes.Get(ctx, "node-a")
	require.NoError(t, err)
	assert.Contains(t, node.Flags, linstor.FlagEvacuate)

	// Evacuating again is idempotent: the flag is only added once.
	require.NoError(t, c.Nodes.Evacuate(ctx, "node-a", lapi.NodeEvacuate{}))
	node, err = c.Nodes.Get(ctx, "node-a")
	require.NoError(t, err)
	assert.Equal(t, []string{linstor.FlagEvacuate}, node.Flags)

	require.NoError(t, c.Nodes.Restore(ctx, "node-a", lapi.NodeRestore{}))
	node, err = c.Nodes.Get(ctx, "node-a")
	require.NoError(t, err)
	assert.NotContains(t, node.Flags, linstor.FlagEvacuate)

	err = c.Nodes.Evacuate(ctx, "missing", lapi.NodeEvacuate{})
	assert.ErrorIs(t, err, lapi.NotFoundError)

	err = c.Nodes.Restore(ctx, "missing", lapi.NodeRestore{})
	assert.ErrorIs(t, err, lapi.NotFoundError)
}

func TestNodeLost(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-a")))

	// An ONLINE node cannot be declared lost.
	err := c.Nodes.Lost(ctx, "node-a")
	assert.ErrorContains(t, err, "OFFLINE")

	f.SetConnectionStatus("node-a", "OFFLINE")

	require.NoError(t, c.Nodes.Lost(ctx, "node-a"))
	_, err = c.Nodes.Get(ctx, "node-a")
	assert.ErrorIs(t, err, lapi.NotFoundError)
}

func TestResourceGroups(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	groups, err := c.ResourceGroups.GetAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, groups)

	require.NoError(t, c.ResourceGroups.Create(ctx, lapi.ResourceGroup{Name: "rg1"}))

	groups, err = c.ResourceGroups.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "rg1", groups[0].Name)

	err = c.ResourceGroups.Create(ctx, lapi.ResourceGroup{Name: "rg1"})
	assert.ErrorContains(t, err, "already exists")
}

func TestResourceDefinitions(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	create := lapi.ResourceDefinitionCreate{
		ResourceDefinition: lapi.ResourceDefinition{Name: "res1"},
	}
	require.NoError(t, c.ResourceDefinitions.Create(ctx, create))

	rds, err := c.ResourceDefinitions.GetAll(ctx, lapi.RDGetAllRequest{})
	require.NoError(t, err)
	require.Len(t, rds, 1)
	assert.Equal(t, "res1", rds[0].Name)

	err = c.ResourceDefinitions.Create(ctx, create)
	assert.ErrorContains(t, err, "already exists")

	require.NoError(t, c.ResourceDefinitions.Delete(ctx, "res1"))

	rds, err = c.ResourceDefinitions.GetAll(ctx, lapi.RDGetAllRequest{})
	require.NoError(t, err)
	assert.Empty(t, rds)
}

func TestResourceDefinitionTruncate(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	err := c.ResourceDefinitions.Truncate(ctx, "missing")
	assert.ErrorIs(t, err, lapi.NotFoundError)

	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-a")))
	require.NoError(t, c.ResourceDefinitions.Create(ctx, lapi.ResourceDefinitionCreate{
		ResourceDefinition: lapi.ResourceDefinition{Name: "res1"},
	}))
	require.NoError(t, c.Resources.Create(ctx, lapi.ResourceCreate{
		Resource: lapi.Resource{Name: "res1", NodeName: "node-a"},
	}))

	// Truncate removes the resources but keeps the resource definition.
	require.NoError(t, c.ResourceDefinitions.Truncate(ctx, "res1"))

	_, err = c.Resources.Get(ctx, "res1", "node-a")
	assert.ErrorIs(t, err, lapi.NotFoundError)

	rds, err := c.ResourceDefinitions.GetAll(ctx, lapi.RDGetAllRequest{})
	require.NoError(t, err)
	require.Len(t, rds, 1)
	assert.Equal(t, "res1", rds[0].Name)

	// With DeleteEmptyResourceDefinition the now-empty resource definition
	// is deleted as well.
	require.NoError(t, c.Resources.Create(ctx, lapi.ResourceCreate{
		Resource: lapi.Resource{Name: "res1", NodeName: "node-a"},
	}))
	require.NoError(t, c.ResourceDefinitions.Truncate(ctx, "res1", &lapi.ResourceDefinitionTruncateOpts{
		DeleteEmptyResourceDefinition: true,
	}))

	rds, err = c.ResourceDefinitions.GetAll(ctx, lapi.RDGetAllRequest{})
	require.NoError(t, err)
	assert.Empty(t, rds)
}

func TestResources(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-a")))
	require.NoError(t, c.ResourceDefinitions.Create(ctx, lapi.ResourceDefinitionCreate{
		ResourceDefinition: lapi.ResourceDefinition{Name: "res1"},
	}))

	// A resource cannot be created for an unknown resource definition...
	err := c.Resources.Create(ctx, lapi.ResourceCreate{
		Resource: lapi.Resource{Name: "missing", NodeName: "node-a"},
	})
	assert.ErrorIs(t, err, lapi.NotFoundError)

	// ...nor on an unknown node.
	err = c.Resources.Create(ctx, lapi.ResourceCreate{
		Resource: lapi.Resource{Name: "res1", NodeName: "missing"},
	})
	assert.ErrorIs(t, err, lapi.NotFoundError)

	require.NoError(t, c.Resources.Create(ctx, lapi.ResourceCreate{
		Resource: lapi.Resource{Name: "res1", NodeName: "node-a"},
	}))

	resource, err := c.Resources.Get(ctx, "res1", "node-a")
	require.NoError(t, err)
	assert.Equal(t, "res1", resource.Name)
	assert.Equal(t, "node-a", resource.NodeName)

	view, err := c.Resources.GetResourceView(ctx)
	require.NoError(t, err)
	require.Len(t, view, 1)
	assert.Equal(t, "res1", view[0].Name)

	// The resource view can be filtered by node.
	view, err = c.Resources.GetResourceView(ctx, &lapi.ListOpts{Node: []string{"other"}})
	require.NoError(t, err)
	assert.Empty(t, view)

	view, err = c.Resources.GetResourceView(ctx, &lapi.ListOpts{Node: []string{"node-a"}})
	require.NoError(t, err)
	assert.Len(t, view, 1)

	// A node still holding a resource cannot be deleted.
	err = c.Nodes.Delete(ctx, "node-a")
	assert.ErrorContains(t, err, "still contains resource")

	// Deleting the resource definition also removes its resources.
	require.NoError(t, c.ResourceDefinitions.Delete(ctx, "res1"))
	_, err = c.Resources.Get(ctx, "res1", "node-a")
	assert.ErrorIs(t, err, lapi.NotFoundError)

	require.NoError(t, c.Nodes.Delete(ctx, "node-a"))
}

func TestStoragePools(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-a")))

	pools, err := c.Nodes.GetStoragePools(ctx, "node-a")
	require.NoError(t, err)
	assert.Empty(t, pools)

	_, err = c.Nodes.GetStoragePool(ctx, "node-a", "pool1")
	assert.ErrorIs(t, err, lapi.NotFoundError)

	require.NoError(t, c.Nodes.CreateStoragePool(ctx, "node-a", lapi.StoragePool{
		StoragePoolName: "pool1",
		ProviderKind:    lapi.LVM_THIN,
	}))

	pool, err := c.Nodes.GetStoragePool(ctx, "node-a", "pool1")
	require.NoError(t, err)
	assert.Equal(t, "pool1", pool.StoragePoolName)
	assert.Equal(t, "node-a", pool.NodeName)
	assert.Equal(t, lapi.LVM_THIN, pool.ProviderKind)

	// Storage pools are scoped to their node.
	pools, err = c.Nodes.GetStoragePools(ctx, "node-a")
	require.NoError(t, err)
	require.Len(t, pools, 1)

	pools, err = c.Nodes.GetStoragePools(ctx, "other")
	require.NoError(t, err)
	assert.Empty(t, pools)

	err = c.Nodes.CreateStoragePool(ctx, "node-a", lapi.StoragePool{
		StoragePoolName: "pool1",
		ProviderKind:    lapi.LVM_THIN,
	})
	assert.ErrorContains(t, err, "already exists")

	// Modify overrides and deletes properties.
	require.NoError(t, c.Nodes.ModifyStoragePool(ctx, "node-a", "pool1", lapi.GenericPropsModify{
		OverrideProps: lapi.OverrideProps{"Foo": "bar", "Baz": "qux"},
	}))
	pool, err = c.Nodes.GetStoragePool(ctx, "node-a", "pool1")
	require.NoError(t, err)
	assert.Equal(t, "bar", pool.Props["Foo"])
	assert.Equal(t, "qux", pool.Props["Baz"])

	require.NoError(t, c.Nodes.ModifyStoragePool(ctx, "node-a", "pool1", lapi.GenericPropsModify{
		DeleteProps: lapi.DeleteProps{"Foo"},
	}))
	pool, err = c.Nodes.GetStoragePool(ctx, "node-a", "pool1")
	require.NoError(t, err)
	assert.NotContains(t, pool.Props, "Foo")
	assert.Equal(t, "qux", pool.Props["Baz"])

	err = c.Nodes.ModifyStoragePool(ctx, "node-a", "missing", lapi.GenericPropsModify{})
	assert.ErrorIs(t, err, lapi.NotFoundError)

	require.NoError(t, c.Nodes.DeleteStoragePool(ctx, "node-a", "pool1"))
	_, err = c.Nodes.GetStoragePool(ctx, "node-a", "pool1")
	assert.ErrorIs(t, err, lapi.NotFoundError)
}

func TestCreateDevicePool(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-a")))

	require.NoError(t, c.Nodes.CreateDevicePool(ctx, "node-a", lapi.PhysicalStorageCreate{
		ProviderKind: lapi.LVM,
		DevicePaths:  []string{"/dev/sdb"},
		WithStoragePool: lapi.PhysicalStorageStoragePoolCreate{
			Name:  "pool1",
			Props: map[string]string{"k": "v"},
		},
	}))

	assert.Equal(t, 1, f.DevicePoolCreations())

	pools := f.StoragePools()
	require.Len(t, pools, 1)
	assert.Equal(t, "pool1", pools[0].StoragePoolName)
	assert.Equal(t, "node-a", pools[0].NodeName)
	assert.Equal(t, lapi.LVM, pools[0].ProviderKind)
	assert.Equal(t, "v", pools[0].Props["k"])

	// The registered storage pool is also visible through the regular API.
	pool, err := c.Nodes.GetStoragePool(ctx, "node-a", "pool1")
	require.NoError(t, err)
	assert.Equal(t, "pool1", pool.StoragePoolName)

	// Registering the same storage pool again fails, but the call is still
	// counted as a device-pool creation attempt.
	err = c.Nodes.CreateDevicePool(ctx, "node-a", lapi.PhysicalStorageCreate{
		ProviderKind:    lapi.LVM,
		DevicePaths:     []string{"/dev/sdc"},
		WithStoragePool: lapi.PhysicalStorageStoragePoolCreate{Name: "pool1"},
	})
	assert.ErrorContains(t, err, "already exists")
	assert.Equal(t, 2, f.DevicePoolCreations())
}

func TestNodesGetAll(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	nodes, err := c.Nodes.GetAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, nodes)

	nodeA := satelliteNode("node-a")
	nodeA.Props = map[string]string{"Aux/zone": "east"}
	require.NoError(t, c.Nodes.Create(ctx, nodeA))
	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-b")))

	nodes, err = c.Nodes.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)

	// Filter by node name.
	nodes, err = c.Nodes.GetAll(ctx, &lapi.ListOpts{Node: []string{"node-a"}})
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "node-a", nodes[0].Name)

	// Filter by property.
	nodes, err = c.Nodes.GetAll(ctx, &lapi.ListOpts{Prop: []string{"Aux/zone=east"}})
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "node-a", nodes[0].Name)

	nodes, err = c.Nodes.GetAll(ctx, &lapi.ListOpts{Prop: []string{"Aux/zone=west"}})
	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestVolumeDefinitions(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	// Volume definitions require an existing resource definition.
	err := c.ResourceDefinitions.CreateVolumeDefinition(ctx, "missing", lapi.VolumeDefinitionCreate{})
	assert.ErrorIs(t, err, lapi.NotFoundError)

	require.NoError(t, c.ResourceDefinitions.Create(ctx, lapi.ResourceDefinitionCreate{
		ResourceDefinition: lapi.ResourceDefinition{Name: "res1"},
	}))

	// The volume number is assigned automatically when not requested.
	require.NoError(t, c.ResourceDefinitions.CreateVolumeDefinition(ctx, "res1", lapi.VolumeDefinitionCreate{}))

	vds, err := c.ResourceDefinitions.GetVolumeDefinitions(ctx, "res1")
	require.NoError(t, err)
	require.Len(t, vds, 1)
	require.NotNil(t, vds[0].VolumeNumber)
	assert.Equal(t, int32(0), *vds[0].VolumeNumber)

	// An explicit volume number is honored...
	two := int32(2)
	require.NoError(t, c.ResourceDefinitions.CreateVolumeDefinition(ctx, "res1", lapi.VolumeDefinitionCreate{
		VolumeDefinition: lapi.VolumeDefinition{VolumeNumber: &two},
	}))

	// ...and reusing it is rejected.
	err = c.ResourceDefinitions.CreateVolumeDefinition(ctx, "res1", lapi.VolumeDefinitionCreate{
		VolumeDefinition: lapi.VolumeDefinition{VolumeNumber: &two},
	})
	assert.ErrorContains(t, err, "already exists")

	// Modify overrides and deletes properties.
	require.NoError(t, c.ResourceDefinitions.ModifyVolumeDefinition(ctx, "res1", 2, lapi.VolumeDefinitionModify{
		GenericPropsModify: lapi.GenericPropsModify{OverrideProps: lapi.OverrideProps{"Foo": "bar"}},
	}))
	vds, err = c.ResourceDefinitions.GetVolumeDefinitions(ctx, "res1")
	require.NoError(t, err)
	assert.Equal(t, "bar", volumeDefinition(t, vds, 2).Props["Foo"])

	require.NoError(t, c.ResourceDefinitions.ModifyVolumeDefinition(ctx, "res1", 2, lapi.VolumeDefinitionModify{
		GenericPropsModify: lapi.GenericPropsModify{DeleteProps: lapi.DeleteProps{"Foo"}},
	}))
	vds, err = c.ResourceDefinitions.GetVolumeDefinitions(ctx, "res1")
	require.NoError(t, err)
	assert.NotContains(t, volumeDefinition(t, vds, 2).Props, "Foo")

	// Modifying an unknown volume definition fails.
	err = c.ResourceDefinitions.ModifyVolumeDefinition(ctx, "res1", 9, lapi.VolumeDefinitionModify{})
	assert.ErrorIs(t, err, lapi.NotFoundError)

	// Deleting the resource definition also removes its volume definitions.
	require.NoError(t, c.ResourceDefinitions.Delete(ctx, "res1"))
	_, err = c.ResourceDefinitions.GetVolumeDefinitions(ctx, "res1")
	assert.ErrorIs(t, err, lapi.NotFoundError)
}

func TestResourceDefinitionsWithVolumeDefinitions(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	require.NoError(t, c.ResourceDefinitions.Create(ctx, lapi.ResourceDefinitionCreate{
		ResourceDefinition: lapi.ResourceDefinition{Name: "res1", Props: map[string]string{"Aux/tier": "gold"}},
	}))
	require.NoError(t, c.ResourceDefinitions.Create(ctx, lapi.ResourceDefinitionCreate{
		ResourceDefinition: lapi.ResourceDefinition{Name: "res2", Props: map[string]string{"Aux/tier": "silver"}},
	}))
	require.NoError(t, c.ResourceDefinitions.CreateVolumeDefinition(ctx, "res1", lapi.VolumeDefinitionCreate{}))

	// Volume definitions are omitted unless explicitly requested.
	rds, err := c.ResourceDefinitions.GetAll(ctx, lapi.RDGetAllRequest{})
	require.NoError(t, err)
	require.Len(t, rds, 2)
	for _, rd := range rds {
		assert.Empty(t, rd.VolumeDefinitions, "volume definitions should be omitted for %s", rd.Name)
	}

	rds, err = c.ResourceDefinitions.GetAll(ctx, lapi.RDGetAllRequest{WithVolumeDefinitions: true})
	require.NoError(t, err)
	for _, rd := range rds {
		if rd.Name == "res1" {
			assert.Len(t, rd.VolumeDefinitions, 1)
		}
	}

	// Filter by resource definition name.
	rds, err = c.ResourceDefinitions.GetAll(ctx, lapi.RDGetAllRequest{ResourceDefinitions: []string{"res2"}})
	require.NoError(t, err)
	require.Len(t, rds, 1)
	assert.Equal(t, "res2", rds[0].Name)

	// Filter by property.
	rds, err = c.ResourceDefinitions.GetAll(ctx, lapi.RDGetAllRequest{Props: []string{"Aux/tier=gold"}})
	require.NoError(t, err)
	require.Len(t, rds, 1)
	assert.Equal(t, "res1", rds[0].Name)
}

func TestResourcesGetAll(t *testing.T) {
	f := fakelinstor.New()
	defer f.Server.Close()

	c := newClient(t, f)
	ctx := context.Background()

	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-a")))
	require.NoError(t, c.Nodes.Create(ctx, satelliteNode("node-b")))
	require.NoError(t, c.ResourceDefinitions.Create(ctx, lapi.ResourceDefinitionCreate{
		ResourceDefinition: lapi.ResourceDefinition{Name: "res1"},
	}))
	require.NoError(t, c.Resources.Create(ctx, lapi.ResourceCreate{
		Resource: lapi.Resource{Name: "res1", NodeName: "node-a"},
	}))
	require.NoError(t, c.Resources.Create(ctx, lapi.ResourceCreate{
		Resource: lapi.Resource{Name: "res1", NodeName: "node-b"},
	}))

	all, err := c.Resources.GetAll(ctx, "res1")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// Filter by node.
	filtered, err := c.Resources.GetAll(ctx, "res1", &lapi.ListOpts{Node: []string{"node-a"}})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, "node-a", filtered[0].NodeName)

	// An unknown resource definition simply has no resources.
	none, err := c.Resources.GetAll(ctx, "missing")
	require.NoError(t, err)
	assert.Empty(t, none)
}
