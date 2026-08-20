#!/bin/sh

set -eu

repo="https://github.com/dovixman/harness-profiles"
install_dir="${HP_INSTALL_DIR:-$HOME/.local/bin}"
version="${HP_VERSION:-latest}"

for command in curl grep tar; do
	if ! command -v "$command" >/dev/null 2>&1; then
		printf 'error: %s is required\n' "$command" >&2
		exit 1
	fi
done

case "$(uname -s)" in
	Darwin) os=Darwin ;;
	Linux) os=Linux ;;
	*)
		printf 'error: unsupported operating system: %s\n' "$(uname -s)" >&2
		exit 1
		;;
esac

case "$(uname -m)" in
	x86_64 | amd64) arch=x86_64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*)
		printf 'error: unsupported architecture: %s\n' "$(uname -m)" >&2
		exit 1
		;;
esac

case "$version" in
	latest) release_url="$repo/releases/latest/download" ;;
	v*) release_url="$repo/releases/download/$version" ;;
	*)
		printf 'error: HP_VERSION must be latest or a tag such as v1.2.3\n' >&2
		exit 1
		;;
esac

if [ -n "${HP_DOWNLOAD_BASE_URL:-}" ]; then
	release_url=$HP_DOWNLOAD_BASE_URL
fi

asset="hp_${os}_${arch}.tar.gz"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

curl -fsSL "$release_url/$asset" -o "$tmp_dir/$asset"
curl -fsSL "$release_url/checksums.txt" -o "$tmp_dir/checksums.txt"

checksum_line=$(grep "[[:space:]]$asset\$" "$tmp_dir/checksums.txt") || true
if [ -z "$checksum_line" ]; then
	printf 'error: checksum not found for %s\n' "$asset" >&2
	exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
	(cd "$tmp_dir" && printf '%s\n' "$checksum_line" | sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
	(cd "$tmp_dir" && printf '%s\n' "$checksum_line" | shasum -a 256 -c -)
else
	printf 'error: sha256sum or shasum is required\n' >&2
	exit 1
fi

tar -xzf "$tmp_dir/$asset" -C "$tmp_dir" hp

mkdir -p "$install_dir"
cp "$tmp_dir/hp" "$install_dir/hp"
chmod 0755 "$install_dir/hp"

printf 'hp installed at %s/hp\n' "$install_dir"

case ":$PATH:" in
	*":$install_dir:"*) ;;
	*) printf 'Add %s to your PATH to run hp.\n' "$install_dir" ;;
esac
