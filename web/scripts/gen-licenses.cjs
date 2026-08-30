#!/usr/bin/env node
// Generate third-party license notices for all installed dependencies.
// Walks the pnpm virtual store (node_modules/.pnpm) so transitive deps are
// covered regardless of the package manager's layout.
// Output: licenses.txt in the web/ directory (consumed by the Docker build).
const fs = require('node:fs');
const path = require('node:path');

const webDir = path.join(__dirname, '..');
const storeDir = path.join(webDir, 'node_modules', '.pnpm');
const licenseFileRe = /^(licen[cs]e|copying|notice)(\.|$)/i;

const entries = fs.existsSync(storeDir) ? fs.readdirSync(storeDir) : [];
const seen = new Set();
let out = '';

for (const entry of entries.sort()) {
  // entry looks like "axios@1.20.0" or "@element-plus+icons-vue@2.3.2"
  const base = path.join(storeDir, entry, 'node_modules');
  if (!fs.existsSync(base)) continue;
  for (const dir of fs.readdirSync(base)) {
    const pkgDirs = dir.startsWith('@')
      ? fs.readdirSync(path.join(base, dir)).map((n) => path.join(base, dir, n))
      : [path.join(base, dir)];
    for (const pkgDir of pkgDirs) {
      const pkgJsonPath = path.join(pkgDir, 'package.json');
      if (!fs.existsSync(pkgJsonPath)) continue;
      const pkg = JSON.parse(fs.readFileSync(pkgJsonPath, 'utf8'));
      const key = `${pkg.name}@${pkg.version}`;
      if (seen.has(key)) continue;
      seen.add(key);
      out += '='.repeat(72) + '\n' + key + '\nLicense: ' + (pkg.license || 'UNKNOWN') + '\n' + '-'.repeat(72) + '\n';
      for (const f of fs.readdirSync(pkgDir)) {
        if (licenseFileRe.test(f) && fs.statSync(path.join(pkgDir, f)).isFile()) {
          out += fs.readFileSync(path.join(pkgDir, f), 'utf8').trim() + '\n';
        }
      }
      out += '\n';
    }
  }
}

fs.writeFileSync(path.join(webDir, 'licenses.txt'), out);
console.log('licenses.txt written:', seen.size, 'packages');
