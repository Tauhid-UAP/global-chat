package chat

import (
	"time"
)

type WebsocketDurationControlConfig struct {
	PingInterval time.Duration
	PongDeadline time.Duration
	WriteDeadline time.Duration
}