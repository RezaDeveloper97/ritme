import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { AdminFormScreen } from '@/screens/admins';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('admins');
  return { title: t('new') };
}

export default function AdminsNewPage() {
  return <AdminFormScreen id={null} />;
}
