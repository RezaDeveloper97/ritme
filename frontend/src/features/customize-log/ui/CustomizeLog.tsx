'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react';

import { type LogPreferences, useLogPreferences } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
} from '@/shared/ui';

import { useResetLogPreferences, useSaveLogPreferences } from '../api/mutations';
import {
  applyDraft,
  type CustomizeDraft,
  diffDraft,
  draftFromPrefs,
  isDirty,
  isHidden,
  isPinned,
  moveCategory,
  pinsFull,
  toggleHidden,
  togglePin,
} from '../model/draft';
import { useReorder } from '../model/use-reorder';
import { type CategoryLook, CategoryRow } from './CategoryRow';
import { CustomItems } from './CustomItems';

export interface CustomizeLogProps {
  /** Icon + tone per category (owned by `features/log-day`, passed in by the screen). */
  lookOf: (code: string) => CategoryLook;
  /** Leaves the screen (back arrow, after «ذخیره و خروج»). */
  onExit: () => void;
  /** Taxonomy mode; omit for the user's current life-stage mode. */
  mode?: string;
}

interface Drafts {
  base: CustomizeDraft;
  draft: CustomizeDraft;
}

function LoadingState({ label }: { label: string }) {
  return (
    <SkeletonGroup label={label} className="lcz-skel">
      <Skeleton width="medium" />
      <div className="nb-card lcz-skel-card">
        {Array.from({ length: 8 }, (_, i) => (
          <div key={i} className="lcz-skel-row">
            <Skeleton shape="circle" />
            <Skeleton width="medium" />
            <Skeleton shape="block" className="lcz-skel-switch" />
          </div>
        ))}
      </div>
      <Skeleton shape="block" className="lcz-skel-add" />
    </SkeletonGroup>
  );
}

/**
 * Log customisation (B-N3-04, nbl_/nbd_Log_Customize): the mode's categories in her order — drag (or
 * ↑/↓) to reorder, the pin puts a category on the quick tiles (at most 8), the switch shows / hides it on
 * the log sheet — plus her custom items and «بازگشت به پیش‌فرض». Order / visibility / tiles are a draft
 * saved in one PUT with only the lists she changed (optimistic, rolled back on failure).
 */
export function CustomizeLog({ lookOf, onExit, mode }: CustomizeLogProps) {
  const t = useTranslations('logCustomize');
  const loc = useLocale() as Locale;
  const query = useLogPreferences(mode);
  const save = useSaveLogPreferences(mode);
  const reset = useResetLogPreferences(mode);
  const ids = useId();
  const [drafts, setDrafts] = useState<Drafts | null>(null);
  const [leaving, setLeaving] = useState(false);
  const [confirmReset, setConfirmReset] = useState(false);
  const [status, setStatus] = useState('');
  const [pinRefused, setPinRefused] = useState(false);
  const refocus = useRef<{ code: string; target: 'grip' | 'up' | 'down' } | null>(null);

  const data = query.data;
  // Follow the server while nothing is being edited (first load, a refetch, a custom item write).
  useEffect(() => {
    if (!data) return;
    setDrafts((prev) => {
      if (prev && isDirty(prev.base, prev.draft)) return prev;
      const fresh = draftFromPrefs(data);
      return { base: fresh, draft: fresh };
    });
  }, [data]);

  const draft = drafts?.draft;
  const order = draft?.order ?? [];
  const max = data?.maxPinned ?? 8;
  const dirty = drafts ? isDirty(drafts.base, drafts.draft) : false;

  const edit = (fn: (d: CustomizeDraft) => CustomizeDraft) => {
    save.reset();
    setDrafts((prev) => (prev ? { ...prev, draft: fn(prev.draft) } : prev));
  };

  const labelOf = (code: string) => data?.categories.find((c) => c.code === code)?.label ?? code;
  const announceMove = (index: number) => {
    const code = order[index];
    if (code) {
      setStatus(
        t('moved', {
          label: labelOf(code),
          position: formatNumber(index + 1, loc),
          total: formatNumber(order.length, loc),
        }),
      );
    }
  };

  const move = (from: number, to: number) => edit((d) => moveCategory(d, from, to));
  const reorder = useReorder(
    order.length,
    (from, to, via) => {
      // A keyboard move keeps focus on the moved row's handle once React has reordered the list.
      const code = order[from];
      if (via === 'key' && code) refocus.current = { code, target: 'grip' };
      move(from, to);
    },
    announceMove,
  );

  // The announcement reads the list after the move, so it runs once the new order has rendered.
  useLayoutEffect(() => {
    const pending = refocus.current;
    if (!pending) return;
    refocus.current = null;
    announceMove(order.indexOf(pending.code));
    const wanted = document.getElementById(`lcz-${pending.target}-${pending.code}`);
    const el = wanted && !(wanted as HTMLButtonElement).disabled ? wanted : document.getElementById(`lcz-grip-${pending.code}`);
    if (el && document.activeElement !== el) el.focus();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- runs per reorder only
  }, [order]);

  const doSave = (after?: () => void) => {
    if (!drafts || !data) return;
    const sent = drafts.draft;
    save.mutate(
      { changes: diffDraft(drafts.base, sent), optimistic: applyDraft(data, sent) },
      {
        onSuccess: () => {
          // What she sent is what is stored now (the response leaves hidden categories' tiles out).
          setDrafts((prev) => (prev ? { base: sent, draft: prev.draft } : prev));
          setStatus(t('saved'));
          after?.();
        },
      },
    );
  };

  const doReset = () => {
    reset.mutate(undefined, {
      onSuccess: (prefs: LogPreferences) => {
        const fresh = draftFromPrefs(prefs);
        setDrafts({ base: fresh, draft: fresh });
        setConfirmReset(false);
        setStatus(t('reset.done'));
      },
    });
  };

  const back = () => (dirty ? setLeaving(true) : onExit());

  let body;
  if (query.isPending || (data && !drafts)) {
    body = <LoadingState label={t('loading')} />;
  } else if (query.isError || !data || !draft) {
    body = (
      <EmptyState
        icon="warning"
        title={t('loadError.title')}
        body={t('loadError.body')}
        action={
          <PrimaryButton block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('loadError.retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!order.length) {
    body = <EmptyState icon="grid" title={t('empty.title')} body={t('empty.body')} />;
  } else {
    const full = pinsFull(draft, max);
    const helpId = `${ids}-help`;
    const counterId = `${ids}-count`;
    body = (
      <>
        <div className="lcz-hint">
          <p className="lcz-hint-text">
            <Icon name="pin" size={14} strokeWidth={2.2} className="lcz-hint-pin" />
            {t('hint', { max: formatNumber(max, loc) })}
          </p>
          <span id={counterId} className={full ? 'lcz-count is-full' : 'lcz-count'}>
            {t('pinCount', { count: formatNumber(draft.pinned.length, loc), max: formatNumber(max, loc) })}
          </span>
        </div>
        {full && pinRefused ? (
          <p className="lcz-note" role="status">
            {t('pinsFull', { max: formatNumber(max, loc) })}
          </p>
        ) : null}
        <p id={helpId} className="sr-only">
          {t('dragHelp')}
        </p>
        <ul className="nb-card lcz-list" aria-label={t('listLabel')}>
          {order.map((code, index) => (
            <CategoryRow
              key={code}
              code={code}
              label={labelOf(code)}
              look={lookOf(code)}
              index={index}
              total={order.length}
              hidden={isHidden(draft, code)}
              pinned={isPinned(draft, code)}
              pinLocked={full}
              dragging={reorder.dragging === index}
              handle={reorder.handle(index)}
              rowRef={reorder.rowRef(index)}
              helpId={helpId}
              counterId={counterId}
              onMove={(to) => {
                refocus.current = { code, target: to < index ? 'up' : 'down' };
                move(index, to);
              }}
              onToggleHidden={() => edit((d) => toggleHidden(d, code))}
              onTogglePin={() => {
                const refused = full && !isPinned(draft, code);
                setPinRefused(refused);
                if (!refused) edit((d) => togglePin(d, code, drafts?.base.pinned ?? [], max));
              }}
            />
          ))}
        </ul>
        <CustomItems prefs={data} mode={mode} announce={setStatus} />
        {!data.isDefault || dirty ? (
          confirmReset ? (
            <div className="nb-card lcz-reset-confirm" role="group" aria-labelledby={`${ids}-reset`}>
              <p id={`${ids}-reset`} className="lcz-item-q">
                {t('reset.confirm')}
              </p>
              {reset.isError ? (
                <p className="lcz-field-err" role="alert">
                  {t('reset.error')}
                </p>
              ) : null}
              <div className="lcz-item-acts">
                <button type="button" className="lcz-text-btn is-danger" disabled={reset.isPending} onClick={doReset}>
                  {t('reset.yes')}
                </button>
                <button type="button" className="lcz-text-btn" onClick={() => setConfirmReset(false)}>
                  {t('reset.no')}
                </button>
              </div>
            </div>
          ) : (
            <button type="button" className="lcz-reset" onClick={() => setConfirmReset(true)}>
              <Icon name="refresh" size={16} />
              {t('reset.action')}
            </button>
          )
        ) : null}
        {dirty ? (
          <div className="lcz-foot">
            <div className="lcz-foot-text">
              <span className="lcz-foot-title">{t('unsaved')}</span>
              {save.isError ? (
                <span className="lcz-foot-err" role="alert">
                  {t('saveError')}
                </span>
              ) : null}
            </div>
            <SecondaryButton
              variant="text"
              block={false}
              onClick={() => setDrafts((prev) => (prev ? { ...prev, draft: prev.base } : prev))}
            >
              {t('discard')}
            </SecondaryButton>
            <PrimaryButton block={false} loading={save.isPending} className="lcz-foot-save" onClick={() => doSave()}>
              {t('save')}
            </PrimaryButton>
          </div>
        ) : null}
      </>
    );
  }

  return (
    <div className="lcz">
      <ScreenHeader title={t('title')} subtitle={t('subtitle')} onBack={back} backLabel={t('back')} />
      {body}
      <p className="sr-only" role="status" aria-live="polite">
        {status}
      </p>
      <AppSheet
        open={leaving}
        onClose={() => setLeaving(false)}
        title={t('leave.title')}
        footer={
          <div className="lcz-leave-acts">
            <PrimaryButton loading={save.isPending} onClick={() => doSave(onExit)}>
              {t('leave.save')}
            </PrimaryButton>
            <SecondaryButton onClick={onExit}>{t('leave.discard')}</SecondaryButton>
          </div>
        }
      >
        <p className="lcz-leave-body">{t('leave.body')}</p>
        {save.isError ? (
          <p className="lcz-field-err" role="alert">
            {t('saveError')}
          </p>
        ) : null}
      </AppSheet>
    </div>
  );
}
