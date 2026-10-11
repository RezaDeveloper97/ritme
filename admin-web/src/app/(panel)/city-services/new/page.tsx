import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CityServiceFormScreen } from '@/screens/city-services';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('cityServices');
  return { title: t('new') };
}

export default function Page() {
  return <CityServiceFormScreen id={null} />;
}
