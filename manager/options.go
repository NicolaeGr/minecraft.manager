package manager

import "time"

type Options struct {
	WorkingDir string
	RunScript  string // default: "./run.sh"

	// RCON, parsed from server.properties if not set
	RCONHost     string
	RCONPort     int
	RCONPassword string

	// Dial & command timeouts
	DialTimeout    time.Duration
	CommandTimeout time.Duration
	// Reconnect/backoff
	MinBackoff time.Duration
	MaxBackoff time.Duration
}
