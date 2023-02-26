package tart

import (
	"os"
	"path"
)

// packersdk.TartVMArtifact implementation
type TartVMArtifact struct {
	VMName string
	// StateData should store data such as GeneratedData
	// to be shared with post-processors
	StateData map[string]interface{}
}

func (*TartVMArtifact) BuilderId() string {
	return BuilderId
}

func (a *TartVMArtifact) Files() []string {
	baseDir := a.vmDirPath()
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return []string{}
	}
	result := make([]string, len(entries))
	for index, entry := range entries {
		result[index] = path.Join(entry.Name())
	}
	return result
}

func (a *TartVMArtifact) Id() string {
	return a.VMName
}

func (a *TartVMArtifact) String() string {
	return a.VMName
}

func (a *TartVMArtifact) State(name string) interface{} {
	return a.StateData[name]
}

func (a *TartVMArtifact) Destroy() error {
	return os.RemoveAll(a.vmDirPath())
}

func (a *TartVMArtifact) vmDirPath() string {
	return PathInTartHome("vms", a.VMName)
}
