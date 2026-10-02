'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { LogCategory, LogParam } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatDecimal } from '@/shared/lib/date';
import { ChipGroup, Icon, IconCircle, type IconName, PillChip, PrimaryButton, SecondaryButton, type Tone } from '@/shared/ui';

import { groupItems, hasHeavyBleeding, includedItems, type ReviewGroup, type ReviewItem } from '../model/review';
import { diaryLook } from './look';

interface VoiceReviewProps {
  items: readonly ReviewItem[];
  transcript: string;
  categories: readonly LogCategory[];
  iconOf: (category: string) => IconName;
  toneOf: (category: string) => Tone;
  onToggle: (id: string) => void;
  onChoose: (id: string, index: number) => void;
  onRevalue: (id: string, value: string, label: string) => void;
  /** Pencil of a log group: merge and open that section in «ثبت دستی». */
  onEdit: (category: string) => void;
  /** «چیزی جا افتاده؟ اضافه کن»: merge and switch to «ثبت دستی». */
  onAdd: () => void;
  onConfirm: () => void;
  onAgain: () => void;
  saving: boolean;
  error: string | null;
}

/** A single-choice param short enough for the board's segmented row (bleeding «لکه · کم · متوسط · زیاد»). */
function segmentedParam(categories: readonly LogCategory[], item: ReviewItem): LogParam | null {
  if (item.target !== 'log' || item.choices.length > 1) return null;
  const p = categories.find((c) => c.code === item.choice.category)?.params.find((x) => x.code === item.choice.param);
  return p && p.type === 'single' && p.options.length >= 2 && p.options.length <= 5 ? p : null;
}

/**
 * nbl_Voice_Review «این‌ها را فهمیدیم · درست است؟»: summary + the full transcript, one card per category /
 * diary with its chips (tap = leave out / bring back), a segmented row for short single-choice params, the
 * «مطمئن نیستیم · منظورت کدام بود؟» chooser from `options[]`, «چیزی جا افتاده؟», the heavy-bleeding note and
 * «تأیید و ثبت N مورد» / «دوباره بگو». The API returns no per-item quote, so none is shown (the transcript
 * is one tap away under the summary).
 */
export function VoiceReview({
  items,
  transcript,
  categories,
  iconOf,
  toneOf,
  onToggle,
  onChoose,
  onRevalue,
  onEdit,
  onAdd,
  onConfirm,
  onAgain,
  saving,
  error,
}: VoiceReviewProps) {
  const t = useTranslations('voiceLog');
  const locale = useLocale() as Locale;
  const label = (text: string) => formatDecimal(text, locale);
  const chosen = includedItems(items);
  const groups = groupItems(items);

  const look = (g: ReviewGroup): { icon: IconName; tone: Tone; title: string } => {
    if (g.target === 'log') {
      return { icon: iconOf(g.category), tone: toneOf(g.category), title: categories.find((c) => c.code === g.category)?.label ?? g.category };
    }
    const d = diaryLook(g.target);
    return { ...d, title: t(`targets.${g.target}`) };
  };

  return (
    <div className="vlog vlog-review">
      <header className="vlog-step-head">
        <h3 className="vlog-step-title">{t('review.title')}</h3>
        <p className="vlog-step-sub">{t('review.sub')}</p>
      </header>

      <section className="nb-card vlog-summary" aria-labelledby="vlog-summary-title">
        <h4 id="vlog-summary-title" className="vlog-summary-title">
          <Icon name="sparkle" size={16} className="vlog-summary-icon" />
          {t('review.summary')}
        </h4>
        <p className="vlog-summary-text">{chosen.length ? chosen.map((i) => label(i.choice.label)).join(t('separator')) : t('review.confirm', { count: 0 })}</p>
        <details className="vlog-transcript">
          <summary>{t('review.transcript')}</summary>
          <p className="vlog-transcript-text">«{transcript}»</p>
        </details>
      </section>

      <section className="nb-card vlog-groups">
        {groups.map((g) => {
          const l = look(g);
          return (
            <div key={g.key} className="vlog-group">
              <div className="vlog-group-head">
                <IconCircle icon={l.icon} tone={l.tone} size="sm" />
                <h4 className="vlog-group-title">{l.title}</h4>
                {g.items.some((i) => i.choices.length > 1) ? <span className="vlog-unsure">{t('review.unsure')}</span> : null}
                {g.target === 'log' ? (
                  <button
                    type="button"
                    className="vlog-edit"
                    aria-label={t('review.edit', { label: l.title })}
                    onClick={() => onEdit(g.category)}
                  >
                    <Icon name="pencil" size={16} />
                  </button>
                ) : null}
              </div>
              {g.items.map((item) => {
                const seg = segmentedParam(categories, item);
                if (item.choices.length > 1) {
                  return (
                    <div key={item.id} className="vlog-choose">
                      <p className="vlog-which">{t('review.which')}</p>
                      <ChipGroup label={t('review.which')}>
                        {item.choices.map((c, index) => (
                          <PillChip
                            key={`${c.category}.${c.param}.${c.item ?? ''}`}
                            pressed={item.included && c === item.choice}
                            mode="multi"
                            tone={l.tone}
                            className="vlog-chip"
                            onClick={() => (item.included && c === item.choice ? onToggle(item.id) : onChoose(item.id, index))}
                          >
                            {label(c.label)}
                          </PillChip>
                        ))}
                      </ChipGroup>
                    </div>
                  );
                }
                if (seg) {
                  return (
                    <div key={item.id} className="vlog-seg" role="radiogroup" aria-label={seg.label}>
                      {seg.options.map((o) => {
                        const on = item.included && item.choice.value === o.value;
                        return (
                          <button
                            key={o.value}
                            type="button"
                            role="radio"
                            aria-checked={on}
                            className="vlog-seg-opt"
                            onClick={() => (on ? onToggle(item.id) : onRevalue(item.id, o.value, `${seg.label} · ${o.label}`))}
                          >
                            {o.label}
                          </button>
                        );
                      })}
                    </div>
                  );
                }
                return null;
              })}
              {g.items.some((i) => i.choices.length === 1 && !segmentedParam(categories, i)) ? (
                <ChipGroup label={l.title}>
                  {g.items
                    .filter((i) => i.choices.length === 1 && !segmentedParam(categories, i))
                    .map((item) => (
                      <PillChip
                        key={item.id}
                        pressed={item.included}
                        mode="multi"
                        tone={l.tone}
                        className="vlog-chip"
                        onClick={() => onToggle(item.id)}
                      >
                        {label(item.choice.label)}
                      </PillChip>
                    ))}
                </ChipGroup>
              ) : null}
            </div>
          );
        })}
        <p className="vlog-help">{t('review.help')}</p>
      </section>

      <button type="button" className="vlog-add" onClick={onAdd}>
        <Icon name="plus" size={18} />
        {t('review.add')}
      </button>

      {hasHeavyBleeding(items) ? (
        <aside role="note" className="vlog-alert">
          <IconCircle icon="warning" tone="period" size="sm" />
          <div>
            <p className="vlog-alert-title">{t('review.noteTitle')}</p>
            <p className="vlog-alert-body">{t('review.noteBody')}</p>
          </div>
        </aside>
      ) : null}

      <div className="vlog-cta">
        {error ? (
          <p className="vlog-error" role="alert">
            {error}
          </p>
        ) : null}
        <PrimaryButton icon="check" onClick={onConfirm} loading={saving} disabled={chosen.length === 0}>
          {saving ? t('review.saving') : t('review.confirm', { count: chosen.length })}
        </PrimaryButton>
        <SecondaryButton variant="text" icon="mic" onClick={onAgain} disabled={saving}>
          {t('review.again')}
        </SecondaryButton>
      </div>
    </div>
  );
}
