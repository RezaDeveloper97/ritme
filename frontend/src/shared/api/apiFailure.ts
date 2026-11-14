/**
 * A tiny fan-out for "some request failed" so a cross-cutting feature (the AI gate) can react to refusals that
 * surfaces forget to handle, without `shared` importing it. The listener is registered by the shell-mounted
 * feature; callers (the QueryClient caches, the SSE stream that bypasses `apiClient`) just report.
 */
type Listener = (error: unknown) => void;

let listener: Listener | null = null;

/** Registers the one listener; returns the cleanup. */
export function setApiFailureListener(next: Listener): () => void {
  listener = next;
  return () => {
    if (listener === next) listener = null;
  };
}

export function reportApiFailure(error: unknown): void {
  try {
    listener?.(error);
  } catch {
    /* a listener must never break the request path */
  }
}
