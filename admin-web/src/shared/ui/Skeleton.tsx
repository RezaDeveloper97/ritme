import { cn } from '@/shared/lib';

export function Skeleton({ className }: { className?: string }) {
  return <span className={cn('skeleton h-4', className)} aria-hidden="true" />;
}
