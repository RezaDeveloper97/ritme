'use client';

/**
 * Title row shared by the analysis cards. Logical alignment so it flips
 * correctly in both directions (§12).
 */
export function SectionHead({ title }: { title: string }) {
  return (
    <div className="sec-head-row">
      <span className="sec-head-t">{title}</span>
      <span />
    </div>
  );
}
