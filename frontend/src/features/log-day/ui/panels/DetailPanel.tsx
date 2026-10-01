'use client';

import { useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { AppSheet } from '@/shared/sheet';
import { PrimaryButton, SearchField } from '@/shared/ui';

interface DetailPanelProps {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle: string;
  /** «تأیید» keeps the edits in the sheet's draft; «ذخیره» (Log_Measure) also saves the day. */
  action: 'confirm' | 'save';
  onAction: () => void;
  busy?: boolean;
  /** A search typed here jumps back to the sheet's full list, filtered. */
  onSearch: (query: string) => void;
  children: ReactNode;
}

/**
 * The shell of a log detail panel (Log_Bleeding / Log_Pain / Log_Measure): an inline `AppSheet` that rises
 * over the log sheet, centred title + day, the shared search field, the panel's cards and one pill action.
 */
export function DetailPanel({ open, onClose, title, subtitle, action, onAction, busy, onSearch, children }: DetailPanelProps) {
  const t = useTranslations('logSheet');
  const [query, setQuery] = useState('');
  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      className="lday-panel"
      title={
        <span className="lday-ttl">
          <span className="lday-ttl-main">{title}</span>
          <span className="lday-ttl-sub">{subtitle}</span>
        </span>
      }
      footer={
        <PrimaryButton onClick={onAction} loading={busy}>
          {action === 'save' ? t('panel.save') : t('panel.confirm')}
        </PrimaryButton>
      }
    >
      <div className="lday-panel-body">
        <SearchField
          value={query}
          onValueChange={setQuery}
          label={t('search.label')}
          clearLabel={t('search.clear')}
          placeholder={t('search.placeholder')}
          onSubmit={(q) => {
            if (!q.trim()) return;
            setQuery('');
            onSearch(q);
          }}
        />
        {children}
      </div>
    </AppSheet>
  );
}
