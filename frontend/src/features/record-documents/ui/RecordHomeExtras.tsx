'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId } from 'react';

import {
  CATEGORY_LOOK,
  type CategoryKey,
  type RecordCategories,
  type RecordExtras,
  useRecordCategories,
  useRecordExtras,
  useSaveRecordExtras,
} from '@/entities/health-record';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, IconCircle, SecondaryButton, Skeleton, SkeletonGroup, Switch } from '@/shared/ui';

import { useRecordHomeSheet } from '../model/sheet-store';
import { ExtrasSheet } from './ExtrasSheet';
import { UploadDocumentSheet } from './UploadDocumentSheet';

/** Grid tiles in board order: the API categories (documents per kind, surgeries, family history) + «همه اسناد». */
const TILES: readonly (CategoryKey | 'all')[] = [
  'imaging',
  'visit',
  'labs',
  'prescription',
  'hospital',
  'other',
  'surgeries',
  'family_history',
  'all',
];
const TIMELINE_KIND: Partial<Record<CategoryKey, string>> = {
  labs: 'lab',
  imaging: 'imaging',
  visit: 'visit',
  prescription: 'prescription',
  hospital: 'hospital',
  other: 'other',
};

/**
 * Canvas additions to bloom's record home (CB-REC-04, nbl_Rec_Home) on top of B-N6-03's sections: the drug-allergy
 * card with its emergency-card flag, the category grid with counts, the «needs review» card and «افزودن سند».
 * Insurance links (CB-INS) are not built yet and stay out.
 */
export function RecordHomeExtras() {
  const t = useTranslations('record');
  const router = useRouter();
  const categories = useRecordCategories();
  const extras = useRecordExtras();
  const openSheet = useRecordHomeSheet((s) => s.open);

  const open = (key: CategoryKey | 'all') => {
    if (key === 'surgeries' || key === 'family_history') openSheet(key);
    else if (key === 'all') router.push('/record/timeline');
    else router.push(`/record/timeline?kind=${TIMELINE_KIND[key] ?? 'all'}`);
  };

  return (
    <>
      {extras.data ? <AllergyCard extras={extras.data} /> : null}
      {categories.isPending ? (
        <SkeletonGroup label={t('timeline.loading')} className="rec-grid">
          {TILES.map((k) => (
            <Skeleton key={k} shape="card" className="rec-tile-skel" />
          ))}
        </SkeletonGroup>
      ) : categories.data ? (
        <>
          <CategoryGrid data={categories.data} onOpen={open} />
          <ReviewCard count={categories.data.needsReviewCount} onOpen={() => router.push('/record/timeline')} />
        </>
      ) : null}
      <div className="rec-home-actions">
        <SecondaryButton icon="box" onClick={() => router.push('/record/timeline')}>
          {t('home.allDocuments')}
        </SecondaryButton>
        <SecondaryButton icon="plus" onClick={() => openSheet('upload')}>
          {t('home.addDocument')}
        </SecondaryButton>
      </div>
    </>
  );
}

/** The record-home sheets; mount outside the screen's scroller (next to the screen's other sheets). */
export function RecordHomeSheets() {
  const router = useRouter();
  const extras = useRecordExtras();
  const sheet = useRecordHomeSheet((s) => s.sheet);
  const close = useRecordHomeSheet((s) => s.close);
  // a sheet left open must not reappear when the user comes back to /record
  useEffect(() => close, [close]);
  const section = sheet === 'surgeries' || sheet === 'family_history' ? sheet : null;
  return (
    <>
      {extras.data ? <ExtrasSheet section={section} extras={extras.data} onClose={close} /> : null}
      <UploadDocumentSheet
        open={sheet === 'upload'}
        onClose={close}
        onCreated={(doc) => {
          close();
          router.push(`/record/documents/${doc.id}`);
        }}
      />
    </>
  );
}

function AllergyCard({ extras }: { extras: RecordExtras }) {
  const t = useTranslations('record.allergy');
  const save = useSaveRecordExtras();
  const descId = useId();
  const list = extras.allergies ?? [];
  if (list.length === 0) return null;
  const on = save.isPending && save.variables?.allergiesOnEmergencyCard !== undefined ? save.variables.allergiesOnEmergencyCard : extras.allergiesOnEmergencyCard;
  return (
    <Card className="rec-allergy">
      <IconCircle icon="warning" tone="danger" size="md" />
      <div className="rec-allergy-text">
        <p className="rec-allergy-title">{t('title', { list: list.join('، ') })}</p>
        <p id={descId} className="rec-allergy-sub">
          {on ? t('onCard') : t('offCard')}
        </p>
        {save.isError ? (
          <p className="hrec-error" role="alert">
            {t('saveError')}
          </p>
        ) : null}
      </div>
      <Switch
        checked={on}
        label={t('toggle')}
        describedBy={descId}
        disabled={save.isPending}
        onCheckedChange={(next) => save.mutate({ allergiesOnEmergencyCard: next })}
      />
    </Card>
  );
}

function CategoryGrid({ data, onOpen }: { data: RecordCategories; onOpen: (key: CategoryKey | 'all') => void }) {
  const t = useTranslations('record.categories');
  const locale = useLocale() as Locale;
  const titleId = useId();
  const countOf = (k: CategoryKey | 'all') => (k === 'all' ? data.allCount : data.counts[k]);
  return (
    <section aria-labelledby={titleId} className="rec-grid-wrap">
      <h2 id={titleId} className="sr-only">
        {t('title')}
      </h2>
      <ul className="rec-grid">
        {TILES.map((k) => {
          const look = CATEGORY_LOOK[k];
          const count = countOf(k);
          return (
            <li key={k}>
              <button type="button" className="rec-tile" onClick={() => onOpen(k)}>
                <IconCircle icon={look.icon} tone={look.tone} size="md" />
                <span className="rec-tile-name">{t(k)}</span>
                <span className="rec-tile-count">{t(`count.${k}`, { count, n: formatNumber(count, locale) })}</span>
              </button>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

function ReviewCard({ count, onOpen }: { count: number; onOpen: () => void }) {
  const t = useTranslations('record.pending');
  const locale = useLocale() as Locale;
  if (count <= 0) return null;
  return (
    <Card className="rec-review">
      <IconCircle icon="warning" tone="warm" size="md" />
      <div className="rec-review-text">
        <p className="rec-review-title">{t('needsReview', { count, n: formatNumber(count, locale) })}</p>
        <p className="rec-review-sub">{t('needsReviewBody')}</p>
      </div>
      <button type="button" className="rec-pill-btn" onClick={onOpen}>
        {t('open')}
      </button>
    </Card>
  );
}
