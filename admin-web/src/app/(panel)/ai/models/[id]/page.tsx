import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { ModelFormScreen } from '@/screens/ai-credits';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('aiCredits.models');
  return { title: t('edit') };
}

export default async function AiModelEditPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <ModelFormScreen id={id} />;
}
