/** Weeks split into the three trimesters (1–13, 14–27, 28–40) plus any stored week above 40. */
export interface Band<T> {
  key: 'first' | 'second' | 'third' | 'beyond';
  from: number;
  to: number;
  weeks: T[];
}

export function byTrimester<T extends { week: number }>(weeks: readonly T[]): Band<T>[] {
  const bands: Band<T>[] = [
    { key: 'first', from: 1, to: 13, weeks: [] },
    { key: 'second', from: 14, to: 27, weeks: [] },
    { key: 'third', from: 28, to: 40, weeks: [] },
    { key: 'beyond', from: 41, to: 42, weeks: [] },
  ];
  for (const w of [...weeks].sort((a, b) => a.week - b.week)) {
    const band = bands.find((b) => w.week >= b.from && w.week <= b.to) ?? bands[3]!;
    band.weeks.push(w);
  }
  return bands.filter((b) => b.weeks.length > 0);
}
