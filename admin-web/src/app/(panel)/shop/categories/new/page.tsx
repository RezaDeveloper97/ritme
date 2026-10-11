import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ShopCategoryFormScreen } from '@/screens/shop';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shop');
  return { title: t('newCategory') };
}

export default function ShopCategoryNewPage() {
  return <ShopCategoryFormScreen id={null} />;
}
