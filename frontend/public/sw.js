// Kill-switch service worker.
//
// This project does NOT use a service worker. A stale worker from a previous
// app that ran on this port kept throwing:
//   "Failed to execute 'put' on 'Cache': Request scheme 'chrome-extension' is unsupported"
//
// The browser periodically re-fetches /sw.js for any registered scope. Serving
// this file makes it replace the old worker with one that unregisters itself
// and wipes its caches, so the error stops for good.

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys();
      await Promise.all(keys.map((key) => caches.delete(key)));
      await self.registration.unregister();
      const clientList = await self.clients.matchAll({ type: 'window' });
      clientList.forEach((client) => client.navigate(client.url));
    })(),
  );
});
