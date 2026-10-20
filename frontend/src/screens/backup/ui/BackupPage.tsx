'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useModuleEnabled } from '@/entities/app-module';
import { useExportData, useExportPdf } from '@/features/manage-account';
import { readLastSync } from '@/shared/api';
import { useDirection, useRouter, type Locale } from '@/shared/i18n';
import { diffInDays, formatLongDate, formatTimeOfDay, today } from '@/shared/lib/date';
import { Icon, InfoNote, ListGroup, ListRow, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

/** `undefined` until read on the device (no server/client mismatch), then epoch ms or null (nothing written yet). */
function useLastSync(): number | null | undefined {
  const [at, setAt] = useState<number | null | undefined>(undefined);
  useEffect(() => setAt(readLastSync()), []);
  return at;
}

/**
 * Backup & export (B-N10-07, `nbl_Me_Hub` «پشتیبان و خروجی داده») at `/profile/backup`. There is no cloud-drive
 * backup: her data already lives in her account on the server, so «last backup» = the last successful write from this
 * device (QUESTIONS Q11, `shared/api/lastSync`). Manual export reuses the JSON/PDF export and the doctor report builder.
 */
export function BackupPage() {
  const t = useTranslations('me');
  const tp = useTranslations('me.privacy');
  const router = useRouter();
  const recordOn = useModuleEnabled('record'); // the doctor report is a health-record export
  const loc = useLocale() as Locale;
  const dir = useDirection();
  const lastSync = useLastSync();
  const json = useExportData();
  const pdf = useExportPdf();
  const exporting = json.isPending || pdf.isPending;

  const exportPdf = () =>
    pdf.exportPdf({
      title: tp('mine.pdfTitle'),
      subtitle: tp('mine.pdfSubtitle', { date: formatLongDate(today(), loc) }),
      footer: tp.raw('mine.pdfFooter') as string,
      dir,
      locale: loc,
    });

  let when: string | null = null;
  if (lastSync) {
    const d = new Date(lastSync);
    const days = diffInDays(today(), d);
    const time = formatTimeOfDay(d, loc);
    when =
      days <= 0 ? t('backup.syncToday', { time })
      : days === 1 ? t('backup.syncYesterday', { time })
      : t('backup.syncOn', { date: formatLongDate(d, loc), time });
  }

  return (
    <div className="view bkp-page">
      <SkyLayer />
      <div className="scroll bkp-scroll">
        <ScreenHeader title={t('backup.title')} onBack={() => router.push('/profile')} backLabel={t('backup.back')} />

        <section className="nb-card bkp-sync" aria-labelledby="bkp-sync-label">
          <span className="bkp-sync-ic" aria-hidden>
            <Icon name="check" size={22} />
          </span>
          <div className="bkp-sync-body">
            <div id="bkp-sync-label" className="bkp-sync-label">
              {t('backup.lastSync')}
            </div>
            {lastSync === undefined ? (
              <SkeletonGroup label={t('backup.loading')}>
                <Skeleton shape="line" width="medium" />
              </SkeletonGroup>
            ) : (
              <div className="bkp-sync-value">{when ?? t('backup.syncNone')}</div>
            )}
          </div>
        </section>
        <p className="bkp-text">{t('backup.explain')}</p>

        <section className="bkp-sec" aria-labelledby="bkp-export">
          <h2 id="bkp-export" className="bkp-label">
            {t('backup.exportTitle')}
          </h2>
          <ListGroup>
            <ListRow
              icon="download"
              title={t('backup.json')}
              description={json.isPending ? tp('mine.exporting') : t('backup.jsonSub')}
              onClick={exporting ? undefined : () => json.exportData()}
            />
            <ListRow
              icon="fileDoc"
              iconTone="bloom"
              title={t('backup.pdf')}
              description={pdf.isPending ? tp('mine.exporting') : t('backup.pdfSub')}
              onClick={exporting ? undefined : exportPdf}
            />
            {recordOn ? (
              <ListRow
                icon="fileDoc"
                iconTone="brand"
                title={t('backup.report')}
                description={t('backup.reportSub')}
                onClick={() => router.push('/record/export')}
              />
            ) : null}
          </ListGroup>
          {json.isError || pdf.isError ? (
            <p className="me-inline-error" role="alert">
              {t('rows.exportError')}
            </p>
          ) : null}
        </section>

        <InfoNote icon="lock">{t('backup.privacy')}</InfoNote>
      </div>
    </div>
  );
}
