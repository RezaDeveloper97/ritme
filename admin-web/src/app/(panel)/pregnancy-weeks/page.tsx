import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PregnancyWeeksScreen } from '@/screens/pregnancy-weeks';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('pregnancyWeeks');
  return { title: t('title') };
}

export default function PregnancyWeeksPage() {
  return <PregnancyWeeksScreen />;
}
