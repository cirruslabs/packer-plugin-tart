package tart

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func TartExec(args ...string) (string, error) {
	var out bytes.Buffer

	log.Printf("Executing tart: %#v", args)
	cmd := exec.Command("tart", args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()

	outString := strings.TrimSpace(out.String())

	if _, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("tart error: %s", outString)
	}

	return outString, err
}
