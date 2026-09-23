import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { RecommendationsScreen } from '@/screens/recommendations';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('recommendations');
  return { title: t('title') };
}

export default function RecommendationsPage() {
  return <RecommendationsScreen />;
}
