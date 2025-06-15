packer {
  required_plugins {
    tart = {
      version = ">= 1.11.1"
      source  = "github.com/cirruslabs/tart"
    }
  }
}

# Example using RAW disk format (default)
source "tart-cli" "raw-disk" {
  from_ipsw    = "latest"
  vm_name      = "macos-raw-disk"
  cpu_count    = 4
  memory_gb    = 8
  disk_size_gb = 50
  disk_format  = "raw"  # This is the default, so it can be omitted
  ssh_username = "admin"
  ssh_password = "admin"
  ssh_timeout  = "120s"
}

# Example using ASIF disk format (requires macOS 26+ for creation)
source "tart-cli" "asif-disk" {
  from_ipsw    = "latest"
  vm_name      = "macos-asif-disk"
  cpu_count    = 4
  memory_gb    = 8
  disk_size_gb = 50
  disk_format  = "asif"  # High-performance sparse disk format
  ssh_username = "admin"
  ssh_password = "admin"
  ssh_timeout  = "120s"
}

build {
  sources = ["source.tart-cli.raw-disk", "source.tart-cli.asif-disk"]

  provisioner "shell" {
    inline = [
      "echo 'VM created with disk format: ${source.name}'",
      "diskutil info / | grep 'File System Personality'"
    ]
  }
}
