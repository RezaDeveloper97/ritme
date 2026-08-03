// Generates public/version.json from package.json before every build.
//
// version            → package.json "version" (bump every release)
// minSupportedVersion → package.json "pwa.minSupportedVersion" — bump ONLY for
//                       critical releases (breaking API change, security fix);
//                       clients below it get a blocking forced-update screen.
//
// The file must be served with Cache-Control: no-store (see next.config.ts)
// and is never precached by the service worker.
import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const pkg = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));

const payload = {
  version: pkg.version,
  minSupportedVersion: pkg.pwa?.minSupportedVersion ?? '0.0.0',
  releaseNotes: pkg.pwa?.releaseNotes ?? '',
};

writeFileSync(join(root, 'public', 'version.json'), `${JSON.stringify(payload, null, 2)}\n`);

// Stamp the service worker from its template so every release ships a
// byte-different worker — that's what makes the browser fire `updatefound`.
const swTemplate = readFileSync(join(root, 'scripts', 'sw.template.js'), 'utf8');
writeFileSync(join(root, 'public', 'sw.js'), swTemplate.replaceAll('__APP_VERSION__', pkg.version));

console.log(`version.json + sw.js → ${payload.version} (min supported: ${payload.minSupportedVersion})`);
