export type SaveStatus = "idle" | "saving" | "saved" | "queued" | "error";

/**
 * The status once the outbox has replayed: a day that was only queued becomes
 * «saved» when *its* entry reached the server. Any other status (a newer save,
 * an error, a reset) is left alone.
 */
export function statusAfterSync(
  status: SaveStatus,
  queuedKey: string | null,
  sentKeys: readonly string[],
): SaveStatus {
  return status === "queued" && queuedKey !== null && sentKeys.includes(queuedKey)
    ? "saved"
    : status;
}

/** Only a save the server has confirmed shows the green «ذخیره شد» button. */
export function isConfirmedSave(status: SaveStatus): boolean {
  return status === "saved";
}
