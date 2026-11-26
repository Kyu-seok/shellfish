// Builds the npm packages for one version from GoReleaser's binaries:
//
//   node npm/build.mjs <version> <dist> <out>
//
// <dist> holds GoReleaser's artifacts.json and the binaries it lists. <out>
// gets one package per platform, @kyu-seok/shellfish-<platform>-<arch>, whose
// os and cpu make npm install only the one matching the machine, and the main
// package, @kyu-seok/shellfish, which lists them all as optionalDependencies
// and runs the installed one (npm/shellfish/bin/shellfish.js).
import fs from "node:fs";
import path from "node:path";

const [version, dist, out] = process.argv.slice(2);
if (!version || !dist || !out) {
  console.error("usage: node npm/build.mjs <version> <dist> <out>");
  process.exit(2);
}

const root = path.dirname(path.dirname(new URL(import.meta.url).pathname));
const main = JSON.parse(fs.readFileSync(path.join(root, "npm/shellfish/package.json"), "utf8"));

// Go's names for each platform and architecture, as Node names them.
const platforms = { darwin: "darwin", linux: "linux", windows: "win32" };
const archs = { amd64: "x64", arm64: "arm64" };
const wanted = ["darwin-arm64", "darwin-x64", "linux-arm64", "linux-x64", "win32-x64"];

const artifacts = JSON.parse(fs.readFileSync(path.join(dist, "artifacts.json"), "utf8"));
const binaries = artifacts.filter((a) => a.type === "Binary");

fs.rmSync(out, { recursive: true, force: true });
const optional = {};
for (const b of binaries) {
  const target = `${platforms[b.goos]}-${archs[b.goarch]}`;
  const name = `@kyu-seok/shellfish-${target}`;
  const dir = path.join(out, `shellfish-${target}`);
  const exe = b.goos === "windows" ? "shellfish.exe" : "shellfish";
  // artifacts.json paths are relative to where GoReleaser ran, with dist/ first.
  const from = path.join(dist, path.relative("dist", b.path));
  fs.mkdirSync(path.join(dir, "bin"), { recursive: true });
  fs.copyFileSync(from, path.join(dir, "bin", exe));
  fs.chmodSync(path.join(dir, "bin", exe), 0o755);
  fs.copyFileSync(path.join(root, "LICENSE"), path.join(dir, "LICENSE"));
  write(path.join(dir, "package.json"), {
    name,
    version,
    description: `The shellfish binary for ${target}. Install @kyu-seok/shellfish instead.`,
    homepage: main.homepage,
    repository: main.repository,
    license: main.license,
    os: [platforms[b.goos]],
    cpu: [archs[b.goarch]],
    files: ["bin"],
    preferUnplugged: true,
    publishConfig: main.publishConfig,
  });
  optional[name] = version;
}

const missing = wanted.filter((t) => !optional[`@kyu-seok/shellfish-${t}`]);
if (missing.length > 0 || Object.keys(optional).length !== wanted.length) {
  console.error(`npm/build.mjs: want binaries for ${wanted.join(", ")}; missing ${missing.join(", ") || "none"}, found ${Object.keys(optional).join(", ")}`);
  process.exit(1);
}

const dir = path.join(out, "shellfish");
fs.cpSync(path.join(root, "npm/shellfish"), dir, { recursive: true });
fs.chmodSync(path.join(dir, "bin/shellfish.js"), 0o755);
for (const f of ["README.md", "LICENSE"]) {
  fs.copyFileSync(path.join(root, f), path.join(dir, f));
}
write(path.join(dir, "package.json"), { ...main, version, optionalDependencies: optional });
console.log(`npm/build.mjs: built @kyu-seok/shellfish ${version} for ${wanted.join(", ")} in ${out}`);

function write(file, json) {
  fs.writeFileSync(file, JSON.stringify(json, null, 2) + "\n");
}
