import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

// Unit tests cover pure domain logic and the centralized date layer
// (React- and locale-free), so the default Node environment is enough.
export default defineConfig({
  // tsconfig keeps `jsx: preserve` for Next; tests that render a shared/ui
  // primitive to static markup (its a11y contract) need the automatic runtime.
  esbuild: { jsx: 'automatic' },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
});
