'use client';

import { useTranslations } from 'next-intl';

import { type Child, ChildAvatar, useChildren } from '@/entities/child';
import { Link } from '@/shared/i18n';
import { Icon, PrimaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';

import { toggleChildId } from '../model/draft';

interface ChildrenPickerProps {
  /** Selected child ids. */
  value: readonly number[];
  onChange: (ids: number[]) => void;
  disabled?: boolean;
  /** Show the «افزودن فرزند» link (leaves the current screen for `/children/new`). */
  showAdd?: boolean;
}

/** The owner's own children — a spouse can only be given children she owns (a shared one isn't hers to share). */
export function ownChildren(children: readonly Child[]): Child[] {
  return children.filter((c) => c.role === 'owner');
}

/**
 * «کدام فرزند مشترک است؟» (nbl_Hamdam_Children): one checkbox card per child
 * the owner registered (`GET /children`), for the spouse wizard (`child_ids`
 * on create) and the companion detail editor (PUT /companions/{id}/children).
 */
export function ChildrenPicker({ value, onChange, disabled, showAdd = true }: ChildrenPickerProps) {
  const t = useTranslations('companions.flow.children');
  const query = useChildren();

  if (query.isPending) {
    return (
      <SkeletonGroup label={t('loading')} className="cmp-kids">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  }
  if (query.isError || !query.data) {
    return (
      <div className="nb-card cmp-kids-msg">
        <p className="cmp-soon-body">{t('errorBody')}</p>
        <PrimaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
          {t('retry')}
        </PrimaryButton>
      </div>
    );
  }

  const kids = ownChildren(query.data.children);
  const add =
    showAdd && query.data.canAdd ? (
      <Link href="/children/new" className="cmp-add-child is-live">
        <Icon name="plus" size={18} strokeWidth={2.2} />
        {t('add')}
      </Link>
    ) : null;

  if (kids.length === 0) {
    return (
      <>
        <div className="nb-card cmp-soon">
          <span className="cmp-soon-icon" aria-hidden>
            <Icon name="sprout" size={22} />
          </span>
          <div className="cmp-soon-text">
            <b className="cmp-soon-title">{t('emptyTitle')}</b>
            <p className="cmp-soon-body">{t('emptyBody')}</p>
          </div>
        </div>
        {add}
      </>
    );
  }

  return (
    <>
      <ul className="cmp-kids">
        {kids.map((c) => (
          <li key={c.id}>
            <label className="cmp-kid">
              <ChildAvatar id={c.id} name={c.name} initial={c.initial} sex={c.sex} size={48} />
              <span className="cmp-kid-text">
                <b className="cmp-kid-name">{c.name}</b>
                <span className="cmp-kid-age">{c.age.label}</span>
              </span>
              <input
                type="checkbox"
                className="cmp-kid-check"
                checked={value.includes(c.id)}
                disabled={disabled}
                onChange={() => onChange(toggleChildId(value, c.id))}
              />
            </label>
          </li>
        ))}
      </ul>
      {add}
    </>
  );
}

/** Names of the selected children, in list order (summary rows, family strips). */
export function useChildNames(ids: readonly number[]): Array<{ id: number; name: string }> {
  const query = useChildren({ enabled: ids.length > 0 });
  const all = query.data?.children ?? [];
  return ids.flatMap((id) => {
    const c = all.find((x) => x.id === id);
    return c ? [{ id, name: c.name }] : [];
  });
}
