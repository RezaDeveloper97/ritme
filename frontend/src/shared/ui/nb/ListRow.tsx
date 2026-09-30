'use client';

import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import { useDirection } from '@/shared/i18n';

import { Icon, type IconName } from '../Icon';
import { IconCircle } from './IconCircle';
import type { Tone } from './tone';

interface ListRowProps {
  title: ReactNode;
  description?: ReactNode;
  icon?: IconName;
  iconTone?: Tone;
  /** Ring around the icon disc (settings rows). */
  iconOutlined?: boolean;
  /** Short value at the end («فارسی», «۲ روز قبل»). */
  value?: ReactNode;
  /** Control at the end — a {@link Switch}, a pill… Not combined with `onClick`. */
  trailing?: ReactNode;
  /** Makes the whole row one button and shows a forward chevron. */
  onClick?: () => void;
  /** Sets the row's id prefix: the title gets `${id}-title` (for `Switch labelledBy`). */
  id?: string;
  className?: string;
}

/** 52–56px settings/list row: icon disc, label 13.5/700 (+ caption), value/chevron at the end. */
export function ListRow({
  title,
  description,
  icon,
  iconTone = 'brand',
  iconOutlined,
  value,
  trailing,
  onClick,
  id,
  className,
}: ListRowProps) {
  const rtl = useDirection() === 'rtl';
  const content = (
    <>
      {icon ? <IconCircle icon={icon} tone={iconTone} outlined={iconOutlined} /> : null}
      <span className="nb-row-text">
        <span id={id ? `${id}-title` : undefined} className="nb-row-title">
          {title}
        </span>
        {description ? <span className="nb-row-desc">{description}</span> : null}
      </span>
      {value ? <span className="nb-row-value">{value}</span> : null}
      {trailing}
      {onClick ? <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="nb-row-chev" /> : null}
    </>
  );
  return onClick ? (
    <button type="button" className={clsx('nb-row', 'is-action', className)} onClick={onClick}>
      {content}
    </button>
  ) : (
    <div className={clsx('nb-row', className)}>{content}</div>
  );
}

interface ListGroupProps {
  /** Optional group heading inside the card («یادآورها»). */
  title?: ReactNode;
  className?: string;
  children: ReactNode;
}

/** Rows grouped in one flat card with 1px dividers. */
export function ListGroup({ title, className, children }: ListGroupProps) {
  return (
    <section className={clsx('nb-card', 'nb-list', className)}>
      {title ? <h2 className="nb-list-title">{title}</h2> : null}
      <div className="nb-list-rows">{children}</div>
    </section>
  );
}
