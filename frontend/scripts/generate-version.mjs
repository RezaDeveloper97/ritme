// Generates public/version.json and public/sw.js before every build (prebuild).
//
// version             → package.json "version" (bump every release; drives the
//                       human-facing "new version" toast).
// minSupportedVersion → package.json "pwa.minSupportedVersion" — bump ONLY for
//                       critical releases (breaking API change, security fix);
//                       clients below it get a blocking forced-update screen.
// buildId             → unique per build (RITME_BUILD_ID if set, e.g. a git
//                       SHA from CI; otherwise time + random). It is stamped
//                       into sw.js so EVERY deploy ships a byte-different
//                       worker, even one that forgot to bump `version` — that
//                       is what fires `updatefound`, refreshes the precache
//                       and runs the old-cache cleanup (pwa-audit S-4).
//                       next.config.ts reads it back from version.json to use
//                       as Next's own BUILD_ID and NEXT_PUBLIC_BUILD_ID, so the
//                       worker, the bundle and version.json agree.
//
// version.json must be served with Cache-Control: no-store (see next.config.ts)
// and is never cached by the service worker.
import { randomBytes } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const pkg = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));

function makeBuildId() {
  const fromEnv = process.env.RITME_BUILD_ID?.trim();
  // Also used as a cache name and Next's build directory name — keep it tame.
  if (fromEnv && /^[A-Za-z0-9._-]{1,64}$/.test(fromEnv)) {
    // An env id alone would repeat across rebuilds of one commit; the random
    // suffix keeps the "every build is a new worker" guarantee.
    return `${fromEnv}-${randomBytes(3).toString('hex')}`;
  }
  return `${Date.now().toString(36)}-${randomBytes(4).toString('hex')}`;
}

const buildId = makeBuildId();

const payload = {
  version: pkg.version,
  minSupportedVersion: pkg.pwa?.minSupportedVersion ?? '0.0.0',
  releaseNotes: pkg.pwa?.releaseNotes ?? '',
  buildId,
};

writeFileSync(join(root, 'public', 'version.json'), `${JSON.stringify(payload, null, 2)}\n`);

const swTemplate = readFileSync(join(root, 'scripts', 'sw.template.js'), 'utf8');
writeFileSync(
  join(root, 'public', 'sw.js'),
  swTemplate.replaceAll('__APP_VERSION__', pkg.version).replaceAll('__BUILD_ID__', buildId),
);

console.log(
  `version.json + sw.js → ${payload.version} build ${buildId} (min supported: ${payload.minSupportedVersion})`,
);
