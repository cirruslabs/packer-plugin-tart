
The `tart` builder is used to create macOS and Linux VMs for Apple Silicon powered by [Tart virtualization](https://github.com/cirruslabs/tart).

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
    gridscale = {
      version = ">= 1.6.1"
      source  = "github.com/cirruslabs/tart"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
$ packer plugins install github.com/cirruslabs/tart
```

### Components

#### Builders

- [tart](/packer/integrations/cirruslabs/tart/latest/components/builder/tart) - The builder is used to create macOS and Linux VMs for Apple Silicon powered by [Tart virtualization](https://github.com/cirruslabs/tart).

### Getting Started

Here is how you can install Tart, pull a remote macOS virtual machine and run it:

```bash
brew install cirruslabs/cli/tart
tart clone ghcr.io/cirruslabs/macos-ventura-vanilla:latest ventura-vanilla
tart run ventura-vanilla
```

