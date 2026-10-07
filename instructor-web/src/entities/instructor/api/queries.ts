'use client';

import { useQuery } from '@tanstack/react-query';

import { fetchMe, instructorKeys } from './instructor-api';

export function useInstructorMe(enabled = true) {
  return useQuery({
    queryKey: instructorKeys.me(),
    queryFn: ({ signal }) => fetchMe(signal),
    enabled,
  });
}
