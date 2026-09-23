'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError } from '@/shared/api';
import {
  Badge,
  Button,
  FormPage,
  LoadGate,
  PageHeader,
  TextArea,
  TextInput,
  toast,
  useNotifyError,
} from '@/shared/ui';

import { messagesApi, type Message } from '../api/messages';
import { fieldKind, fromDraft, toDraft } from '../lib/payload';
import { useMessageLabels } from './labels';

/** /messages/:id — edit one smart message's payload, keeping its shape (Blade messages.edit). */
export function MessageEditScreen({ id }: { id: number }) {
  const t = useTranslations('smartMessages');
  const detail = messagesApi.useDetail(id);
  return (
    <LoadGate queries={[detail]} header={<PageHeader title={t('edit')} backHref="/messages" backLabel={t('backToList')} />}>
      {() => (detail.data ? <MessageForm message={detail.data.message} /> : null)}
    </LoadGate>
  );
}

function MessageForm({ message }: { message: Message }) {
  const t = useTranslations('smartMessages');
  const tc = useTranslations('crud');
  const router = useRouter();
  const labels = useMessageLabels();
  const notifyError = useNotifyError();
  const save = messagesApi.useSave(message.id);
  const approve = messagesApi.useAction('approve');
  const toggle = messagesApi.useAction('toggle');
  const [draft, setDraft] = useState<Record<string, string>>(() => toDraft(message.payload));
  const dir = labels.direction(message.locale);
  const backHref = `/messages?group=${encodeURIComponent(message.group)}`;

  const submit = () =>
    save.mutate(
      { payload: fromDraft(message.payload, draft) },
      {
        onSuccess: () => {
          toast.success(tc('saved'));
          router.push(backHref);
        },
        onError: notifyError,
      },
    );

  const keys = Object.keys(message.payload);

  return (
    <FormPage
      title={labels.group(message.group)}
      backHref={backHref}
      backLabel={t('backToList')}
      meta={
        <>
          <Badge>
            {t('key')}: <span dir="ltr">{message.item_key}</span>
          </Badge>
          <Badge tone="brand">{labels.locale(message.locale)}</Badge>
          {message.is_approved ? <Badge tone="green">{t('approved')}</Badge> : <Badge tone="amber">{t('pending')}</Badge>}
          {message.is_active ? null : <Badge tone="red">{tc('inactive')}</Badge>}
        </>
      }
      headerActions={
        <>
          <Button
            loading={approve.isPending}
            onClick={() =>
              approve.mutate(
                { id: message.id },
                { onSuccess: () => toast.success(message.is_approved ? t('unapprovedToast') : t('approvedToast')), onError: notifyError },
              )
            }
          >
            {message.is_approved ? t('unapprove') : t('approve')}
          </Button>
          <Button
            loading={toggle.isPending}
            onClick={() => toggle.mutate({ id: message.id }, { onSuccess: () => toast.success(tc('statusChanged')), onError: notifyError })}
          >
            {message.is_active ? tc('deactivate') : tc('activate')}
          </Button>
        </>
      }
      onSubmit={submit}
      submitLabel={tc('saveChanges')}
      saving={save.isPending}
    >
      <p className="field-hint m-0">{t('editHint')}</p>
      {keys.length === 0 ? <p className="m-0 text-ink-3">{t('emptyPayload')}</p> : null}
      {keys.map((key) => {
        const original = message.payload[key] ?? '';
        const kind = fieldKind(original);
        const label = (
          <>
            <span dir="ltr">{key}</span>
            {kind === 'list' ? <span className="ms-2 font-normal text-muted">{t('listHint')}</span> : null}
          </>
        );
        const common = {
          label,
          dir,
          value: draft[key] ?? '',
          error: fieldError(save.error, `payload.${key}`),
        };
        return kind === 'line' ? (
          <TextInput key={key} {...common} onChange={(e) => setDraft((d) => ({ ...d, [key]: e.target.value }))} />
        ) : (
          <TextArea
            key={key}
            {...common}
            rows={kind === 'list' ? Math.max(3, Array.isArray(original) ? original.length : 3) : 4}
            onChange={(e) => setDraft((d) => ({ ...d, [key]: e.target.value }))}
          />
        );
      })}
    </FormPage>
  );
}
