// Anti-parpadeo: fija el tema antes de pintar. Lee axiom.theme (light|dark|system,
// por defecto system) y resuelve system con matchMedia. Todo en try/catch.
// Va como script externo para cumplir la CSP (sin script en línea).
(function () {
  var pref = 'system';
  try {
    var stored = window.localStorage.getItem('axiom.theme');
    if (stored === 'light' || stored === 'dark' || stored === 'system') pref = stored;
  } catch (e) { /* sin almacenamiento: system */ }
  var theme = pref;
  if (pref === 'system') {
    try { theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'; }
    catch (e) { theme = 'light'; }
  }
  document.documentElement.dataset.theme = theme;
})();
