import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PhaseContentsScreen } from '@/screens/phase-contents';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('phaseContents');
  return { title: t('title') };
}

export default function PhaseContentsPage() {
  return <PhaseContentsScreen />;
}
