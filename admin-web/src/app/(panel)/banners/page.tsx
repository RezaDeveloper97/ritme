import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { BannersScreen } from '@/screens/banners';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('banners');
  return { title: t('title') };
}

export default function BannersPage() {
  return <BannersScreen />;
}
