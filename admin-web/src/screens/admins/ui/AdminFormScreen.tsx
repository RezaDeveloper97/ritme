'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { RequireSuper, useCurrentAdmin } from '@/features/auth';
import type { Admin } from '@/entities/admin';
import { fieldError } from '@/shared/api';
import { FormPage, LoadGate, PageHeader, Select, Switch, TextInput, toast, useNotifyError } from '@/shared/ui';

import { adminsApi, type AdminInput } from '../api/admins';

/** /admins/new and /admins/:id (super admins; Blade admins.form). */
export function AdminFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('admins');
  const detail = adminsApi.useDetail(id);
  return (
    <RequireSuper>
      <LoadGate
        queries={[detail]}
        header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/admins" backLabel={t('backToList')} />}
      >
        {() => <AdminForm id={id} row={detail.data?.admin ?? null} />}
      </LoadGate>
    </RequireSuper>
  );
}

function AdminForm({ id, row }: { id: number | null; row: Admin | null }) {
  const t = useTranslations('admins');
  const tr = useTranslations('roles');
  const tc = useTranslations('crud');
  const router = useRouter();
  const me = useCurrentAdmin();
  const notifyError = useNotifyError();
  const save = adminsApi.useSave(id);
  const isSelf = row !== null && row.id === me?.id;

  const [name, setName] = useState(row?.name ?? '');
  const [email, setEmail] = useState(row?.email ?? '');
  const [password, setPassword] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [role, setRole] = useState<string>(row?.role ?? 'editor');
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (field: string) => fieldError(save.error, field);

  const submit = () => {
    const body: AdminInput = { name: name.trim(), email: email.trim() };
    if (password || id === null) {
      body.password = password;
      body.password_confirmation = confirmation;
    }
    // A super admin can't change their own role or deactivate themselves (cannot_modify_self).
    if (!isSelf) {
      body.role = role;
      body.is_active = active;
    } else {
      body.role = row?.role;
    }
    save.mutate(body, {
      onSuccess: () => {
        toast.success(id === null ? tc('created') : tc('saved'));
        router.push('/admins');
      },
      onError: notifyError,
    });
  };

  return (
    <FormPage
      title={id === null ? t('new') : row?.name ?? t('edit')}
      backHref="/admins"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? t('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <TextInput label={t('name')} value={name} onChange={(e) => setName(e.target.value)} required maxLength={255} error={err('name')} />
        <TextInput
          label={t('email')}
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          dir="ltr"
          autoComplete="off"
          error={err('email')}
        />
        <TextInput
          label={t('password')}
          hint={id === null ? t('passwordHint') : t('passwordKeep')}
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required={id === null}
          minLength={8}
          maxLength={72}
          autoComplete="new-password"
          error={err('password')}
        />
        <TextInput
          label={t('passwordConfirm')}
          type="password"
          value={confirmation}
          onChange={(e) => setConfirmation(e.target.value)}
          required={id === null || password !== ''}
          autoComplete="new-password"
          error={err('password_confirmation')}
        />
        <Select
          label={t('role')}
          hint={isSelf ? t('selfRole') : undefined}
          value={role}
          disabled={isSelf}
          onChange={(e) => setRole(e.target.value)}
          options={[
            { value: 'editor', label: `${tr('editor')} (${t('editorScope')})` },
            { value: 'super', label: `${tr('super')} (${t('superScope')})` },
          ]}
          error={err('role')}
        />
        <Switch
          label={tc('isActive')}
          hint={isSelf ? t('selfActive') : undefined}
          checked={active}
          disabled={isSelf}
          onChange={setActive}
        />
      </div>
    </FormPage>
  );
}
