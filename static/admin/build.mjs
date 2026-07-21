// Build pipeline for the listmonk SSR admin frontend (Bun).
// Output (dist/) is stuffed into the Go binary and served at /admin/static/*.
// Run: `bun run build` (one-shot) or `bun run watch` (rebuild on change).

import { readdir, rm, mkdir, cp } from 'node:fs/promises';
import { watch as fsWatch } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.dirname(fileURLToPath(import.meta.url));
const srcJS = path.join(root, 'src/js');
const dist = path.join(root, 'dist');
const watch = process.argv.includes('--watch');

// Third-party runtime libs copied from node_modules into dist/vendor/. These are loaded
// as global <script>/<link> tags in base.html not bundled.
const vendor = [
  ['node_modules/alpinejs/dist/cdn.min.js', 'vendor/alpinejs.min.js'],
  ['node_modules/@knadh/oat/oat.min.js', 'vendor/oat.min.js'],
  ['node_modules/@knadh/oat/oat.min.css', 'vendor/oat.min.css'],
  ['node_modules/chart.js/dist/chart.umd.js', 'vendor/chart.min.js'],
];

async function build() {
  // Fresh /dist dir.
  await rm(dist, { recursive: true, force: true });
  await mkdir(dist, { recursive: true });

  // Verbatim static assets (fonts, icons, images), authored stylesheet, vendored libs.
  await cp(path.join(root, 'src/static'), dist, { recursive: true });
  await cp(path.join(root, 'src/css/style.css'), path.join(dist, 'style.css'));
  for (const [from, to] of vendor) {
    await cp(path.join(root, from), path.join(dist, to));
  }

  // Entry points = modules loaded directly by a <script> tag: the global main.js and
  // every per-view module under src/js/views/.
  const views = (await readdir(path.join(srcJS, 'views')))
    .filter((f) => f.endsWith('.js'))
    .map((f) => path.join(srcJS, 'views', f));

  const result = await Bun.build({
    entrypoints: [path.join(srcJS, 'main.js'), path.join(srcJS, 'code-editor.js'), ...views],
    outdir: path.join(dist, 'js'),
    root: srcJS,
    splitting: true,
    format: 'esm',
    minify: true,
    target: 'browser',
    naming: { entry: '[dir]/[name].[ext]', chunk: 'chunks/[name]-[hash].[ext]' },
  });

  if (!result.success) {
    for (const log of result.logs) console.error(log);
    throw new AggregateError(result.logs, 'admin build failed');
  }
  console.log(`built ${result.outputs.length} files -> dist/js`);
}

await build();

if (watch) {
  console.log('watching src/ for changes…');
  let timer = null;
  fsWatch(path.join(root, 'src'), { recursive: true }, () => {
    clearTimeout(timer);
    timer = setTimeout(() => build().catch((e) => console.error(e)), 100);
  });
}
