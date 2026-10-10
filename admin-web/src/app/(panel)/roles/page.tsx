import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { RolesScreen } from '@/screens/access-control';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('access');
  return { title: t('rolesTitle') };
}

export default function Page() {
  return <RolesScreen />;
}
