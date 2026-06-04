#!/usr/bin/env node
// sdk-install.js — install only mendixmodelsdk/src/gen (TypeScript definitions)
// without pulling any transitive dependencies (got, mobx, eventsource, etc.).
//
// Usage:  npm run sdk-install
// Result: node_modules/mendixmodelsdk/src/gen/ populated with .ts files only.

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const pkg = require(path.join(root, 'package.json'));
const version = pkg.dependencies.mendixmodelsdk;
const tgz = `mendixmodelsdk-${version}.tgz`;
const tgzPath = path.join(root, tgz);
const destDir = path.join(root, 'node_modules', 'mendixmodelsdk');

// Step 1: Download the package tarball only (no dependency resolution).
if (!fs.existsSync(tgzPath)) {
  console.log(`Downloading mendixmodelsdk@${version}...`);
  execSync(`npm pack mendixmodelsdk@${version} --silent`, { stdio: 'inherit', cwd: root });
}

// Step 2: Extract only package/src/gen/ from the tarball.
fs.mkdirSync(path.join(destDir, 'src', 'gen'), { recursive: true });
console.log(`Extracting src/gen/ ...`);
execSync(
  `tar -xzf "${tgz}" --strip-components=1 -C node_modules/mendixmodelsdk package/src/gen`,
  { stdio: 'inherit', cwd: root, shell: true }
);

// Step 3: Clean up tarball.
fs.unlinkSync(tgzPath);
console.log(`Done: node_modules/mendixmodelsdk/src/gen/ is ready (no transitive deps installed).`);
