package Session

import (
	"context"
	"time"
)

// SyncAdapter only handles communication with a remote session store.
// Serialization and deserialization are owned by the framework.
type SyncAdapter interface {
	// Pull retrieves one serialized session from the remote store.
	// found distinguishes a missing session from a remote-store failure.
	Pull(ctx context.Context, sessionID string) (payload []byte, found bool, err error)

	// Push sends one framework-serialized session to the remote store.
	// ttl is the remaining lifetime that should be applied remotely.
	Push(ctx context.Context, sessionID string, payload []byte, ttl time.Duration) error
}
