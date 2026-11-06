// Datos de Inicio y de la barra lateral: una sola caché en state.overview con una
// entrada por fuente ({status: loading | ready | error, data, error}).
//
// - Se refresca al cargar, con «Actualizar» y con el sondeo de 60 s; nunca al
//   cambiar de ruta, así que navegar no dispara peticiones.
// - Las peticiones GET simultáneas se comparten en api() (ver core/api.js): las
//   pantallas que ya consultan la misma ruta no duplican la llamada.
// - Cada fuente falla por separado: una que falle no vacía el resto de Inicio.
// - Con datos previos, un refresco no vuelve a «cargando»: se muestran los
//   anteriores hasta que llega la respuesta (sin parpadeo).

import { api } from './api.js';
import { state, set, on } from './state.js';

export const SOURCES = {
  increments: '/api/increments',
  doctor: '/api/ecosystem/doctor',
  sync: '/api/workspace/specs/sync-status',
  inbox: '/api/skills/inbox',
  semantic: '/api/semantic/status',
  specs: '/api/archive/specs'
};

// Estas fuentes devuelven una lista; un `null` (lista vacía en Go) se normaliza a [].
const LIST_SOURCES = new Set(['increments', 'inbox']);

// El sondeo de 60 s solo repite lo que alimenta las insignias y el aviso de specs;
// el estado semántico y el catálogo de specs se actualizan al cargar y con Actualizar.
export const POLLED_SOURCES = ['increments', 'doctor', 'sync', 'inbox'];

state.overview = Object.fromEntries(Object.keys(SOURCES).map(key => [key, { status: 'loading', data: null, error: '' }]));

function publish(key, entry) {
  set('overview', { ...state.overview, [key]: entry });
}

// ---- Fuentes que dependen del proyecto abierto -------------------------------------------
//
// El Doctor mira el proyecto abierto. Al cambiar de proyecto el informe anterior se
// invalida (resetProjectData) y lo que llegue de una petición lanzada antes del cambio
// no se publica: la época sube con cada cambio, y el informe lleva además la raíz que
// examinó (`project`), que debe ser la del proyecto abierto.

const PROJECT_SOURCES = ['doctor'];
let projectEpoch = 0;

function normalizePath(path) {
  return String(path || '').split('\\').join('/').replace(/\/+$/, '').toLowerCase();
}

// activeProjectPath devuelve la ruta del proyecto abierto según la lista de proyectos, o ''.
export function activeProjectPath() {
  const data = state.projects;
  if (!data || !Array.isArray(data.projects)) return '';
  const project = data.projects.find(p => p && (p.id === data.active_workspace || p.path === data.active_workspace));
  return project && project.path ? String(project.path) : '';
}

// isReportForProject: un informe sin `project`, o sin proyecto abierto conocido con el
// que compararlo, se acepta; si ambos existen, tienen que ser el mismo.
export function isReportForProject(report, expected) {
  if (!report || !report.project || !expected) return true;
  return normalizePath(report.project) === normalizePath(expected);
}

// resetProjectData olvida lo que se sabía del proyecto anterior (el Doctor vuelve a
// «cargando») e ignora las respuestas que sigan en vuelo.
export function resetProjectData() {
  projectEpoch += 1;
  for (const key of PROJECT_SOURCES) publish(key, { status: 'loading', data: null, error: '' });
}

async function refreshSource(key, retry = true) {
  const epoch = projectEpoch;
  const expected = PROJECT_SOURCES.includes(key) ? activeProjectPath() : '';
  try {
    const data = await api(SOURCES[key]);
    if (epoch !== projectEpoch) return; // llegó tarde: el proyecto cambió y otra petición publicará
    if (!isReportForProject(data, expected)) {
      // Pudo compartirse una petición lanzada antes del cambio: se repite una vez.
      if (retry) return refreshSource(key, false);
      publish(key, { status: 'error', data: state.overview[key].data, error: 'El diagnóstico recibido no corresponde al proyecto abierto.' });
      return;
    }
    publish(key, { status: 'ready', data: LIST_SOURCES.has(key) ? data || [] : data, error: '' });
  } catch (err) {
    console.debug(`Error cargando ${SOURCES[key]}:`, err);
    if (epoch !== projectEpoch) return;
    publish(key, { status: 'error', data: state.overview[key].data, error: err.message });
  }
}

// refreshOverview vuelve a consultar las fuentes indicadas (todas por defecto).
export function refreshOverview(keys = Object.keys(SOURCES)) {
  return Promise.all(keys.filter(key => SOURCES[key]).map(key => refreshSource(key)));
}

// Cuando la pantalla de Incrementos recarga la lista (Continuar, Nuevo incremento…),
// la caché se pone al día sin otra petición.
on('increments', (list) => publish('increments', { status: 'ready', data: list || [], error: '' }));

// sourceData devuelve los datos de una fuente o null si aún no hay.
export function sourceData(key) {
  const entry = state.overview[key];
  return entry && entry.data !== null && entry.data !== undefined ? entry.data : null;
}

// doctorSummary resume el informe de salud: comprobaciones con status != "ok".
export function doctorSummary(report) {
  const checks = report && Array.isArray(report.checks) ? report.checks : [];
  const notOk = checks.filter(c => c.status !== 'ok');
  const errors = notOk.filter(c => c.status === 'error').length;
  return { healthy: !!(report && report.healthy), notOk: notOk.length, errors, warnings: notOk.length - errors, first: notOk[0] || null };
}
