'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { KIND_LOOK, type TimelineItem, useRecordTimeline } from '@/entities/health-record';
import { UploadDocumentSheet } from '@/features/record-documents';
import type { Locale } from '@/shared/i18n';
import { formatDayMonth, fromApiDate } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import { EmptyState, IconCircle, PrimaryButton, SecondaryButton, Skeleton } from '@/shared/ui';
import '@/shared/ui/styles/ins.css';

interface ClaimDocPickerProps {
  open: boolean;
  onClose: () => void;
  /** Record document ids that are already on the claim (shown as taken). */
  takenIds: readonly number[];
  /** The chosen record document (an existing one, or one just added with the camera). */
  onPick: (doc: { id: number; kind: string; title: string | null; date: string | null }) => void;
}

/**
 * «از پرونده سلامت» (CB-INS-04, nbl_Ins_ClaimNew / _ClaimDetail): her own record documents, newest first, or a new one
 * from the camera through the record's upload sheet. Nothing here leaves her account and no document is logged.
 */
export function ClaimDocPicker({ open, onClose, takenIds, onPick }: ClaimDocPickerProps) {
  const t = useTranslations('insurance.docPicker');
  const tk = useTranslations('record.kinds');
  const locale = useLocale() as Locale;
  const timeline = useRecordTimeline('all');
  const [camera, setCamera] = useState(false);

  const items: TimelineItem[] = (timeline.data?.pages ?? []).flatMap((p) => p.months.flatMap((m) => m.items)).filter((i) => i.type === 'document');

  return (
    <>
      <AppSheet
        open={open && !camera}
        onClose={onClose}
        size="full"
        title={t('title')}
        footer={
          <PrimaryButton icon="camera" onClick={() => setCamera(true)}>
            {t('camera')}
          </PrimaryButton>
        }
      >
        <p className="ins-hint">{t('hint')}</p>
        {timeline.isPending ? (
          <Skeleton shape="card" />
        ) : timeline.isError ? (
          <EmptyState
            icon="fileDoc"
            title={t('error')}
            action={
              <SecondaryButton block={false} onClick={() => void timeline.refetch()}>
                {t('retry')}
              </SecondaryButton>
            }
          />
        ) : items.length === 0 ? (
          <EmptyState icon="fileDoc" title={t('empty')} />
        ) : (
          <ul className="ins-docs ins-picker">
            {items.map((d) => {
              const taken = takenIds.includes(d.id);
              const look = KIND_LOOK[d.kind];
              return (
                <li key={d.id}>
                  <button
                    type="button"
                    className="ins-pick"
                    disabled={taken}
                    {...pilot(`ins-docs.pick:${d.id}`)}
                    onClick={() => {
                      onPick({ id: d.id, kind: d.kind, title: d.title, date: d.dateKnown ? d.date : null });
                      onClose();
                    }}
                  >
                    <IconCircle icon={look.icon} tone={look.tone} />
                    <span className="ins-pick-text">
                      <span className="ins-pick-t">{d.title ?? tk(d.kind)}</span>
                      <span className="ins-pick-s">{d.dateKnown ? formatDayMonth(fromApiDate(d.date), locale) : tk(d.kind)}</span>
                    </span>
                    {taken ? <span className="ins-pick-taken">{t('taken')}</span> : null}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </AppSheet>
      <UploadDocumentSheet
        open={open && camera}
        onClose={() => setCamera(false)}
        onCreated={(doc) => {
          setCamera(false);
          onPick({ id: doc.id, kind: doc.kind, title: doc.title, date: doc.date });
          onClose();
        }}
      />
    </>
  );
}
