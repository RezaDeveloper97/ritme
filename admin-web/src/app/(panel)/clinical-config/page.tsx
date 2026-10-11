import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ClinicalConfigScreen } from '@/screens/clinical-config';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('clinicalConfig');
  return { title: t('title') };
}

export default function Page() {
  return <ClinicalConfigScreen />;
}
