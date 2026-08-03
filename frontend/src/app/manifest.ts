import type { MetadataRoute } from 'next';

// Served at /manifest.webmanifest. Persian is the product's default locale;
// start_url is "/" so the middleware redirects to the user's locale.
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: 'ریتمی — همراه سلامت زنان',
    short_name: 'ریتمی',
    description: 'پیگیری چرخه قاعدگی و بارداری',
    id: '/',
    start_url: '/',
    scope: '/',
    display: 'standalone',
    orientation: 'portrait',
    dir: 'rtl',
    lang: 'fa',
    background_color: '#F2ECFF',
    theme_color: '#F2ECFF',
    icons: [
      { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
      { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
      { src: '/icons/maskable-192.png', sizes: '192x192', type: 'image/png', purpose: 'maskable' },
      { src: '/icons/maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
    ],
  };
}
