// Página sin conexión: sondea /api/health y recarga en cuanto el servidor responde.
// Script clásico y externo (la CSP no admite script en línea).
(function () {
  var DEFAULT_PORT = '47700';
  var POLL_MS = 3000;

  // La aplicación instalada queda ligada a su puerto: si no es el fijo, la orden
  // tiene que pedir ese mismo puerto.
  document.getElementById('offline-origin').textContent = window.location.origin;
  if (window.location.port && window.location.port !== DEFAULT_PORT) {
    var command = document.querySelector('[data-command]');
    command.textContent = 'axiom ui --detach --port ' + window.location.port;
  }

  var status = document.getElementById('offline-status');
  var checking = false;

  function check() {
    if (checking) return;
    checking = true;
    fetch('/api/health', { cache: 'no-store' })
      .then(function (response) {
        if (!response.ok) throw new Error('estado ' + response.status);
        status.textContent = 'El servidor ya responde. Recargando…';
        window.location.reload();
      })
      .catch(function () {
        checking = false;
      });
  }

  document.getElementById('offline-retry').addEventListener('click', check);
  window.setInterval(check, POLL_MS);
})();
