'use client';

import { clsx } from 'clsx';

import { childTone } from '../model/format';
import { useChildPhoto } from '../model/photo';
import type { ChildSex } from '../model/types';

interface ChildAvatarProps {
  id: number;
  name: string;
  initial?: string;
  sex: ChildSex | null;
  /** 36 (strips) · 48 (companion card) · 60 (list) · 84 (child home). */
  size?: 36 | 48 | 60 | 84;
  /** Decorative next to a visible name (default); otherwise the name is its label. */
  decorative?: boolean;
  className?: string;
}

/**
 * The child's round avatar: the on-device photo when one was picked
 * (`model/photo.ts`, never uploaded), else the initial on the sex tone
 * (artboards: pink girl, teal boy).
 */
export function ChildAvatar({ id, name, initial, sex, size = 60, decorative = true, className }: ChildAvatarProps) {
  const photo = useChildPhoto(id);
  const letter = initial || Array.from(name.trim())[0] || '';
  return (
    <span
      className={clsx('chd-avatar', `is-${size}`, `nb-tone-${childTone(sex)}`, className)}
      role={decorative ? undefined : 'img'}
      aria-label={decorative ? undefined : name}
      aria-hidden={decorative ? true : undefined}
    >
      {photo ? (
        // A blob: URL of a local file — next/image can't optimise it.
        // eslint-disable-next-line @next/next/no-img-element
        <img src={photo} alt="" className="chd-avatar-img" />
      ) : (
        <span aria-hidden>{letter}</span>
      )}
    </span>
  );
}
