package client

import (
	"testing"

	linstor "github.com/LINBIT/golinstor"
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
		r := ApiCallRc{RetCode: c.rc}

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
