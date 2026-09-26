import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CheckupTypeFormScreen } from '@/screens/checkup-types';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('checkupTypes');
  return { title: t('new') };
}

export default function CheckupTypesNewPage() {
  return <CheckupTypeFormScreen id={null} />;
}
