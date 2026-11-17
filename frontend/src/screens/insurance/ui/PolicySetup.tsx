'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  categoryKey,
  type InsurancePolicy,
  useInsurers,
  useSaveInsurancePolicy,
} from '@/entities/insurance';
import type { Locale } from '@/shared/i18n';
import { type DateParts, formatLongDate, fromApiDate, partsToDate, toApiDate, toParts } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import {
  CalendarPicker,
  Icon,
  InfoNote,
  LocaleNumberField,
  PillChip,
  PrimaryButton,
  SecondaryButton,
} from '@/shared/ui';

import {
  draftFromPolicy,
  draftProblems,
  draftToPayload,
  emptyDraft,
  FORM_CATEGORIES,
  MAX_MEMBERS,
  MEMBER_KINDS,
  type PolicyDraft,
} from '../model/setup';

type DateField = 'validFrom' | 'validTo';

function DateSheet({ value, title, onClose, onPick }: { value: string | null; title: string; onClose: () => void; onPick: (api: string) => void }) {
  const t = useTranslations('insurance.setup');
  const locale = useLocale() as Locale;
  const [parts, setParts] = useState<DateParts | null>(value ? toParts(fromApiDate(value), locale) : null);
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={title}
      footer={
        <div className="ins-sheet-btns">
          <SecondaryButton block={false} onClick={onClose}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} disabled={!parts} onClick={() => parts && onPick(toApiDate(partsToDate(parts, locale)))} {...pilot('ins-setup.date-done')}>
            {t('done')}
          </PrimaryButton>
        </div>
      }
    >
      <CalendarPicker value={parts} onSelect={setParts} pilot={pilot('ins-setup.calendar', 'calendar')} />
    </AppSheet>
  );
}

/**
 * Setup / edit form of the self-tracked policy (CB-INS-03). Everything typed here stays in her own record: nothing is
 * sent to an insurer. A form, so the screen mounts no bottom nav (IA_Nav).
 */
export function PolicySetup({ policy, onDone, onShowInfo }: { policy: InsurancePolicy | null; onDone: () => void; onShowInfo: () => void }) {
  const t = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const insurers = useInsurers();
  // An existing policy is saved by id (it may be a renewal that has not started yet); none → PUT /policy creates it.
  const save = useSaveInsurancePolicy(policy?.id ?? null);
  const [draft, setDraft] = useState<PolicyDraft>(() => (policy ? draftFromPolicy(policy, t('setup.selfName')) : emptyDraft(t('setup.selfName'))));
  const [picker, setPicker] = useState<DateField | null>(null);
  const [touched, setTouched] = useState(false);

  const problems = draftProblems(draft);
  const patch = (next: Partial<PolicyDraft>) => setDraft((d) => ({ ...d, ...next }));
  const patchCoverage = (category: string, next: Partial<PolicyDraft['coverages'][string]>) =>
    setDraft((d) => ({ ...d, coverages: { ...d.coverages, [category]: { ...d.coverages[category], ...next } } }));
  const dateText = (api: string | null) => (api ? formatLongDate(fromApiDate(api), locale) : t('setup.notSet'));

  const submit = () => {
    setTouched(true);
    if (Object.keys(problems).length > 0) return;
    save.mutate(draftToPayload(draft), { onSuccess: onDone });
  };

  return (
    <div className="ins-form">
      {policy ? null : <p className="ins-intro">{t('setup.intro')}</p>}

      <section className="nb-card ins-card" aria-labelledby="ins-f-insurer">
        <h2 id="ins-f-insurer" className="ins-h">
          {t('setup.insurer')}
        </h2>
        <div className="ins-chips" {...pilot('ins-setup.insurer', 'choice')}>
          {(insurers.data ?? []).map((i) => (
            <PillChip
              key={i.id}
              pressed={!draft.custom && draft.insurerId === i.id}
              {...pilotOption(String(i.id))}
              onPressedChange={() => patch({ insurerId: i.id, custom: false })}
            >
              {i.name}
            </PillChip>
          ))}
          <PillChip pressed={draft.custom} onPressedChange={() => patch({ insurerId: null, custom: true })} {...pilotOption('other')}>
            {t('setup.insurerOther')}
          </PillChip>
        </div>
        {draft.custom || (insurers.isSuccess && insurers.data.length === 0) ? (
          <label className="fld-label">
            <span className="fld-label-t">{t('setup.insurerName')}</span>
            <input
              className="field fld-input"
              value={draft.insurerName}
              maxLength={120}
              autoComplete="off"
              onChange={(e) => patch({ insurerName: e.target.value, custom: true, insurerId: null })}
              {...pilot('ins-setup.insurer-name', 'text')}
            />
          </label>
        ) : null}
        {touched && problems.insurer ? (
          <p className="ins-problem" role="alert">
            {t('setup.problems.insurer')}
          </p>
        ) : null}
        <label className="fld-label">
          <span className="fld-label-t">{t('setup.plan')}</span>
          <input className="field fld-input" value={draft.planName} maxLength={120} autoComplete="off" onChange={(e) => patch({ planName: e.target.value })} {...pilot('ins-setup.plan', 'text')} />
        </label>
        <label className="fld-label">
          <span className="fld-label-t">{t('setup.number')}</span>
          <input
            className="field fld-input ins-ltr"
            value={draft.policyNumber}
            maxLength={40}
            inputMode="numeric"
            autoComplete="off"
            onChange={(e) => patch({ policyNumber: e.target.value })}
            {...pilot('ins-setup.number', 'text')}
          />
          <span className="ins-hint">{policy?.numberLast4 ? t('setup.numberKept') : t('setup.numberHint')}</span>
        </label>
      </section>

      <section className="nb-card ins-card" aria-labelledby="ins-f-dates">
        <h2 id="ins-f-dates" className="ins-h">
          {t('setup.validFrom')} / {t('setup.validTo')}
        </h2>
        <div className="ins-dates">
          {(['validFrom', 'validTo'] as const).map((field) => (
            <button key={field} type="button" className="ins-date" onClick={() => setPicker(field)} {...pilot(`ins-setup.date:${field}`)}>
              <span className="ins-date-l">{t(field === 'validFrom' ? 'setup.validFrom' : 'setup.validTo')}</span>
              <span className="ins-date-v">
                <Icon name="calendar" size={16} />
                {dateText(draft[field])}
              </span>
            </button>
          ))}
        </div>
        {touched && problems.dates ? (
          <p className="ins-problem" role="alert">
            {t('setup.problems.dates')}
          </p>
        ) : null}
        <LocaleNumberField
          label={t('setup.deadline')}
          locale={locale}
          value={draft.claimDeadlineDays}
          max={365}
          pilot={pilot('ins-setup.deadline')}
          onChange={(v) => patch({ claimDeadlineDays: v === undefined ? undefined : Math.min(365, v) })}
        />
      </section>

      <section className="nb-card ins-card" aria-labelledby="ins-f-members">
        <h2 id="ins-f-members" className="ins-h">
          {t('setup.membersTitle')}
        </h2>
        {draft.members.map((m, i) => (
          <div key={m.id ?? `new-${i}`} className="ins-member">
            <div className="ins-chips" {...pilot(`ins-setup.member-kind:${i}`, 'choice')}>
              {MEMBER_KINDS.map((k) => (
                <PillChip
                  key={k}
                  pressed={m.kind === k}
                  {...pilotOption(k)}
                  onPressedChange={() => patch({ members: draft.members.map((x, j) => (j === i ? { ...x, kind: k } : x)) })}
                >
                  {t(`setup.memberKinds.${k}`)}
                </PillChip>
              ))}
            </div>
            <div className="ins-member-row">
              <label className="fld-label ins-member-name">
                <span className="fld-label-t">{t('setup.memberName')}</span>
                <input
                  className="field fld-input"
                  value={m.name}
                  maxLength={64}
                  autoComplete="off"
                  onChange={(e) => patch({ members: draft.members.map((x, j) => (j === i ? { ...x, name: e.target.value } : x)) })}
                  {...pilot(`ins-setup.member-name:${i}`, 'text')}
                />
              </label>
              {draft.members.length > 1 ? (
                <button type="button" className="ins-icon-btn" aria-label={t('setup.removeMember')} onClick={() => patch({ members: draft.members.filter((_, j) => j !== i) })}>
                  <Icon name="trash" size={18} />
                </button>
              ) : null}
            </div>
          </div>
        ))}
        {touched && problems.member ? (
          <p className="ins-problem" role="alert">
            {t('setup.problems.member')}
          </p>
        ) : null}
        {draft.members.length < MAX_MEMBERS ? (
          <SecondaryButton icon="plus" variant="text" block={false} onClick={() => patch({ members: [...draft.members, { kind: 'spouse', name: '' }] })} {...pilot('ins-setup.member-add')}>
            {t('setup.addMember')}
          </SecondaryButton>
        ) : null}
      </section>

      <section className="nb-card ins-card" aria-labelledby="ins-f-cov">
        <h2 id="ins-f-cov" className="ins-h">
          {t('setup.coverageTitle')}
        </h2>
        <p className="ins-hint">{t('setup.coverageHint')}</p>
        {FORM_CATEGORIES.map((category) => {
          const c = draft.coverages[category];
          return (
            <fieldset key={category} className="ins-cov">
              <legend className="ins-cov-t">{t(`categories.${categoryKey(category)}`)}</legend>
              <div className="ins-cov-fields">
                <LocaleNumberField label={t('setup.cap')} locale={locale} value={c.capMillions} max={99999} pilot={pilot(`ins-setup.cap:${category}`)} onChange={(v) => patchCoverage(category, { capMillions: v })} />
                <LocaleNumberField label={t('setup.deductible')} locale={locale} value={c.deductiblePct} max={100} pilot={pilot(`ins-setup.deductible:${category}`)} onChange={(v) => patchCoverage(category, { deductiblePct: v === undefined ? undefined : Math.min(100, v) })} />
                {category === 'delivery' ? (
                  <LocaleNumberField label={t('setup.waiting')} locale={locale} value={c.waitingMonths} max={36} pilot={pilot(`ins-setup.waiting:${category}`)} onChange={(v) => patchCoverage(category, { waitingMonths: v === undefined ? undefined : Math.min(36, v) })} />
                ) : null}
              </div>
            </fieldset>
          );
        })}
      </section>

      <InfoNote icon="lock">{t('setup.privacy')}</InfoNote>
      {save.isError ? (
        <p className="ins-problem" role="alert">
          {t('setup.saveError')}
        </p>
      ) : null}
      <PrimaryButton loading={save.isPending} onClick={submit} {...pilot('ins-setup.save')}>
        {t('setup.save')}
      </PrimaryButton>
      {policy ? (
        <SecondaryButton variant="text" onClick={onDone}>
          {t('setup.cancel')}
        </SecondaryButton>
      ) : (
        <SecondaryButton variant="text" icon="info" onClick={onShowInfo}>
          {t('setup.howLink')}
        </SecondaryButton>
      )}

      {picker ? (
        <DateSheet
          title={t(picker === 'validFrom' ? 'setup.validFrom' : 'setup.validTo')}
          value={draft[picker]}
          onClose={() => setPicker(null)}
          onPick={(api) => {
            patch({ [picker]: api });
            setPicker(null);
          }}
        />
      ) : null}
    </div>
  );
}
