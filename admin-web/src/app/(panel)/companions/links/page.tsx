import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CompanionLinksScreen } from '@/screens/companions';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('companions.links');
  return { title: t('title') };
}

export default function CompanionLinksPage() {
  return <CompanionLinksScreen />;
}
