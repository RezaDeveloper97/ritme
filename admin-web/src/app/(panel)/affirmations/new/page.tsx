import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { AffirmationFormScreen } from '@/screens/affirmations';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('affirmations');
  return { title: t('new') };
}

export default function AffirmationsNewPage() {
  return <AffirmationFormScreen id={null} />;
}
