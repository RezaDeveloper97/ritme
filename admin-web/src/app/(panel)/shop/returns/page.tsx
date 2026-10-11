import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ShopReturnsScreen } from '@/screens/shop';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shopAdmin');
  return { title: t('titles.returns') };
}

export default function Page() {
  return <ShopReturnsScreen />;
}
