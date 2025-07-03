package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	linstor "github.com/LINBIT/golinstor"
	"github.com/LINBIT/golinstor/client"
)

func TestRcIs(t *testing.T) {
	cases := []struct {
		descr    string
		rc       int64
		mask     uint64
		expectIs bool
	}{{
		descr:    "exact match",
		rc:       -4611686018406153244,
		mask:     linstor.FailNotEnoughNodes,
		expectIs: true,
	}, {
		descr:    "wrong value",
		rc:       -4611686018406153244,
		mask:     linstor.FailInvldNodeName,
		expectIs: false,
	}}

	for _, c := range cases {
		r := client.ApiCallRc{RetCode: c.rc}

		is := r.Is(c.mask)

		if is && !c.expectIs {
			t.Errorf("Case \"%s\": mask unexpectedly matches", c.descr)
			t.Errorf("RetCode: %064b", uint64(c.rc))
			t.Errorf("Mask:    %064b", c.mask)
			t.Errorf("And:     %064b", uint64(c.rc)&c.mask)
		}
		if !is && c.expectIs {
			t.Errorf("Case \"%s\": mask unexpectedly does not match", c.descr)
			t.Errorf("RetCode: %064b", uint64(c.rc))
			t.Errorf("Mask:    %064b", c.mask)
			t.Errorf("And:     %064b", uint64(c.rc)&c.mask)
		}
	}
}

func TestApiCallErrorIs(t *testing.T) {
	var err error
	// Random, observed error
	err = client.ApiCallError{
		{
			RetCode: 4611686018481137428,
			Message: "Tie breaker marked for deletion",
			ObjRefs: map[string]string{
				"RscDfn": "test1",
				"Node":   "n2.example.com",
			},
		},
		{
			RetCode: 53739522,
			Message: "Node: n2.example.com, Resource: test1 preparing for deletion.",
			Details: "Node: n2.example.com, Resource: test1 UUID is: 54cb972d-13b1-4894-aa34-65c87b53e3af",
			ObjRefs: map[string]string{
				"RscDfn": "test1",
				"Node":   "n2.example.com",
			},
		},
		{
			RetCode: -4611686018373647389,
			Message: "No connection to satellite 'n2.example.com'",
			ObjRefs: map[string]string{
				"RscDfn": "test1",
				"Node":   "n2.example.com",
			},
		},
		{
			RetCode: 53739523,
			Message: "(n3.example.com) Resource 'test1' [DRBD] adjusted.",
			ObjRefs: map[string]string{
				"RscDfn": "test1",
				"Node":   "n2.example.com",
			},
		},
		{
			RetCode: 53739523,
			Message: "Preparing deletion of resource on 'n3.example.com'",
			ObjRefs: map[string]string{
				"RscDfn": "test1",
				"Node":   "n2.example.com",
			},
		},
		{
			RetCode: 53739523,
			Message: "(n1.example.com) Resource 'test1' [DRBD] adjusted.",
			ObjRefs: map[string]string{
				"RscDfn": "test1",
				"Node":   "n2.example.com",
			},
		},
		{
			RetCode: 53739523,
			Message: "Preparing deletion of resource on 'n1.example.com'",
			ObjRefs: map[string]string{
				"RscDfn": "test1",
				"Node":   "n2.example.com",
			},
		},
		{
			RetCode: -4611686018373647386,
			Message: "Deletion of resource 'test1' on node 'n2.example.com' failed due to an unhandled exception of type DelayedApiRcException. Exceptions have been converted to responses",
			Details: "Node: n2.example.com, Resource: test1",
			ErrorReportIds: []string{
				"68663093-00000-000003",
			},
			ObjRefs: map[string]string{
				"RscDfn": "test1",
				"Node":   "n2.example.com",
			},
		},
	}

	assert.True(t, client.IsApiCallError(err, linstor.FailNotConnected))
	assert.True(t, client.IsApiCallError(err, linstor.FailUnknownError))
	assert.False(t, client.IsApiCallError(err, linstor.FailNotEnoughNodes))
	assert.False(t, client.IsApiCallError(err, linstor.FailAccDeniedCommand))
}
