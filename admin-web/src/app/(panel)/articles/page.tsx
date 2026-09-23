import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ArticlesScreen } from '@/screens/articles';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('articles');
  return { title: t('title') };
}

export default function ArticlesPage() {
  return <ArticlesScreen />;
}
