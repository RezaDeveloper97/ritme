'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState, type FormEvent } from 'react';

import { useNumber } from '@/shared/lib';
import { Badge, Button, DataTable, Panel, TextInput, type Column } from '@/shared/ui';

import { useCatalogGroups, type CatalogGroup } from '../api/catalog';
import { CODE_PATTERN, MAX_CODE_LEN, isValidCode } from '../lib/payload';

/** /catalog — the group picker: every group that has items, plus opening a new one (catalog.md §3). */
export function CatalogGroupsScreen() {
  const t = useTranslations('catalog');
  const n = useNumber();
  const router = useRouter();
  const groups = useCatalogGroups();

  const columns: Column<CatalogGroup>[] = [
    {
      key: 'group',
      header: t('group'),
      cell: (g) => (
        <span dir="ltr" className="font-bold">
          {g.group}
        </span>
      ),
    },
    { key: 'items', header: t('itemsCount'), cell: (g) => n(g.items_count), className: 'cell-num' },
    {
      key: 'active',
      header: t('activeCount'),
      cell: (g) =>
        g.active_count < g.items_count ? <Badge tone="amber">{n(g.active_count)}</Badge> : n(g.active_count),
      className: 'cell-num',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <Panel title={t('title')} bodyClassName="">
        <p className="panel-note">{t('intro')}</p>
        <DataTable
          columns={columns}
          items={groups.data?.items}
          rowKey={(g) => g.group}
          loading={groups.isPending}
          error={groups.error}
          onRetry={() => groups.refetch()}
          onRowClick={(g) => router.push(`/catalog/${g.group}`)}
          emptyText={t('noGroups')}
          caption={t('title')}
        />
      </Panel>
      <NewGroupPanel onOpen={(group) => router.push(`/catalog/${group}`)} />
    </div>
  );
}

function NewGroupPanel({ onOpen }: { onOpen: (group: string) => void }) {
  const t = useTranslations('catalog');
  const [group, setGroup] = useState('');
  const [error, setError] = useState<string | undefined>();

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const value = group.trim();
    if (!isValidCode(value)) {
      setError(t('codeInvalid'));
      return;
    }
    onOpen(value);
  };

  return (
    <Panel title={t('newGroup')}>
      <form onSubmit={submit} className="flex flex-col gap-3 sm:flex-row sm:items-start">
        <TextInput
          className="flex-1"
          label={t('group')}
          hint={t('newGroupHint')}
          value={group}
          onChange={(e) => {
            setGroup(e.target.value);
            setError(undefined);
          }}
          dir="ltr"
          maxLength={MAX_CODE_LEN}
          pattern={CODE_PATTERN}
          placeholder="teen_faq"
          error={error}
          required
        />
        <Button type="submit" variant="primary" className="sm:mt-6">
          {t('openGroup')}
        </Button>
      </form>
    </Panel>
  );
}
