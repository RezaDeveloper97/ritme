import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { MessagesScreen } from '@/screens/messages';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('smartMessages');
  return { title: t('title') };
}

export default function MessagesPage() {
  return <MessagesScreen />;
}
