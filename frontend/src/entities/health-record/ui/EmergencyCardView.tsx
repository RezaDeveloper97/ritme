'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

import type { EmergencyCard } from '../api/emergency';
import { CHRONIC_ILLNESSES } from '../model/types';

function isChronic(code: string): code is (typeof CHRONIC_ILLNESSES)[number] {
  return (CHRONIC_ILLNESSES as readonly string[]).includes(code);
}

/**
 * The read-only emergency card (nbl_Rec_Emergency, CB-REC-03 data): name + blood type, then allergy, condition,
 * regular medication, pregnancy, insurance (masked) and the emergency contact (a `tel:` link). Strings come from
 * `common.emergencyCard` so it can render on the app-lock screen, where only the shell namespaces are loaded.
 * `variant="lock"` is the app-lock view (CB-PRIV-01): the server already sends only the minimal card (first name,
 * no insurance) and the view never renders insurance either.
 */
export function EmergencyCardView({ card, variant = 'owner' }: { card: EmergencyCard; variant?: 'owner' | 'lock' }) {
  const t = useTranslations('common.emergencyCard');
  const loc = useLocale() as Locale;
  const none = t('none');
  const join = (items: string[]) => (items.length ? items.join(t('separator')) : none);

  const rows: { key: string; label: string; value: string }[] = [
    { key: 'allergies', label: t('allergies'), value: join(card.allergies ?? []) },
    {
      key: 'conditions',
      label: t('conditions'),
      value: join(card.conditions.map((c) => (isChronic(c) ? t(`chronic.${c}`) : c))),
    },
    {
      key: 'medications',
      label: t('medications'),
      value: join(card.medications.map((m) => (m.dose ? `${m.title} ${m.dose}` : m.title))),
    },
  ];
  if (card.pregnancyWeek !== null) {
    rows.push({ key: 'status', label: t('status'), value: t('pregnant', { week: formatNumber(card.pregnancyWeek, loc) }) });
  }
  if (card.insurance && variant === 'owner') {
    // LTR isolate (U+2066 … U+2069): «•••• 4821» keeps its order inside RTL text.
    const masked = `⁦${card.insurance.masked}⁩`;
    rows.push({ key: 'insurance', label: t('insurance'), value: card.insurance.label ? `${card.insurance.label} · ${masked}` : masked });
  }
  const contact = card.contact;
  // The visible label is a name or relation only; the number itself stays in the tel: href (LOW-2).
  const contactName = contact?.name || contact?.relation || '';

  return (
    <article className="nb-card ecard" aria-labelledby="ecard-name">
      <header className="ecard-head">
        <h2 id="ecard-name" className="ecard-name">
          {card.name || t('unnamed')}
        </h2>
        {card.bloodType ? (
          <span className="ecard-blood" aria-label={`${t('bloodType')} ${card.bloodType}`}>
            <bdi dir="ltr">{card.bloodType}</bdi>
          </span>
        ) : null}
      </header>
      <dl className="ecard-rows">
        {rows.map((r) => (
          <div key={r.key} className="ecard-row">
            <dt className="ecard-label">{r.label}</dt>
            <dd className="ecard-value">{r.value}</dd>
          </div>
        ))}
        <div className="ecard-row">
          <dt className="ecard-label">{t('contact')}</dt>
          <dd className="ecard-value">
            {contact?.phone ? (
              <a className="ecard-call" href={`tel:${contact.phone}`}>
                {contactName ? t('call', { name: contactName }) : t('callGeneric')}
              </a>
            ) : (
              contactName || none
            )}
          </dd>
        </div>
      </dl>
    </article>
  );
}
