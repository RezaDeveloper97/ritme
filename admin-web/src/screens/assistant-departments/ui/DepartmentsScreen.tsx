'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCurrentAdmin } from '@/features/auth';
import { fieldError } from '@/shared/api';
import { useContentLanguages, useLocalized } from '@/shared/i18n';
import { blankToNull } from '@/shared/lib';
import {
  Badge,
  Button,
  ErrorState,
  PageHeader,
  Panel,
  Skeleton,
  Switch,
  TextArea,
  TextInput,
  TranslatableField,
  toast,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { useApproveDepartment, useDepartments, useSaveDepartment, type Department } from '../api/departments';

/** /assistant/departments — prompts, guardrails and copy of the AI assistant's departments (super only, audit-logged). */
export function DepartmentsScreen() {
  const t = useTranslations('assistantDepartments');
  const isSuper = useCurrentAdmin()?.role === 'super';
  const query = useDepartments();
  const [editing, setEditing] = useState<number | 'new' | null>(null);
  const localize = useLocalized();

  if (!isSuper) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader title={t('title')} />
        <Panel>
          <p className="m-0 text-ink-3">{t('superOnly')}</p>
        </Panel>
      </div>
    );
  }
  const current = query.data?.items.find((d) => d.id === editing) ?? null;

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('title')}
        actions={
          <Button variant="primary" onClick={() => setEditing('new')}>
            {t('new')}
          </Button>
        }
      />
      <p className="panel-note m-0">{t('intro')}</p>
      {query.error && !query.data ? (
        <Panel>
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </Panel>
      ) : query.isPending ? (
        <Panel>
          <Skeleton className="h-16 w-full" />
        </Panel>
      ) : (
        <div className="grid gap-4 lg:grid-cols-[minmax(0,340px)_1fr]">
          <Panel title={t('list')} bodyClassName="p-2">
            <ul className="m-0 flex list-none flex-col gap-1 p-0">
              {query.data.items.map((d) => (
                <li key={d.id}>
                  <button
                    type="button"
                    onClick={() => setEditing(d.id)}
                    aria-current={editing === d.id ? 'true' : undefined}
                    className={`flex min-h-11 w-full flex-col items-start gap-1 rounded-lg border border-transparent bg-transparent p-3 text-start ${editing === d.id ? 'bg-brand-soft border-brand' : 'hover:bg-surface-2'}`}
                  >
                    <span className="font-semibold" dir="auto">
                      {localize(d.name) || d.code}
                    </span>
                    <span className="flex flex-wrap gap-1">
                      <Badge tone={d.is_active ? 'green' : 'neutral'}>{d.is_active ? t('active') : t('inactive')}</Badge>
                      {d.needs_review ? <Badge tone="amber">{t('needsReview')}</Badge> : <Badge tone="data">{t('approved')}</Badge>}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </Panel>
          {editing === 'new' || current ? (
            <DepartmentForm key={editing === 'new' ? 'new' : current?.id} row={current} limits={query.data.limits} onDone={() => setEditing(null)} />
          ) : (
            <Panel>
              <p className="m-0 text-muted">{t('pick')}</p>
            </Panel>
          )}
        </div>
      )}
    </div>
  );
}

const listText = (v: Record<string, string[]>, lang: string) => (v[lang] ?? []).join('\n');

function DepartmentForm({ row, limits, onDone }: { row: Department | null; limits: Record<string, number>; onDone: () => void }) {
  const t = useTranslations('assistantDepartments');
  const notifyError = useNotifyError();
  const languages = useContentLanguages();
  const save = useSaveDepartment(row?.id ?? null);
  const approve = useApproveDepartment();
  const [code, setCode] = useState(row?.code ?? '');
  const [icon, setIcon] = useState(row?.icon ?? '');
  const [name, setName] = useState<Translations>(row?.name ?? {});
  const [tagline, setTagline] = useState<Translations>(row?.tagline ?? {});
  const [intro, setIntro] = useState<Translations>(row?.intro ?? {});
  const [doesNot, setDoesNot] = useState<Translations>(
    Object.fromEntries(Object.keys(row?.does_not_do ?? {}).map((l) => [l, listText(row!.does_not_do, l)])),
  );
  const [quick, setQuick] = useState<Translations>(
    Object.fromEntries(Object.keys(row?.quick_replies ?? {}).map((l) => [l, listText(row!.quick_replies, l)])),
  );
  const [prompt, setPrompt] = useState(row?.system_prompt ?? '');
  const [guardrails, setGuardrails] = useState(row?.guardrails ?? '');
  const [active, setActive] = useState(row?.is_active ?? false);
  const err = (f: string) => fieldError(save.error, f);

  const cleanT = (v: Translations) => {
    const out: Translations = {};
    for (const [k, x] of Object.entries(v)) if (x.trim()) out[k] = x;
    return Object.keys(out).length ? out : null;
  };
  const toLists = (v: Translations) => {
    const out: Record<string, string[]> = {};
    for (const [k, x] of Object.entries(v)) {
      const items = x.split('\n').map((l) => l.trim()).filter(Boolean);
      if (items.length) out[k] = items;
    }
    return Object.keys(out).length ? out : null;
  };

  const submit = () =>
    save.mutate(
      {
        ...(row ? {} : { code: code.trim() }),
        icon: blankToNull(icon),
        name,
        tagline: cleanT(tagline),
        intro: cleanT(intro),
        does_not_do: toLists(doesNot),
        quick_replies: toLists(quick),
        system_prompt: prompt,
        guardrails: blankToNull(guardrails),
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(t('saved'));
          if (!row) onDone();
        },
        onError: notifyError,
      },
    );

  const langs = (languages.data?.languages ?? []).map((l) => l.code);
  const listField = (label: string, value: Translations, set: (v: Translations) => void, name: string) => (
    <TranslatableField name={name} label={label} value={value} onChange={set} kind="textarea" errors={undefined} />
  );

  return (
    <Panel title={row ? t('edit') : t('new')}>
      <form
        className="flex flex-col gap-5"
        onSubmit={(e) => {
          e.preventDefault();
          if (!save.isPending) submit();
        }}
      >
        {row?.needs_review ? (
          <div className="rounded-lg border border-line p-3 text-sm" role="status">
            <p className="m-0 mb-2">{t('reviewNotice')}</p>
            <Button
              variant="primary"
              size="sm"
              loading={approve.isPending}
              onClick={() => approve.mutate(row.id, { onSuccess: () => toast.success(t('approvedToast')), onError: notifyError })}
            >
              {t('approve')}
            </Button>
          </div>
        ) : null}
        {row ? (
          <p className="m-0 text-sm text-ink-3">
            {t('code')}: <span dir="ltr">{row.code}</span>
          </p>
        ) : (
          <TextInput label={t('code')} hint={t('codeHint')} value={code} maxLength={32} dir="ltr" required error={err('code')} onChange={(e) => setCode(e.target.value)} />
        )}
        <TextInput label={t('icon')} value={icon} maxLength={32} dir="ltr" error={err('icon')} onChange={(e) => setIcon(e.target.value)} />
        <TranslatableField name="name" label={t('name')} value={name} onChange={setName} required maxLength={120} />
        <TranslatableField name="tagline" label={t('tagline')} value={tagline} onChange={setTagline} maxLength={limits.text ?? 600} />
        <TranslatableField name="intro" label={t('intro2')} value={intro} onChange={setIntro} kind="textarea" maxLength={limits.text ?? 600} />
        {listField(t('doesNotDo'), doesNot, setDoesNot, 'does_not_do')}
        {listField(t('quickReplies'), quick, setQuick, 'quick_replies')}
        <p className="m-0 text-xs text-muted">{t('listHint', { count: limits.list_items ?? 10, languages: langs.join(', ') })}</p>
        <TextArea
          label={t('systemPrompt')}
          hint={t('systemPromptHint', { max: limits.system_prompt ?? 4000 })}
          value={prompt}
          rows={10}
          dir="ltr"
          required
          maxLength={limits.system_prompt ?? 4000}
          error={err('system_prompt')}
          onChange={(e) => setPrompt(e.target.value)}
        />
        <TextArea
          label={t('guardrails')}
          hint={t('guardrailsHint', { max: limits.guardrails ?? 2000 })}
          value={guardrails}
          rows={5}
          dir="ltr"
          maxLength={limits.guardrails ?? 2000}
          error={err('guardrails')}
          onChange={(e) => setGuardrails(e.target.value)}
        />
        <Switch label={t('activeLabel')} hint={row?.needs_review ? t('activeBlocked') : t('activeHint')} checked={active} onChange={setActive} />
        {err('is_active') ? <span className="field-error" role="alert">{err('is_active')}</span> : null}
        <div className="flex gap-2">
          <Button type="submit" variant="primary" loading={save.isPending}>
            {row ? t('save') : t('create')}
          </Button>
          <Button onClick={onDone}>{t('close')}</Button>
        </div>
      </form>
    </Panel>
  );
}
