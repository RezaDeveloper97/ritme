import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { InsuranceCentresScreen } from '@/screens/insurance-centers';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('insuranceCenters');
  return { title: t('title') };
}

export default function Page() {
  return <InsuranceCentresScreen />;
}
