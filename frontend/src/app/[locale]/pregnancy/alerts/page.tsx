import { redirect } from 'next/navigation';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/pregnancy/alerts` — the old pregnancy-only screen; alerts now live in the notifications sheet (urgent ones also
 * as a modal). The URL keeps working: it opens that sheet on «امروز» (`/home` carries `?sheet=` through its hand-off). */
export default async function PregnancyAlertsRoute({ params }: Props) {
  const { locale } = await params;
  redirect(`/${locale}/home?sheet=notifications`);
}
