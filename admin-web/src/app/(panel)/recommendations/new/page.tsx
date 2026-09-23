import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { RecommendationFormScreen } from '@/screens/recommendations';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('recommendations');
  return { title: t('new') };
}

export default function RecommendationsNewPage() {
  return <RecommendationFormScreen id={null} />;
}
