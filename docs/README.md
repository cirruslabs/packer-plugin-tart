# Tart Plugin

## Requirements

Please install the latest version of [`tart`](https://github.com/cirruslabs/tart):

```console
brew install cirruslabs/cli/tart
```

## Installation

### Using pre-built releases

#### Using the `packer init` command

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

To install this plugin, copy and paste this code into your Packer configuration .
Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    parallels = {
      version = ">= 0.1.0"
      source  = "github.com/cirruslabs/tart"
    }
  }
}
```

#### Manual installation

You can find pre-built binary releases of the plugin [here](https://github.com/cirruslabs/packer-plugin-tart/releases).
Once you have downloaded the latest archive, uncompress it to retrieve the plugin binary file.
To install the plugin, please follow the Packer documentation on
[installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


#### From Source

If you prefer to build the plugin from its source code, clone the GitHub
repository locally and run the command `go build` from the root
directory. Upon successful compilation, a `packer-plugin-tart` plugin
binary file can be found in the root directory.
To install the compiled plugin, please follow the official Packer documentation
on [installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).
