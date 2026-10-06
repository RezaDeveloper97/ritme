import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { WhoScreen } from '@/screens/child-content';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('childContent');
  return { title: t('pages.who') };
}

export default function ChildContentWhoPage() {
  return <WhoScreen />;
}
