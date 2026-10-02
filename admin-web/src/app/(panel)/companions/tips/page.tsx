import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CompanionTipsScreen } from '@/screens/companions';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('companions.tips');
  return { title: t('title') };
}

export default function CompanionTipsPage() {
  return <CompanionTipsScreen />;
}
