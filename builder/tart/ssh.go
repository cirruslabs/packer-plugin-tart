package tart

import (
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TartMachineIP(vmName string) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		return TartExec("ip", "--wait", "120", vmName)
	}
}
