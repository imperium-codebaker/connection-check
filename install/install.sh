#!/bin/sh
# Downloads Connection Check for this computer and starts it.
# Usage: curl -fsSL <release>/install.sh | sh -s -- --site 'Name=host[:port][@airport]' [--site ...]
set -eu

repo="imperium-codebaker/connection-check"

case "$(uname -s)" in
  Darwin) os=darwin; dir="$HOME/Library/Application Support/connection-check" ;;
  Linux)  os=linux;  dir="${XDG_DATA_HOME:-$HOME/.local/share}/connection-check" ;;
  *) echo "Connection Check supports macOS, Linux and Windows." >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "Unsupported processor: $(uname -m)" >&2; exit 1 ;;
esac

mkdir -p "$dir"
bin="$dir/connection-check"
echo "Downloading Connection Check ($os/$arch)..."
curl -fsSL -o "$bin.tmp" "https://github.com/$repo/releases/latest/download/connection-check-$os-$arch"
chmod +x "$bin.tmp"
mv "$bin.tmp" "$bin"

# Linux: ordinary users can't send pings on many distributions; offer the one-time fix.
if [ "$os" = linux ]; then
  range=$(cat /proc/sys/net/ipv4/ping_group_range 2>/dev/null || echo "0 0")
  low=${range%%[[:space:]]*}
  high=${range##*[[:space:]]}
  gid=$(id -g)
  if [ "$gid" -lt "$low" ] || [ "$gid" -gt "$high" ]; then
    echo
    echo "This computer blocks pings from ordinary users, so some routers won't show."
    echo "To allow them (the same setting Fedora uses), your password is needed once."
    printf "Allow pings for ordinary users? [Y/n] "
    answer=$(head -n 1 </dev/tty || echo n)
    case "$answer" in
      n*|N*) echo "Continuing with a partial view." ;;
      *) echo 'net.ipv4.ping_group_range = 0 2147483647' | sudo tee /etc/sysctl.d/60-ping-group.conf >/dev/null \
           && sudo sysctl -q --system ;;
    esac
  fi
fi

exec "$bin" "$@" </dev/null
