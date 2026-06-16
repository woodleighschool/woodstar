package adminapi

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/santa"
)

// The host detail contributors are adminapi's only job in host enrichment:
// attach a capability's loaded host state onto the composite response, and stay
// absent when the capability has no loader. The DB-backed state itself (santa
// config matching, munki observations) is tested in those packages.

type fakeSantaLoader struct{ state *santa.HostState }

func (f fakeSantaLoader) LoadHostState(context.Context, int64) (*santa.HostState, error) {
	return f.state, nil
}

type fakeMunkiLoader struct{ state *munki.HostState }

func (f fakeMunkiLoader) LoadHostState(context.Context, int64) (*munki.HostState, error) {
	return f.state, nil
}

func TestSantaContributorAttachesLoadedState(t *testing.T) {
	state := &santa.HostState{Version: "2026.2"}
	var body HostDetail
	if err := newSantaHostDetailContributor(fakeSantaLoader{state}).
		ContributeHostDetail(context.Background(), 7, &body); err != nil {
		t.Fatalf("contribute santa detail: %v", err)
	}
	if body.Santa != state {
		t.Fatalf("body.Santa = %+v, want loaded state", body.Santa)
	}
}

func TestMunkiContributorAttachesLoadedState(t *testing.T) {
	state := &munki.HostState{Version: "7.1.2"}
	var body HostDetail
	if err := newMunkiHostDetailContributor(fakeMunkiLoader{state}).
		ContributeHostDetail(context.Background(), 7, &body); err != nil {
		t.Fatalf("contribute munki detail: %v", err)
	}
	if body.Munki != state {
		t.Fatalf("body.Munki = %+v, want loaded state", body.Munki)
	}
}

func TestHostDetailContributorsAbsentWithoutLoader(t *testing.T) {
	if c := newSantaHostDetailContributor(nil); c != nil {
		t.Fatalf("santa contributor = %v, want nil without a loader", c)
	}
	if c := newMunkiHostDetailContributor(nil); c != nil {
		t.Fatalf("munki contributor = %v, want nil without a loader", c)
	}
}
