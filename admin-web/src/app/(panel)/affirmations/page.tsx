import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { AffirmationsScreen } from '@/screens/affirmations';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('affirmations');
  return { title: t('title') };
}

export default function AffirmationsPage() {
  return <AffirmationsScreen />;
}
