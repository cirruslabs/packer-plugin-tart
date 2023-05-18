package tart

import (
	"context"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TartMachineIP(ctx context.Context, vmName string) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		return TartExec(ctx, "ip", "--wait", "120", vmName)
	}
}
