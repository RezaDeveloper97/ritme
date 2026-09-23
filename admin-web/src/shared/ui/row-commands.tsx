'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';

import { Button } from './Button';
import { confirm } from './confirm';
import { RowActions } from './RowActions';
import { toast } from './toast';
import { useNotifyError } from './use-notify-error';

type Callbacks = { onSuccess?: () => void; onError?: (error: unknown) => void };
interface Mutation<V> {
  mutate: (variables: V, callbacks?: Callbacks) => void;
  isPending: boolean;
  variables?: V;
}

/**
 * Toggle + delete for a list row, with the confirm dialog and toasts every
 * content list shares. Pass the resource's `useAction('toggle')` / `useRemove()`.
 */
export function useRowCommands({
  toggle,
  remove,
  confirmDelete,
}: {
  toggle?: Mutation<{ id: number }>;
  remove?: Mutation<number>;
  confirmDelete?: string;
}) {
  const t = useTranslations('crud');
  const notifyError = useNotifyError();
  return {
    toggle: (id: number) =>
      toggle?.mutate({ id }, { onSuccess: () => toast.success(t('statusChanged')), onError: notifyError }),
    remove: async (id: number) => {
      if (!remove) return;
      if (!(await confirm({ message: confirmDelete ?? t('confirmDelete'), confirmLabel: t('delete'), tone: 'danger' }))) return;
      remove.mutate(id, { onSuccess: () => toast.success(t('deleted')), onError: notifyError });
    },
    isToggling: (id: number) => Boolean(toggle?.isPending && toggle.variables?.id === id),
    isRemoving: (id: number) => Boolean(remove?.isPending && remove.variables === id),
  };
}

/** Edit link + optional toggle + optional delete, as one table cell. */
export function CrudRowActions({
  id,
  editHref,
  commands,
  active,
  activeLabels,
  canDelete = true,
}: {
  id: number;
  editHref: string;
  commands: ReturnType<typeof useRowCommands>;
  /** Current on/off state; omit for resources without a toggle. */
  active?: boolean;
  /** [label when on (turns it off), label when off] — defaults to deactivate/activate. */
  activeLabels?: readonly [string, string];
  canDelete?: boolean;
}) {
  const t = useTranslations('crud');
  const [offLabel, onLabel] = activeLabels ?? [t('deactivate'), t('activate')];
  return (
    <RowActions>
      <Link href={editHref} className="btn btn-sm">
        {t('edit')}
      </Link>
      {active !== undefined ? (
        <Button size="sm" onClick={() => commands.toggle(id)} loading={commands.isToggling(id)}>
          {active ? offLabel : onLabel}
        </Button>
      ) : null}
      {canDelete ? (
        <Button size="sm" variant="danger" onClick={() => commands.remove(id)} loading={commands.isRemoving(id)}>
          {t('delete')}
        </Button>
      ) : null}
    </RowActions>
  );
}
