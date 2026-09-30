import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CatalogGroupsScreen } from '@/screens/catalog';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('catalog');
  return { title: t('title') };
}

export default function CatalogPage() {
  return <CatalogGroupsScreen />;
}
