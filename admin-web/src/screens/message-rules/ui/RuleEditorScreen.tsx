'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { isApiError } from '@/shared/api';
import { Button, LoadGate, PageHeader, Panel, Select, Switch, TextArea, TextInput, confirm, toast, useNotifyError } from '@/shared/ui';

import { KINDS, LEVELS, useCreateRule, useDeleteRule, useRule, useRuleEnums, useUpdateRule, type RuleEnums } from '../api/rules';
import { BANDS, emptyForm, fromRule, needsApproval, toPayload, type RuleForm, type VariantForm } from '../lib/rule-form';
import { ConditionBuilder } from './ConditionBuilder';
import { ApprovalPanel, ConflictPanel, PreviewPanel, VariantStats } from './RuleSidePanels';

/** /messages/rules/:id|new — the message editor (Admin_MessageEditor artboard). */
export function RuleEditorScreen({ id }: { id: number | null }) {
  const t = useTranslations('messageRules');
  const detail = useRule(id);
  const enums = useRuleEnums();
  const header = <PageHeader title={id === null ? t('newTitle') : t('editTitle')} backHref="/messages/rules" backLabel={t('backToList')} />;
  return (
    <LoadGate queries={[detail, enums]} header={header}>
      {() => <Editor id={id} detail={detail.data} enums={enums.data as RuleEnums} />}
    </LoadGate>
  );
}

function Editor({ id, detail, enums }: { id: number | null; detail: ReturnType<typeof useRule>['data']; enums: RuleEnums }) {
  const t = useTranslations('messageRules');
  const router = useRouter();
  const notifyError = useNotifyError();
  const [form, setForm] = useState<RuleForm>(() => (detail ? fromRule(detail.rule) : emptyForm()));
  const create = useCreateRule();
  const update = useUpdateRule();
  const remove = useDeleteRule();
  const saving = create.isPending || update.isPending;
  const error = create.error ?? update.error;
  const err = (field: string) => (isApiError(error) ? error.field(field) : undefined);
  const set = <K extends keyof RuleForm>(k: K, v: RuleForm[K]) => setForm((f) => ({ ...f, [k]: v }));
  const setVariant = (which: 'a' | 'b', k: keyof VariantForm, v: string) => setForm((f) => ({ ...f, [which]: { ...f[which], [k]: v } }));
  const rule = detail?.rule;
  const payload = () => toPayload(form);
  const band = BANDS[form.kind];
  const medical = needsApproval(form.level, form.kind);
  const withdraws = rule?.status === 'approved' && medical;

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

  const variantFields = (which: 'a' | 'b') => (
    <div className="flex flex-col gap-3">
      <TextInput
        label={which === 'a' ? t('titleA') : t('titleB')}
        value={form[which].title}
        onChange={(e) => setVariant(which, 'title', e.target.value)}
        error={which === 'a' ? err('content.a') : err('content.b')}
      />
      <TextArea label={t('body')} rows={3} value={form[which].body} onChange={(e) => setVariant(which, 'body', e.target.value)} />
      <div className="grid gap-3 sm:grid-cols-2">
        <TextInput label={t('cta')} value={form[which].cta_label} onChange={(e) => setVariant(which, 'cta_label', e.target.value)} />
        <TextInput label={t('ctaTarget')} dir="ltr" value={form[which].cta_target} onChange={(e) => setVariant(which, 'cta_target', e.target.value)} placeholder="ritme://analysis/symptoms" />
      </div>
    </div>
  );

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={rule ? rule.title : t('newTitle')}
        backHref="/messages/rules"
        backLabel={t('backToList')}
        meta={rule ? t('version', { n: rule.version }) : undefined}
        actions={
          <>
            {rule ? (
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
            <Button variant="primary" loading={saving} onClick={save}>
              {t('saveDraft')}
            </Button>
          </>
        }
      />
      {withdraws ? (
        <p role="status" className="m-0 rounded-lg border border-line bg-surface-2 p-3 text-[13px]">
          {t('editWithdraws')}
        </p>
      ) : null}

      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="flex min-w-0 flex-col gap-4">
          <Panel title={t('sectionContent')}>
            <div className="flex flex-col gap-4">
              <p className="m-0 text-[13px] text-ink-3" dir="ltr">
                {t('variablesHint')} {enums.variables.map((v) => `{${v}}`).join(' ')}
              </p>
              <TextInput label={t('internalTitle')} value={form.title} onChange={(e) => set('title', e.target.value)} error={err('title')} required />
              {variantFields('a')}
              <Switch label={t('variantsEnabled')} hint={t('variantsHint')} checked={form.variants_enabled} onChange={(v) => set('variants_enabled', v)} />
              {form.variants_enabled ? (
                <>
                  {variantFields('b')}
                  <TextInput label={t('abSplit')} value={form.ab_split} onChange={(e) => set('ab_split', e.target.value)} error={err('ab_split')} hint={t('abSplitHint')} type="number" />
                </>
              ) : null}
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

          <ConditionBuilder
            groups={form.groups}
            exceptions={form.exceptions}
            ops={enums.ops}
            onGroups={(g) => set('groups', g)}
            onExceptions={(e) => set('exceptions', e)}
            error={err('conditions') ?? err('exceptions')}
          />

          <Panel title={t('sectionLevel')}>
            <div className="flex flex-col gap-3">
              <Select
                label={t('level')}
                value={form.level}
                onChange={(e) => set('level', e.target.value)}
                options={LEVELS.map((l) => ({ value: l, label: t(`levels.${l}`) }))}
                error={err('level')}
              />
              <p className="m-0 text-[13px] text-ink-3">{t(`levelHints.${form.level as 'info'}`)}</p>
              {medical ? <p className="m-0 text-[13px]">{t('levelMedicalNote')}</p> : null}
            </div>
          </Panel>

          <Panel title={t('sectionDelivery')}>
            <div className="grid gap-3 sm:grid-cols-2">
              <Select
                label={t('placement')}
                value={form.placement}
                onChange={(e) => set('placement', e.target.value)}
                options={enums.placements.map((p) => ({ value: p, label: t(`placements.${p}` as 'placements.smart_tip') }))}
                error={err('placement')}
              />
              <Select
                label={t('kind')}
                value={form.kind}
                onChange={(e) => {
                  const kind = e.target.value;
                  const b = BANDS[kind];
                  setForm((f) => ({ ...f, kind, priority: b && (Number(f.priority) < b.min || Number(f.priority) > b.max) ? String(b.min === b.max ? b.min : Math.min(b.max, Math.max(b.min, b.min + 10))) : f.priority }));
                }}
                options={KINDS.map((k) => ({ value: k, label: t(`kinds.${k}`) }))}
                error={err('kind')}
              />
              <TextInput
                label={t('priority')}
                value={form.priority}
                onChange={(e) => set('priority', e.target.value)}
                error={err('priority')}
                hint={band ? t('priorityBand', { min: band.min, max: band.max }) : undefined}
                type="number"
              />
              <TextInput label={t('capCount')} value={form.cap_count} onChange={(e) => set('cap_count', e.target.value)} error={err('cap_count')} hint={t('capHint')} type="number" />
              <Select
                label={t('capPeriod')}
                value={form.cap_period}
                onChange={(e) => set('cap_period', e.target.value)}
                options={enums.cap_periods.map((p) => ({ value: p, label: t(`capPeriods.${p}` as 'capPeriods.day') }))}
                error={err('cap_period')}
              />
              <TextInput label={t('cooldown')} value={form.cooldown_days} onChange={(e) => set('cooldown_days', e.target.value)} error={err('cooldown_days')} hint={t('cooldownHint')} type="number" />
              <TextInput label={t('hoursFrom')} value={form.hours_from} onChange={(e) => set('hours_from', e.target.value)} error={err('hours_from')} hint={t('hoursHint')} type="number" />
              <TextInput label={t('hoursTo')} value={form.hours_to} onChange={(e) => set('hours_to', e.target.value)} error={err('hours_to')} type="number" />
              <TextInput label={t('startsOn')} dir="ltr" value={form.starts_on} onChange={(e) => set('starts_on', e.target.value)} error={err('starts_on')} placeholder="2026-10-01" />
              <TextInput label={t('endsOn')} dir="ltr" value={form.ends_on} onChange={(e) => set('ends_on', e.target.value)} error={err('ends_on')} placeholder={t('noEnd')} />
              <TextInput label={t('audience')} value={form.audience_percent} onChange={(e) => set('audience_percent', e.target.value)} error={err('audience_percent')} type="number" />
              <TextInput label={t('goal')} value={form.goal} onChange={(e) => set('goal', e.target.value)} error={err('goal')} hint={t('goalHint')} />
            </div>
          </Panel>
        </div>

        <div className="flex min-w-0 flex-col gap-4">
          {rule ? <ApprovalPanel rule={rule} /> : <Panel title={t('approvalTitle')}><p className="m-0 text-[13px] text-ink-3">{medical ? t('approvalHint') : t('approvalNotNeeded')}</p></Panel>}
          <PreviewPanel payload={payload} ruleId={rule?.id} />
          <ConflictPanel payload={payload} ruleId={rule?.id} initial={detail?.conflicts} />
          {detail ? <VariantStats a={detail.variant_stats_7d.a} b={detail.variant_stats_7d.b} /> : null}
          {detail && detail.versions.length > 0 ? (
            <Panel title={t('versions')}>
              <ul className="m-0 list-none p-0 text-[13px]">
                {detail.versions.map((v) => (
                  <li key={v.version}>
                    {t('version', { n: v.version })}
                    {v.created_at ? ` · ${v.created_at.slice(0, 10)}` : ''}
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
