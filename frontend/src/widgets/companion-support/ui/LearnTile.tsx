'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import type { LearnTopic } from '../model/learn';
import '@/shared/ui/styles/cmh.css';

interface LearnTileProps {
  topic: LearnTopic;
  /** `short` = the home/personal label, `long` = the learn page title. */
  label?: 'short' | 'long';
  /** Navigate (home, personal panel) … */
  href?: string;
  /** … or act in place (the learn page opens the topic). */
  onOpen?: () => void;
  tall?: boolean;
}

/** A soft-tinted learn tile (compLearn `learn-tile`); the saturated prototype gradients map to tint gradients. */
export function LearnTile({ topic, label = 'short', href, onOpen, tall }: LearnTileProps) {
  const t = useTranslations('companionHome.learn.topics');
  const className = clsx('cmh2-tile', `is-${topic.accent}`, tall && 'is-tall');
  const body = (
    <>
      <span className="cmh2-tile-ic" aria-hidden>
        <Icon name={topic.icon} size={18} />
      </span>
      <span className="cmh2-tile-t">{t(`${topic.id}.${label}`)}</span>
    </>
  );
  if (href) {
    return (
      <Link href={href} className={className}>
        {body}
      </Link>
    );
  }
  return (
    <button type="button" className={className} onClick={onOpen}>
      {body}
    </button>
  );
}
