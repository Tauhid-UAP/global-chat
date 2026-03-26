package chat

import "encoding/json"

type RequestWebSocketMessage struct {
	Type string `json:"Type"`
	Data json.RawMessage `json:"Data"`
}
