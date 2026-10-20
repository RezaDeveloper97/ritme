import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { AppModulesScreen } from '@/screens/app-modules';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('appModules');
  return { title: t('title') };
}

export default function Page() {
  return <AppModulesScreen />;
}
