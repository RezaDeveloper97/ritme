import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { InfoSectionFormScreen } from '@/screens/info-sections';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('infoSections');
  return { title: t('edit') };
}

export default async function InfoSectionsIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <InfoSectionFormScreen id={id} />;
}
