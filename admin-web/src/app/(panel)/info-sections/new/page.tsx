import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { InfoSectionFormScreen } from '@/screens/info-sections';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('infoSections');
  return { title: t('new') };
}

export default function InfoSectionsNewPage() {
  return <InfoSectionFormScreen id={null} />;
}
