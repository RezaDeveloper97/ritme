import { Suspense } from 'react';

import { LoginFlow } from '@/features/otp-login';
import { ThemeToggle } from '@/shared/ui';

/** /login — OTP sign-in (nbl_Onb_Phone / nbl_Onb_OTP styling, instructor copy). */
export function LoginScreen() {
  return (
    <div className="auth-page">
      <div className="auth-page__glow" aria-hidden="true" />
      <div className="auth-page__theme">
        <ThemeToggle />
      </div>
      <Suspense>
        <LoginFlow />
      </Suspense>
    </div>
  );
}
