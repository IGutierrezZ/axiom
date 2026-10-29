// Estructura de la aplicación: barra lateral, barra superior (miga, actualizar,
// nuevo incremento), menú plegable en anchos estrechos y aviso de specs.

import { api, escapeHtml } from './core/api.js';
import { $, toast, setBusy } from './core/components.js';
import { isArchived } from './core/increment.js';
import { doctorSummary, sourceData, refreshOverview, POLLED_SOURCES } from './core/overview.js';
import { hrefFor } from './core/router.js';
import { state, on } from './core/state.js';

const ROUTE_LABELS = {
  inicio: 'Inicio',
  incrementos: 'Incrementos',
  incremento: 'Incrementos',
  specs: 'Specs vivas',
  skills: 'Skills',
  codigo: 'Código y grafo',
  ecosistema: 'Ecosistema'
};

export function routeLabel(route) {
  return ROUTE_LABELS[route.name] || 'Axiom';
}

// setActiveNav marca la entrada activa con aria-current="page".
export function setActiveNav(route) {
  const active = route.name === 'incremento' ? 'incrementos' : route.name;
  document.querySelectorAll('.nav-item[data-nav]').forEach(link => {
    if (link.getAttribute('data-nav') === active) link.setAttribute('aria-current', 'page');
    else link.removeAttribute('aria-current');
  });
}

// renderBreadcrumb pinta la miga: proyecto / sección [/ incremento].
export function renderBreadcrumb(route) {
  const list = document.getElementById('breadcrumb');
  if (!list) return;
  const project = (state.workspace && state.workspace.name) || 'Proyecto';
  const items = [`<li>${escapeHtml(project)}</li>`];
  if (route.name === 'incremento') {
    items.push(`<li><a href="${hrefFor('incrementos')}">Incrementos</a></li>`);
    items.push(`<li aria-current="page">${escapeHtml(route.change)}</li>`);
  } else {
    items.push(`<li aria-current="page">${escapeHtml(routeLabel(route))}</li>`);
  }
  list.innerHTML = items.join('');
}

// ---- Menú plegable (≤ 859 px) ----------------------------------------------

function setMenu(open) {
  const sidebar = document.getElementById('sidebar');
  const toggle = document.getElementById('menu-toggle');
  if (!sidebar || !toggle) return;
  sidebar.setAttribute('data-open', String(open));
  toggle.setAttribute('aria-expanded', String(open));
  toggle.setAttribute('aria-label', open ? 'Cerrar el menú' : 'Abrir el menú');
}

export function closeMenu() {
  setMenu(false);
}

function initMenu() {
  const toggle = document.getElementById('menu-toggle');
  if (!toggle) return;
  toggle.addEventListener('click', () => {
    setMenu(toggle.getAttribute('aria-expanded') !== 'true');
  });
  document.getElementById('sidebar-body').addEventListener('click', (event) => {
    if (event.target.closest('a.nav-item')) closeMenu();
  });
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && toggle.getAttribute('aria-expanded') === 'true' && !document.querySelector('dialog[open]')) {
      setMenu(false);
      toggle.focus();
    }
  });
}

// ---- Aviso de specs por detrás del remoto (ODD-5.4) -------------------------

// El estado sale de la caché de Inicio (state.overview.sync): una sola consulta
// alimenta el aviso de la barra lateral y el de Inicio.
export function renderSyncBanner() {
  const banner = $('#specs-sync-banner');
  const textEl = $('#specs-sync-text');
  if (!banner) return;
  const status = sourceData('sync');
  if (!status || !status.is_git_repo || !(status.behind > 0)) {
    banner.hidden = true;
    return;
  }
  banner.hidden = false;
  const behind = status.behind;
  textEl.innerHTML = `Specs: <span class="mono">${escapeHtml(status.branch || 'main')}</span> · ${behind} por detrás`;
  banner.title = `El repositorio de especificaciones está ${behind} commit(s) por detrás del remoto (${status.branch} @ ${status.remote}). Haz pull antes de operar.`;
}

export function checkSpecsSync() {
  return refreshOverview(['sync']);
}

let reloadWorkspaceFn = async () => {};

// pullSpecs hace pull del repositorio de specs y refresca el estado. Lo usan el
// botón de la barra lateral y el de Inicio.
export async function pullSpecs(btn) {
  const restore = setBusy(btn, 'Sincronizando…');
  try {
    await api('/api/workspace/specs/pull', { method: 'POST' });
    toast('Repositorio de especificaciones sincronizado con éxito.', 'ok');
    await checkSpecsSync();
    await reloadWorkspaceFn();
  } catch (err) {
    toast(`Error haciendo pull: ${err.message}`, 'bad');
  } finally {
    restore();
  }
}

function initSpecsPull(reloadWorkspace) {
  reloadWorkspaceFn = reloadWorkspace;
  const btn = $('#btn-specs-pull');
  if (!btn) return;
  btn.addEventListener('click', () => pullSpecs(btn));
}

// ---- Insignias de la barra lateral -------------------------------------------
// Incrementos (activos), Skills (propuestas pendientes) y Ecosistema (punto de
// aviso). Salen de la misma caché: se repintan cuando llegan datos, no al navegar.

function setBadge(id, text, hiddenText) {
  const el = document.getElementById(id);
  if (!el) return;
  el.hidden = !text;
  if (!text) return;
  el.innerHTML = `<span aria-hidden="true">${escapeHtml(text)}</span><span class="sr-only">${escapeHtml(hiddenText)}</span>`;
}

export function renderNavBadges() {
  const increments = sourceData('increments');
  const active = increments ? increments.filter(inc => !isArchived(inc)).length : 0;
  setBadge('badge-incrementos', active > 0 ? String(active) : '', ` ${active === 1 ? 'incremento activo' : 'incrementos activos'}`);

  const inbox = sourceData('inbox');
  const pending = Array.isArray(inbox) ? inbox.length : 0;
  setBadge('badge-skills', pending > 0 ? String(pending) : '', ` ${pending === 1 ? 'propuesta pendiente' : 'propuestas pendientes'}`);

  const eco = document.getElementById('badge-ecosistema');
  if (!eco) return;
  const report = sourceData('doctor');
  const sum = doctorSummary(report);
  eco.hidden = !report || sum.notOk === 0;
  if (eco.hidden) return;
  const parts = [];
  if (sum.errors > 0) parts.push(`${sum.errors} ${sum.errors === 1 ? 'error' : 'errores'}`);
  if (sum.warnings > 0) parts.push(`${sum.warnings} ${sum.warnings === 1 ? 'aviso' : 'avisos'}`);
  const text = parts.join(' y ');
  eco.innerHTML = `<span class="nav-dot${sum.errors > 0 ? ' is-bad' : ''}" aria-hidden="true"></span><span class="sr-only">${escapeHtml(text)}</span>`;
  eco.title = text;
}

// ---- Cabecera del selector de proyecto --------------------------------------

export function renderProjectBlock() {
  const ws = state.workspace;
  const nameEl = $('#ws-name');
  const subEl = $('#ws-topology');
  const initialsEl = $('#project-initials');
  const trigger = $('#project-trigger');
  if (!nameEl) return;
  if (!ws) return;
  const roles = ws.roles ? Object.keys(ws.roles).length : 0;
  nameEl.textContent = ws.name || 'Workspace';
  subEl.textContent = `${ws.topology || 'monorepo'} · ${roles} ${roles === 1 ? 'rol' : 'roles'}`;
  const words = String(ws.name || '?').split(/[\s_\-./\\]+/).filter(Boolean);
  initialsEl.textContent = (words.length > 1 ? words[0][0] + words[1][0] : (words[0] || '?').slice(0, 2)).toUpperCase();
  trigger.setAttribute('aria-label', `Cambiar de proyecto: ${ws.name || 'Workspace'}`);
}

// initShell enlaza lo propio de la estructura. Las acciones que dependen de
// datos (refrescar, nuevo incremento) las registra main.js.
export function initShell({ onRefresh, reloadWorkspace }) {
  initMenu();
  initSpecsPull(reloadWorkspace);
  on('workspace', renderProjectBlock);
  on('overview', renderSyncBanner);
  on('overview', renderNavBadges);

  const refresh = $('#btn-refresh');
  refresh.addEventListener('click', async () => {
    refresh.disabled = true;
    refresh.setAttribute('aria-busy', 'true');
    try {
      await onRefresh();
    } finally {
      refresh.disabled = false;
      refresh.removeAttribute('aria-busy');
    }
  });

  // Sondeo cada 60 s (solo con la pestaña visible): sincronización remota de specs
  // e insignias. No se repite al cambiar de ruta.
  setInterval(() => {
    if (!document.hidden) refreshOverview(POLLED_SOURCES);
  }, 60000);
}

