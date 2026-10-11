import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { KpiScreen } from '@/screens/kpi';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('kpi');
  return { title: t('title') };
}

export default function Page() {
  return <KpiScreen />;
}
