import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { EngineScreen } from '@/screens/message-rules';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('messageEngine');
  return { title: t('title') };
}

export default function MessageEnginePage() {
  return <EngineScreen />;
}
