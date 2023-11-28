# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "Tart"
  description = "Create macOS and Linux VMs for Apple Silicon powered by Tart virtualization."
  identifier = "packer/cirruslabs/tart"
  component {
    type = "builder"
    name = "Tart"
    slug = "tart"
  }
}
