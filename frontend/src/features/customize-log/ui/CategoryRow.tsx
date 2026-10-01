'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';

import { Icon, IconCircle, type IconName, Switch, type Tone } from '@/shared/ui';

import type { ReorderHandle } from '../model/use-reorder';

export interface CategoryLook {
  icon: IconName;
  tone: Tone;
}

interface CategoryRowProps {
  code: string;
  label: string;
  look: CategoryLook;
  index: number;
  total: number;
  hidden: boolean;
  pinned: boolean;
  /** Pinning is refused (8 tiles already; the tap explains why); unpinning still works. */
  pinLocked: boolean;
  dragging: boolean;
  handle: ReorderHandle;
  rowRef: (el: HTMLElement | null) => void;
  /** id of the visually hidden drag instructions. */
  helpId: string;
  /** id of the pin counter (why a pin is disabled). */
  counterId: string;
  onMove: (to: number) => void;
  onToggleHidden: () => void;
  onTogglePin: () => void;
}

/**
 * One row of nbl_Log_Customize: drag handle · tinted icon · name · pin · show switch. The handle is a
 * real button (pointer drag + ↑/↓ keys); the explicit «up / down» buttons stay out of sight until they
 * get keyboard focus, so screen-reader and switch users can reorder without a drag.
 */
export function CategoryRow({
  code,
  label,
  look,
  index,
  total,
  hidden,
  pinned,
  pinLocked,
  dragging,
  handle,
  rowRef,
  helpId,
  counterId,
  onMove,
  onToggleHidden,
  onTogglePin,
}: CategoryRowProps) {
  const t = useTranslations('logCustomize');
  const pinDisabled = pinLocked && !pinned;
  return (
    <li ref={rowRef} className={clsx('lcz-row', hidden && 'is-hidden', dragging && 'is-dragging')}>
      <button
        type="button"
        id={`lcz-grip-${code}`}
        className="lcz-grip"
        aria-label={t('dragHandle', { label })}
        aria-describedby={helpId}
        {...handle}
      >
        <Icon name="grip" size={18} strokeWidth={2.6} />
      </button>
      <span className="lcz-moves">
        <button
          type="button"
          id={`lcz-up-${code}`}
          className="lcz-move"
          aria-label={t('moveUp', { label })}
          disabled={index === 0}
          onClick={() => onMove(index - 1)}
        >
          <Icon name="chevronUp" size={16} strokeWidth={2.2} />
        </button>
        <button
          type="button"
          id={`lcz-down-${code}`}
          className="lcz-move"
          aria-label={t('moveDown', { label })}
          disabled={index === total - 1}
          onClick={() => onMove(index + 1)}
        >
          <Icon name="chevronDown" size={16} strokeWidth={2.2} />
        </button>
      </span>
      <IconCircle icon={look.icon} tone={look.tone} size="sm" className="lcz-icon" />
      <span className="lcz-name">
        {label}
      </span>
      <button
        type="button"
        className="lcz-pin"
        aria-pressed={pinned}
        aria-label={t('pin', { label })}
        aria-describedby={pinDisabled ? counterId : undefined}
        aria-disabled={pinDisabled || undefined}
        onClick={onTogglePin}
      >
        <span className="lcz-pin-disc">
          <Icon name="pin" size={15} strokeWidth={2} />
        </span>
      </button>
      <Switch checked={!hidden} label={t('show', { label })} onCheckedChange={onToggleHidden} className="lcz-switch" />
    </li>
  );
}
