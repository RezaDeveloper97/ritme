import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ShopCategoriesScreen } from '@/screens/shop';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shop');
  return { title: t('categories') };
}

export default function ShopCategoriesPage() {
  return <ShopCategoriesScreen />;
}
