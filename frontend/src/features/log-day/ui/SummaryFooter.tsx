'use client';

import { useTranslations } from 'next-intl';

import { PrimaryButton } from '@/shared/ui';

import type { LogEntry } from '../model/draft';

interface SummaryFooterProps {
  entries: readonly LogEntry[];
  dirty: boolean;
  saving: boolean;
  saveError: boolean;
  justSaved: boolean;
  onSave: () => void;
}

/** Sticky «۴ مورد ثبت شده · گرفتگی شکم، زودرنج…» + «ذخیره» (nbl_Log_Sheet_Cycle footer). */
export function SummaryFooter({ entries, dirty, saving, saveError, justSaved, onSave }: SummaryFooterProps) {
  const t = useTranslations('logSheet');
  const count = entries.length;
  const list = entries.map((e) => e.label).join(t('separator'));
  return (
    <div className="lday-foot">
      <div className="lday-foot-text" aria-live="polite">
        <span className="lday-foot-count">
          {saveError ? t('footer.error') : justSaved ? t('footer.saved') : t('footer.count', { count })}
        </span>
        {list && !saveError ? <span className="lday-foot-list">{list}</span> : null}
      </div>
      <PrimaryButton block={false} className="lday-foot-save" onClick={onSave} disabled={!dirty} loading={saving}>
        {saving ? t('footer.saving') : t('footer.save')}
      </PrimaryButton>
    </div>
  );
}
