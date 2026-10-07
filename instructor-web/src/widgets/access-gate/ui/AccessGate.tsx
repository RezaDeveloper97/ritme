'use client';

import { useTranslations } from 'next-intl';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useState, type ReactNode } from 'react';

import {
  accessOf,
  pathForAccess,
  useInstructorMe,
  type Instructor,
  type InstructorAccess,
} from '@/entities/instructor';
import { isApiError } from '@/shared/api';
import { useErrorMessage } from '@/shared/i18n';
import { hasAuthToken } from '@/shared/session';
import { Button, ErrorState, Skeleton } from '@/shared/ui';

interface AccessGateProps {
  /** Access states this route renders; any other state is sent to its own route. */
  allow: readonly InstructorAccess[];
  children: (instructor: Instructor | null, access: InstructorAccess) => ReactNode;
}

/**
 * Client gate for every signed-in route: no token → /login; otherwise GET /me
 * decides between «درخواست» (/apply), «در انتظار تأیید» (/pending) and the panel.
 * The API remains the authority (403 instructor_* on every panel call).
 */
export function AccessGate({ allow, children }: AccessGateProps) {
  const router = useRouter();
  const pathname = usePathname();
  const t = useTranslations('common');
  const errorMessage = useErrorMessage();
  const [signedIn, setSignedIn] = useState<boolean | null>(null);

  useEffect(() => {
    const ok = hasAuthToken();
    setSignedIn(ok);
    if (!ok) {
      const next = pathname && pathname !== '/' ? `?next=${encodeURIComponent(pathname)}` : '';
      router.replace(`/login${next}`);
    }
  }, [router, pathname]);

  const me = useInstructorMe(signedIn === true);
  const access = me.data ? accessOf(me.data.instructor) : null;
  const allowed = access !== null && allow.includes(access);

  useEffect(() => {
    if (access !== null && !allow.includes(access)) router.replace(pathForAccess(access));
  }, [access, allow, router]);

  if (me.isError && !(isApiError(me.error) && me.error.status === 401)) {
    return (
      <div className="gate-center">
        <ErrorState
          title={t('loadFailed')}
          message={errorMessage(me.error)}
          action={
            <Button variant="ghost" onClick={() => me.refetch()} loading={me.isFetching}>
              {t('retry')}
            </Button>
          }
        />
      </div>
    );
  }

  if (!allowed || !me.data) return <GateSkeleton />;
  return <>{children(me.data.instructor, access)}</>;
}

function GateSkeleton() {
  const t = useTranslations('common');
  return (
    <div className="gate-skeleton" role="status" aria-label={t('loading')}>
      <div className="gate-skeleton__head">
        <Skeleton className="skeleton--avatar" />
        <div className="gate-skeleton__lines">
          <Skeleton className="skeleton--line-sm" />
          <Skeleton className="skeleton--line" />
        </div>
      </div>
      <Skeleton className="skeleton--card" />
      <Skeleton className="skeleton--card" />
    </div>
  );
}
