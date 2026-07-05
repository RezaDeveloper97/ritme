interface HomeIndicatorProps {
  light?: boolean;
}

/** iOS-style home swipe indicator pill. */
export function HomeIndicator({ light }: HomeIndicatorProps) {
  return (
    <div className="home-indicator">
      <div className={`home-indicator-bar${light ? ' light' : ''}`} />
    </div>
  );
}
