package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/LINBIT/golinstor/client"
)

type resourceState struct {
	hasQuorum bool
	isWatched bool
}

type haResources struct {
	resources map[string]resourceState
	sync.Mutex
}

// LostResourceUser is a struct that exposes the "may promote" state of a DRBD resource
// If a resource may be promoted (i.e., may be switched to Primary) after some grace period, this usually means that its user (that had the resource promoted) failed. It could also happen that the user just terminated/gets rescheduled,... It is up to the user of this API to decide.
// This also means that the user (e.g., some k8s pod) needs to be restarted/rescheduled.
// The LostResourceUser is generic, it sends the names of resources that lost their user on the channel C.
type LostResourceUser struct {
	ctx              context.Context
	cancel           context.CancelFunc
	client           *client.Client
	mayPromoteStream *client.DRBDMayPromoteStream
	haResources      haResources
	initialDelay     time.Duration
	existingDelay    time.Duration

	C chan string // DRBD resource names of resources that may be promoted.
}

const (
	INITIAL_DELAY_DEFAULT  = 1 * time.Minute
	EXISTING_DELAY_DEFAULT = 45 * time.Second
)

// Option represents a configuration option of the LostResourceUser
type Option func(*LostResourceUser) error

// WithDelay sets the "initial delay" (for not yet seen resources) and the "existing delay" (for already known resources).
func WithDelay(initial, existing time.Duration) Option {
	return func(ha *LostResourceUser) error {
		ha.initialDelay = initial
		ha.existingDelay = existing
		return nil
	}
}

// NewLostResourceUser creates a new LostResourceUser. It takes a context, a Go LINSTOR client, and options as its input.
func NewLostResourceUser(ctx context.Context, client *client.Client, options ...Option) (*LostResourceUser, error) {
	// TODO: we only add to the map, we should have a GC that iterates over all the non-watched(?) and rms them.
	mayPromoteStream, err := client.Events.DRBDPromotion(ctx, "current")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	lr := &LostResourceUser{
		ctx:              ctx,
		cancel:           cancel,
		client:           client,
		mayPromoteStream: mayPromoteStream,
		// haResources:      haResources,
		haResources: haResources{
			resources: make(map[string]resourceState),
		},
		initialDelay:  INITIAL_DELAY_DEFAULT,
		existingDelay: EXISTING_DELAY_DEFAULT,
		C:             make(chan string),
	}

	for _, opt := range options {
		if err := opt(lr); err != nil {
			return nil, err
		}
	}

	go func() {
		for {
			select {
			case ev, ok := <-lr.mayPromoteStream.Events:
				if !ok {
					lr.Stop()
					close(lr.C)
					return
				}
				if !ev.MayPromote {
					continue
				}

				resName := ev.ResourceName

				watch, dur := lr.resShouldWatch(resName)
				if !watch {
					continue
				}
				go lr.watch(resName, dur)
			case <-lr.ctx.Done():
				lr.mayPromoteStream.Close()
				// would now receive in the event case, so
				close(lr.C)
				return
			}
		}
	}()

	return lr, nil
}

// Stop terminates all helper Go routines and closes the connection to the events stream.
func (rl *LostResourceUser) Stop() {
	rl.cancel()
}

func (lr *LostResourceUser) watch(resName string, dur time.Duration) {
	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		break
	case <-lr.ctx.Done():
		return
	}

	// reevaluate the current state
	ress, err := lr.client.Resources.GetAll(lr.ctx, resName)
	// here we might delete it, or reset isWatched
	lr.haResources.Lock()
	defer lr.haResources.Unlock()

	if err == client.NotFoundError {
		// looks like it got deleted. but anyways, nothing we can do, rm it from our dict
		delete(lr.haResources.resources, resName)
		return
	} else if err != nil {
		lr.Stop()
		return
	}

	oneMayPromote := false
	for _, r := range ress {
		if r.LayerObject.Type != client.DRBD {
			delete(lr.haResources.resources, resName)
			return
		}
		if r.LayerObject.Drbd.MayPromote {
			oneMayPromote = true
			break
		}
	}

	if oneMayPromote {
		lr.C <- resName
	}

	res := lr.haResources.resources[resName]
	// if we introduce a GC we need to check for ok here ^^
	// but currently all the deletes are here under this lock
	res.isWatched = false
	lr.haResources.resources[resName] = res
}

func (ha *LostResourceUser) resHasQuorum(resName string) (bool, error) {
	rd, err := ha.client.ResourceDefinitions.Get(ha.ctx, resName)
	if err != nil {
		return false, err
	}

	val, ok := rd.Props["DrbdOptions/Resource/quorum"]
	if !ok || val == "off" {
		return false, nil
	}

	return true, nil
}

func (ha *LostResourceUser) resShouldWatch(resName string) (bool, time.Duration) {
	long, short := ha.initialDelay, ha.existingDelay

	ha.haResources.Lock()
	defer ha.haResources.Unlock()

	res, ok := ha.haResources.resources[resName]

	if ok { // existing resource
		if res.isWatched {
			return false, 0
		}

		if !res.hasQuorum {
			return false, 0
		}

		res.isWatched = true
		ha.haResources.resources[resName] = res
		return true, short
	}

	// new resource
	hasQuorum, err := ha.resHasQuorum(resName)
	if err != nil {
		// hope for better times...
		return false, 0
	}
	// create the map entry
	ha.haResources.resources[resName] = resourceState{
		hasQuorum: hasQuorum,
		isWatched: hasQuorum, // not a typo, if it hasQuorum, we will watch it
	}
	if !hasQuorum {
		return false, 0
	}
	// new one with quorum, give it some time...
	return true, long
}
