
The `tart` builder is used to create macOS and Linux VMs for Apple Silicon powered by [Tart](https://tart.run/).

Here are some highlights of Tart:

- Tart uses Apple's own `Virtualization.Framework` for [near-native performance](https://browser.geekbench.com/v5/cpu/compare/14966395?baseline=14966339).
- Push/Pull virtual machines from any OCI-compatible container registry.
- Built-in CI integration.
- Use this Tart Packer Plugin to automate VM creation.

### Installation

To install this plugin, copy and paste this code into your Packer configuration, then run [`packer init`](https://developer.hashicorp.com/packer/docs/commands/init).

```hcl
packer {
  required_plugins {
    tart = {
      version = ">= 1.11.1"
      source  = "github.com/cirruslabs/tart"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
$ packer plugins install github.com/cirruslabs/tart
```

#### Installing Tart

The plugin requires a working installation of Tart. To install and verify your installation:

```bash
brew install cirruslabs/cli/tart
tart clone ghcr.io/cirruslabs/macos-ventura-vanilla:latest ventura-vanilla
tart run ventura-vanilla
```

Or follow the quick start guide [here](https://tart.run/quick-start/).

### Components

#### Builders

- [tart](/packer/integrations/cirruslabs/tart/latest/components/builder/tart) - The builder is used to create macOS and Linux VMs for Apple Silicon powered by [Tart](https://tart.run/).

