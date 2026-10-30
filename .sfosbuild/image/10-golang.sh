#!/bin/sh
# Official Go for rpmbuild of the daemon (native to SFOS_ARCH).
set -eu

GO_VERSION=1.27.3

case "${SFOS_ARCH}" in
aarch64)  GO_TARBALL_ARCH=linux-arm64  GO_SHA256=a06e99309dbcbdca1a8ccd2282c3ee146e55930886e26b4f1193269587165f49 ;;
armv7hl)  GO_TARBALL_ARCH=linux-armv6l GO_SHA256=e1379a2fe77bd30fa29833074388247e7c65416e09279f746f20de2d5cf4dfea ;;
i486)     GO_TARBALL_ARCH=linux-386    GO_SHA256=d6f4e11dd7157030d7ac11c51c8f73201a2db932d593c3106b1da11cc32bfdf8 ;;
*)
	echo "10-golang.sh: unknown SFOS_ARCH=${SFOS_ARCH:-}" >&2
	exit 1
	;;
esac

zypper --non-interactive in --force-resolution curl ca-certificates
tarball="go${GO_VERSION}.${GO_TARBALL_ARCH}.tar.gz"
tmp="/tmp/${tarball}"
curl -fL --retry 3 -o "$tmp" "https://go.dev/dl/${tarball}"
echo "${GO_SHA256}  ${tmp}" | sha256sum -c
rm -rf /usr/local/go
tar -C /usr/local -xzf "$tmp"
rm -f "$tmp"
ln -sf /usr/local/go/bin/go /usr/bin/go
go version
