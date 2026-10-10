import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { InboxScreen } from '@/screens/telemed';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('telemed.inbox');
  return { title: t('title') };
}

export default function Page() {
  return <InboxScreen />;
}
