import { FA_DECIMAL_SEPARATOR } from '../model/bbt';

/**
 * A formatted temperature (`formatBbt`) for display in Lalezar. Lalezar draws the
 * Persian decimal separator «٫» as a wide spaced comma («۳۶ , ۵۲»), so the
 * separator alone is set in the body font (audit #23); digits stay in the
 * caller's face.
 */
export function BbtNumber({ text, className }: { text: string; className?: string }) {
  const at = text.indexOf(FA_DECIMAL_SEPARATOR);
  if (at < 0) return <span className={className}>{text}</span>;
  return (
    <span className={className}>
      {text.slice(0, at)}
      <span className="font-sans">{FA_DECIMAL_SEPARATOR}</span>
      {text.slice(at + 1)}
    </span>
  );
}
