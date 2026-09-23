'use client';

import { useTranslations } from 'next-intl';

import { Icon } from '@/shared/ui';

import { useLogout } from '../api/queries';

export function LogoutButton({ className = 'nav-link' }: { className?: string }) {
  const t = useTranslations('auth');
  const logout = useLogout();
  return (
    <button
      type="button"
      className={`${className} w-full cursor-pointer border-0 bg-transparent text-start font-[inherit] text-danger-deep`}
      onClick={() => logout.mutate()}
      disabled={logout.isPending}
    >
      <Icon name="logout" className="rtl:-scale-x-100" />
      {t('logout')}
    </button>
  );
}
