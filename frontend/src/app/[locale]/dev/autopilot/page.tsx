import { setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { AutopilotLabPage } from '@/screens/autopilot-lab';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** Dev-only playground of the UI autopilot (docs/autopilot.md): run a script, read the agent manifest. Not shipped. */
export default async function AutopilotLabRoute({ params }: Props) {
  if (process.env.NODE_ENV === 'production') notFound();
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="autopilotLab">
      <AutopilotLabPage />
    </RouteMessages>
  );
}
