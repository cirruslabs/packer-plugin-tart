# Packer Plugin Tart

The `Tart` multi-component plugin can be used with HashiCorp [Packer](https://www.packer.io)
to create custom macOS images. For the full list of available features for this plugin see [docs](https://developer.hashicorp.com/packer/integrations/cirruslabs/tart/latest/components/builder/tart).

> [!IMPORTANT]
>
> **macOS 15 (Sequoia)**
>
> In case you've upgraded and encountering an issue below:
>
> ```
> ssh: connect to host [...] port 22: No route to host
> ```
>
> This is likely related to the [newly introduced "Local Network" permission](https://developer.apple.com/documentation/technotes/tn3179-understanding-local-network-privacy) on macOS Sequoia and the fact that GitLab Runner's binary might have no `LC_UUID` identifier, which is critical for the local network privacy mechanism.
>
> We've already [submitted a fix to Packer](https://github.com/hashicorp/packer/pull/13214), but the next release is planned for Jan, 2025.
>
> For now, you can use a [nightly version](https://github.com/hashicorp/packer/releases/tag/nightly) of Packer.
>
> We have [an issue](https://github.com/cirruslabs/packer-plugin-tart/issues/79) for this, so don't hesitate to ask any questions and subscribe to get updates.

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
    tart = {
      version = ">= 1.11.1"
      source  = "github.com/cirruslabs/tart"
    }
  }
}
```


#### Manual installation

You can find pre-built binary releases of the plugin [here](https://github.com/cirruslabs/packer-plugin-tart/releases).
Once you have downloaded the latest archive corresponding to your target OS,
uncompress it to retrieve the plugin binary file corresponding to your platform.
To install the plugin, please follow the Packer documentation on
[installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


### From Sources

If you prefer to build the plugin from sources, clone the GitHub repository
locally and run the command `go build` from the root
directory. Upon successful compilation, a `packer-plugin-tart` plugin
binary file can be found in the root directory.
To install the compiled plugin, please follow the official Packer documentation
on [installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


### Configuration

For more information on how to configure the plugin, please read the
documentation located on the [HashiCorp's website](https://developer.hashicorp.com/packer/plugins/builders/tart).

## Contributing

* If you think you've found a bug in the code or you have a question regarding
  the usage of this software, please reach out to us by opening an issue in
  this GitHub repository.
* Contributions to this project are welcome: if you want to add a feature or a
  fix a bug, please do so by opening a Pull Request in this GitHub repository.
  In case of feature contribution, we kindly ask you to open an issue to
  discuss it beforehand.
