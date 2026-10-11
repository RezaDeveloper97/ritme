import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ShopProductFormScreen } from '@/screens/shop';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shop');
  return { title: t('new') };
}

export default function ShopNewPage() {
  return <ShopProductFormScreen id={null} />;
}
