'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import type { LogCategory } from '@/entities/health-log';
import { Icon, type IconName, InfoNote, ListGroup, ListRow, type Tone } from '@/shared/ui';

import { statusCategories } from '../model/review';

interface VoiceEntryProps {
  mode: string | null;
  categories: readonly LogCategory[];
  entries: readonly { key: string; label: string; summary: string }[];
  iconOf: (category: string) => IconName;
  toneOf: (category: string) => Tone;
  /** Status line under the mic (permission wait, too-short clip); the start hint otherwise. */
  status: string | null;
  onStart: () => void;
  openSection: (category: string) => void;
  /** An error / nothing-found card shown under the hero. */
  notice?: ReactNode;
}

const EXAMPLES = ['pain', 'pill', 'flash'] as const;

/**
 * nbl_Voice_Entry: «بگو امروز چطور بودی» hero with the big mic, «مثلاً بگو» example quotes, «ثبت‌های امروز»
 * (what is already logged in three of the mode's categories — a row opens that section in «ثبت دستی») and
 * the privacy note.
 */
export function VoiceEntry({ mode, categories, entries, iconOf, toneOf, status, onStart, openSection, notice }: VoiceEntryProps) {
  const t = useTranslations('voiceLog');
  const rows = statusCategories(mode, categories);
  const sep = t('separator');
  const summaryOf = (code: string) => {
    const list = entries.filter((e) => e.key.startsWith(`${code}.`)).map((e) => e.summary);
    return list.length ? list.join(sep) : t('today.empty');
  };
  return (
    <div className="vlog">
      <section className="nb-card vlog-hero" aria-labelledby="vlog-hero-title">
        <h3 id="vlog-hero-title" className="vlog-hero-title">
          {t('title')}
        </h3>
        <p className="vlog-lead">{t('lead')}</p>
        <span className="vlog-halo">
          <button type="button" className="vlog-go" aria-label={t('start')} onClick={onStart}>
            <Icon name="mic" size={40} strokeWidth={1.9} />
          </button>
        </span>
        <p className="vlog-hint" aria-live="polite">
          {status ?? t('startHint')}
        </p>
      </section>

      {notice}

      <section className="vlog-sec" aria-labelledby="vlog-ex-title">
        <h3 id="vlog-ex-title" className="vlog-sec-title">
          {t('examples.title')}
        </h3>
        <ul className="vlog-exs">
          {EXAMPLES.map((key) => (
            <li key={key} className="vlog-ex">
              {t(`examples.${key}`)}
            </li>
          ))}
        </ul>
      </section>

      {rows.length ? (
        <section className="vlog-sec" aria-labelledby="vlog-today-title">
          <h3 id="vlog-today-title" className="vlog-sec-title">
            {t('today.title')}
          </h3>
          <ListGroup className="vlog-today">
            {rows.map((cat) => (
              <ListRow
                key={cat.code}
                icon={iconOf(cat.code)}
                iconTone={toneOf(cat.code)}
                title={cat.label}
                description={summaryOf(cat.code)}
                onClick={() => openSection(cat.code)}
              />
            ))}
          </ListGroup>
        </section>
      ) : null}

      <InfoNote icon="shield" className="vlog-privacy">
        {t('privacy')}
      </InfoNote>
    </div>
  );
}
