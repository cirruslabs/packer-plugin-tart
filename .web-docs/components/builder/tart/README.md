Type: `tart`

The `tart` builder is used to create macOS and Linux VMs for Apple Silicon powered by [Tart](https://tart.run/).

Make sure you have followed the [installation instructions](/packer/integrations/cirruslabs/tart#installation) before continuing.

## Example Usage

Here is a basic example of creating a macOS virtual machine:

```hcl
variable "macos_version" {
  type =  string
  default = "ventura"
}

source "tart-cli" "tart" {
  vm_base_name = "${var.macos_version}-vanilla"
  vm_name      = "${var.macos_version}-base"
  cpu_count    = 4
  memory_gb    = 8
  disk_size_gb = 50
  ssh_username = "admin"
  ssh_password = "admin"
  ssh_timeout  = "120s"
}

build {
  sources = ["source.tart-cli.tart"]

  provisioner "shell" {
    inline = ["echo 'Disabling spotlight indexing...'", "sudo mdutil -a -i off"]
  }

  # more provisioners
}
```

For more advanced examples, please refer to the [`example/` directory](https://github.com/cirruslabs/packer-plugin-tart/tree/main/example) and the [macOS](https://github.com/cirruslabs/macos-image-templates) and [Linux](https://github.com/cirruslabs/linux-image-templates) Packer templates for [Cirrus CI](https://cirrus-ci.org/).


## Configuration Reference

- `vm_name` (string) - The name of the VM to create (only when `from_ipsw`, `from_iso` or `vm_base_name` are used) and run.

#### Optional:

- `allow_insecure` (boolean) — When cloning the image, connect to the OCI registry via an insecure HTTP protocol.
- `pull_concurrency` (boolean) — Amount of layers to pull concurrently from an OCI registry when pulling the image. Default is 4 for Tart 2.0.0+.
- `cpu_count` (number) - Amount of virtual CPUs to use for the new VM. Overrides `tart create` default value when using `from_ipsw` and `from_iso` and VM settings when using `vm_base_name`.
- `create_grace_time` (duration string | ex: "1h5m2s") — Time to wait after finishing the installation process. Can be used to work around the issue when Virtualization.Framework's installation process is still running in the background for some time after `tart create` had already finished.
- `disk_size_gb` — Disk size in GB to use for the new VM. Overrides `tart create` default value when using `from_ipsw` and `from_iso` and VM settings when using `vm_base_name`.
- `recovery_partition` (string) — Behavior with the respect to the macOS recovery partition. Set to `"delete"` to delete it (allows for disk resize, decreases the VM image size, but prevents the updates from working), `"keep"` to keep it as is (prevents the disk resize, increases the VM image size, but allows the updates to proceed) or `"relocate"` to move the partition to the end of disk (allows for disk resize, increases the VM image size, but allows the updates to proceed). Defaults to `""` which is currently equivalent to `"delete"`, but this may change in the future.
- `display` (string) — VM display resolution in a format of `<width>x<height>` (e.g. `1200x800`) to use for the new VM. Overrides `tart create` default value when using `from_ipsw` and `from_iso` and VM settings when using `vm_base_name`.
- `from_ipsw` (string) - Location of an IPSW file to initialize a macOS virtual machine from. Can be either an absolute path to a file on disk, URL to fetch a remote file or `latest`. Mutually exclusive with `from_iso` and `vm_base_name`.
- `from_iso` (list(string)) - Location of the ISO files to initialize a Linux virtual machine from. All values should represent an absolute path to a file on disk. Mutually exclusive with `from_ipsw` and `vm_base_name`.
- `headless` (bool) - Whether to show graphics interface of a VM. Useful for debugging the `boot_command`.
- `memory_gb` (number) - Amount of unified memory in GB to use for the new VM. Overrides `tart create` default value when using `from_ipsw` and `from_iso` and VM settings when using `vm_base_name`.
- `recovery` (bool) — Whether to boot the VM in recovery mode. Useful for disabling the System Integrity Protection automatically for the already created VMs.
- `rosetta` (string) - Whether to enable Rosetta support of a Linux guest VM. Useful for running non-arm64 binaries in the guest VM. A common used value is `rosetta`, for further details and explanation run `tart run --help`.
- `run_extra_args` (list(string)) - Extra arguments to pass to `tart run` command. For example, you can enable bridged networking by specifying `--net-bridged=en0`.
- `ip_extra_args` (list(string)) - Extra arguments to pass to `tart ip` command. For example, you can use a different resolver in case of bridged network by specifying `--resolver=arp`.
- `vm_base_name` (string) - The name of the VM to be used for the initial cloning. Can be either a local VM or a remote VM that will be pulled from a registry. Mutually exclusive with `from_ipsw` and `from_iso`.

### SSH connection configuration

- `ssh_username` (string) - Username to use for the communication over SSH to run provision steps.
- `ssh_password` (string) - Password to use for the communication over SSH to run provision steps.

### HTTP server configuration

<!-- Code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; DO NOT EDIT MANUALLY -->

Packer will create an http server serving `http_directory` when it is set, a
random free port will be selected and the architecture of the directory
referenced will be available in your builder.

Example usage from a builder:

```
wget http://{{ .HTTPIP }}:{{ .HTTPPort }}/foo/bar/preseed.cfg
```

<!-- End of code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; -->


#### Optional:

<!-- Code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; DO NOT EDIT MANUALLY -->

- `http_directory` (string) - Path to a directory to serve using an HTTP server. The files in this
  directory will be available over HTTP that will be requestable from the
  virtual machine. This is useful for hosting kickstart files and so on.
  By default this is an empty string, which means no HTTP server will be
  started. The address and port of the HTTP server will be available as
  variables in `boot_command`. This is covered in more detail below.

- `http_content` (map[string]string) - Key/Values to serve using an HTTP server. `http_content` works like and
  conflicts with `http_directory`. The keys represent the paths and the
  values contents, the keys must start with a slash, ex: `/path/to/file`.
  `http_content` is useful for hosting kickstart files and so on. By
  default this is empty, which means no HTTP server will be started. The
  address and port of the HTTP server will be available as variables in
  `boot_command`. This is covered in more detail below.
  Example:
  ```hcl
    http_content = {
      "/a/b"     = file("http/b")
      "/foo/bar" = templatefile("${path.root}/preseed.cfg", { packages = ["nginx"] })
    }
  ```

- `http_port_min` (int) - These are the minimum and maximum port to use for the HTTP server
  started to serve the `http_directory`. Because Packer often runs in
  parallel, Packer will choose a randomly available port in this range to
  run the HTTP server. If you want to force the HTTP server to be on one
  port, make this minimum and maximum port the same. By default the values
  are `8000` and `9000`, respectively.

- `http_port_max` (int) - HTTP Port Max

- `http_bind_address` (string) - This is the bind address for the HTTP server. Defaults to 0.0.0.0 so that
  it will work with any network interface.

<!-- End of code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; -->


### Boot configuration

<!-- Code generated from the comments of the BootConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

The boot configuration is very important: `boot_command` specifies the keys
to type when the virtual machine is first booted in order to start the OS
installer. This command is typed after boot_wait, which gives the virtual
machine some time to actually load.

The boot_command is an array of strings. The strings are all typed in
sequence. It is an array only to improve readability within the template.

There are a set of special keys available. If these are in your boot
command, they will be replaced by the proper key:

-   `<bs>` - Backspace

-   `<del>` - Delete

-   `<enter> <return>` - Simulates an actual "enter" or "return" keypress.

-   `<esc>` - Simulates pressing the escape key.

-   `<tab>` - Simulates pressing the tab key.

-   `<f1> - <f12>` - Simulates pressing a function key.

-   `<up> <down> <left> <right>` - Simulates pressing an arrow key.

-   `<spacebar>` - Simulates pressing the spacebar.

-   `<insert>` - Simulates pressing the insert key.

-   `<home> <end>` - Simulates pressing the home and end keys.

  - `<pageUp> <pageDown>` - Simulates pressing the page up and page down
    keys.

-   `<menu>` - Simulates pressing the Menu key.

-   `<leftAlt> <rightAlt>` - Simulates pressing the alt key.

-   `<leftCtrl> <rightCtrl>` - Simulates pressing the ctrl key.

-   `<leftShift> <rightShift>` - Simulates pressing the shift key.

-   `<leftSuper> <rightSuper>` - Simulates pressing the ⌘ or Windows key.

  - `<wait> <wait5> <wait10>` - Adds a 1, 5 or 10 second pause before
    sending any additional keys. This is useful if you have to generally
    wait for the UI to update before typing more.

  - `<waitXX>` - Add an arbitrary pause before sending any additional keys.
    The format of `XX` is a sequence of positive decimal numbers, each with
    optional fraction and a unit suffix, such as `300ms`, `1.5h` or `2h45m`.
    Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`. For
    example `<wait10m>` or `<wait1m20s>`.

  - `<XXXOn> <XXXOff>` - Any printable keyboard character, and of these
    "special" expressions, with the exception of the `<wait>` types, can
    also be toggled on or off. For example, to simulate ctrl+c, use
    `<leftCtrlOn>c<leftCtrlOff>`. Be sure to release them, otherwise they
    will be held down until the machine reboots. To hold the `c` key down,
    you would use `<cOn>`. Likewise, `<cOff>` to release.

  - `{{ .HTTPIP }} {{ .HTTPPort }}` - The IP and port, respectively of an
    HTTP server that is started serving the directory specified by the
    `http_directory` configuration parameter. If `http_directory` isn't
    specified, these will be blank!

-   `{{ .Name }}` - The name of the VM.

Example boot command. This is actually a working boot command used to start an
CentOS 6.4 installer:

In JSON:

```json
"boot_command": [

	   "<tab><wait>",
	   " ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/centos6-ks.cfg<enter>"
	]

```

In HCL2:

```hcl
boot_command = [

	   "<tab><wait>",
	   " ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/centos6-ks.cfg<enter>"
	]

```

The example shown below is a working boot command used to start an Ubuntu
12.04 installer:

In JSON:

```json
"boot_command": [

	"<esc><esc><enter><wait>",
	"/install/vmlinuz noapic ",
	"preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg ",
	"debian-installer=en_US auto locale=en_US kbd-chooser/method=us ",
	"hostname={{ .Name }} ",
	"fb=false debconf/frontend=noninteractive ",
	"keyboard-configuration/modelcode=SKIP keyboard-configuration/layout=USA ",
	"keyboard-configuration/variant=USA console-setup/ask_detect=false ",
	"initrd=/install/initrd.gz -- <enter>"

]
```

In HCL2:

```hcl
boot_command = [

	"<esc><esc><enter><wait>",
	"/install/vmlinuz noapic ",
	"preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg ",
	"debian-installer=en_US auto locale=en_US kbd-chooser/method=us ",
	"hostname={{ .Name }} ",
	"fb=false debconf/frontend=noninteractive ",
	"keyboard-configuration/modelcode=SKIP keyboard-configuration/layout=USA ",
	"keyboard-configuration/variant=USA console-setup/ask_detect=false ",
	"initrd=/install/initrd.gz -- <enter>"

]
```

For more examples of various boot commands, see the sample projects from our
[community templates page](https://packer.io/community-tools#templates).

<!-- End of code generated from the comments of the BootConfig struct in bootcommand/config.go; -->


#### Optional:

<!-- Code generated from the comments of the BootConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

- `boot_keygroup_interval` (duration string | ex: "1h5m2s") - Time to wait after sending a group of key pressses. The value of this
  should be a duration. Examples are `5s` and `1m30s` which will cause
  Packer to wait five seconds and one minute 30 seconds, respectively. If
  this isn't specified, a sensible default value is picked depending on
  the builder type.

- `boot_wait` (duration string | ex: "1h5m2s") - The time to wait after booting the initial virtual machine before typing
  the `boot_command`. The value of this should be a duration. Examples are
  `5s` and `1m30s` which will cause Packer to wait five seconds and one
  minute 30 seconds, respectively. If this isn't specified, the default is
  `10s` or 10 seconds. To set boot_wait to 0s, use a negative number, such
  as "-1s"

- `boot_command` ([]string) - This is an array of commands to type when the virtual machine is first
  booted. The goal of these commands should be to type just enough to
  initialize the operating system installer. Special keys can be typed as
  well, and are covered in the section below on the boot command. If this
  is not specified, it is assumed the installer will start itself.

<!-- End of code generated from the comments of the BootConfig struct in bootcommand/config.go; -->


### VNC configuration

<!-- Code generated from the comments of the VNCConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

The boot command "typed" character for character over a VNC connection to
the machine, simulating a human actually typing the keyboard.

Keystrokes are typed as separate key up/down events over VNC with a default
100ms delay. The delay alleviates issues with latency and CPU contention.
You can tune this delay on a per-builder basis by specifying
"boot_key_interval" in your Packer template.

<!-- End of code generated from the comments of the VNCConfig struct in bootcommand/config.go; -->


#### Optional:

<!-- Code generated from the comments of the VNCConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

- `disable_vnc` (bool) - Whether to create a VNC connection or not. A boot_command cannot be used
  when this is true. Defaults to false.

- `boot_key_interval` (duration string | ex: "1h5m2s") - Time in ms to wait between each key press

<!-- End of code generated from the comments of the VNCConfig struct in bootcommand/config.go; -->
