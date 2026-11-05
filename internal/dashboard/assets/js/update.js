// Aviso de versión nueva de Axiom (P2): comprueba la última versión publicada al
// cargar el dashboard y, solo si hay una más reciente, muestra el aviso con
// «Actualizar», el enlace a la release y «Comprobar de nuevo».
//
// El servidor guarda la comprobación en memoria unas horas (GET /api/system/version);
// «Comprobar de nuevo» la omite con ?refresh=1. Sin aviso si está al día, si es una
// compilación de desarrollo o si la comprobación falla: nada de eso interrumpe.

import { api, escapeHtml } from './core/api.js';
import { $, button, toast, setBusy } from './core/components.js';
import { icon } from './core/icons.js';
import { state, set, on } from './core/state.js';

state.version = null; // GET /api/system/version

// Solo se enlazan releases https: la URL viene de la API, no se inserta a ciegas.
function safeReleaseUrl(url) {
  return typeof url === 'string' && /^https:\/\//i.test(url) ? url : '';
}

// renderUpdateBanner pinta el aviso según state.version.
export function renderUpdateBanner() {
  const banner = $('#update-banner');
  if (!banner) return;
  const info = state.version;
  if (!info || info.status !== 'update_available' || !info.update_available) {
    banner.hidden = true;
    banner.innerHTML = '';
    return;
  }
  const release = safeReleaseUrl(info.release_url);
  const link = release ? `<a class="link-btn" href="${escapeHtml(release)}" target="_blank" rel="noopener noreferrer">Ver la release</a>` : '';
  banner.hidden = false;
  banner.innerHTML = `${icon('download', 18)}<p class="grow">Hay una versión nueva de Axiom: <span class="mono">${escapeHtml(info.current)}</span> → <strong class="mono">${escapeHtml(info.latest)}</strong>. ${link}</p>`
    + button({ label: 'Actualizar', iconName: 'download', variant: 'primary', size: 'sm', attrs: 'data-action="update-now"' })
    + button({ label: 'Comprobar de nuevo', iconName: 'refresh', size: 'sm', attrs: 'data-action="update-recheck"' });
}

// checkVersion consulta la versión. Un fallo de red o de la API se queda en silencio:
// el aviso es una ayuda, no una función de la que dependa el resto del dashboard.
export async function checkVersion({ refresh = false } = {}) {
  try {
    const info = await api(refresh ? '/api/system/version?refresh=1' : '/api/system/version');
    set('version', info);
    return info;
  } catch (err) {
    console.debug('No se pudo comprobar la versión:', err);
    return null;
  }
}

async function recheck(btn) {
  const restore = setBusy(btn, 'Comprobando…');
  const info = await checkVersion({ refresh: true });
  restore();
  if (info && info.status === 'update_available') return;
  if (info && info.status === 'up_to_date') toast('Axiom ya está en la última versión.', 'ok');
  else if (info && info.status === 'dev_build') toast('Esta es una compilación de desarrollo: no se compara con las releases.', 'warn');
  else toast('No se pudo comprobar la versión nueva. Inténtalo más tarde.', 'bad');
}

async function startUpdate(btn) {
  const restore = setBusy(btn, 'Actualizando…');
  try {
    await api('/api/system/update', { method: 'POST' });
  } catch (err) {
    toast(`No se pudo iniciar la actualización: ${err.message}`, 'bad');
  } finally {
    restore();
  }
}

export function initUpdate() {
  on('version', renderUpdateBanner);
  const banner = $('#update-banner');
  if (banner) {
    banner.addEventListener('click', (event) => {
      const target = event.target.closest('button[data-action]');
      if (!target) return;
      if (target.dataset.action === 'update-recheck') recheck(target);
      if (target.dataset.action === 'update-now') startUpdate(target);
    });
  }
  checkVersion();
}
