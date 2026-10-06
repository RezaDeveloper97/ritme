import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { LearnScreen } from '@/screens/child-content';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('childContent');
  return { title: t('pages.learn') };
}

export default function ChildContentLearnPage() {
  return <LearnScreen />;
}
