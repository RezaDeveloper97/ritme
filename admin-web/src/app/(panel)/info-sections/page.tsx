import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { InfoSectionsScreen } from '@/screens/info-sections';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('infoSections');
  return { title: t('title') };
}

export default function InfoSectionsPage() {
  return <InfoSectionsScreen />;
}
