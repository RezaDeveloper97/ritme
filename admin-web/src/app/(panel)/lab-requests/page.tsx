import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { LabRequestsScreen } from '@/screens/partner-lab';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('partnerLab.requests');
  return { title: t('title') };
}

export default function Page() {
  return <LabRequestsScreen />;
}
