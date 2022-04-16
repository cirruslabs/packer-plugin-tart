packer {
  required_plugins {
    tart = {
      version = ">=v0.1.0"
      source  = "github.com/hashicorp/tart"
    }
  }
}

source "tart-my-builder" "foo-example" {
  mock = local.foo
}

source "tart-my-builder" "bar-example" {
  mock = local.bar
}

build {
  sources = [
    "source.tart-my-builder.foo-example",
  ]

  source "source.tart-my-builder.bar-example" {
    name = "bar"
  }

  provisioner "tart-my-provisioner" {
    only = ["tart-my-builder.foo-example"]
    mock = "foo: ${local.foo}"
  }

  provisioner "tart-my-provisioner" {
    only = ["tart-my-builder.bar"]
    mock = "bar: ${local.bar}"
  }

  post-processor "tart-my-post-processor" {
    mock = "post-processor mock-config"
  }
}
