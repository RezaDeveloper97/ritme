import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ModelFormScreen } from '@/screens/ai-credits';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('aiCredits.models');
  return { title: t('new') };
}

export default function AiModelNewPage() {
  return <ModelFormScreen id={null} />;
}
