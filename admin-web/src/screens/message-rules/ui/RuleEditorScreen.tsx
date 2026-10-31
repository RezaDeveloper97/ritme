'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState, type ReactNode } from 'react';

import { useCan } from '@/features/auth';
import { ConditionBuilder, useFacts, usesFertilityFact, type FactRegistry } from '@/features/rule-conditions';
import { useContentLanguages } from '@/shared/i18n';
import { formatDate } from '@/shared/lib';
import { Badge, Button, LoadGate, PageHeader, Panel, Select, TextArea, TextInput, confirm, toast, useNotifyError } from '@/shared/ui';

import { KINDS, LEVELS, useCreateRule, useDeleteRule, useRule, useRuleEnums, useUpdateRule, type RuleEnums } from '../api/rules';
import { authorityFor } from '../lib/authority';
import { bannerKey, fieldError } from '../lib/errors';
import {
  BANDS,
  FERTILE_TOPIC,
  MAX_VARIANT_KEYS,
  emptyForm,
  emptyRow,
  fromRule,
  needsApproval,
  overlayText,
  setOverlayField,
  toPayload,
  weightSum,
  type OverlayField,
  type RuleForm,
  type VariantRow,
} from '../lib/rule-form';
import { ApprovalBanner } from './ApprovalBanner';
import { MatrixSelect, type MatrixOption } from './MatrixSelect';
import { ApprovalPanel, ConflictPanel, PreviewPanel, VariantStats } from './RuleSidePanels';

/** /messages/rules/:id|new — the message editor (v6: facts, N variants, caps, topics, schedule, approval). */
export function RuleEditorScreen({ id }: { id: number | null }) {
  const t = useTranslations('messageRules');
  const detail = useRule(id);
  const enums = useRuleEnums();
  const facts = useFacts();
  const header = <PageHeader title={id === null ? t('newTitle') : t('editTitle')} backHref="/messages/rules" backLabel={t('backToList')} />;
  return (
    <LoadGate queries={[detail, enums, facts]} header={header}>
      {() => <Editor id={id} detail={detail.data} enums={enums.data as RuleEnums} registry={facts.data as FactRegistry} />}
    </LoadGate>
  );
}

/** A group of inputs that is disabled as a whole (read-only rule / locked structure). */
function Lock({ disabled, children }: { disabled: boolean; children: ReactNode }) {
  return (
    <fieldset disabled={disabled} className="m-0 min-w-0 border-0 p-0">
      {children}
    </fieldset>
  );
}

function Editor({
  id,
  detail,
  enums,
  registry,
}: {
  id: number | null;
  detail: ReturnType<typeof useRule>['data'];
  enums: RuleEnums;
  registry: FactRegistry;
}) {
  const t = useTranslations('messageRules');
  const locale = useLocale();
  const router = useRouter();
  const notifyError = useNotifyError();
  const canClinical = useCan('clinical', 'manage');
  const [form, setForm] = useState<RuleForm>(() => (detail ? fromRule(detail.rule) : emptyForm()));
  const contentLangs = useContentLanguages().data;
  // The language list is data: every non-default content language plus any language already in the stored overlay.
  const overlayLangs = [
    ...(contentLangs?.languages.filter((l) => !l.is_default) ?? []).map((l) => ({ code: l.code, name: l.name, direction: l.direction })),
    ...Object.keys(form.translations ?? {})
      .filter((c) => c !== contentLangs?.default && !contentLangs?.languages.some((l) => l.code === c))
      .map((c) => ({ code: c, name: c, direction: 'ltr' as const })),
  ];
  const create = useCreateRule();
  const update = useUpdateRule();
  const remove = useDeleteRule();
  const saving = create.isPending || update.isPending;
  const error = create.error ?? update.error;
  const err = (field: string) => fieldError(error, field);
  const banner = bannerKey(error);
  const set = <K extends keyof RuleForm>(k: K, v: RuleForm[K]) => setForm((f) => ({ ...f, [k]: v }));
  const rule = detail?.rule;
  const auth = authorityFor(rule ?? null, canClinical, enums.approval_kinds);
  const payload = () => toPayload(form, registry.facts, registry.fertility_facts);
  const band = BANDS[form.kind];
  const medical = needsApproval(form.level, form.kind);
  const withdraws = rule?.status === 'approved' && medical;
  const structureLocked = auth.readOnly || auth.copyOnly;
  const fertile = usesFertilityFact(form.conditions, registry.fertility_facts);
  const sum = weightSum(form.variants);
  const isSafety = form.kind === 'safety';

  const save = () => {
    if (id === null) {
      create.mutate(payload(), {
        onSuccess: (res) => {
          toast.success(t('created'));
          const created = (res as { rule?: { id: number } } | undefined)?.rule?.id;
          router.replace(created ? `/messages/rules/${created}` : '/messages/rules');
        },
        onError: notifyError,
      });
      return;
    }
    update.mutate(
      { id, body: payload() },
      { onSuccess: () => toast.success(withdraws ? t('savedApprovalWithdrawn') : t('saved')), onError: notifyError },
    );
  };

  // ---- the three matrices (T-M10-02): level × kind, slot × kind; invalid options stay visible, disabled, with a reason
  const levelOpts: MatrixOption[] = LEVELS.map((l) => ({
    value: l,
    label: t(`levels.${l}`),
    reason: (enums.level_kinds[l] ?? [form.kind]).includes(form.kind) ? undefined : t('reasonLevelKind', { kind: t(`kinds.${form.kind as 'safety'}`) }),
  }));
  const kindOpts: MatrixOption[] = KINDS.map((k) => ({
    value: k,
    label: t(`kinds.${k}`),
    reason: !auth.canAuthorKind(k)
      ? t('reasonClinicalOnly')
      : !(enums.level_kinds[form.level] ?? [k]).includes(k)
        ? t('reasonKindLevel')
        : !(enums.slot_kinds[form.placement] ?? [k]).includes(k)
          ? t('reasonKindSlot')
          : undefined,
  }));
  const slotOpts: MatrixOption[] = enums.placements.map((p) => ({
    value: p,
    label: t(`placements.${p}` as 'placements.push'),
    reason: (enums.slot_kinds[p] ?? [form.kind]).includes(form.kind) ? undefined : t('reasonSlotKind', { kind: t(`kinds.${form.kind as 'safety'}`) }),
  }));

  const setRow = (i: number, patch: Partial<VariantRow>) => set('variants', form.variants.map((r, j) => (j === i ? { ...r, ...patch } : r)));
  const addVariant = () => {
    const key = MAX_VARIANT_KEYS.find((k) => !form.variants.some((v) => v.key === k));
    if (key) set('variants', [...form.variants, emptyRow(key, '50')]);
  };
  const maxVariants = isSafety ? 1 : Math.min(enums.max_variants, MAX_VARIANT_KEYS.length);
  const topicOn = (tp: string) => form.topics.includes(tp) || (tp === FERTILE_TOPIC && fertile);

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={rule ? rule.title : t('newTitle')}
        backHref="/messages/rules"
        backLabel={t('backToList')}
        meta={rule ? `${t('version', { n: rule.version })} · ${rule.type} / ${rule.slot} / ${rule.name}` : undefined}
        actions={
          <>
            {rule && auth.canDelete ? (
              <Button
                variant="danger"
                loading={remove.isPending}
                onClick={async () => {
                  if (!(await confirm({ message: t('confirmDelete') }))) return;
                  remove.mutate(rule.id, {
                    onSuccess: () => {
                      toast.success(t('deleted'));
                      router.replace('/messages/rules');
                    },
                    onError: notifyError,
                  });
                }}
              >
                {t('delete')}
              </Button>
            ) : null}
            {auth.canSave ? (
              <Button variant="primary" loading={saving} onClick={save}>
                {t('saveDraft')}
              </Button>
            ) : null}
          </>
        }
      />
      {rule ? <ApprovalBanner rule={rule} approved={detail?.approved} /> : null}
      {auth.readOnly ? (
        <p role="status" className="m-0 rounded-lg border border-line bg-surface-2 p-3 text-[13px]">
          {rule?.locked ? t('lockedReadOnly') : t('clinicalReadOnly')}
        </p>
      ) : auth.copyOnly ? (
        <p role="status" className="m-0 rounded-lg border border-line bg-surface-2 p-3 text-[13px]">
          {t('lockedCopyOnly')}
        </p>
      ) : null}
      {banner ? (
        <p role="alert" className="m-0 rounded-lg border border-red-500/40 bg-red-500/10 p-3 text-[13px]">
          {t(banner as 'errLocked')}
        </p>
      ) : null}
      {withdraws ? (
        <p role="status" className="m-0 rounded-lg border border-line bg-surface-2 p-3 text-[13px]">
          {t('editWithdraws')}
        </p>
      ) : null}

      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="flex min-w-0 flex-col gap-4">
          <Lock disabled={auth.readOnly}>
            <Panel title={t('sectionContent')}>
              <div className="flex flex-col gap-4">
                <p className="m-0 text-[13px] text-ink-3" dir="ltr">
                  {t('variablesHint')} {enums.variables.map((v) => `{${v}}`).join(' ')}
                </p>
                <div className="grid gap-3 sm:grid-cols-2">
                  <TextInput label={t('internalTitle')} value={form.title} onChange={(e) => set('title', e.target.value)} error={err('title')} required disabled={structureLocked} />
                  <TextInput label={t('ruleKey')} dir="ltr" value={form.rule_key} onChange={(e) => set('rule_key', e.target.value)} error={err('rule_key')} hint={t('ruleKeyHint')} disabled={structureLocked} />
                </div>

                <div className="flex flex-wrap items-center gap-2" aria-live="polite">
                  <h3 className="m-0 text-[14px] font-semibold">{t('variantsTitle')}</h3>
                  <Badge tone={sum === 100 || form.variants.length === 1 ? 'green' : 'neutral'}>{t('weightSum', { n: sum })}</Badge>
                  <span className="text-[13px] text-ink-3">{t('variantsCount', { n: form.variants.length, max: maxVariants })}</span>
                </div>
                {isSafety ? <p className="m-0 text-[13px] text-ink-3">{t('safetySingleVariant')}</p> : null}
                {err('variants') ? (
                  <span className="field-error" role="alert">
                    {err('variants')}
                  </span>
                ) : null}
                {form.variants.map((v, i) => (
                  <div key={v.key} className="flex flex-col gap-3 rounded-lg border border-line p-3">
                    <div className="flex flex-wrap items-end gap-3">
                      <span className="rounded bg-surface-2 px-2 py-1 text-[13px] font-semibold" dir="ltr">
                        {v.key}
                      </span>
                      <TextInput
                        label={t('weight')}
                        type="number"
                        value={v.weight}
                        onChange={(e) => setRow(i, { weight: e.target.value })}
                        hint={sum > 0 ? t('weightShare', { p: Math.round((100 * (Number(v.weight) || 0)) / sum) }) : undefined}
                        disabled={structureLocked}
                      />
                      {i > 0 ? (
                        <Button size="sm" variant="ghost" disabled={structureLocked} onClick={() => set('variants', form.variants.filter((_, j) => j !== i))}>
                          {t('removeVariant')}
                        </Button>
                      ) : null}
                    </div>
                    <TextInput
                      label={t('titleOf', { key: v.key })}
                      value={v.title}
                      onChange={(e) => setRow(i, { title: e.target.value })}
                      error={i === 0 ? err('content.a') : v.key === 'b' ? err('content.b') : i === 1 ? err('content.variants') : undefined}
                    />
                    <TextArea label={t('body')} rows={3} value={v.body} onChange={(e) => setRow(i, { body: e.target.value })} />
                    <div className="grid gap-3 sm:grid-cols-2">
                      <TextInput label={t('cta')} value={v.cta_label} onChange={(e) => setRow(i, { cta_label: e.target.value })} />
                      <TextInput label={t('ctaTarget')} dir="ltr" value={v.cta_target} onChange={(e) => setRow(i, { cta_target: e.target.value })} placeholder="ritme://analysis/symptoms" />
                    </div>
                    {overlayLangs.length > 0 ? (
                      <div className="flex flex-col gap-3 rounded-lg bg-surface-2 p-3">
                        <p className="m-0 text-[12px] text-ink-3">{t('overlayHint')}</p>
                        {overlayLangs.map((lang) => {
                          const o = overlayText(form.translations, lang.code, v.key);
                          const setO = (field: OverlayField, value: string) => set('translations', setOverlayField(form.translations, lang.code, v.key, field, value));
                          const dir = lang.direction;
                          return (
                            <div key={lang.code} className="flex flex-col gap-3">
                              <h4 className="m-0 text-[13px] font-semibold">{t('overlayTitle', { lang: lang.name })}</h4>
                              <TextInput label={t('titleOf', { key: v.key })} dir={dir} value={o.title} onChange={(e) => setO('title', e.target.value)} placeholder={v.title} />
                              <TextArea label={t('body')} dir={dir} rows={3} value={o.body} onChange={(e) => setO('body', e.target.value)} placeholder={v.body} />
                              <div className="grid gap-3 sm:grid-cols-2">
                                <TextInput label={t('cta')} dir={dir} value={o.cta_label} onChange={(e) => setO('cta_label', e.target.value)} placeholder={v.cta_label} />
                                <TextInput label={t('ctaTarget')} dir="ltr" value={o.cta_target} onChange={(e) => setO('cta_target', e.target.value)} placeholder={v.cta_target} />
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    ) : null}
                    {form.level === 'suggestion' ? <p className="m-0 text-[12px] text-ink-3">{t('suggestionCta')}</p> : null}
                  </div>
                ))}
                <div>
                  <Button size="sm" disabled={structureLocked || form.variants.length >= maxVariants} onClick={addVariant}>
                    {t('addVariant')}
                  </Button>
                </div>
                {form.level === 'worth_followup' || form.level === 'follow_soon' ? (
                  <>
                    <TextArea label={t('seen')} rows={2} value={form.seen} onChange={(e) => set('seen', e.target.value)} error={err('content.seen')} required />
                    <TextArea label={t('confidence')} rows={2} value={form.confidence} onChange={(e) => set('confidence', e.target.value)} error={err('content.confidence')} required />
                  </>
                ) : null}
                {form.level === 'follow_soon' ? (
                  <TextInput label={t('contact')} dir="ltr" value={form.contact} onChange={(e) => set('contact', e.target.value)} error={err('content.contact')} hint={t('contactHint')} required />
                ) : null}
              </div>
            </Panel>
          </Lock>

          <Lock disabled={structureLocked}>
            <Panel title={t('ruleTitle')}>
              <ConditionBuilder
                value={form.conditions}
                onChange={(c) => set('conditions', c)}
                facts={registry.facts}
                error={err('conditions')}
              />
            </Panel>

            <div className="mt-4 flex flex-col gap-4">
              <Panel title={t('sectionLevel')}>
                <div className="grid gap-3 sm:grid-cols-2">
                  <MatrixSelect label={t('level')} value={form.level} options={levelOpts} onChange={(v) => set('level', v)} error={err('level')} hint={t(`levelHints.${form.level as 'info'}`)} />
                  <MatrixSelect
                    label={t('kind')}
                    value={form.kind}
                    options={kindOpts}
                    error={err('kind')}
                    onChange={(kind) => {
                      const b = BANDS[kind];
                      setForm((f) => ({
                        ...f,
                        kind,
                        priority: b && (Number(f.priority) < b.min || Number(f.priority) > b.max) ? String(b.min === b.max ? b.min : Math.min(b.max, Math.max(b.min, b.min + 10))) : f.priority,
                      }));
                    }}
                  />
                  <MatrixSelect label={t('placement')} value={form.placement} options={slotOpts} onChange={(v) => set('placement', v)} error={err('placement')} />
                  <TextInput label={t('priority')} value={form.priority} onChange={(e) => set('priority', e.target.value)} error={err('priority')} hint={band ? t('priorityBand', { min: band.min, max: band.max }) : undefined} type="number" />
                </div>
                {medical ? <p className="m-0 mt-3 text-[13px]">{t('levelMedicalNote')}</p> : null}
              </Panel>

              <Panel title={t('sectionTopics')}>
                <div className="flex flex-col gap-2">
                  <div className="flex flex-wrap gap-2">
                    {enums.topics.map((tp) => {
                      const on = topicOn(tp);
                      const forced = tp === FERTILE_TOPIC && fertile;
                      return (
                        <button
                          key={tp}
                          type="button"
                          aria-pressed={on}
                          disabled={forced}
                          onClick={() => set('topics', on ? form.topics.filter((x) => x !== tp) : [...form.topics, tp])}
                          className={`rounded-full border px-3 py-1 text-[13px] ${on ? 'border-brand bg-brand/10 font-semibold' : 'border-line'}`}
                          dir="ltr"
                        >
                          {tp}
                        </button>
                      );
                    })}
                  </div>
                  <p className="m-0 text-[13px] text-ink-3">{fertile ? t('topicRequired') : t('topicsHint')}</p>
                  {err('topics') ? (
                    <span className="field-error" role="alert">
                      {err('topics')}
                    </span>
                  ) : null}
                </div>
              </Panel>

              <Panel title={t('sectionDelivery')}>
                <div className="flex flex-col gap-4">
                  {isSafety ? (
                    <p className="m-0 text-[13px] text-ink-3">{t('safetyNoCaps')}</p>
                  ) : (
                    <div className="flex flex-col gap-2">
                      <h3 className="m-0 text-[14px] font-semibold">{t('capsTitle')}</h3>
                      <p className="m-0 text-[13px] text-ink-3">{t('capHint')}</p>
                      {form.caps.map((c, i) => (
                        <div key={i} className="flex flex-wrap items-end gap-2">
                          <TextInput label={t('capCount')} type="number" value={c.count} onChange={(e) => set('caps', form.caps.map((x, j) => (j === i ? { ...x, count: e.target.value } : x)))} />
                          <Select
                            label={t('capPeriod')}
                            value={c.period}
                            onChange={(e) => set('caps', form.caps.map((x, j) => (j === i ? { ...x, period: e.target.value } : x)))}
                            options={enums.cap_periods.map((p) => ({ value: p, label: t(`capPeriods.${p}` as 'capPeriods.day') }))}
                          />
                          <Button size="sm" variant="ghost" onClick={() => set('caps', form.caps.filter((_, j) => j !== i))} aria-label={t('removeCap')}>
                            ×
                          </Button>
                        </div>
                      ))}
                      {err('caps') ? (
                        <span className="field-error" role="alert">
                          {err('caps')}
                        </span>
                      ) : null}
                      <div>
                        <Button size="sm" onClick={() => set('caps', [...form.caps, { count: '1', period: 'week' }])}>
                          {t('addCap')}
                        </Button>
                      </div>
                    </div>
                  )}
                  <TextInput label={t('cooldown')} value={form.cooldown_days} onChange={(e) => set('cooldown_days', e.target.value)} error={err('cooldown_days')} hint={t('cooldownHint')} type="number" />
                  <TextInput label={t('audience')} value={form.audience_percent} onChange={(e) => set('audience_percent', e.target.value)} error={err('audience_percent')} type="number" disabled={isSafety} />
                </div>
              </Panel>

              <Panel title={t('sectionSchedule')}>
                <div className="flex flex-col gap-4">
                  <p className="m-0 text-[13px] text-ink-3">{t('userLocalNote')}</p>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <TextInput label={t('startsAt')} dir="ltr" type="datetime-local" value={form.starts_at} onChange={(e) => set('starts_at', e.target.value)} error={err('starts_at')} hint={form.starts_on && !form.starts_at ? t('legacyDate', { d: form.starts_on }) : undefined} />
                    <TextInput label={t('endsAt')} dir="ltr" type="datetime-local" value={form.ends_at} onChange={(e) => set('ends_at', e.target.value)} error={err('ends_at')} hint={form.ends_on && !form.ends_at ? t('legacyDate', { d: form.ends_on }) : t('noEnd')} />
                  </div>
                  <div className="flex flex-col gap-2">
                    <h3 className="m-0 text-[14px] font-semibold">{t('hoursTitle')}</h3>
                    <p className="m-0 text-[13px] text-ink-3">{t('hoursHint')}</p>
                    {form.hours.map((h, i) => (
                      <div key={i} className="flex flex-wrap items-end gap-2">
                        <TextInput label={t('hoursFrom')} type="number" value={h.from} onChange={(e) => set('hours', form.hours.map((x, j) => (j === i ? { ...x, from: e.target.value } : x)))} />
                        <TextInput label={t('hoursTo')} type="number" value={h.to} onChange={(e) => set('hours', form.hours.map((x, j) => (j === i ? { ...x, to: e.target.value } : x)))} />
                        <Button size="sm" variant="ghost" onClick={() => set('hours', form.hours.filter((_, j) => j !== i))} aria-label={t('removeWindow')}>
                          ×
                        </Button>
                      </div>
                    ))}
                    {err('hours') ? (
                      <span className="field-error" role="alert">
                        {err('hours')}
                      </span>
                    ) : null}
                    <div>
                      <Button size="sm" onClick={() => set('hours', [...form.hours, { from: '8', to: '22' }])}>
                        {t('addWindow')}
                      </Button>
                    </div>
                  </div>
                </div>
              </Panel>

              <Panel title={t('sectionGoal')}>
                <div className="grid gap-3 sm:grid-cols-2">
                  <Select
                    label={t('goalEvent')}
                    value={form.goal_event}
                    onChange={(e) => set('goal_event', e.target.value)}
                    error={err('goal_event')}
                    options={[{ value: '', label: t('noGoalEvent') }, ...enums.goal_events.map((g) => ({ value: g, label: t(`goalEvents.${g}` as 'goalEvents.log_saved') }))]}
                  />
                  <TextInput label={t('goal')} value={form.goal} onChange={(e) => set('goal', e.target.value)} error={err('goal')} hint={t('goalHint')} />
                </div>
              </Panel>
            </div>
          </Lock>
        </div>

        <div className="flex min-w-0 flex-col gap-4">
          {rule ? (
            <ApprovalPanel rule={rule} canToggle={rule.is_active ? auth.canSwitchOff : auth.canPublish} />
          ) : (
            <Panel title={t('approvalTitle')}>
              <p className="m-0 text-[13px] text-ink-3">{medical ? t('approvalHint') : t('approvalNotNeeded')}</p>
            </Panel>
          )}
          <PreviewPanel payload={payload} ruleId={rule?.id} />
          <ConflictPanel payload={payload} ruleId={rule?.id} initial={detail?.conflicts} />
          {detail ? <VariantStats a={detail.variant_stats_7d.a} b={detail.variant_stats_7d.b} /> : null}
          {detail && detail.versions.length > 0 ? (
            <Panel title={t('versions')}>
              <ul className="m-0 list-none p-0 text-[13px]">
                {detail.versions.map((v) => (
                  <li key={v.version}>
                    {t('version', { n: v.version })}
                    {v.created_at ? ` · ${formatDate(v.created_at, locale)}` : ''}
                  </li>
                ))}
              </ul>
            </Panel>
          ) : null}
        </div>
      </div>
    </div>
  );
}
