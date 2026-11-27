#!/bin/sh
# Installs the packages npm/build.mjs made in <out> the way a user would and
# runs the command. The project depends only on @kyu-seok/shellfish; npm picks
# the platform package itself from its optionalDependencies by os and cpu,
# served from the packed tarballs instead of the registry. The command must
# print exactly "shellfish v<version>" and pass a bad argument's exit 2
# through the launcher.
#
#   npm/smoke.sh <out> <version>
set -eu
out=$(cd "$1" && pwd)
want="shellfish v$2"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
for dir in "$out"/shellfish*; do
  (cd "$dir" && npm pack --silent --pack-destination "$tmp" >/dev/null)
done
cd "$tmp"
node -e '
const fs = require("node:fs");
const tgz = fs.readdirSync(".").filter((f) => f.endsWith(".tgz"));
const main = tgz.find((f) => /^kyu-seok-shellfish-\d/.test(f));
const overrides = {};
for (const f of tgz) {
  const m = f.match(/^kyu-seok-(shellfish-[a-z0-9]+-[a-z0-9]+)-\d/);
  if (m) overrides["@kyu-seok/" + m[1]] = "file:./" + f;
}
fs.writeFileSync("package.json", JSON.stringify({
  name: "smoke", private: true,
  dependencies: { "@kyu-seok/shellfish": "file:./" + main },
  overrides,
}));
'
npm install --no-audit --no-fund --loglevel=error >/dev/null
installed=$(ls node_modules/@kyu-seok)
target=$(node -p 'process.platform + "-" + process.arch')
if [ "$installed" != "$(printf 'shellfish\nshellfish-%s' "$target")" ]; then
  echo "npm smoke: installed $(echo $installed), want shellfish and shellfish-$target only" >&2
  exit 1
fi
got=$(./node_modules/.bin/shellfish --version) || {
  echo "npm smoke: --version failed" >&2
  exit 1
}
if [ "$got" != "$want" ]; then
  echo "npm smoke: --version printed '$got', want '$want'" >&2
  exit 1
fi
status=0
./node_modules/.bin/shellfish --no-such-flag >/dev/null 2>&1 || status=$?
if [ "$status" -ne 2 ]; then
  echo "npm smoke: a bad argument exited $status through the launcher, want 2" >&2
  exit 1
fi
echo "npm smoke: $got, from shellfish-$target"
