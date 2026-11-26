#!/bin/sh
# Publishes the packages npm/build.mjs made in <out>: every platform package
# first, then the main package, so its optionalDependencies already exist
# when it appears. A prerelease version (one with a "-", like 0.2.0-rc.1) goes
# out under the dist-tag next, so a plain install never picks it up. Extra
# arguments go to npm publish, like --dry-run.
#
#   npm/publish.sh <out> [npm publish arguments...]
set -eu
out=$1
shift
version=$(node -p "require('./$out/shellfish/package.json').version")
case $version in
*-*) set -- --tag next "$@" ;;
esac
for dir in "$out"/shellfish-* "$out/shellfish"; do
  npm publish "$dir" --access public "$@"
done
