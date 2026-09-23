import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PhaseContentFormScreen } from '@/screens/phase-contents';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('phaseContents');
  return { title: t('new') };
}

export default function PhaseContentsNewPage() {
  return <PhaseContentFormScreen id={null} />;
}
