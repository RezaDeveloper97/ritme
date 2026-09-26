import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { CheckupTypeFormScreen } from '@/screens/checkup-types';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('checkupTypes');
  return { title: t('edit') };
}

export default async function CheckupTypesIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <CheckupTypeFormScreen id={id} />;
}
