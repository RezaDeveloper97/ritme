import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ShopStockScreen } from '@/screens/shop';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shopAdmin');
  return { title: t('titles.stock') };
}

export default function Page() {
  return <ShopStockScreen />;
}
