// Build pipeline for the listmonk SSR admin frontend (Bun).
// Output (dist/) is stuffed into the Go binary and served at /admin/static/*.
// Run: `bun run build` (one-shot) or `bun run watch` (rebuild on change).

import { readdir, rm, mkdir, cp, readFile, writeFile, unlink } from 'node:fs/promises';
import { watch as fsWatch } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.dirname(fileURLToPath(import.meta.url));
const srcJS = path.join(root, 'assets/js');
const dist = path.join(root, 'dist');
const watch = process.argv.includes('--watch');

// Types to always compress and bundle as .gz.
const COMPRESS_EXT = ['.js', '.css', '.svg'];

async function build() {
  // Fresh /dist dir.
  await rm(dist, { recursive: true, force: true });
  await mkdir(dist, { recursive: true });

  // Verbatim static assets (icons, images) and stylesheets.
  await cp(path.join(root, 'assets/static'), dist, { recursive: true });
  await cp(path.join(root, 'assets/css/style.css'), path.join(dist, 'style.css'));
  // Rich text editor styles, loaded only on pages that render the editor partial.
  await cp(path.join(root, 'assets/css/richtext.css'), path.join(dist, 'richtext.css'));

  // Oat CSS is loaded in <head> as is.
  await cp(path.join(root, 'node_modules/@knadh/oat/oat.min.css'), path.join(dist, 'oat.min.css'));

  // Entry points = modules loaded directly by a <script> tag: the global main.js and
  // every per-view module under assets/js/views/.
  const views = (await readdir(path.join(srcJS, 'views')))
    .filter((f) => f.endsWith('.js'))
    .map((f) => path.join(srcJS, 'views', f));

  const result = await Bun.build({
    entrypoints: [
      path.join(srcJS, 'main.js'),
      path.join(srcJS, 'code-editor.js'),
      path.join(srcJS, 'richtext-editor.js'),
      path.join(srcJS, 'visual-editor.js'),
      ...views,
    ],
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

  await compressAssets();
}

async function compressAssets() {
  const entries = await readdir(dist, { recursive: true, withFileTypes: true });
  let n = 0;

  for (const e of entries) {
    if (!e.isFile() || !COMPRESS_EXT.includes(path.extname(e.name))) continue;

    const abs = path.join(e.parentPath ?? e.path, e.name);
    await writeFile(`${abs}.gz`, Bun.gzipSync(await readFile(abs), { level: 9 }));
    await unlink(abs);
    n++;
  }

  console.log(`compressed ${n} assets -> .gz`);
}

await build();

if (watch) {
  console.log('watching assets/ for changes…');
  let timer = null;
  fsWatch(path.join(root, 'assets'), { recursive: true }, () => {
    clearTimeout(timer);
    timer = setTimeout(() => build().catch((e) => console.error(e)), 100);
  });
}
