'use client';

import { useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import { useCan } from '@/features/auth';
import { useSegments } from '@/entities/segment';
import { Button, Field } from '@/shared/ui';

import { addSegment, removeSegment } from '../lib/segment-condition';

/**
 * Target the rule at saved segments: each pick is a `segment contains <id>` condition (all of them must hold).
 * Shows segment names; an id whose segment was deleted stays visible, flagged, and can be cleared.
 */
export function SegmentPicker({ ids, onChange, disabled }: { ids: readonly string[]; onChange: (ids: string[]) => void; disabled?: boolean }) {
  const t = useTranslations('messageRules.segmentPicker');
  const canView = useCan('segments');
  const query = useSegments(canView);
  const selectId = useId();
  const [pick, setPick] = useState('');
  const items = query.data?.items ?? [];
  const nameOf = (id: string) => items.find((s) => String(s.id) === id);
  const loaded = canView && !!query.data;
  const choices = items.filter((s) => !ids.includes(String(s.id)));

  return (
    <div className="mb-4 rounded-lg border border-line bg-surface-2 p-3" data-testid="segment-picker">
      <Field label={t('label')} htmlFor={selectId} hint={canView ? t('hint') : t('noAccess')}>
        <div className="flex flex-wrap items-center gap-2">
          <select id={selectId} className="input min-w-48 flex-1" value={pick} disabled={disabled || !loaded} onChange={(e) => setPick(e.target.value)}>
            <option value="">{query.isLoading ? t('loading') : choices.length === 0 ? t('none') : t('choose')}</option>
            {choices.map((s) => (
              <option key={s.id} value={String(s.id)}>{s.name}</option>
            ))}
          </select>
          <Button type="button" disabled={disabled || pick === ''} onClick={() => { onChange(addSegment(ids, pick)); setPick(''); }}>
            {t('add')}
          </Button>
        </div>
      </Field>
      {ids.length > 0 ? (
        <ul className="mt-2 flex flex-wrap gap-2">
          {ids.map((id) => {
            const seg = nameOf(id);
            const gone = loaded && !seg;
            return (
              <li key={id} className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-sm ${gone ? 'border-danger text-danger' : 'border-line'}`}>
                <span>{seg ? seg.name : gone ? t('deleted', { id }) : t('unknown', { id })}</span>
                <button type="button" className="text-ink-3 hover:text-ink" aria-label={t('clear')} disabled={disabled} onClick={() => onChange(removeSegment(ids, id))}>
                  ×
                </button>
              </li>
            );
          })}
        </ul>
      ) : null}
    </div>
  );
}
