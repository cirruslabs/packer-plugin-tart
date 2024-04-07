package version

import (
	"github.com/hashicorp/packer-plugin-sdk/version"
)

var (
	// Version is the main version number that is being run at the moment.
	Version = "" // Will be set when building

	// PluginVersion is used by the plugin set to allow Packer to recognize
	// what version this plugin is.
	PluginVersion = version.NewRawVersion(Version)
)
