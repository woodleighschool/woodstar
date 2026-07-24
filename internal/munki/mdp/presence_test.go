package mdp

import (
	"reflect"
	"testing"
)

func TestPresenceRevealsRejectedWorkerAfterConnectedWorkerDisconnects(t *testing.T) {
	presence := NewPresence()
	connected := DistributionPointWorker{
		Compatible:      true,
		ProtocolVersion: new(1),
		BuildVersion:    "worker-current",
	}
	rejected := DistributionPointWorker{
		ProtocolVersion: new(2),
		BuildVersion:    "worker-next",
	}
	presence.Connect(1, connected)
	presence.Reject(1, rejected)

	worker, ok := presence.Worker(1)
	if !ok || !reflect.DeepEqual(worker, connected) {
		t.Fatalf("Worker = (%+v, %t), want connected worker %+v", worker, ok, connected)
	}

	presence.Disconnect(1)
	worker, ok = presence.Worker(1)
	if !ok || !reflect.DeepEqual(worker, rejected) {
		t.Fatalf("Worker = (%+v, %t), want rejected worker %+v", worker, ok, rejected)
	}
}

func TestPresenceRecordsRejectedWorkerWithoutConnection(t *testing.T) {
	presence := NewPresence()
	rejected := DistributionPointWorker{
		ProtocolVersion: new(0),
		BuildVersion:    "worker-old",
	}
	presence.Reject(1, rejected)

	worker, ok := presence.Worker(1)
	if !ok || !reflect.DeepEqual(worker, rejected) {
		t.Fatalf("Worker = (%+v, %t), want rejected worker %+v", worker, ok, rejected)
	}
}

func TestPresenceConnectionSupersedesOlderRejectedWorker(t *testing.T) {
	presence := NewPresence()
	presence.Reject(1, DistributionPointWorker{
		ProtocolVersion: new(0),
		BuildVersion:    "worker-old",
	})
	presence.Connect(1, DistributionPointWorker{
		Compatible:      true,
		ProtocolVersion: new(1),
		BuildVersion:    "worker-current",
	})

	presence.Disconnect(1)
	if worker, ok := presence.Worker(1); ok {
		t.Fatalf("Worker = %+v, want superseded rejection cleared", worker)
	}
}
