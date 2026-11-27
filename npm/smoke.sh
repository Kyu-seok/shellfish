#!/bin/sh
# Installs the packages npm/build.mjs made in <out> the way a user would, for
# this machine's platform, and runs the command: the launcher must find the
# binary, run it, and pass its exit status through.
#
#   npm/smoke.sh <out>
set -eu
out=$(cd "$1" && pwd)
target=$(node -p 'process.platform + "-" + process.arch')
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
for dir in "$out/shellfish" "$out/shellfish-$target"; do
  (cd "$dir" && npm pack --silent --pack-destination "$tmp" >/dev/null)
done
cd "$tmp"
npm init -y >/dev/null
npm install --no-audit --no-fund --loglevel=error ./*.tgz >/dev/null
version=$(./node_modules/.bin/shellfish --version)
case $version in
"shellfish v"*) echo "npm smoke: $version" ;;
*)
  echo "npm smoke: --version printed '$version'" >&2
  exit 1
  ;;
esac
status=0
./node_modules/.bin/shellfish --no-such-flag >/dev/null 2>&1 || status=$?
if [ "$status" -ne 2 ]; then
  echo "npm smoke: a bad argument exited $status through the launcher, want 2" >&2
  exit 1
fi
