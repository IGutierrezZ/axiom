// Service worker de la aplicación instalada. Existe para que Edge y Chrome
// ofrezcan instalar el dashboard y para una sola cosa más: si el servidor local
// no responde a una navegación, muestra /offline.html, que explica cómo
// arrancarlo. La aplicación instalada no puede lanzar procesos, así que sin esta
// página solo quedaría el error genérico del navegador.
//
// Todo lo demás va a la red: el dashboard sirve datos vivos del proyecto y una
// copia obsoleta sería peor que un fallo de red. Solo se guardan, al instalar, la
// página sin conexión y sus recursos; también ellos se piden primero a la red y
// nunca se guardan respuestas en tiempo de ejecución.
//
// Al cambiar la página sin conexión, sube la versión de OFFLINE_CACHE: el
// navegador solo reinstala el service worker cuando cambia este fichero.

const OFFLINE_CACHE = 'axiom-offline-v1';
const OFFLINE_URL = '/offline.html';
const OFFLINE_ASSETS = [
  '/offline.html',
  '/offline.css',
  '/offline.js',
  '/tokens.css',
  '/js/theme-init.js',
  '/icons/icon.svg',
];

self.addEventListener('install', event => {
  self.skipWaiting();
  event.waitUntil(
    caches.open(OFFLINE_CACHE)
      .then(cache => cache.addAll(OFFLINE_ASSETS.map(path => new Request(path, { cache: 'reload' }))))
      // Sin la página sin conexión el dashboard funciona igual: no se bloquea la instalación.
      .catch(() => {})
  );
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys()
      .then(keys => Promise.all(keys.filter(key => key !== OFFLINE_CACHE).map(key => caches.delete(key))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', event => {
  const request = event.request;
  if (request.method !== 'GET') return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  if (request.mode === 'navigate') {
    event.respondWith(fetch(request).catch(() => fromCache(OFFLINE_URL)));
    return;
  }
  if (OFFLINE_ASSETS.includes(url.pathname)) {
    event.respondWith(fetch(request).catch(() => fromCache(url.pathname)));
  }
  // El resto, API incluida, sigue el camino normal de red: sin respondWith.
});

function fromCache(path) {
  return caches.match(path).then(response => response || Response.error());
}
