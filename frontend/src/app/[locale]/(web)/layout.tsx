import type { Metadata } from 'next';
import type { ReactNode } from 'react';

import { siteOrigin } from './site-origin';

/** Public web pages (places, store, business, d, emergency): relative canonical / OG URLs resolve against `SITE_URL`. */
export async function generateMetadata(): Promise<Metadata> {
  const origin = await siteOrigin();
  return origin ? { metadataBase: new URL(origin) } : {};
}

export default function WebLayout({ children }: { children: ReactNode }) {
  return children;
}
