#!/bin/zsh

set -eu

if [ ! -f ubuntu-22.04.4-live-server-arm64.iso ]; then
	echo "Downloading image..."
	curl -fSLO https://cdimage.ubuntu.com/releases/22.04/release/ubuntu-22.04.4-live-server-arm64.iso
fi

if [ -d cidata ]; then
	rm -rf cidata*
fi

mkdir cidata
touch cidata/meta-data
cat << 'EOF' > cidata/user-data
#cloud-config
autoinstall:
  version: 1
  identity:
    hostname: ubuntu-server
    # the password is ubuntu, needs to be encoded to /etc/shadow format
    password: "$6$exDY1mhS4KUYCE/2$zmn9ToZwTKLhCw.b4/b.ZRTIZM30JZ4QrOQ2aOXJ8yk96xpcCof0kxKwuX1kqLG/ygbJ1f8wxED22bTL4F46P0"
    username: ubuntu

  storage:
    swap:
      size: 0
    layout:
      name: direct

  ssh:
    install-server: true
    allow-pw: true

  late-commands:
    - "echo 'ubuntu ALL=(ALL) NOPASSWD: ALL' > /target/etc/sudoers.d/ubuntu-nopasswd"

  shutdown: "reboot"
EOF
hdiutil makehybrid -o cidata.iso cidata -joliet -iso
