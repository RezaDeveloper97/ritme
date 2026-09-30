import { setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { UiKitPage } from '@/screens/ui-kit';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** Dev-only showcase of the Night & Bloom primitives (B-N1-03). Not shipped. */
export default async function UiKitRoute({ params }: Props) {
  if (process.env.NODE_ENV === 'production') notFound();
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="uiKit">
      <UiKitPage />
    </RouteMessages>
  );
}
