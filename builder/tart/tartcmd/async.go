package tartcmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"log"
	"os/exec"
	"strings"
)

var ErrOK = errors.New("tart command terminated without error")

type Handle struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	cmd *exec.Cmd

	stdout *lockedBuffer
	stderr *lockedBuffer
}

func Async(ctx context.Context, args []string, env []string) *Handle {
	log.Printf("Executing tart: %#v", args)

	ctx, cancel := context.WithCancelCause(ctx)

	cmd := exec.CommandContext(ctx, tartCommandName, args...)

	if len(env) != 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}

	var stderr lockedBuffer
	cmd.Stderr = &stderr
	var stdout lockedBuffer
	cmd.Stdout = &stdout

	go func() {
		convertErr := func(err error) error {
			if err == nil {
				return ErrOK
			}

			if errors.Is(err, exec.ErrNotFound) {
				return fmt.Errorf("%w: %s command not found in PATH, make sure Tart is installed",
					ErrTartNotFound, tartCommandName)
			}

			if _, ok := err.(*exec.ExitError); ok {
				// Tart command failed, redefine the error
				// to be the Tart-specific output
				return fmt.Errorf("%w: %q", ErrTartFailed, firstNonEmptyLine(stderr.String(), stdout.String()))
			}

			return err
		}

		cancel(convertErr(cmd.Run()))
	}()

	return &Handle{
		ctx:    ctx,
		cancel: cancel,
		cmd:    cmd,
		stderr: &stderr,
		stdout: &stdout,
	}
}

func (handle *Handle) Ctx() context.Context {
	return handle.ctx
}

func (handle *Handle) Err() error {
	if err := context.Cause(handle.ctx); err != nil && !errors.Is(err, ErrOK) {
		return err
	}

	return nil
}

func (handle *Handle) Stdout() string {
	return strings.TrimSpace(handle.stdout.String())
}

const handleKey = "tart-cmd-handle"

func GetHandle(state multistep.StateBag) *Handle {
	return state.Get(handleKey).(*Handle)
}

func SetHandle(state multistep.StateBag, tartCmdHandle *Handle) {
	state.Put(handleKey, tartCmdHandle)
}
