'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { type CityServiceCatalogOption, type DirectoryFilters, EMPTY_FILTERS } from '@/entities/city-service';
import { AppSheet } from '@/shared/sheet';
import { PrimaryButton, SecondaryButton } from '@/shared/ui';

interface Props {
  open: boolean;
  onClose: () => void;
  value: DirectoryFilters;
  onApply: (next: DirectoryFilters) => void;
  ageRanges: CityServiceCatalogOption[];
  amenities: CityServiceCatalogOption[];
  /** Sort modes the API offers; «distance» needs the position (the parent asks for it on the tap). */
  sorts: string[];
  /** Whether the position is already known (else the distance sort explains it will ask). */
  nearReady: boolean;
}

/**
 * CB-DIR-07 filter sheet for the directory list and map: child age range, open now, amenities and the sort.
 * Edits a draft; «نمایش نتایج» applies, «حذف فیلترها» clears. Nothing here is stored or sent until applied.
 */
export function DirectoryFilterSheet({ open, onClose, value, onApply, ageRanges, amenities, sorts, nearReady }: Props) {
  const t = useTranslations('cityServices');
  const [draft, setDraft] = useState(value);
  useEffect(() => {
    if (open) setDraft(value);
  }, [open, value]);

  const toggleAmenity = (code: string) =>
    setDraft((d) => ({ ...d, amenities: d.amenities.includes(code) ? d.amenities.filter((a) => a !== code) : [...d.amenities, code] }));

  const sortOptions = (['rank', 'distance', 'rating'] as const).filter((s) => sorts.length === 0 || sorts.includes(s));
  const chip = (key: string, label: string, on: boolean, onClick: () => void, role: 'radio' | 'checkbox') => (
    <button key={key} type="button" role={role} aria-checked={on} className={clsx('chip csv-chip', on && 'on')} onClick={onClick}>
      {label}
    </button>
  );

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('filters.title')}
      footer={
        <div className="dfs-footer">
          <SecondaryButton variant="text" onClick={() => setDraft(EMPTY_FILTERS)}>
            {t('filters.clear')}
          </SecondaryButton>
          <PrimaryButton
            onClick={() => {
              onApply(draft);
              onClose();
            }}
          >
            {t('filters.apply')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="dfs-body">
        {ageRanges.length > 0 ? (
          <section className="dfs-sec">
            <h3 className="dfs-h">{t('filters.age')}</h3>
            <div role="radiogroup" aria-label={t('filters.age')} className="dfs-chips">
              {chip('any', t('filters.anyAge'), draft.age === null, () => setDraft((d) => ({ ...d, age: null })), 'radio')}
              {ageRanges.map((a) =>
                chip(a.code, a.name ?? a.code, draft.age === a.code, () => setDraft((d) => ({ ...d, age: a.code })), 'radio'),
              )}
            </div>
          </section>
        ) : null}
        <section className="dfs-sec">
          <label className="dfs-row">
            <span className="dfs-h">{t('filters.openNow')}</span>
            <input
              type="checkbox"
              role="switch"
              className="dfs-switch"
              checked={draft.openNow}
              onChange={(e) => setDraft((d) => ({ ...d, openNow: e.target.checked }))}
            />
          </label>
        </section>
        {amenities.length > 0 ? (
          <section className="dfs-sec">
            <h3 className="dfs-h">{t('filters.amenities')}</h3>
            <div role="group" aria-label={t('filters.amenities')} className="dfs-chips">
              {amenities.map((a) => chip(a.code, a.name ?? a.code, draft.amenities.includes(a.code), () => toggleAmenity(a.code), 'checkbox'))}
            </div>
          </section>
        ) : null}
        <section className="dfs-sec">
          <h3 className="dfs-h">{t('filters.sort')}</h3>
          <div role="radiogroup" aria-label={t('filters.sort')} className="dfs-chips">
            {sortOptions.map((s) => chip(s, t(`filters.sorts.${s}`), draft.sort === s, () => setDraft((d) => ({ ...d, sort: s })), 'radio'))}
          </div>
          {draft.sort === 'distance' && !nearReady ? <p className="dfs-note">{t('filters.distanceNote')}</p> : null}
        </section>
      </div>
    </AppSheet>
  );
}
