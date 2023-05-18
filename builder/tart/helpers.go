package tart

import (
	"os"
	"path"
)

func PathInTartHome(elem ...string) string {
	if home := os.Getenv("TART_HOME"); home != "" {
		return path.Join(home, path.Join(elem...))
	}
	userHome, _ := os.UserHomeDir()
	return path.Join(userHome, ".tart", path.Join(elem...))
}
