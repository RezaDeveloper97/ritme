import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

// Pure logic (api client, date layer, list params) and server-rendered markup of
// shared/ui components (react-dom/server), so the Node environment is enough.
export default defineConfig({
  esbuild: { jsx: 'automatic' },
  test: {
    environment: 'node',
    include: ['src/**/*.test.{ts,tsx}'],
  },
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
});
