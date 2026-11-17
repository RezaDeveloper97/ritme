'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  CLAIM_COST_TYPES,
  type ClaimDocLabel,
  costTypeKey,
  maskedTail,
  useCreateClaim,
  useInsurancePolicy,
} from '@/entities/insurance';
import { KIND_LOOK, type TimelineItemKind } from '@/entities/health-record';
import { type Locale, useRouter } from '@/shared/i18n';
import { type DateParts, formatLongDate, fromApiDate, partsToDate, toApiDate, today, toParts } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import {
  Avatar,
  CalendarPicker,
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  LocaleNumberField,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';
import { ClaimDocPicker } from '@/widgets/claim-documents';

import { type ClaimDraft, type DraftDoc, DOC_SLOTS, draftProblems, draftToPayload, MAX_AMOUNT, MAX_CENTRE, withDoc } from '../model/draft';

function DateSheet({ value, onClose, onPick }: { value: string; onClose: () => void; onPick: (api: string) => void }) {
  const t = useTranslations('insurance.claimNew');
  const locale = useLocale() as Locale;
  const [parts, setParts] = useState<DateParts | null>(toParts(fromApiDate(value), locale));
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={t('date')}
      footer={
        <div className="ins-sheet-btns">
          <SecondaryButton block={false} onClick={onClose}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} disabled={!parts} onClick={() => parts && onPick(toApiDate(partsToDate(parts, locale)))} {...pilot('ins-claim-new.date-done')}>
            {t('done')}
          </PrimaryButton>
        </div>
      }
    >
      <CalendarPicker value={parts} onSelect={setParts} pilot={pilot('ins-claim-new.calendar', 'calendar')} />
    </AppSheet>
  );
}

function Form({ memberOptions, ibanLast4, defaultMemberId }: { memberOptions: { id: number; name: string }[]; ibanLast4: string | null; defaultMemberId: number | null }) {
  const t = useTranslations('insurance');
  const tn = useTranslations('insurance.claimNew');
  const tk = useTranslations('record.kinds');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const create = useCreateClaim();
  const [draft, setDraft] = useState<ClaimDraft>(() => ({
    memberId: defaultMemberId,
    costType: 'paraclinic',
    centreName: '',
    serviceDate: toApiDate(today()),
    amount: undefined,
    docs: [],
  }));
  const [pickFor, setPickFor] = useState<ClaimDocLabel | null>(null);
  const [datePicker, setDatePicker] = useState(false);
  const [touched, setTouched] = useState(false);

  const problems = draftProblems(draft);
  const patch = (next: Partial<ClaimDraft>) => setDraft((d) => ({ ...d, ...next }));
  const docOf = (label: ClaimDocLabel): DraftDoc | undefined => draft.docs.find((d) => d.label === label);
  const extras = draft.docs.filter((d) => d.label === 'other');
  const tail = maskedTail(ibanLast4);

  const submit = () => {
    setTouched(true);
    if (problems.length > 0) return;
    create.mutate(draftToPayload(draft), { onSuccess: (claim) => router.push(`/insurance/claims/${claim.id}`) });
  };

  const kindOf = (kind: string): TimelineItemKind => (kind in KIND_LOOK ? (kind as TimelineItemKind) : 'other');
  const docLook = (kind: string) => KIND_LOOK[kindOf(kind)];

  return (
    <>
      {memberOptions.length > 0 ? (
        <section className="ins-sec" aria-labelledby="ins-n-who">
          <h2 id="ins-n-who" className="ins-label">
            {tn('who')}
          </h2>
          <div className="ins-whos" role="group" aria-labelledby="ins-n-who" {...pilot('ins-claim-new.who', 'choice')}>
            {memberOptions.map((m) => (
              <button key={m.id} type="button" className="ins-who" aria-pressed={draft.memberId === m.id} {...pilotOption(String(m.id))} onClick={() => patch({ memberId: m.id })}>
                <Avatar name={m.name} size="md" />
                <span className="ins-who-t">{m.name}</span>
              </button>
            ))}
          </div>
        </section>
      ) : null}

      <section className="ins-sec" aria-labelledby="ins-n-type">
        <h2 id="ins-n-type" className="ins-label">
          {tn('costType')}
        </h2>
        <div className="ins-chips" role="group" aria-labelledby="ins-n-type" {...pilot('ins-claim-new.cost-type', 'choice')}>
          {CLAIM_COST_TYPES.map((c) => (
            <PillChip key={c} pressed={draft.costType === c} onPressedChange={() => patch({ costType: c })} {...pilotOption(c)}>
              {t(`costTypes.${costTypeKey(c)}`)}
            </PillChip>
          ))}
        </div>
      </section>

      <section className="nb-card ins-card" aria-label={tn('details')}>
        <label className="fld-label">
          <span className="fld-label-t">{tn('centre')}</span>
          <input
            className="field fld-input"
            value={draft.centreName}
            maxLength={MAX_CENTRE}
            autoComplete="off"
            placeholder={tn('centrePlaceholder')}
            onChange={(e) => patch({ centreName: e.target.value })}
            {...pilot('ins-claim-new.centre', 'text')}
          />
        </label>
        <button type="button" className="ins-date ins-date-wide" onClick={() => setDatePicker(true)} {...pilot('ins-claim-new.date')}>
          <span className="ins-date-l">{tn('date')}</span>
          <span className="ins-date-v">
            <Icon name="calendar" size={16} />
            {formatLongDate(fromApiDate(draft.serviceDate), locale)}
          </span>
        </button>
        <LocaleNumberField
          label={tn('amount')}
          locale={locale}
          value={draft.amount}
          max={MAX_AMOUNT}
          pilot={pilot('ins-claim-new.amount')}
          onChange={(v) => patch({ amount: v })}
        />
        {touched && problems.includes('amount') ? (
          <p className="ins-problem" role="alert">
            {tn('problems.amount')}
          </p>
        ) : null}
      </section>

      <section className="ins-sec" aria-labelledby="ins-n-docs">
        <div className="ins-sec-head">
          <h2 id="ins-n-docs" className="ins-label">
            {tn('docs')}
          </h2>
          <button type="button" className="ins-more ins-more-btn" onClick={() => setPickFor('other')} {...pilot('ins-claim-new.doc-add:other')}>
            {tn('fromRecord')}
          </button>
        </div>
        <ul className="nb-card ins-docs ins-doc-slots">
          {DOC_SLOTS.map((slot) => {
            const attached = docOf(slot);
            const look = attached ? docLook(attached.kind) : null;
            return (
              <li key={slot} className="ins-slot">
                <IconCircle icon={look?.icon ?? 'fileDoc'} tone={attached ? 'data' : 'neutral'} />
                <span className="ins-claim-text">
                  <span className="ins-claim-t">{tn(`slots.${slot}`)}</span>
                  <span className="ins-claim-s">{attached ? (attached.title ?? tk(kindOf(attached.kind))) : tn(`slotHints.${slot}`)}</span>
                </span>
                {attached ? (
                  <StatusPill tone="data" icon="check">
                    {tn('attached')}
                  </StatusPill>
                ) : (
                  <SecondaryButton block={false} onClick={() => setPickFor(slot)} {...pilot(`ins-claim-new.doc-add:${slot}`)}>
                    {tn('add')}
                  </SecondaryButton>
                )}
              </li>
            );
          })}
          {extras.map((d) => (
            <li key={d.id} className="ins-slot">
              <IconCircle icon={docLook(d.kind).icon} tone="data" />
              <span className="ins-claim-text">
                <span className="ins-claim-t">{d.title ?? tk(kindOf(d.kind))}</span>
                <span className="ins-claim-s">{tn('slots.other')}</span>
              </span>
              <button type="button" className="ins-icon-btn" aria-label={tn('removeDoc')} onClick={() => patch({ docs: draft.docs.filter((x) => x.id !== d.id) })}>
                <Icon name="trash" size={18} />
              </button>
            </li>
          ))}
        </ul>
      </section>

      {tail ? (
        <section className="nb-card ins-iban" aria-label={tn('iban')}>
          <span className="ins-claim-t">{tn('iban')}</span>
          <bdi dir="ltr" className="ins-iban-v">
            {tn('ibanMask', { tail: ibanLast4 ?? '' })}
          </bdi>
        </section>
      ) : null}

      <InfoNote icon="lock">{tn('selfTrack')}</InfoNote>
      {create.isError ? (
        <p className="ins-problem" role="alert">
          {tn('saveError')}
        </p>
      ) : null}

      <div className="ins-cta">
        <PrimaryButton icon="check" loading={create.isPending} onClick={submit} {...pilot('ins-claim-new.submit')}>
          {tn('submit')}
        </PrimaryButton>
      </div>

      {datePicker ? (
        <DateSheet
          value={draft.serviceDate}
          onClose={() => setDatePicker(false)}
          onPick={(api) => {
            patch({ serviceDate: api });
            setDatePicker(false);
          }}
        />
      ) : null}
      <ClaimDocPicker
        open={pickFor !== null}
        onClose={() => setPickFor(null)}
        takenIds={draft.docs.map((d) => d.id)}
        onPick={(doc) => {
          const label = pickFor ?? 'other';
          setDraft((d) => ({ ...d, docs: withDoc(d.docs, { id: doc.id, label, title: doc.title, kind: doc.kind }) }));
        }}
      />
    </>
  );
}

/**
 * New claim (CB-INS-04, nbl_Ins_ClaimNew) at `/insurance/claims/new`: person, cost type, centre, date, invoice amount
 * and the documents from her own record (or the camera). It only records the claim for her to follow — nothing is sent
 * to an insurer. A form, so no bottom nav (IA_Nav).
 */
export function InsuranceClaimNewPage() {
  const t = useTranslations('insurance');
  const router = useRouter();
  const query = useInsurancePolicy();
  const policy = query.data?.policy ?? null;

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="ins-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="shield"
        title={t('error')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!policy) {
    body = (
      <EmptyState
        icon="shield"
        title={t('claimNew.noPolicy')}
        action={<PrimaryButton onClick={() => router.push('/insurance')}>{t('setup.save')}</PrimaryButton>}
      />
    );
  } else {
    const self = policy.members.find((m) => m.kind === 'self') ?? policy.members[0] ?? null;
    body = (
      <Form
        memberOptions={policy.members.map((m) => ({ id: m.id, name: m.name }))}
        ibanLast4={policy.ibanLast4}
        defaultMemberId={self?.id ?? null}
      />
    );
  }

  return (
    <div className="view ins-page ins-has-cta">
      <SkyLayer />
      <div className="scroll ins-scroll">
        <ScreenHeader title={t('claimNew.title')} onBack={() => router.push('/insurance/claims')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}
