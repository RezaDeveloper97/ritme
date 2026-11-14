'use client';

import { useEffect, useState } from 'react';

import { type BillingChannel, getBillingChannel } from './billing';

/**
 * The billing channel of this runtime. `web` on the server and during
 * hydration (the bridge only exists in the browser); `ready` flips once the
 * real channel is known, so screens can hold their first paint until then.
 */
export function useBillingChannel(): { channel: BillingChannel; ready: boolean } {
  const [state, setState] = useState<{ channel: BillingChannel; ready: boolean }>({ channel: 'web', ready: false });
  useEffect(() => {
    setState({ channel: getBillingChannel(), ready: true });
  }, []);
  return state;
}
