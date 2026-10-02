'use client';

import { useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useState } from 'react';

import { fieldErrorsOf } from '@/shared/api';
import { useContentLanguages, type ContentLanguage } from '@/shared/i18n';
import { cn, useNumber } from '@/shared/lib';
import {
  Badge,
  Button,
  Icon,
  LinkTabs,
  LoadGate,
  Panel,
  PageHeader,
  TextArea,
  TextInput,
  confirm,
  toast,
  useNotifyError,
  type BadgeTone,
} from '@/shared/ui';

import {
  useCompanionTips,
  useResetCompanionTips,
  useSaveCompanionTips,
  type PhaseTips,
  type TipSource,
  type TipsList,
} from '../api/companions';
import { draftOf, fillName, move, sameDraft, tipsBody, type LocaleDraft } from '../lib/tips';

/** /companions/tips?phase= — «نکته‌های همدم»: note + up to three tips per partner phase and language (B-N4-07). */
export function CompanionTipsScreen() {
  const t = useTranslations('companions.tips');
  const tips = useCompanionTips();
  const languages = useContentLanguages();
  return (
    <LoadGate queries={[tips, languages]} header={<PageHeader title={t('title')} />}>
      {() =>
        tips.data && languages.data ? (
          <TipsEditor
            data={tips.data}
            languages={[...languages.data.languages].sort((a, b) => Number(b.is_default) - Number(a.is_default))}
          />
        ) : null
      }
    </LoadGate>
  );
}

const SOURCE_TONE: Record<TipSource, BadgeTone> = { locale: 'green', default_language: 'amber', built_in: 'neutral' };

function SourceBadge({ source }: { source: TipSource }) {
  const t = useTranslations('companions.tips.source');
  return <Badge tone={SOURCE_TONE[source]}>{t(source)}</Badge>;
}

type Edits = Record<string, Record<string, LocaleDraft>>;

function TipsEditor({ data, languages }: { data: TipsList; languages: ContentLanguage[] }) {
  const t = useTranslations('companions.tips');
  const tc = useTranslations('crud');
  const n = useNumber();
  const search = useSearchParams();
  const notifyError = useNotifyError();
  const save = useSaveCompanionTips();
  const reset = useResetCompanionTips();

  const phase = data.phases.includes(search.get('phase') ?? '') ? (search.get('phase') as string) : (data.phases[0] ?? 'general');
  const item: PhaseTips | undefined = data.items.find((i) => i.phase === phase);
  const [lang, setLang] = useState(languages[0]?.code ?? data.default_locale);
  const [edits, setEdits] = useState<Edits>({});
  const [sampleName, setSampleName] = useState('');

  const serverDraft = (p: string, code: string) => draftOf(data.items.find((i) => i.phase === p)?.texts[code]);
  const draftFor = (p: string, code: string) => edits[p]?.[code] ?? serverDraft(p, code);
  const dirtyCodes = (p: string) =>
    languages.map((l) => l.code).filter((code) => edits[p]?.[code] && !sameDraft(edits[p][code], serverDraft(p, code)));
  const isDirty = (p: string) => dirtyCodes(p).length > 0;

  const current = languages.find((l) => l.code === lang) ?? languages[0];
  const texts = item?.texts[lang];
  const draft = draftFor(phase, lang);
  const setDraft = (next: LocaleDraft) => setEdits((all) => ({ ...all, [phase]: { ...all[phase], [lang]: next } }));
  const errors = fieldErrorsOf(save.error) ?? {};
  const err = (path: string) => errors[`texts.${lang}.${path}`]?.[0];
  const max = data.tips_per_phase;
  const name = sampleName.trim() || t('sampleNameDefault');
  const phaseLabel = (p: string) => {
    const key = `phase.${p}` as 'phase.general';
    return t.has(key) ? t(key) : p;
  };

  const discardPhase = (p: string) =>
    setEdits((all) => {
      const next = { ...all };
      delete next[p];
      return next;
    });

  const submit = () => {
    const codes = dirtyCodes(phase);
    if (codes.length === 0) return;
    save.mutate(
      { phase, body: tipsBody(edits[phase] ?? {}, codes) },
      {
        onSuccess: () => {
          toast.success(tc('saved'));
          discardPhase(phase);
        },
        onError: notifyError,
      },
    );
  };

  const resetLanguage = async () => {
    const ok = await confirm({ message: t('resetConfirm', { language: current?.name ?? lang }), tone: 'danger', confirmLabel: t('reset') });
    if (!ok) return;
    reset.mutate(
      { phase, locale: lang },
      {
        onSuccess: () => {
          toast.success(t('resetDone'));
          setEdits((all) => {
            const forPhase = { ...all[phase] };
            delete forPhase[lang];
            return { ...all, [phase]: forPhase };
          });
        },
        onError: notifyError,
      },
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <LinkTabs
        label={t('phases')}
        items={data.phases.map((p) => ({
          key: p,
          href: p === data.phases[0] ? '/companions/tips' : `/companions/tips?phase=${p}`,
          label: isDirty(p) ? `${phaseLabel(p)} •` : phaseLabel(p),
          active: p === phase,
        }))}
      />
      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,22rem)]">
        <Panel
          title={t('phaseTitle', { phase: phaseLabel(phase) })}
          actions={texts?.customized ? <Badge tone="green">{t('customized')}</Badge> : <Badge>{t('notCustomized')}</Badge>}
        >
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              submit();
            }}
          >
            <p className="field-hint m-0">
              {t('intro')} {t('placeholders')}: <span dir="ltr">{data.placeholders.map((p) => `{${p}}`).join(' ')}</span>
            </p>
            <div role="tablist" aria-label={t('languages')} className="flex flex-wrap gap-1.5">
              {languages.map((l) => (
                <button
                  key={l.code}
                  type="button"
                  role="tab"
                  aria-selected={l.code === lang}
                  className={cn('btn btn-sm', l.code === lang ? 'btn-primary' : 'btn-ghost')}
                  onClick={() => setLang(l.code)}
                >
                  {l.name}
                  <span className="lang-code">{l.code}</span>
                  {dirtyCodes(phase).includes(l.code) ? <span aria-label={t('unsaved')}>•</span> : null}
                  {Object.keys(errors).some((k) => k.startsWith(`texts.${l.code}.`)) ? <Icon name="alert" size={14} /> : null}
                </button>
              ))}
            </div>

            <div role="tabpanel" className="flex flex-col gap-4" dir={current?.direction} lang={lang}>
              <div className="flex flex-col gap-1.5">
                <TextArea
                  label={t('note')}
                  hint={t('noteHint')}
                  rows={3}
                  maxLength={data.limits.note}
                  value={draft.note}
                  onChange={(e) => setDraft({ ...draft, note: e.target.value })}
                  error={err('note')}
                />
                {texts ? (
                  <span className="flex items-center gap-2 text-xs text-muted">
                    {t('now')} <SourceBadge source={texts.note.source} />
                  </span>
                ) : null}
              </div>

              <fieldset className="m-0 flex min-w-0 flex-col gap-3 border-0 p-0">
                <legend className="field-label mb-1.5 p-0">
                  {t('tipsLabel')}{' '}
                  <span className="text-xs font-normal text-muted">({t('countOf', { count: n(draft.tips.length), max: n(max) })})</span>
                </legend>
                {draft.tips.length === 0 ? <p className="field-hint m-0">{t('noTips')}</p> : null}
                {draft.tips.map((tip, i) => (
                  <div key={i} className="flex flex-col gap-3 rounded-xl border border-line bg-surface-2 p-3">
                    <div className="flex items-center gap-1.5">
                      <strong className="text-sm">{t('tipN', { n: n(i + 1) })}</strong>
                      {texts?.tips[i] && !edits[phase]?.[lang] ? <SourceBadge source={texts.tips[i].source} /> : null}
                      <span className="flex-1" />
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={i === 0}
                        aria-label={t('moveUp')}
                        onClick={() => setDraft({ ...draft, tips: move(draft.tips, i, -1) })}
                      >
                        ↑
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={i === draft.tips.length - 1}
                        aria-label={t('moveDown')}
                        onClick={() => setDraft({ ...draft, tips: move(draft.tips, i, 1) })}
                      >
                        ↓
                      </Button>
                      <Button
                        size="sm"
                        variant="danger"
                        onClick={() => setDraft({ ...draft, tips: draft.tips.filter((_, j) => j !== i) })}
                      >
                        {t('remove')}
                      </Button>
                    </div>
                    <TextInput
                      label={t('tipTitle')}
                      required
                      maxLength={data.limits.title}
                      value={tip.title}
                      onChange={(e) => setDraft({ ...draft, tips: draft.tips.map((x, j) => (j === i ? { ...x, title: e.target.value } : x)) })}
                      error={err(`tips.${i}.title`)}
                    />
                    <TextArea
                      label={t('tipBody')}
                      rows={2}
                      maxLength={data.limits.body}
                      value={tip.body}
                      onChange={(e) => setDraft({ ...draft, tips: draft.tips.map((x, j) => (j === i ? { ...x, body: e.target.value } : x)) })}
                      error={err(`tips.${i}.body`)}
                    />
                  </div>
                ))}
                {err('tips') ? <span className="field-error">{err('tips')}</span> : null}
                <div>
                  <Button
                    size="sm"
                    disabled={draft.tips.length >= max}
                    onClick={() => setDraft({ ...draft, tips: [...draft.tips, { title: '', body: '' }] })}
                  >
                    <Icon name="plus" size={14} />
                    {t('addTip')}
                  </Button>
                </div>
              </fieldset>
            </div>

            <div className="flex flex-wrap items-center gap-2 border-t border-line pt-4">
              <Button type="submit" variant="primary" loading={save.isPending} disabled={!isDirty(phase)}>
                {tc('saveChanges')}
              </Button>
              {isDirty(phase) ? (
                <Button variant="ghost" onClick={() => discardPhase(phase)}>
                  {t('discard')}
                </Button>
              ) : null}
              <span className="flex-1" />
              {texts?.customized ? (
                <Button variant="ghost" loading={reset.isPending} onClick={() => void resetLanguage()}>
                  {t('reset')}
                </Button>
              ) : null}
            </div>
            <p className="field-hint m-0">{t('saveHint')}</p>
          </form>
        </Panel>

        <Panel title={t('preview')}>
          <div className="flex flex-col gap-3">
            <TextInput
              label={t('sampleName')}
              placeholder={t('sampleNameDefault')}
              maxLength={40}
              value={sampleName}
              onChange={(e) => setSampleName(e.target.value)}
            />
            <p className="field-hint m-0">{t('previewHint')}</p>
            {(['light', 'dark'] as const).map((theme) => (
              <TipsPreview key={theme} theme={theme} draft={draft} name={name} dir={current?.direction} lang={lang} />
            ))}
          </div>
        </Panel>
      </div>
    </div>
  );
}

/** The companion panel's note + «امروز چه کار کنی؟» card with the sample name filled in. */
function TipsPreview({
  theme,
  draft,
  name,
  dir,
  lang,
}: {
  theme: 'light' | 'dark';
  draft: LocaleDraft;
  name: string;
  dir?: 'rtl' | 'ltr';
  lang: string;
}) {
  const t = useTranslations('companions.tips');
  const visible = draft.tips.filter((x) => x.title.trim() !== '');
  return (
    <div data-theme={theme} className="rounded-xl border border-line bg-[var(--page)] p-3 text-[var(--ink)]">
      <span className="mb-2 block text-xs text-[var(--muted)]">{t(`theme.${theme}`)}</span>
      <div dir={dir} lang={lang} className="flex flex-col gap-2.5">
        {draft.note.trim() ? (
          <p className="m-0 rounded-xl bg-[var(--pink-bg)] p-3 text-[13px] leading-6">{fillName(draft.note.trim(), name)}</p>
        ) : (
          <p className="m-0 text-xs text-[var(--muted)]">{t('previewNoNote')}</p>
        )}
        <strong className="text-sm">{t('previewHeading')}</strong>
        {visible.length === 0 ? <p className="m-0 text-xs text-[var(--muted)]">{t('previewNoTips')}</p> : null}
        {visible.map((tip, i) => (
          <div key={i} className="flex items-start gap-2.5 rounded-xl bg-[var(--surface)] p-3">
            <span className="mt-0.5 grid size-7 shrink-0 place-items-center rounded-full bg-[var(--data-soft)] text-[var(--data-deep)]">
              <Icon name="heart" size={15} />
            </span>
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="text-[13px] font-semibold">{fillName(tip.title.trim(), name)}</span>
              {tip.body.trim() ? <span className="text-xs leading-5 text-[var(--ink-3)]">{fillName(tip.body.trim(), name)}</span> : null}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
