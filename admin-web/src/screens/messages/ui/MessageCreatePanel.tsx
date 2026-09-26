'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { Badge, Button, ErrorState, Panel, Skeleton, Switch, toast, useNotifyError } from '@/shared/ui';

import { messagesApi, useRegistryItem, type RegistryItem } from '../api/messages';
import { normalizeDraft, seedDraft, type Draft } from '../lib/schema-form';
import { useMessageLabels } from './labels';
import { SchemaForm } from './SchemaForm';

export interface CreateTarget {
  group: string;
  item_key: string;
  locale: string;
}

/** «ایجاد» a missing row of a registered group: the item's schema, prefilled from its template (§13). */
export function MessageCreatePanel({ target, onClose }: { target: CreateTarget; onClose: () => void }) {
  const t = useTranslations('smartMessages');
  const labels = useMessageLabels();
  const item = useRegistryItem(target.group, target.item_key, target.locale);
  return (
    <Panel
      title={t('createTitle', {
        group: labels.group(target.group),
        key: target.item_key,
      })}
      actions={
        <Button size="sm" variant="ghost" onClick={onClose}>
          {t('cancel')}
        </Button>
      }
    >
      {item.error && !item.data ? (
        <ErrorState error={item.error} onRetry={() => item.refetch()} />
      ) : !item.data ? (
        <Skeleton className="h-40 w-full" />
      ) : (
        <CreateForm key={`${target.group}/${target.item_key}/${target.locale}`} item={item.data} target={target} onClose={onClose} />
      )}
    </Panel>
  );
}

function CreateForm({ item, target, onClose }: { item: RegistryItem; target: CreateTarget; onClose: () => void }) {
  const t = useTranslations('smartMessages');
  const tc = useTranslations('crud');
  const labels = useMessageLabels();
  const notifyError = useNotifyError();
  const save = messagesApi.useSave(null);
  const [draft, setDraft] = useState<Draft>(() => seedDraft(item.fields, item.template));
  const [active, setActive] = useState(true);

  const submit = () =>
    save.mutate(
      {
        group: target.group,
        item_key: target.item_key,
        locale: target.locale,
        payload: normalizeDraft(item.fields, draft),
        is_active: active,
        is_approved: true,
      },
      {
        onSuccess: () => {
          toast.success(tc('created'));
          onClose();
        },
        onError: notifyError,
      },
    );

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
    >
      <div className="flex flex-wrap gap-1.5">
        <Badge tone="brand">{labels.locale(target.locale)}</Badge>
        {item.existing?.map((e) => (
          <Badge key={e.id} tone="green">
            {t('existsIn', { locale: labels.locale(e.locale) })}
          </Badge>
        ))}
      </div>
      <p className="field-hint m-0">{t('createHint')}</p>
      {item.placeholders && item.placeholders.length > 0 ? (
        <p className="field-hint m-0">
          {t('placeholders')}: <span dir="ltr">{item.placeholders.map((p) => `{${p}}`).join(' ')}</span>
        </p>
      ) : null}
      <SchemaForm fields={item.fields} value={draft} onChange={setDraft} error={save.error} dir={labels.direction(target.locale)} />
      <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      <div>
        <Button type="submit" variant="primary" loading={save.isPending}>
          {tc('create')}
        </Button>
      </div>
    </form>
  );
}
