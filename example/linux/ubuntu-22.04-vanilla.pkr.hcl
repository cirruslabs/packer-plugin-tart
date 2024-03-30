packer {
  required_plugins {
    tart = {
      version = ">= 1.10.0"
      source  = "github.com/cirruslabs/tart"
    }
  }
}

source "tart-cli" "tart" {
  from_iso     = ["cidata.iso", "ubuntu-22.04.4-live-server-arm64.iso"]
  vm_name      = "ubuntu-22.04-vanilla"
  cpu_count    = 4
  memory_gb    = 8
  disk_size_gb = 8
  boot_command = [
    # grub
    "<wait5s><enter>",
    # autoinstall prompt
    "<wait30s>yes<enter>",
  ]
  ssh_password = "ubuntu"
  ssh_username = "ubuntu"
  ssh_timeout  = "300s"
}

build {
  sources = ["source.tart-cli.tart"]

  provisioner "shell" {
    environment_vars = ["DEBIAN_FRONTEND=noninteractive"]
    execute_command = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
    inline = [
      "apt-get update",
      "apt-get install -y hello"
    ]
  }
}
