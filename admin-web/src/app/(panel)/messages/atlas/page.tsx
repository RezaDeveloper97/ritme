import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { MessageAtlasScreen } from '@/screens/message-atlas';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('messageAtlas');
  return { title: t('title') };
}

export default function MessageAtlasPage() {
  return <MessageAtlasScreen />;
}
