packer {
  required_plugins {
    tart = {
      version = ">= 1.10.0"
      source  = "github.com/cirruslabs/tart"
    }
  }
}

source "tart-cli" "tart" {
  vm_base_name        = "ubuntu-22.04-vanilla"
  vm_name             = "ubuntu-22.04-hello"
  cpu_count           = 4
  memory_gb           = 8
  disk_size_gb        = 10
  headless            = true
  disable_vnc         = true
  ssh_password        = "ubuntu"
  ssh_username        = "ubuntu"
  ssh_timeout         = "120s"
}

build {
  sources = ["source.tart-cli.tart"]

  # resize the disk
  provisioner "shell" {
    inline = [
      "sudo growpart /dev/vda 2",
      "sudo resize2fs /dev/vda2"
    ]
  }

  provisioner "shell" {
    inline = [
      "hello"
    ]
  }
}
