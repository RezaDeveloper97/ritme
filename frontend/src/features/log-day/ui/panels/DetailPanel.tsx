'use client';

import { useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { pilot } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import { PrimaryButton, SearchField } from '@/shared/ui';

import { logPilotId } from '../../model/pilot-targets';
import '@/shared/ui/styles/lday.css';

interface DetailPanelProps {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle: string;
  /** «تأیید»: keeps the edits in the sheet's draft (the footer «ذخیره» saves the day). */
  onAction: () => void;
  /** A search typed here jumps back to the sheet's full list, filtered. */
  onSearch: (query: string) => void;
  children: ReactNode;
}

/**
 * The shell of a log detail panel (Log_Pain — bleeding and measurements are edited in their accordion): an inline `AppSheet` that rises
 * over the log sheet, centred title + day, the shared search field, the panel's cards and one pill action.
 */
export function DetailPanel({ open, onClose, title, subtitle, onAction, onSearch, children }: DetailPanelProps) {
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
        <PrimaryButton onClick={onAction} {...pilot(logPilotId.panelConfirm)}>
          {t('panel.confirm')}
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
