/** Label of an enum value from an `{value, label}` list; the raw value when unknown. */
export function optionLabel(
  options: ReadonlyArray<{ value: string; label: string }> | undefined,
  value: string | null | undefined,
): string {
  if (!value) return '';
  return options?.find((o) => o.value === value)?.label ?? value;
}
