import { Link } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { Icon } from '@/shared/ui';
import '@/shared/ui/styles/rmd.css';

/** A list section's title row with its «+ افزودن» link to the matching form. */
export function SectionHead({
  id,
  title,
  addHref,
  addText,
  addLabel,
  pilotKey,
}: {
  id: string;
  title: string;
  addHref: string;
  addText: string;
  /** The accessible name — «افزودن» alone doesn't say what is added. */
  addLabel: string;
  /** The stable code of what is added (`medication` / `appointment`) — the autopilot name. */
  pilotKey: string;
}) {
  return (
    <div className="rmd-sec-head">
      <h2 id={id} className="rmd-sec-title">
        {title}
      </h2>
      <Link href={addHref} className="rmd-sec-add" aria-label={addLabel} {...pilot(`care.add:${pilotKey}`)}>
        <Icon name="plus" size={14} strokeWidth={2.4} />
        {addText}
      </Link>
    </div>
  );
}
