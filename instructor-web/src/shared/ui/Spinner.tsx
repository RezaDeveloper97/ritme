import { cn } from '@/shared/lib';

/** Ring spinner in currentColor; `sm` (20px) sits inside buttons. */
export function Spinner({ size = 'md', label }: { size?: 'sm' | 'md'; label?: string }) {
  return (
    <span
      className={cn('spinner', size === 'sm' && 'spinner--sm')}
      role={label ? 'status' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
    />
  );
}
