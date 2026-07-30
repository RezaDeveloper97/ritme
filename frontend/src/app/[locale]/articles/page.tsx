import { setRequestLocale } from 'next-intl/server';

import { ArticlesPage } from '@/screens/articles';

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function ArticlesRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <ArticlesPage />;
}
