'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import type { GrantStatus } from '../model/schema';
import { fetchGroup, fetchGroups, fetchStudents, revokeGrant, studentKeys } from './student-api';

export const useStudents = (status: GrantStatus | 'all') =>
  useQuery({ queryKey: studentKeys.list(status), queryFn: ({ signal }) => fetchStudents(status, signal) });
export const useGroups = () => useQuery({ queryKey: studentKeys.groups(), queryFn: ({ signal }) => fetchGroups(signal) });
export const useGroup = (id: number) =>
  useQuery({ queryKey: studentKeys.group(id), queryFn: ({ signal }) => fetchGroup(id, signal) });

/** Revoke one grant; every student/group/course-access view refetches. */
export function useRevokeGrant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => revokeGrant(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: studentKeys.all });
      void qc.invalidateQueries({ queryKey: ['courses'] });
    },
  });
}
