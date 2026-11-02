"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";

import { api } from "@/shared/api";

/** Saved segments (capability `segments`, admin-api.md §6c). Replaces the old browser-local list. */

const segmentSchema = z.object({
  id: z.number(),
  name: z.string(),
  filters: z.record(z.string(), z.string()),
  valid: z.boolean().optional(),
  created_at: z.string().nullable().optional(),
});
export type Segment = z.infer<typeof segmentSchema>;

const listSchema = z.object({
  items: z.array(segmentSchema),
  options: z.record(z.string(), z.array(z.string())).optional(),
});

const sizeSchema = segmentSchema.extend({
  size: z.number().nullable(),
  size_suppressed: z.boolean(),
});

const key = ["users", "segments"] as const;

export function useSegments(enabled = true) {
  return useQuery({
    queryKey: key,
    queryFn: ({ signal }) =>
      api.get("/segments", { schema: listSchema, signal }),
    enabled,
  });
}

/** Today's size; null (suppressed) below 10 users. */
export function useSegmentSize(id: number | null) {
  return useQuery({
    queryKey: [...key, id, "size"],
    queryFn: ({ signal }) =>
      api.get(`/segments/${id}`, { schema: sizeSchema, signal }),
    enabled: id !== null,
  });
}

export function useSaveSegment() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      name,
      filters,
    }: {
      id?: number;
      name: string;
      filters: Partial<Record<string, string>>;
    }) =>
      id
        ? api.put(`/segments/${id}`, { name, filters })
        : api.post("/segments", { name, filters }),
    onSuccess: () => client.invalidateQueries({ queryKey: key }),
  });
}

export function useDeleteSegment() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.delete(`/segments/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: key }),
  });
}
