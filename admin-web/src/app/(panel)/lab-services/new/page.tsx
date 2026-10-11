import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { LabServiceFormScreen } from '@/screens/partner-lab';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('partnerLab.services');
  return { title: t('new') };
}

export default function Page() {
  return <LabServiceFormScreen id={null} />;
}
