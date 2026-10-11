import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { LabServicesScreen } from '@/screens/partner-lab';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('partnerLab.services');
  return { title: t('title') };
}

export default function Page() {
  return <LabServicesScreen />;
}
