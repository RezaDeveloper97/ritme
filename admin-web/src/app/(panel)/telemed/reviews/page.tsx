import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ReviewsScreen } from '@/screens/telemed';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('telemed.reviews');
  return { title: t('title') };
}

export default function Page() {
  return <ReviewsScreen />;
}
