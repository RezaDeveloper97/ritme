'use client';

import { useTranslations } from 'next-intl';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, toIntOrNull } from '@/shared/lib';
import {
  FormPage,
  LoadGate,
  PageHeader,
  Select,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { INFO_GROUPS, INFO_KEY_PATTERN, infoSectionsApi, type InfoSection } from '../api/info-sections';
import { useGroupLabel } from './group-label';

const listHref = (group: string) => (group === 'help' ? '/info-sections' : `/info-sections?group=${group}`);

/** /info-sections/new?group= and /info-sections/:id (Blade info-sections.form). */
export function InfoSectionFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('infoSections');
  const search = useSearchParams();
  const requested = search.get('group') ?? 'help';
  const detail = infoSectionsApi.useDetail(id);
  const group = detail.data?.info_section.group ?? requested;
  const options = infoSectionsApi.useOptions({ group });
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref={listHref(group)} backLabel={t('backToList')} />}
    >
      {() => (
        <InfoSectionForm
          id={id}
          row={detail.data?.info_section ?? null}
          group={options.data?.group ?? group}
          groups={options.data?.groups ?? [...INFO_GROUPS]}
          nextSort={options.data?.next_sort_order ?? 0}
        />
      )}
    </LoadGate>
  );
}

function InfoSectionForm({
  id,
  row,
  group: initialGroup,
  groups,
  nextSort,
}: {
  id: number | null;
  row: InfoSection | null;
  group: string;
  groups: string[];
  nextSort: number;
}) {
  const t = useTranslations('infoSections');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const groupLabel = useGroupLabel();
  const notifyError = useNotifyError();
  const save = infoSectionsApi.useSave(id);

  const [group, setGroup] = useState(row?.group ?? initialGroup);
  const [key, setKey] = useState(row?.key ?? '');
  const [heading, setHeading] = useState<Translations>(row?.heading ?? {});
  const [body, setBody] = useState<Translations>(row?.body ?? {});
  const [linkLabel, setLinkLabel] = useState<Translations>(row?.link_label ?? {});
  const [linkUrl, setLinkUrl] = useState(row?.link_url ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? nextSort));
  const [active, setActive] = useState(row?.is_active ?? true);
  const errors = fieldErrorsOf(save.error);
  const err = (name: string) => fieldError(save.error, name);

  const submit = () =>
    save.mutate(
      {
        group,
        key: blankToNull(key.trim()),
        heading,
        body,
        link_label: linkLabel,
        link_url: blankToNull(linkUrl),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push(listHref(group));
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('newIn', { page: groupLabel(initialGroup) }) : localize(row?.heading) || t('edit')}
      backHref={listHref(initialGroup)}
      backLabel={t('backTo', { page: groupLabel(initialGroup) })}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <Select
        label={t('page')}
        hint={t('pageHint')}
        value={group}
        onChange={(e) => setGroup(e.target.value)}
        options={groups.map((g) => ({ value: g, label: groupLabel(g) }))}
        required
        error={err('group')}
      />
      <TextInput
        label={t('key')}
        hint={t('keyHint')}
        value={key}
        onChange={(e) => setKey(e.target.value)}
        dir="ltr"
        maxLength={64}
        pattern={INFO_KEY_PATTERN}
        autoComplete="off"
        spellCheck={false}
        placeholder="email"
        error={err('key')}
      />
      <TranslatableField name="heading" label={t('heading')} value={heading} onChange={setHeading} required maxLength={200} errors={errors} />
      <TranslatableField name="body" label={t('body')} value={body} onChange={setBody} kind="textarea" required errors={errors} />
      <fieldset className="form-section m-0 flex min-w-0 flex-col gap-4 border-0 p-0">
        <legend className="field-label p-0">{t('contactButton')}</legend>
        <p className="field-hint m-0">{t('contactButtonHint')}</p>
        <TranslatableField name="link_label" label={t('linkLabel')} value={linkLabel} onChange={setLinkLabel} maxLength={60} errors={errors} />
        <TextInput
          label={t('linkUrl')}
          value={linkUrl}
          onChange={(e) => setLinkUrl(e.target.value)}
          dir="ltr"
          maxLength={500}
          placeholder="mailto:support@ritmesalamat.com"
          error={err('link_url')}
        />
      </fieldset>
      <div className="form-grid">
        <TextInput
          label={tc('sortOrder')}
          hint={tc('sortOrderHint')}
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          error={err('sort_order')}
        />
        <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      </div>
    </FormPage>
  );
}
