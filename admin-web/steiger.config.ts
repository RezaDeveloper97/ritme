import fsd from '@feature-sliced/steiger-plugin';
import { defineConfig } from 'steiger';

// FSD-lite boundary check (docs/go-migration/admin-web.md). Screens are named
// `screens`, not `pages` (Next would treat `pages/` as the Pages Router), so
// steiger cannot see references coming from them — hence insignificant-slice
// is off for slices consumed only by screens.
export default defineConfig([
  ...fsd.configs.recommended,
  { ignores: ['**/*.test.ts', '**/*.test.tsx'] },
  {
    files: ['./src/features/**', './src/entities/**', './src/widgets/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
]);
