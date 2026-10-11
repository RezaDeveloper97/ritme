import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { InsurersScreen } from '@/screens/insurance-centers';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('insuranceCenters');
  return { title: t('insurers') };
}

export default function Page() {
  return <InsurersScreen />;
}
