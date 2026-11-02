"use client";

import { useTranslations } from "next-intl";
import { useState } from "react";

import { useCan } from "@/features/auth";
import {
  Badge,
  Button,
  confirm,
  ErrorState,
  Panel,
  Skeleton,
  TextInput,
  toast,
  useNotifyError,
} from "@/shared/ui";
import { formatNumber } from "@/shared/lib";

import type { SegmentFilters } from "../model/segment-filters";
import {
  useDeleteSegment,
  useSaveSegment,
  useSegments,
  useSegmentSize,
  type Segment,
} from "../api/segments";

function SegmentSize({ id }: { id: number }) {
  const t = useTranslations("users");
  const size = useSegmentSize(id);
  if (!size.data) return null;
  return (
    <Badge tone="data">
      {size.data.size_suppressed
        ? t("segmentSizeHidden")
        : t("segmentSize", { count: formatNumber(size.data.size ?? 0, "en") })}
    </Badge>
  );
}

/** Saved segments: list, apply, save the current filters as new / over an existing one, rename, delete (capability `segments`). */
export function SegmentsPanel({
  current,
  activeId,
  canSave,
  onApply,
}: {
  /** The list's active filters, already reduced to the segment vocabulary. */
  current: SegmentFilters;
  activeId: number | null;
  canSave: boolean;
  onApply: (segment: Segment) => void;
}) {
  const t = useTranslations("users");
  const manage = useCan("segments", "manage");
  const view = useCan("segments");
  const notifyError = useNotifyError();
  const query = useSegments(view);
  const save = useSaveSegment();
  const del = useDeleteSegment();
  const [name, setName] = useState("");
  const [editing, setEditing] = useState<Segment | null>(null);

  if (!view) return null;

  const filterText = (f: Record<string, string>) =>
    Object.entries(f)
      .map(
        ([k, v]) =>
          `${t.has(`filterNames.${k}` as "filterNames.plan") ? t(`filterNames.${k}` as "filterNames.plan") : k}: ${v}`,
      )
      .join(" · ") || t("segmentAll");

  const submit = () => {
    const trimmed = name.trim();
    if (!trimmed) return;
    save.mutate(
      { id: editing?.id, name: trimmed, filters: current },
      {
        onSuccess: () => {
          toast.success(t("segmentSaved"));
          setName("");
          setEditing(null);
        },
        onError: notifyError,
      },
    );
  };

  return (
    <Panel title={t("segments")}>
      {query.isPending ? (
        <Skeleton className="h-16 w-full" />
      ) : query.error ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : (
        <div className="flex flex-col gap-3">
          {query.data.items.length === 0 ? (
            <p className="m-0 text-ink-3">{t("segmentsEmpty")}</p>
          ) : (
            <ul className="m-0 flex list-none flex-col gap-2 p-0">
              {query.data.items.map((s) => (
                <li
                  key={s.id}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-line px-3 py-2"
                >
                  <div className="flex min-w-0 flex-col gap-1">
                    <span className="font-medium">
                      {s.name}{" "}
                      {activeId === s.id && (
                        <Badge tone="brand">{t("segmentActive")}</Badge>
                      )}
                    </span>
                    <span className="text-[13px] text-ink-3">
                      {filterText(s.filters)}
                    </span>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <SegmentSize id={s.id} />
                    <Button size="sm" onClick={() => onApply(s)}>
                      {t("segmentApply")}
                    </Button>
                    {manage && (
                      <>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setEditing(s);
                            setName(s.name);
                          }}
                        >
                          {t("segmentEdit")}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          loading={del.isPending && del.variables === s.id}
                          onClick={async () => {
                            if (
                              !(await confirm({
                                message: t("segmentConfirmDelete", {
                                  name: s.name,
                                }),
                              }))
                            )
                              return;
                            del.mutate(s.id, {
                              onSuccess: () =>
                                toast.success(t("segmentDeleted")),
                              onError: notifyError,
                            });
                          }}
                        >
                          {t("segmentDelete")}
                        </Button>
                      </>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}
          {manage && (
            <form
              className="flex flex-wrap items-end gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                submit();
              }}
            >
              <TextInput
                label={
                  editing
                    ? t("segmentEditing", { name: editing.name })
                    : t("segmentName")
                }
                value={name}
                maxLength={80}
                onChange={(e) => setName(e.target.value)}
                className="w-64"
              />
              <Button
                type="submit"
                variant="primary"
                size="sm"
                loading={save.isPending}
                disabled={!name.trim() || !canSave}
              >
                {editing ? t("segmentUpdate") : t("saveSegment")}
              </Button>
              {editing && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setEditing(null);
                    setName("");
                  }}
                >
                  {t("segmentCancel")}
                </Button>
              )}
              <span className="basis-full text-[13px] text-ink-3">
                {canSave ? t("segmentSaveHint") : t("segmentNeedFilters")}
              </span>
            </form>
          )}
        </div>
      )}
    </Panel>
  );
}
