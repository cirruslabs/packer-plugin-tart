package tartcmd

import (
	"context"
)

func Sync(ctx context.Context, args ...string) (string, error) {
	// Run Tart command asynchronously
	handle := Async(ctx, args, nil)

	// Wait for the Tart command to finish
	<-handle.Ctx().Done()

	return handle.Stdout(), handle.Err()
}
