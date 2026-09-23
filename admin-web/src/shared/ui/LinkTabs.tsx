import Link from 'next/link';

/** Section tabs that are links (info-section groups, translation namespaces). */
export function LinkTabs({
  label,
  items,
}: {
  label: string;
  items: ReadonlyArray<{ key: string; href: string; label: string; active: boolean }>;
}) {
  return (
    <nav className="tabs" aria-label={label}>
      {items.map((item) => (
        <Link
          key={item.key}
          href={item.href}
          className="tab"
          aria-current={item.active ? 'page' : undefined}
          scroll={false}
        >
          {item.label}
        </Link>
      ))}
    </nav>
  );
}
