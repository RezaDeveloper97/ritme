import type { HTMLAttributes, ReactNode } from 'react';
import { clsx } from 'clsx';

type CardVariant = 'default' | 'secondary' | 'hero' | 'inset';

interface CardProps extends HTMLAttributes<HTMLElement> {
  /**
   * `default` flat card (surface, 1px line, radius 24) · `secondary` radius 20 ·
   * `hero` `--hero-tint` + `--shadow-hero` · `inset` `--surface-3`, no border.
   */
  variant?: CardVariant;
  /** `section` when the card has its own heading, otherwise a plain `div`. */
  as?: 'div' | 'section' | 'article';
  padding?: 'sm' | 'md' | 'lg' | 'none';
  children: ReactNode;
}

/** Flat Night & Bloom card — no shadow except the hero variant. */
export function Card({ variant = 'default', as: Tag = 'div', padding = 'md', className, children, ...rest }: CardProps) {
  return (
    <Tag
      className={clsx('nb-card', variant !== 'default' && `is-${variant}`, padding !== 'none' && `pad-${padding}`, className)}
      {...rest}
    >
      {children}
    </Tag>
  );
}

/** Hero card: `--hero-tint` gradient + `--shadow-hero`; put a Lalezar number in it. */
export function HeroCard(props: Omit<CardProps, 'variant'>) {
  return <Card {...props} variant="hero" />;
}

interface SectionTitleProps {
  title: ReactNode;
  /** End link, e.g. «جزئیات» / «همه». */
  actionLabel?: ReactNode;
  onAction?: () => void;
  /** Heading level; 2 by default (the screen title is the h1). */
  level?: 2 | 3;
  id?: string;
  className?: string;
}

/** Section heading 15/800 with an optional end link 12.5/800 in `--brand-strong`. */
export function SectionTitle({ title, actionLabel, onAction, level = 2, id, className }: SectionTitleProps) {
  const H = level === 2 ? 'h2' : 'h3';
  return (
    <div className={clsx('nb-sect', className)}>
      <H id={id} className="nb-sect-title">
        {title}
      </H>
      {actionLabel && onAction ? (
        <button type="button" className="nb-sect-link" onClick={onAction}>
          {actionLabel}
        </button>
      ) : null}
    </div>
  );
}
