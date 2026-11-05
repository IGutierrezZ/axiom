// Service worker mínimo: existe solo para que Edge y Chrome ofrezcan instalar
// el dashboard como aplicación. No guarda nada en caché y no intercepta
// ninguna petición: el dashboard sirve datos vivos del proyecto y una copia
// obsoleta sería peor que un fallo de red.
//
// El manejador de fetch está vacío a propósito (sin respondWith): el
// navegador sigue el camino normal de red, pero el service worker cuenta
// como manejador de peticiones para los criterios de instalabilidad.

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', event => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('fetch', () => {
  // Solo red: no se llama a event.respondWith().
});
