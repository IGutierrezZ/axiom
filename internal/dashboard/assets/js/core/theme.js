// Tema claro / oscuro / automático. La preferencia se guarda por usuario en
// localStorage ('axiom.theme': light | dark | system, por defecto system).
// El script en línea de index.html aplica el tema antes de pintar para evitar
// el parpadeo; este módulo se encarga del selector y de seguir al sistema.

const STORAGE_KEY = 'axiom.theme';
const VALID = ['light', 'dark', 'system'];

let current = 'system';

// Toda lectura y escritura de localStorage va en try/catch: puede estar
// bloqueado (modo privado, políticas) y entonces se conserva solo en memoria.
export function readPreference() {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (VALID.includes(stored)) return stored;
  } catch (_) {
    // Sin almacenamiento: se usa el valor por defecto.
  }
  return 'system';
}

export function resolveTheme(pref) {
  if (pref === 'light' || pref === 'dark') return pref;
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  } catch (_) {
    return 'light';
  }
}

function apply(pref) {
  document.documentElement.dataset.theme = resolveTheme(pref);
}

function syncControls(root) {
  root.querySelectorAll('[data-theme-pref]').forEach(btn => {
    btn.setAttribute('aria-pressed', String(btn.getAttribute('data-theme-pref') === current));
  });
}

export function setPreference(pref, root = document) {
  if (!VALID.includes(pref)) return;
  current = pref;
  try {
    window.localStorage.setItem(STORAGE_KEY, pref);
  } catch (_) {
    // Sin almacenamiento: la preferencia vale solo para esta sesión.
  }
  apply(pref);
  syncControls(root);
}

// initTheme enlaza el selector segmentado y la escucha del sistema operativo.
export function initTheme(root = document) {
  current = readPreference();
  apply(current);
  syncControls(root);

  root.querySelectorAll('[data-theme-pref]').forEach(btn => {
    btn.addEventListener('click', () => setPreference(btn.getAttribute('data-theme-pref'), root));
  });

  // Con "Auto" el tema sigue al sistema en vivo.
  try {
    const mql = window.matchMedia('(prefers-color-scheme: dark)');
    const onChange = () => { if (current === 'system') apply('system'); };
    if (mql.addEventListener) mql.addEventListener('change', onChange);
    else if (mql.addListener) mql.addListener(onChange);
  } catch (_) {
    // matchMedia no disponible: el tema queda fijo.
  }
}
