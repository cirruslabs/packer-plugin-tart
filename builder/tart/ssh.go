package tart

import (
	"context"
	"fmt"
	"strings"
)

func TartMachineIP(ctx context.Context, vmName string, ipExtraArgs []string) (string, error) {
	run := func(args ...string) (string, error) {
		out, err := TartExec(ctx, nil, args...)
		if err != nil {
			return "", err
		}
		if out == "" {
			return "", fmt.Errorf("tart ip returned empty output (args=%s)", strings.Join(args, " "))
		}
		return out, nil
	}

	// If the user provided explicit extra args (e.g. --resolver arp), honor them.
	if len(ipExtraArgs) > 0 {
		ipArgs := []string{"ip", "--wait", "120", vmName}
		ipArgs = append(ipArgs, ipExtraArgs...)
		return run(ipArgs...)
	}

	// Best-effort probing to avoid long stalls and handle transient empty output.
	probes := [][]string{
		{"ip", "--wait", "1", vmName},
		{"ip", "--wait", "1", "--resolver", "agent", vmName},
		{"ip", "--wait", "1", "--resolver", "arp", vmName},
	}
	for _, args := range probes {
		out, err := run(args...)
		if err == nil {
			return out, nil
		}
	}

	// Fall back to the default resolver with a longer wait.
	return run("ip", "--wait", "120", vmName)
}
