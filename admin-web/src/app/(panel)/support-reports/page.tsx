import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { SupportReportsScreen } from '@/screens/support-reports';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('supportReports');
  return { title: t('title') };
}

export default function SupportReportsPage() {
  return <SupportReportsScreen />;
}
