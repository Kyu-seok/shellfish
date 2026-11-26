#!/usr/bin/env node
// Runs the shellfish binary for this platform. npm installs it from the one
// optional dependency, @kyu-seok/shellfish-<platform>-<arch>, whose os and
// cpu match this machine (npm/build.mjs makes them).
"use strict";

const { spawnSync } = require("node:child_process");

const target = `${process.platform}-${process.arch}`;
const exe = process.platform === "win32" ? "shellfish.exe" : "shellfish";

let bin;
try {
  bin = require.resolve(`@kyu-seok/shellfish-${target}/bin/${exe}`);
} catch {
  console.error(
    `shellfish: no binary was installed for ${target}.\n` +
      "If you installed with --omit=optional or --no-optional, reinstall without it.\n" +
      "Other ways to install: https://github.com/Kyu-seok/shellfish#install",
  );
  process.exit(1);
}

const result = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(`shellfish: ${result.error.message}`);
  process.exit(1);
}
if (result.signal) {
  process.kill(process.pid, result.signal);
}
process.exit(result.status ?? 1);
