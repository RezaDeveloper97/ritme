/** The child the feeding screen opens without an id: the first own child (the API lists own children youngest first), else the first shared one. */
export function defaultChildId(children: ReadonlyArray<{ id: number; role: string }>): number | null {
  return (children.find((c) => c.role === 'owner') ?? children[0])?.id ?? null;
}
