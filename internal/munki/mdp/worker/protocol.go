package worker

import (
	"context"
	"encoding/json"

	"github.com/coder/websocket"
)

const (
	messageHello      = "hello"
	messageDesiredSet = "desired_set"

	eventPackageSyncing = "package_syncing"
	eventPackageCurrent = "package_current"
	eventPackageError   = "package_error"
)

// serverMessage is an inbound message from Woodstar. hello carries the point
// identity; desired_set carries the full authoritative installer list.
type serverMessage struct {
	Type              string           `json:"type"`
	DistributionPoint pointIdentity    `json:"distribution_point"`
	Packages          []desiredPackage `json:"packages"`
}

// desiredPackage is one installer Woodstar wants this point to mirror. The
// worker fetches a download URL per job, so none is carried here.
type desiredPackage struct {
	PackageID int64  `json:"package_id"`
	Filename  string `json:"filename"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// packageEvent is the worker's outbound report for one package, emitted as each
// mirror job settles rather than as a batch after a full reconcile.
type packageEvent struct {
	Type      string `json:"type"`
	PackageID int64  `json:"package_id"`
	SHA256    string `json:"sha256,omitempty"`
	Error     string `json:"error,omitempty"`
}

func writeJSON(ctx context.Context, ws *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ws.Write(ctx, websocket.MessageText, data)
}
