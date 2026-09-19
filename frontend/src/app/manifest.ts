import type { MetadataRoute } from 'next';

// Served at /manifest.webmanifest.
//
// - `id` stays "/" — changing it would make every existing install a different
//   app in the browser's eyes.
// - `start_url` is the unprefixed splash: "/" cost two redirects (/ → /fa →
//   /fa/splash) on every launch; "/splash" costs one, and the middleware still
//   picks the user's locale (a hard-coded "/fa/splash" would force fa on en
//   users). The manifest text is fa-only by design for M1 (dir/lang below too);
//   a per-locale manifest is future work.
// - `display: fullscreen` is a deliberate product choice (no browser chrome, no
//   status bar — the app should not read as a web page); `display_override`
//   spells out the fallback for platforms without fullscreen (iOS → standalone).
// - One light theme/background colour: the manifest has no per-scheme colours
//   and the app starts light, so the launch splash stays light on purpose.
// - Every icon is generated from one master (see public/icons); maskable icons
//   keep the round mark inside the 80% safe-zone circle on a white plate.
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: 'ریتمی — همراه سلامت زنان',
    short_name: 'ریتمی',
    description: 'پیگیری چرخه قاعدگی و بارداری',
    id: '/',
    start_url: '/splash',
    scope: '/',
    display: 'fullscreen',
    display_override: ['fullscreen', 'standalone'],
    orientation: 'portrait',
    dir: 'rtl',
    lang: 'fa',
    categories: ['health', 'medical', 'lifestyle'],
    background_color: '#F2ECFF',
    theme_color: '#F2ECFF',
    icons: [
      { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
      { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
      { src: '/icons/maskable-192.png', sizes: '192x192', type: 'image/png', purpose: 'maskable' },
      { src: '/icons/maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
    ],
    shortcuts: [
      {
        name: 'ثبت امروز',
        short_name: 'ثبت',
        url: '/fa/log',
        icons: [
          { src: '/icons/shortcut-log-96.png', sizes: '96x96', type: 'image/png' },
          { src: '/icons/shortcut-log-192.png', sizes: '192x192', type: 'image/png' },
        ],
      },
      {
        name: 'تقویم',
        short_name: 'تقویم',
        url: '/fa/calendar',
        icons: [
          { src: '/icons/shortcut-calendar-96.png', sizes: '96x96', type: 'image/png' },
          { src: '/icons/shortcut-calendar-192.png', sizes: '192x192', type: 'image/png' },
        ],
      },
    ],
    screenshots: [
      {
        src: '/screenshots/home-narrow.png',
        sizes: '1080x2340',
        type: 'image/png',
        form_factor: 'narrow',
        label: 'صفحه اصلی ریتمی',
      },
      {
        src: '/screenshots/calendar-narrow.png',
        sizes: '1080x2340',
        type: 'image/png',
        form_factor: 'narrow',
        label: 'تقویم چرخه',
      },
      {
        src: '/screenshots/home-wide.png',
        sizes: '1280x800',
        type: 'image/png',
        form_factor: 'wide',
        label: 'ریتمی روی صفحه بزرگ',
      },
    ],
  };
}
