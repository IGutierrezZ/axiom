// Aviso de versión nueva de Axiom (P2): comprueba la última versión publicada al
// cargar el dashboard y, solo si hay una más reciente, muestra el aviso con
// «Actualizar», el enlace a la release y «Comprobar de nuevo».
//
// El servidor guarda la comprobación en memoria unas horas (GET /api/system/version);
// «Comprobar de nuevo» la omite con ?refresh=1. Sin aviso si está al día, si es una
// compilación de desarrollo o si la comprobación falla: nada de eso interrumpe.
//
// «Actualizar» lanza un trabajo en segundo plano (POST /api/system/update) y la UI
// consulta su estado cada ~1,5 s (GET /api/system/update/status): así la petición no
// depende del tiempo de espera del servidor. Cada fase se muestra en el aviso; si
// falla, se ve el motivo y el comando manual.

import { api, escapeHtml } from './core/api.js';
import { $, button, toast, setBusy } from './core/components.js';
import { icon } from './core/icons.js';
import { state, set, on } from './core/state.js';

state.version = null;   // GET /api/system/version
state.updateJob = null; // GET /api/system/update/status

const POLL_MS = 1500;
// Tolerancia a fallos de red al consultar el estado: mientras el dashboard se
// reinicia el servidor deja de responder unos segundos.
const MAX_POLL_ERRORS = 60;
const RUNNING_PHASES = new Set(['checking', 'upgrading', 'restarting', 'syncing']);
const PHASE_LABEL = {
  checking: 'Comprobando',
  upgrading: 'Actualizando',
  restarting: 'Reiniciando',
  syncing: 'Sincronizando'
};

let pollTimer = null;
let pollErrors = 0;

// Solo se enlazan releases https: la URL viene de la API, no se inserta a ciegas.
function safeReleaseUrl(url) {
  return typeof url === 'string' && /^https:\/\//i.test(url) ? url : '';
}

function dismissKey(job) {
  return `${job.phase}|${job.message}`;
}

// El aviso de un trabajo terminado se puede cerrar; se recuerda en la sesión para
// que no reaparezca al recargar. El acceso al almacenamiento va protegido.
function isDismissed(job) {
  try {
    return window.sessionStorage.getItem('axiom.update.dismissed') === dismissKey(job);
  } catch (err) {
    return false;
  }
}

function rememberDismissed(job) {
  try {
    window.sessionStorage.setItem('axiom.update.dismissed', dismissKey(job));
  } catch (err) {
    // sin almacenamiento: el aviso reaparecerá al recargar, sin más consecuencias
  }
}

// syncSummary resume la sincronización del workspace que acompaña a la actualización;
// la salida del comando va plegada. Si no se ejecutó (omitida) no se dice nada.
function syncSummary(sync) {
  if (!sync || !sync.executed) return '';
  const lines = Array.isArray(sync.output) ? sync.output.filter(Boolean) : [];
  const detail = lines.length
    ? `<details><summary>Ver el detalle de la sincronización</summary><pre class="mono">${escapeHtml(lines.join('\n'))}</pre></details>`
    : '';
  if (sync.success) return `<p>Sincronización del workspace completada.</p>${detail}`;
  const reason = sync.error ? `: ${escapeHtml(sync.error)}` : '';
  return `<p>La sincronización del workspace falló${reason}</p>${detail}`;
}

function manualHint(job) {
  if (!job.manual_hint) return '';
  return `<p>Puedes actualizar a mano con <code class="mono">${escapeHtml(job.manual_hint)}</code></p>`;
}

// renderJob pinta el avance o el resultado del trabajo de actualización.
function renderJob(banner, job) {
  if (RUNNING_PHASES.has(job.phase)) {
    banner.className = 'alert alert-warn update-banner';
    banner.setAttribute('role', 'status');
    banner.innerHTML = `<span class="spinner" aria-hidden="true"></span><p class="grow"><strong>${escapeHtml(PHASE_LABEL[job.phase])}</strong> · ${escapeHtml(job.message)}</p>`;
    return;
  }
  const failed = job.phase === 'failed';
  banner.className = `alert ${failed ? 'alert-bad' : 'alert-ok'} update-banner`;
  banner.setAttribute('role', failed ? 'alert' : 'status');
  const copy = failed && job.manual_hint ? button({ label: 'Copiar orden', iconName: 'copy', size: 'sm', attrs: `data-copy="${escapeHtml(job.manual_hint)}"` }) : '';
  const retry = failed ? button({ label: 'Reintentar', iconName: 'refresh', size: 'sm', attrs: 'data-action="update-now"' }) : '';
  banner.innerHTML = `${icon(failed ? 'alert' : 'check', 18)}<div class="grow"><p><strong>${failed ? 'La actualización ha fallado' : 'Actualización terminada'}</strong> · ${escapeHtml(job.message)}</p>${failed ? manualHint(job) : ''}${syncSummary(job.sync)}</div>`
    + copy + retry
    + button({ label: 'Cerrar', size: 'sm', attrs: 'data-action="update-dismiss"' });
}

// renderUpdateBanner pinta el aviso según state.updateJob (si hay trabajo) o
// state.version.
export function renderUpdateBanner() {
  const banner = $('#update-banner');
  if (!banner) return;
  const job = state.updateJob;
  if (job && job.phase && job.phase !== 'idle' && (job.running || !isDismissed(job))) {
    banner.hidden = false;
    renderJob(banner, job);
    return;
  }
  const info = state.version;
  if (!info || info.status !== 'update_available' || !info.update_available) {
    banner.hidden = true;
    banner.innerHTML = '';
    return;
  }
  const release = safeReleaseUrl(info.release_url);
  const link = release ? `<a class="link-btn" href="${escapeHtml(release)}" target="_blank" rel="noopener noreferrer">Ver la release</a>` : '';
  banner.className = 'alert alert-warn update-banner';
  banner.setAttribute('role', 'status');
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

function stopPolling() {
  if (pollTimer) clearTimeout(pollTimer);
  pollTimer = null;
}

// Tras actualizarse, el servidor viejo se apaga y el nuevo vuelve a servir en el mismo
// puerto: mientras tanto no responde. restartFrom guarda la versión que servía el
// dashboard viejo; cuando el servidor contesta con otra, se recarga la página.
let restartFrom = null;
let restartStarted = 0;
// sawDown: el servidor dejó de responder al menos una vez durante el reinicio. Solo se
// usa si no se conoce la versión anterior (entonces basta con que vuelva a contestar).
let sawDown = false;
const RESTART_TIMEOUT_MS = 90000;
// Una vez lanzada una actualización desde esta página, el reinicio puede pasar sin que
// ningún sondeo vea la fase «restarting» (dura menos que el intervalo de sondeo). Por
// eso la recarga no depende solo de verla: se recarga también cuando responde un
// proceso distinto, es decir, cuando el estado trae otro from_version (el proceso
// relanzado no lo conoce) o cuando hubo fallos de red antes de volver a contestar.
let updateStarted = false;
let startedFrom = '';
let downSeen = false;
let restartHint = '';

function trackUpdate(job) {
  updateStarted = true;
  startedFrom = job.from_version || '';
  downSeen = false;
}

function newProcessAnswered(job) {
  return updateStarted && (downSeen || (job.from_version || '') !== startedFrom);
}
const MANUAL_RELAUNCH = 'axiom ui';

function failRestart(hint) {
  restartFrom = null;
  sawDown = false;
  stopPolling();
  set('updateJob', {
    phase: 'failed', running: false, sync: null,
    message: 'El dashboard no volvió a arrancar con la versión nueva.',
    manual_hint: hint || restartHint || MANUAL_RELAUNCH
  });
}

// pollRestart espera a que responda una versión distinta de la que se sustituyó.
async function pollRestart() {
  stopPolling();
  try {
    const info = await api('/api/system/version');
    if (info && info.current && (restartFrom ? info.current !== restartFrom : sawDown)) {
      window.location.reload();
      return;
    }
  } catch (err) {
    // el servidor está reiniciándose: se sigue esperando
    sawDown = true;
    downSeen = true;
  }
  if (Date.now() - restartStarted > RESTART_TIMEOUT_MS) {
    failRestart(state.updateJob && state.updateJob.manual_hint);
    return;
  }
  // Mientras el servidor viejo aún responde, la versión no cambia: se sigue esperando.
  pollTimer = setTimeout(pollRestart, POLL_MS);
}

function beginRestartWait(job) {
  if (restartFrom !== null) return;
  restartFrom = job.from_version || (state.version && state.version.current) || '';
  restartStarted = Date.now();
  restartHint = job.manual_hint || '';
  sawDown = false;
  pollTimer = setTimeout(pollRestart, POLL_MS);
}

// pollUpdate consulta el estado del trabajo hasta que termina. Los fallos de red se
// toleran un rato (el servidor se reinicia al actualizarse); pasado ese margen se
// muestra un fallo local con el comando manual.
async function pollUpdate() {
  stopPolling();
  try {
    const job = await api('/api/system/update/status');
    pollErrors = 0;
    if (newProcessAnswered(job)) {
      window.location.reload();
      return;
    }
    set('updateJob', job);
    if (job.running && job.phase === 'restarting') {
      beginRestartWait(job);
    } else if (job.running) {
      pollTimer = setTimeout(pollUpdate, POLL_MS);
    }
  } catch (err) {
    downSeen = true;
    pollErrors += 1;
    if (pollErrors >= MAX_POLL_ERRORS) {
      set('updateJob', {
        phase: 'failed', running: false, sync: null,
        message: 'Se perdió la conexión con el dashboard durante la actualización.',
        manual_hint: 'go install github.com/IGutierrezZ/axiom/v3/cmd/axiom@latest'
      });
      return;
    }
    pollTimer = setTimeout(pollUpdate, POLL_MS);
  }
}

async function startUpdate(btn) {
  const restore = setBusy(btn, 'Actualizando…');
  try {
    const job = await api('/api/system/update', { method: 'POST' });
    pollErrors = 0;
    trackUpdate(job);
    set('updateJob', job);
    if (job.running) pollTimer = setTimeout(pollUpdate, POLL_MS);
  } catch (err) {
    toast(`No se pudo iniciar la actualización: ${err.message}`, 'bad');
  } finally {
    restore();
  }
}

function dismiss() {
  if (state.updateJob) rememberDismissed(state.updateJob);
  renderUpdateBanner();
}

// resumeUpdate recupera el estado del trabajo al cargar la página: si había uno en
// curso (o terminado sin cerrar, p. ej. tras el reinicio) se vuelve a mostrar.
async function resumeUpdate() {
  try {
    const job = await api('/api/system/update/status');
    if (job.running) trackUpdate(job);
    set('updateJob', job);
    if (job.running) pollTimer = setTimeout(pollUpdate, POLL_MS);
  } catch (err) {
    console.debug('No se pudo leer el estado de la actualización:', err);
  }
}

export function initUpdate() {
  on('version', renderUpdateBanner);
  on('updateJob', renderUpdateBanner);
  const banner = $('#update-banner');
  if (banner) {
    banner.addEventListener('click', (event) => {
      const target = event.target.closest('button[data-action]');
      if (!target) return;
      if (target.dataset.action === 'update-recheck') recheck(target);
      if (target.dataset.action === 'update-now') startUpdate(target);
      if (target.dataset.action === 'update-dismiss') dismiss();
    });
  }
  checkVersion();
  resumeUpdate();
}
