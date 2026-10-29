// Formatos compartidos de las pantallas de conocimiento. Solo se muestran fechas
// reales: un valor vacío o la fecha cero de Go (0001-01-01) se devuelve como ''.

// parseDate devuelve un Date válido o null.
export function parseDate(iso) {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 2000) return null;
  return d;
}

// formatDay: «9 oct 2026».
export function formatDay(iso) {
  const d = parseDate(iso);
  if (!d) return '';
  try {
    return d.toLocaleDateString('es-ES', { day: 'numeric', month: 'short', year: 'numeric' });
  } catch (_) {
    return d.toISOString().slice(0, 10);
  }
}

// formatDateTime: «9 oct 2026, 7:58».
export function formatDateTime(iso) {
  const d = parseDate(iso);
  if (!d) return '';
  try {
    return d.toLocaleString('es-ES', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
  } catch (_) {
    return d.toISOString().replace('T', ' ').slice(0, 16);
  }
}

// formatNumber agrupa los miles según la configuración regional española.
export function formatNumber(n) {
  const v = Number(n) || 0;
  try {
    return v.toLocaleString('es-ES');
  } catch (_) {
    return String(v);
  }
}

// plural: «1 skill» / «3 skills».
export function plural(n, one, many) {
  return `${formatNumber(n)} ${Number(n) === 1 ? one : many}`;
}

// shortHash recorta una huella SHA-256 para mostrarla; el valor completo va en title.
export function shortHash(hash, len = 12) {
  const h = String(hash || '');
  return h.length > len ? `${h.slice(0, len)}…` : h;
}
