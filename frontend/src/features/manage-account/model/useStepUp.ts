'use client';

import { useCallback, useRef, useState } from 'react';

/** What the step-up sheet needs from its owner (spread onto `<StepUpSheet>`). */
export interface StepUpController {
  open: boolean;
  onClose: () => void;
  onVerified: (token: string) => void;
}

/**
 * Holds one pending account action until the user has confirmed a fresh SMS
 * code (T-M12-06). `request(run)` opens the sheet; a verified code hands its
 * single-use token to `run` exactly once. The token is never stored.
 */
export function useStepUp(): { request: (run: (token: string) => void) => void; sheet: StepUpController } {
  const [open, setOpen] = useState(false);
  const pending = useRef<((token: string) => void) | null>(null);

  const request = useCallback((run: (token: string) => void) => {
    pending.current = run;
    setOpen(true);
  }, []);

  const onClose = useCallback(() => {
    pending.current = null;
    setOpen(false);
  }, []);

  const onVerified = useCallback((token: string) => {
    const run = pending.current;
    pending.current = null;
    setOpen(false);
    run?.(token);
  }, []);

  return { request, sheet: { open, onClose, onVerified } };
}
