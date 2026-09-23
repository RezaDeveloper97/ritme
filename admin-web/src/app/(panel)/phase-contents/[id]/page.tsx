import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { PhaseContentFormScreen } from '@/screens/phase-contents';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('phaseContents');
  return { title: t('edit') };
}

export default async function PhaseContentsIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <PhaseContentFormScreen id={id} />;
}
