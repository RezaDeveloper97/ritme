import { setRequestLocale } from 'next-intl/server';

import { BackupPage } from '@/screens/backup';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/profile/backup` — B-N10-07 (nbl_Me_Hub). */
export default async function ProfileBackupRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileBackup">
      <BackupPage />
    </RouteMessages>
  );
}
