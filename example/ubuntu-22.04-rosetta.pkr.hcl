packer {
  required_plugins {
    tart = {
      version = ">= 0.6.2"
      source  = "github.com/cirruslabs/tart"
    }
  }
}

source "tart-cli" "tart" {
  vm_base_name        = "ubuntu-22.04-vanilla"
  vm_name             = "ubuntu-22.04-rosetta"
  headless            = true
  disable_vnc         = true
  rosetta             = "rosetta"
  ssh_password        = "ubuntu"
  ssh_username        = "ubuntu"
  ssh_timeout         = "120s"
}

build {
  sources = ["source.tart-cli.tart"]

  provisioner "shell" {
    inline = [
      "sudo apt update && sudo apt-get install -y binfmt-support",
    ]
  }

  provisioner "shell" {
    script      = "install_rosetta.sh"
    remote_path = "/tmp/install_rosetta.sh"
  }
}
