package tart

import (
	"context"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"packer-plugin-tart/builder/tart/tartcmd"
)

func TartMachineIP(ctx context.Context, vmName string) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		return tartcmd.Sync(ctx, "ip", "--wait", "120", vmName)
	}
}
