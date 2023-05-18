package tart

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

const tartCommand = "tart"

func PathInTartHome(elem ...string) string {
	if home := os.Getenv("TART_HOME"); home != "" {
		return path.Join(home, path.Join(elem...))
	}
	userHome, _ := os.UserHomeDir()
	return path.Join(userHome, ".tart", path.Join(elem...))
}

func TartExec(ctx context.Context, args ...string) (string, error) {
	var out bytes.Buffer

	log.Printf("Executing tart: %#v", args)
	cmd := exec.CommandContext(ctx, tartCommand, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()

	outString := strings.TrimSpace(out.String())

	if _, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("tart error: %s", outString)
	}

	return outString, err
}
