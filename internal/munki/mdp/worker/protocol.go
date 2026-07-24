package worker

import (
	"context"
	"encoding/json"

	"github.com/coder/websocket"
)

func writeJSON(ctx context.Context, ws *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ws.Write(ctx, websocket.MessageText, data)
}
