import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { DoctorFormScreen } from '@/screens/telemed';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('telemed.doctor');
  return { title: t('new') };
}

export default function Page() {
  return <DoctorFormScreen id={null} />;
}
