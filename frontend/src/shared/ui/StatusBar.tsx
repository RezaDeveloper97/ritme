interface StatusBarProps {
  tone?: 'dark' | 'light';
}

/** Decorative iOS-style status bar. Tone 'light' is used on dark/gradient backgrounds. */
export function StatusBar({ tone = 'dark' }: StatusBarProps) {
  const c = tone === 'light' ? '#fff' : '#11202F';
  return (
    <div className="statusbar" style={{ color: c }}>
      <span className="time">۹:۴۱</span>
      <span className="si">
        {/* signal bars */}
        <svg width="18" height="12" viewBox="0 0 18 12" fill={c}>
          <rect x="0" y="7" width="3" height="5" rx="1" />
          <rect x="5" y="4.5" width="3" height="7.5" rx="1" />
          <rect x="10" y="2" width="3" height="10" rx="1" />
          <rect x="15" y="0" width="3" height="12" rx="1" />
        </svg>
        {/* wifi */}
        <svg width="17" height="12" viewBox="0 0 17 12" fill="none" stroke={c} strokeWidth="1.6" strokeLinecap="round">
          <path d="M1 4.2A11 11 0 0 1 16 4.2" />
          <path d="M3.5 6.6A7.4 7.4 0 0 1 13.5 6.6" />
          <path d="M6 9A3.8 3.8 0 0 1 11 9" />
          <circle cx="8.5" cy="11" r=".6" fill={c} stroke="none" />
        </svg>
        {/* battery */}
        <svg width="26" height="13" viewBox="0 0 26 13" fill="none">
          <rect x="1" y="1" width="21" height="11" rx="3" stroke={c} strokeOpacity=".5" />
          <rect x="3" y="3" width="16" height="7" rx="1.5" fill={c} />
          <rect x="23" y="4" width="2" height="5" rx="1" fill={c} fillOpacity=".5" />
        </svg>
      </span>
    </div>
  );
}
