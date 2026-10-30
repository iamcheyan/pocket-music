// Bundles the pure-logic unit tests with esbuild and runs them through the
// built-in node:test runner. No test framework dependency is added to the app.
import { build } from 'esbuild';
import { readdir, rm, mkdir } from 'node:fs/promises';
import { join, relative } from 'node:path';
import { spawnSync } from 'node:child_process';

const root = new URL('..', import.meta.url).pathname;
const outDir = join(root, '.test-build');

async function collect(dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name.startsWith('.')) continue;
      files.push(...(await collect(full)));
    } else if (entry.name.endsWith('.test.ts')) {
      files.push(full);
    }
  }
  return files;
}

const files = await collect(join(root, 'src'));
if (files.length === 0) {
  console.error('no test files found');
  process.exit(1);
}

await rm(outDir, { recursive: true, force: true });
await mkdir(outDir, { recursive: true });

const built = [];
for (const file of files) {
  const outfile = join(outDir, relative(join(root, 'src'), file).replace(/\.ts$/, '.mjs'));
  built.push(outfile);
  await build({
    entryPoints: [file],
    outfile,
    bundle: true,
    platform: 'node',
    format: 'esm',
    target: 'node22',
    logLevel: 'warning',
  });
}

const result = spawnSync(process.execPath, ['--test', ...built], { stdio: 'inherit' });
process.exit(result.status ?? 1);
