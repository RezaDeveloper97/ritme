import { Card, EmptyState, PrimaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';
import '@/shared/ui/styles/csv.css';

/** Loading skeleton shared by the city-services screens. */
export function CsSkeleton({ label }: { label: string }) {
  return (
    <SkeletonGroup label={label} className="csv-list">
      <Skeleton shape="block" />
      <Skeleton shape="block" />
      <Skeleton shape="block" />
    </SkeletonGroup>
  );
}

/** Error card with a retry button shared by the city-services screens. */
export function CsError({ title, retry, onRetry, busy }: { title: string; retry: string; onRetry: () => void; busy: boolean }) {
  return (
    <Card>
      <EmptyState
        icon="warning"
        title={title}
        action={
          <PrimaryButton onClick={onRetry} loading={busy}>
            {retry}
          </PrimaryButton>
        }
      />
    </Card>
  );
}
