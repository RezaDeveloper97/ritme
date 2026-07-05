import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

// Point the next-intl plugin at our request config, which lives in the
// `shared` layer (domain-agnostic foundation) per FSD.
const withNextIntl = createNextIntlPlugin('./src/shared/i18n/request.ts');

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Emit a self-contained server bundle (.next/standalone) so the Docker
  // runtime image can ship without node_modules or the full source tree.
  output: 'standalone',
};

export default withNextIntl(nextConfig);
