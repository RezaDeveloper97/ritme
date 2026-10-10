'use client';

import { useQuery } from '@tanstack/react-query';

import { courseKeys, fetchCourses, fetchDashboard, fetchOutline } from './course-api';

export function useDashboard() {
  return useQuery({ queryKey: courseKeys.dashboard(), queryFn: ({ signal }) => fetchDashboard(signal) });
}

export function useCourses() {
  return useQuery({ queryKey: courseKeys.list(), queryFn: ({ signal }) => fetchCourses(signal) });
}

export function useOutline(id: number | null) {
  return useQuery({
    queryKey: courseKeys.outline(id ?? 0),
    queryFn: ({ signal }) => fetchOutline(id ?? 0, signal),
    enabled: id !== null && id > 0,
  });
}
