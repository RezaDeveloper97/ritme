import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PackFormScreen } from '@/screens/ai-credits';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('aiCredits.packs');
  return { title: t('new') };
}

export default function AiCreditPackNewPage() {
  return <PackFormScreen id={null} />;
}
