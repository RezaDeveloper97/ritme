import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ShopProductsScreen } from '@/screens/shop';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shop');
  return { title: t('title') };
}

export default function ShopPage() {
  return <ShopProductsScreen />;
}
