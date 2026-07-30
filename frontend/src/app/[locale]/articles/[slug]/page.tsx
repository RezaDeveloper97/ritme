import { setRequestLocale } from 'next-intl/server';

import { ArticlePage } from '@/screens/article';

interface Props {
  params: Promise<{ locale: string; slug: string }>;
}

// The slug is public content (never health data), so it is safe in the URL —
// unlike the cycle phase, which is why /cycle/phase is param-less (§11).
export default async function ArticleRoute({ params }: Props) {
  const { locale, slug } = await params;
  setRequestLocale(locale);
  return <ArticlePage slug={slug} />;
}
