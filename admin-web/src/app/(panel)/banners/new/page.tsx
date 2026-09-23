import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { BannerFormScreen } from '@/screens/banners';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('banners');
  return { title: t('new') };
}

export default function BannersNewPage() {
  return <BannerFormScreen id={null} />;
}
