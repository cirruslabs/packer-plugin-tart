package tart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/shell-local/localexec"
)

const tartCommand = "tart"

type VMInfo struct {
	Disk int64 `json:"disk"`
}

func PathInTartHome(elem ...string) string {
	if home := os.Getenv("TART_HOME"); home != "" {
		return path.Join(home, path.Join(elem...))
	}
	userHome, _ := os.UserHomeDir()
	return path.Join(userHome, ".tart", path.Join(elem...))
}

func TartExec(ctx context.Context, ui packer.Ui, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, tartCommand, args...)

	if ui != nil {
		return "", localexec.RunAndStream(cmd, ui, []string{})
	} else {
		log.Printf("Executing tart: %#v", args)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()

		outString := strings.TrimSpace(out.String())

		if _, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("tart error: %s", outString)
		}

		return outString, err
	}
}

func TartVMInfo(ctx context.Context, ui packer.Ui, vmName string) (*VMInfo, error) {
	output, err := TartExec(ctx, ui, "get", "--format", "json", vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to run \"tart get --format json %s\": %w", vmName, err)
	}

	var vmInfo VMInfo

	if err := json.Unmarshal([]byte(output), &vmInfo); err != nil {
		return nil, fmt.Errorf("failed to parse \"tart get --format json %s\"'s output: %w", vmName, err)
	}

	return &vmInfo, nil
}
