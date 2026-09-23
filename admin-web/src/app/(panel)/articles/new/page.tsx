import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ArticleFormScreen } from '@/screens/articles';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('articles');
  return { title: t('new') };
}

export default function ArticlesNewPage() {
  return <ArticleFormScreen id={null} />;
}
