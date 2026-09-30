'use client';

import { useId, useState, type ReactNode } from 'react';
import { clsx } from 'clsx';

import { Icon, type IconName } from '../Icon';
import { IconCircle } from './IconCircle';
import { toneClass, type Tone } from './tone';

interface AccordionProps {
  title: ReactNode;
  /** One line under the title — «ثبت نشده» or what was logged. */
  summary?: ReactNode;
  icon?: IconName;
  tone?: Tone;
  /** Controlled open state; omit for uncontrolled with `defaultOpen`. */
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** Tints the summary and border in the tone (a category with data). */
  active?: boolean;
  className?: string;
  children: ReactNode;
}

/** Card whose header row toggles a region (`aria-expanded` + `aria-controls`). */
export function Accordion({
  title,
  summary,
  icon,
  tone = 'brand',
  open,
  defaultOpen = false,
  onOpenChange,
  active,
  className,
  children,
}: AccordionProps) {
  const [inner, setInner] = useState(defaultOpen);
  const isOpen = open ?? inner;
  const id = useId();
  const toggle = () => {
    setInner(!isOpen);
    onOpenChange?.(!isOpen);
  };
  return (
    <section className={clsx('nb-card', 'nb-acc', toneClass(tone), active && 'is-active', className)}>
      <h3 className="nb-acc-h">
        <button
          type="button"
          id={`${id}-btn`}
          className="nb-acc-btn"
          aria-expanded={isOpen}
          aria-controls={`${id}-panel`}
          onClick={toggle}
        >
          {icon ? <IconCircle icon={icon} tone={tone} size="sm" /> : null}
          <span className="nb-acc-text">
            <span className="nb-acc-title">{title}</span>
            {summary ? <span className="nb-acc-summary">{summary}</span> : null}
          </span>
          <Icon name="chevronDown" size={18} className="nb-acc-chev" />
        </button>
      </h3>
      <div id={`${id}-panel`} role="region" aria-labelledby={`${id}-btn`} className="nb-acc-panel" hidden={!isOpen}>
        {children}
      </div>
    </section>
  );
}
