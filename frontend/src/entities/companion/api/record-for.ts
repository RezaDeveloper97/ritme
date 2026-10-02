'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';

import { initialRecordTarget, type RecordSection, type RecordTarget, recordTargets } from '../lib/record-for';
import { companionSideKeys, useCompanionLinks } from './companion-side';

export interface RecordForState {
  /** Owners who granted edit on the section (empty = no picker). */
  targets: RecordTarget[];
  /** The picker shows only with at least one target (and never on an owner's record). */
  showPicker: boolean;
  /** null = the viewer's own record. */
  target: number | null;
  setTarget: (target: number | null) => void;
  /** The chosen owner, or null for the viewer. */
  chosen: RecordTarget | null;
  /** False once the links say the viewer may only see the fixed owner's section (save would be a 403). */
  canEdit: boolean;
  /** After a delegated save: the companion home's shared meds / appointments are stale. */
  onSaved: () => void;
  /** A delegated write got 403 (grant revoked meanwhile): back to «خودم» and refetch the links. */
  onRevoked: () => void;
}

/**
 * «ثبت برای چه کسی؟» state of a medication / appointment form (B-N4-06).
 * `forUserId` is the `?for=` owner: preselected on a new record while the
 * viewer may still record for her. `fixedOwner` = editing an owner's record —
 * the target is that owner and there is no picker.
 */
export function useRecordFor(section: RecordSection, forUserId: number | null, fixedOwner = false): RecordForState {
  const queryClient = useQueryClient();
  const links = useCompanionLinks();
  const targets = recordTargets(links.data, section);
  const [target, setTargetState] = useState<number | null>(fixedOwner ? forUserId : null);
  const touched = useRef(false);

  // The links arrive after the first render: apply the `?for=` preselection once.
  const ready = links.data !== undefined;
  useEffect(() => {
    if (fixedOwner || touched.current || !ready) return;
    setTargetState(initialRecordTarget(targets, forUserId));
    // `targets` derives from the links; re-run only when they (or the param) change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready, links.data, forUserId, fixedOwner]);

  const setTarget = (next: number | null) => {
    touched.current = true;
    setTargetState(next);
  };

  const chosen = targets.find((t) => t.ownerId === target) ?? null;
  return {
    targets,
    showPicker: !fixedOwner && targets.length > 0,
    target,
    setTarget,
    chosen: fixedOwner && !chosen && forUserId !== null ? { ownerId: forUserId, name: ownerName(links.data, forUserId), linkId: 0 } : chosen,
    canEdit: !fixedOwner || !ready || targets.some((t) => t.ownerId === forUserId),
    onSaved: () => {
      void queryClient.invalidateQueries({ queryKey: companionSideKeys.home() });
    },
    onRevoked: () => {
      if (!fixedOwner) setTarget(null);
      void queryClient.invalidateQueries({ queryKey: companionSideKeys.all });
    },
  };
}

function ownerName(links: ReturnType<typeof useCompanionLinks>['data'], ownerId: number): string | null {
  return links?.find((l) => l.owner.id === ownerId)?.owner.name ?? null;
}
