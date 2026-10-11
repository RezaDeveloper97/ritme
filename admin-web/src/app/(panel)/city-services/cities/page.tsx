import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CityServiceTaxaScreen } from '@/screens/city-services';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('cityServices');
  return { title: t('cities') };
}

export default function Page() {
  return <CityServiceTaxaScreen kind="cities" />;
}
