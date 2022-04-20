packer {
  required_plugins {
    tart = {
      version = ">= 0.1.0"
      source  = "github.com/cirruslabs/tart"
    }
  }
}

build {
  sources = ["source.tart.example"]
}
