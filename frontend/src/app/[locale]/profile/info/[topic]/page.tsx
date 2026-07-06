import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { INFO_TOPICS, InfoPage, isInfoTopic } from '@/screens/profile-info';
import { routing } from '@/shared/i18n';

interface Props {
  params: Promise<{ locale: string; topic: string }>;
}

export function generateStaticParams() {
  return routing.locales.flatMap((locale) =>
    INFO_TOPICS.map((topic) => ({ locale, topic })),
  );
}

export default async function ProfileInfoRoute({ params }: Props) {
  const { locale, topic } = await params;
  if (!isInfoTopic(topic)) notFound();
  setRequestLocale(locale);
  return <InfoPage topic={topic} />;
}
