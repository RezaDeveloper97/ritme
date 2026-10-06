import { setRequestLocale } from 'next-intl/server';

import { LabUploadPage } from '@/screens/lab-upload';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/labs/new` — nbl_Lab_Upload (B-N6-07). */
export default async function LabUploadPageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="labs">
      <LabUploadPage />
    </RouteMessages>
  );
}
