package worker

import (
	"context"
	"encoding/json"

	"github.com/coder/websocket"
)

const (
	messageHello          = "hello"
	messageDesiredChanged = "desired_changed"
	messageState          = "state"

	packageStatusCurrent = "current"
	packageStatusError   = "error"
)

// serverMessage is an inbound message from Woodstar. hello and desired_changed
// share the desired-set payload; hello additionally carries the point identity.
type serverMessage struct {
	Type              string           `json:"type"`
	DistributionPoint pointIdentity    `json:"distribution_point"`
	Packages          []desiredPackage `json:"packages"`
}

// desiredPackage is one installer Woodstar wants this point to mirror.
type desiredPackage struct {
	PackageID   int64  `json:"package_id"`
	Filename    string `json:"filename"`
	SHA256      string `json:"sha256"`
	SizeBytes   int64  `json:"size_bytes"`
	DisplayName string `json:"display_name"`
	Version     string `json:"version"`
}

// stateMessage is the worker's outbound report, sent after every reconcile.
type stateMessage struct {
	Type     string            `json:"type"`
	Packages []reportedPackage `json:"packages"`
}

// reportedPackage is one desired package's mirror state.
type reportedPackage struct {
	PackageID int64  `json:"package_id"`
	SHA256    string `json:"sha256,omitempty"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

func writeJSON(ctx context.Context, ws *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ws.Write(ctx, websocket.MessageText, data)
}
