import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CityServicesScreen } from '@/screens/city-services';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('cityServices');
  return { title: t('title') };
}

export default function Page() {
  return <CityServicesScreen />;
}
