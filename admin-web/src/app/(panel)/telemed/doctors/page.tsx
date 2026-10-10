import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { DoctorsScreen } from '@/screens/telemed';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('telemed.doctors');
  return { title: t('title') };
}

export default function Page() {
  return <DoctorsScreen />;
}
