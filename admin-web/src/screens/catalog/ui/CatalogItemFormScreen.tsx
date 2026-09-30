'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { toIntOrNull, useNumber } from '@/shared/lib';
import {
  Badge,
  FormPage,
  LoadGate,
  PageHeader,
  Switch,
  TextArea,
  TextInput,
  TranslatableField,
  toast,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { useCatalogItem, useSaveItem, type CatalogItem } from '../api/catalog';
import {
  CODE_PATTERN,
  MAX_BODY_LEN,
  MAX_CODE_LEN,
  MAX_TITLE_LEN,
  audiencesError,
  cleanTranslations,
  formatMeta,
  invalidAudiences,
  isValidCode,
  parseAudiences,
  parseMeta,
} from '../lib/payload';
import { MetaHint } from './MetaHint';

/** /catalog/:group/new and /catalog/:group/:id. */
export function CatalogItemFormScreen({ group, id }: { group: string; id: number | null }) {
  const t = useTranslations('catalog');
  const detail = useCatalogItem(group, id);
  return (
    <LoadGate
      queries={[detail]}
      header={<PageHeader title={id === null ? t('newItem') : t('editItem')} backHref={`/catalog/${group}`} backLabel={t('backToItems')} />}
    >
      {() => <CatalogItemForm group={group} id={id} row={detail.data?.catalog_item ?? null} />}
    </LoadGate>
  );
}

function CatalogItemForm({ group, id, row }: { group: string; id: number | null; row: CatalogItem | null }) {
  const t = useTranslations('catalog');
  const tc = useTranslations('crud');
  const n = useNumber();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = useSaveItem(group, id);

  const [code, setCode] = useState(row?.code ?? '');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [body, setBody] = useState<Translations>(row?.body ?? {});
  const [audiences, setAudiences] = useState((row?.audiences ?? []).join(', '));
  const [meta, setMeta] = useState(formatMeta(row?.meta));
  const [sortOrder, setSortOrder] = useState(row ? String(row.sort_order) : '');
  const [active, setActive] = useState(row?.is_active ?? true);
  const [needsReview, setNeedsReview] = useState(row?.needs_review ?? true);
  const [local, setLocal] = useState<{ code?: string; audiences?: string; meta?: string }>({});

  const apiErrors = fieldErrorsOf(save.error);
  const err = (name: string) => fieldError(save.error, name);
  const back = `/catalog/${group}`;

  const submit = () => {
    const problems: typeof local = {};
    if (id === null && !isValidCode(code.trim())) problems.code = t('codeInvalid');
    const codes = parseAudiences(audiences);
    const bad = invalidAudiences(codes);
    if (bad.length) problems.audiences = t('audiencesInvalid', { codes: bad.join(', ') });
    const parsed = parseMeta(meta);
    if (!parsed.ok) problems.meta = t(`metaError.${parsed.reason}` as 'metaError.json');
    setLocal(problems);
    if (Object.keys(problems).length || !parsed.ok) return;

    const sort = toIntOrNull(sortOrder);
    save.mutate(
      {
        ...(id === null ? { code: code.trim() } : {}),
        title: cleanTranslations(title) ?? {},
        body: cleanTranslations(body),
        audiences: codes.length ? codes : null,
        meta: parsed.value,
        ...(sort === null ? {} : { sort_order: sort }),
        is_active: active,
        needs_review: needsReview,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push(back);
        },
        onError: notifyError,
      },
    );
  };

  return (
    <FormPage
      title={id === null ? t('newItem') : localize(row?.title) || row?.code || t('editItem')}
      meta={
        <>
          <span dir="ltr">{group}</span>
          {row?.needs_review ? <Badge tone="amber">{t('needsReview')}</Badge> : null}
          {row ? <span>{t('orderN', { n: n(row.sort_order) })}</span> : null}
        </>
      }
      backHref={back}
      backLabel={t('backToItems')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <TextInput
        label={t('code')}
        hint={id === null ? t('codeHint') : t('codeFixed')}
        value={code}
        onChange={(e) => setCode(e.target.value)}
        dir="ltr"
        maxLength={MAX_CODE_LEN}
        pattern={CODE_PATTERN}
        required={id === null}
        disabled={id !== null}
        error={local.code ?? err('code')}
      />

      <TranslatableField
        name="title"
        label={t('titleField')}
        value={title}
        onChange={setTitle}
        required
        maxLength={MAX_TITLE_LEN}
        errors={apiErrors}
      />
      <TranslatableField
        name="body"
        label={t('body')}
        kind="textarea"
        value={body}
        onChange={setBody}
        maxLength={MAX_BODY_LEN}
        errors={apiErrors}
      />

      <TextInput
        label={t('audiences')}
        hint={t('audiencesHint')}
        value={audiences}
        onChange={(e) => setAudiences(e.target.value)}
        dir="ltr"
        placeholder="menopause, teen"
        error={local.audiences ?? audiencesError(apiErrors)}
      />

      <div className="flex flex-col gap-3">
        <TextArea
          label={t('meta')}
          hint={t('metaFieldHint')}
          value={meta}
          onChange={(e) => setMeta(e.target.value)}
          dir="ltr"
          rows={8}
          spellCheck={false}
          error={local.meta ?? err('meta')}
        />
        <MetaHint group={group} onInsert={meta.trim() ? undefined : setMeta} />
      </div>

      <TextInput
        type="number"
        label={tc('sortOrder')}
        hint={id === null ? t('sortOrderNewHint') : tc('sortOrderHint')}
        value={sortOrder}
        onChange={(e) => setSortOrder(e.target.value)}
        min={0}
        error={err('sort_order')}
      />

      <Switch label={tc('isActive')} hint={t('activeHint')} checked={active} onChange={setActive} />
      <Switch label={t('needsReviewField')} hint={t('needsReviewHint')} checked={needsReview} onChange={setNeedsReview} />
    </FormPage>
  );
}
