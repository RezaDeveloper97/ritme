import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { RulesListScreen } from '@/screens/message-rules';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('messageRules');
  return { title: t('title') };
}

export default function MessageRulesPage() {
  return <RulesListScreen />;
}
