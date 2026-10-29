// Inicio: qué necesita atención. Titular accionable, aviso de specs, cuatro
// indicadores, lista «Necesita atención» (con la acción de D5), flujo SDD por fase
// y «Axiom te ayuda». Si el proyecto no está configurado (is_configured === false),
// #zero-config-hero sustituye al contenido.
//
// Los datos salen de state.overview (core/overview.js): cada bloque se pinta con
// su propia fuente, así que una que falle o tarde no vacía el resto. Sin datos
// inventados ni edades relativas (D3).

import { api, escapeHtml } from '../core/api.js';
import { $, $all, button, consoleBox, renderConsole, reminder, setBusy, stateEmpty, stateError, stateLoading, toast } from '../core/components.js';
import { icon } from '../core/icons.js';
import { attentionItems, continueIncrement, deriveNextStep, flagBadge, incrementStats, progressBar, typeChip } from '../core/increment.js';
import { doctorSummary, refreshOverview } from '../core/overview.js';
import { hrefFor } from '../core/router.js';
import { state, on } from '../core/state.js';
import { pullSpecs } from '../shell.js';
import { loadIncrements } from './incrementos.js';

let reloadAll = async () => {};

// Cuántas filas se muestran en «Necesita atención»; el resto está en el tablero.
const MAX_ATTENTION = 8;
// Segmentos máximos por fase en el flujo SDD (el número exacto va siempre en texto).
const MAX_SEGMENTS = 10;

// Incrementos con «Continuar» en curso: se pintan deshabilitados aunque llegue
// otro refresco mientras se ejecuta la orden.
const busy = new Set();
// Último HTML pintado por bloque: si no cambia no se toca el DOM (conserva foco y estado).
const painted = new Map();

function plural(n, one, many) {
  return `${n} ${n === 1 ? one : many}`;
}

function source(key) {
  return state.overview[key] || { status: 'loading', data: null, error: '' };
}

function paint(selector, html) {
  const el = $(selector);
  if (!el || painted.get(selector) === html) return;
  painted.set(selector, html);
  el.innerHTML = html;
}

// ---- Titular y aviso de specs -----------------------------------------------------------

function renderHeadline() {
  const title = $('#inicio-title');
  const sub = $('#inicio-sub');
  const inc = source('increments');
  let h1 = 'Inicio';
  let text = 'Cargando el estado de los incrementos…';
  if (inc.status === 'error' && !inc.data) {
    text = 'No se pudo leer el estado de los incrementos.';
  } else if (inc.data) {
    const stats = incrementStats(inc.data);
    const n = attentionItems(inc.data).length;
    if (n > 0) {
      h1 = `${n} ${n === 1 ? 'incremento necesita' : 'incrementos necesitan'} atención`;
      text = 'Un incremento necesita atención cuando está listo para verificar, esperando a roles, listo para archivar o pendiente de spec o de design.';
    } else if (stats.active > 0) {
      h1 = 'Nada necesita atención';
      text = `${plural(stats.active, 'incremento activo avanza', 'incrementos activos avanzan')} sin bloqueos.`;
    } else {
      h1 = 'Aún no hay incrementos activos';
      text = 'Crea el primero con «Nuevo incremento» en la barra superior.';
    }
  }
  if (title.textContent !== h1) title.textContent = h1;
  if (sub.textContent !== text) sub.textContent = text;
}

function renderSyncNotice() {
  const status = source('sync').data;
  if (!status || !status.is_git_repo || !(status.behind > 0)) return paint('#home-sync', '');
  const remote = status.remote ? `${status.remote}/${status.branch || 'main'}` : status.branch || 'main';
  paint('#home-sync', `<div class="alert alert-warn" role="status">${icon('download', 18)}<p class="grow">El repositorio de specs va <strong>${plural(status.behind, 'commit', 'commits')} por detrás</strong> de <span class="mono">${escapeHtml(remote)}</span>. Sincronízalo antes de abrir un incremento nuevo.</p>${button({ label: 'Hacer pull de specs', iconName: 'download', size: 'sm', attrs: 'data-action="pull"' })}</div>`);
}

// ---- Indicadores ------------------------------------------------------------------------------

function tile({ label, value, sub = '', href, extra = '' }) {
  return `<a class="card kpi kpi-link" href="${href}"><span class="kpi-label">${escapeHtml(label)}</span><span class="kpi-value">${value}</span>${extra}<span class="kpi-sub">${sub}</span></a>`;
}

function waitingTile(label, href, entry) {
  const failed = entry.status === 'error';
  return tile({
    label, href, value: '<span class="dim">—</span>',
    sub: failed ? `<span class="text-bad">${icon('alert', 13)}No disponible: ${escapeHtml(entry.error)}</span>` : (entry.status === 'loading' ? 'Cargando…' : 'Sin datos')
  });
}

function incrementTiles() {
  const entry = source('increments');
  const labels = [['Incrementos activos', '#/incrementos'], ['Tareas completadas', '#/incrementos'], ['En verificación', '#/incrementos']];
  if (!entry.data) return labels.map(([label, href]) => waitingTile(label, href, entry));
  const s = incrementStats(entry.data);
  const bugs = s.bugs > 0 ? `<span class="text-bad kpi-bug">${icon('bug', 13)}${plural(s.bugs, 'bug', 'bugs')}</span> · ` : '';
  return [
    tile({ label: 'Incrementos activos', href: '#/incrementos', value: s.active, sub: `${bugs}${plural(s.archived, 'archivado', 'archivados')}` }),
    tile({
      label: 'Tareas completadas', href: '#/incrementos',
      value: `${s.tasksDone}<span class="kpi-of"> / ${s.tasksTotal}</span>`,
      extra: s.tasksTotal > 0 ? progressBar(s.pct, 'Tareas completadas de los incrementos activos') : '',
      sub: s.tasksTotal > 0 ? `${s.pct} % completado` : 'Sin tareas todavía'
    }),
    tile({ label: 'En verificación', href: '#/incrementos', value: s.verifying, sub: `${plural(s.readyForArchive, 'listo', 'listos')} para archivar` })
  ];
}

function healthTile() {
  const entry = source('doctor');
  if (!entry.data) return waitingTile('Salud del ecosistema', '#/ecosistema', entry);
  const sum = doctorSummary(entry.data);
  let status = { tone: 'ok', icon: 'check', text: 'Correcta' };
  if (sum.errors > 0) status = { tone: 'bad', icon: 'x', text: 'Con errores' };
  else if (!sum.healthy || sum.notOk > 0) status = { tone: 'warn', icon: 'alert', text: 'Con avisos' };
  const first = sum.first ? ` · ${escapeHtml(sum.first.name)}` : '';
  const sub = sum.notOk > 0 ? `${plural(sum.notOk, 'comprobación', 'comprobaciones')} sin superar${first}` : 'Todas las comprobaciones en orden';
  return tile({ label: 'Salud del ecosistema', href: '#/ecosistema', value: `<span class="kpi-status text-${status.tone}">${icon(status.icon, 20)}${status.text}</span>`, sub });
}

// ---- Necesita atención ----------------------------------------------------------------------------

// attentionRow aplica D5: «Continuar» es el botón de Axiom (calcula el siguiente
// paso); las fases de agente salen como Recordatorio compacto con la orden copiable.
function attentionRow({ inc, flag }) {
  const name = inc.name;
  const next = deriveNextStep({ summary: inc, barrier_report: null });
  const tasks = inc.tasks_total > 0 ? `${inc.tasks_completed}/${inc.tasks_total} tareas · ${inc.progress_pct} %` : 'Sin tareas todavía';
  const actions = [];
  if (next.canContinue) {
    const running = busy.has(name);
    actions.push(button({
      label: running ? 'Ejecutando…' : 'Continuar', iconName: 'play', size: 'sm', variant: flag.id === 'gverify' ? 'primary' : 'secondary',
      attrs: `data-action="continue" data-name="${escapeHtml(name)}" aria-label="Continuar ${escapeHtml(name)} con Axiom"${running ? ' disabled aria-busy="true"' : ''}`
    }));
  }
  if (flag.id === 'waiting') {
    actions.push(`<a class="btn btn-secondary btn-sm" href="${hrefFor('incremento', { change: name, tab: 'roles' })}" aria-label="Ver la barrera de ${escapeHtml(name)}">${icon('users', 16)}Ver barrera</a>`);
  }
  const rem = next.reminder ? `<div class="att-rem">${reminder({ ...next.reminder, compact: true })}</div>` : '';
  return `<li class="row att-row">
    <div class="row-main"><a class="att-name" href="${hrefFor('incremento', { change: name })}">${escapeHtml(name)}</a><span class="att-detail">${escapeHtml(tasks)}</span></div>
    <div class="att-tags">${typeChip(inc)}<span class="badge badge-mono">${escapeHtml(inc.phase)}</span>${flagBadge(inc)}</div>
    <div class="att-actions">${actions.join('')}</div>
    ${rem}
  </li>`;
}

function renderAttention() {
  const entry = source('increments');
  if (!entry.data) {
    if (entry.status === 'error') {
      return paint('#home-attention', `${stateError(entry.error, 'No se pudo cargar la lista')}<p class="att-foot">${button({ label: 'Reintentar', iconName: 'refresh', size: 'sm', attrs: 'data-action="retry"' })}</p>`);
    }
    return paint('#home-attention', stateLoading('Cargando incrementos…'));
  }
  const items = attentionItems(entry.data);
  if (items.length === 0) {
    return paint('#home-attention', `<div class="att-empty">${stateEmpty('Nada necesita atención', 'Los incrementos activos avanzan sin bloqueos.', 'check')}</div>`);
  }
  const shown = items.slice(0, MAX_ATTENTION);
  const more = items.length > shown.length ? `Mostrando ${shown.length} de ${items.length}. ` : '';
  paint('#home-attention', `<ul class="row-list" role="list" aria-label="Incrementos que necesitan atención">${shown.map(attentionRow).join('')}</ul>
    <p class="att-foot">${icon('terminal', 14)}<span>${more}Las fases de agente (propuesta, apply, verify, archive…) no las ejecuta el dashboard: lánzalas desde tu IDE o CLI con la orden de cada fila. «Continuar» solo ejecuta la orden de Axiom que calcula el siguiente paso.</span></p>`);
}

// ---- Flujo SDD --------------------------------------------------------------------------------------

function renderFlow() {
  const entry = source('increments');
  if (!entry.data) {
    return paint('#home-flow', entry.status === 'error' ? stateError(entry.error, 'No se pudo cargar el flujo') : stateLoading('Cargando el flujo…'));
  }
  const stats = incrementStats(entry.data);
  const cells = stats.byPhase.map((p, i) => {
    const total = Math.min(p.count, MAX_SEGMENTS);
    const segs = Array.from({ length: total }, (_, k) => `<span class="flow-seg${k < p.bugs ? ' is-bug' : ''}"></span>`).join('');
    const bug = p.bugs > 0 ? `<span class="flow-bug">${icon('bug', 12)}${plural(p.bugs, 'bug', 'bugs')}</span>` : '';
    return `<li class="flow-cell${p.count === 0 ? ' is-empty' : ''}"><span class="flow-label mono">0${i + 1} ${escapeHtml(p.id)}</span><span class="flow-count">${p.count}<span class="sr-only"> ${p.count === 1 ? 'incremento' : 'incrementos'} en ${escapeHtml(p.label)}</span></span><span class="flow-bars" aria-hidden="true">${segs}</span>${bug}</li>`;
  }).join('');
  const archive = `<li class="flow-cell flow-archive"><span class="flow-label mono">07 archive</span><span class="flow-count">${stats.archived}<span class="sr-only"> ${stats.archived === 1 ? 'incremento archivado' : 'incrementos archivados'}</span></span><span class="flow-note">en specs vivas</span></li>`;
  paint('#home-flow', `<ol class="flow-list" aria-label="Incrementos activos por fase SDD">${cells}${archive}</ol>
    <ul class="flow-legend"><li><span class="flow-key"></span>Feature · refactor · arquitectura</li><li><span class="flow-key is-bug"></span>${icon('bug', 12)}Bug o fix (con trama y texto)</li></ul>`);
}

// ---- Axiom te ayuda ----------------------------------------------------------------------------------

function helpRow({ href, iconName, title, text, error = false }) {
  const body = `${icon(iconName, 18)}<span class="help-text"><span class="help-title">${escapeHtml(title)}</span><span class="help-sub">${escapeHtml(text)}</span></span>`;
  return error ? `<div class="row help-row is-error">${body}</div>` : `<a class="row help-row" href="${href}">${body}</a>`;
}

function helpRows() {
  const rows = [];
  let waiting = 0;
  const check = (key, label, build) => {
    const entry = source(key);
    if (entry.data) return build(entry.data);
    if (entry.status === 'error') rows.push(helpRow({ iconName: 'alert', title: `No se pudo consultar ${label}`, text: entry.error, error: true }));
    else if (entry.status === 'loading') waiting += 1;
  };

  check('inbox', 'el buzón de skills', (inbox) => {
    if (Array.isArray(inbox) && inbox.length > 0) {
      rows.push(helpRow({ href: '#/skills', iconName: 'cube', title: `${plural(inbox.length, 'skill propuesta', 'skills propuestas')} en el buzón`, text: 'Revísalas y apruébalas o recházalas.' }));
    }
  });
  check('semantic', 'el estado del código', (sem) => {
    if (!(sem.total_symbols > 0)) {
      rows.push(helpRow({ href: '#/codigo', iconName: 'graph', title: 'El grafo de código no está indexado', text: 'Indexa el código para que los agentes puedan consultarlo.' }));
    } else if (Array.isArray(sem.warnings) && sem.warnings.length > 0) {
      rows.push(helpRow({ href: '#/codigo', iconName: 'graph', title: `${plural(sem.warnings.length, 'aviso', 'avisos')} del conector de código`, text: String(sem.warnings[0]) }));
    }
  });
  check('specs', 'las specs vivas', (catalog) => {
    const domains = Array.isArray(catalog.specs) ? catalog.specs.length : 0;
    if (catalog.external_baseline) {
      rows.push(helpRow({ href: '#/specs', iconName: 'book', title: 'La baseline de specs es externa', text: 'Sus documentos no se indexan aquí; consulta los dominios y rutas declarados.' }));
    } else if (domains > 0) {
      rows.push(helpRow({ href: '#/specs', iconName: 'book', title: `${plural(domains, 'dominio', 'dominios')} en specs vivas`, text: `${plural(catalog.total_requirements || 0, 'requisito consolidado', 'requisitos consolidados')}. Consúltalas antes de proponer.` }));
    }
  });
  const doctor = source('doctor').data;
  if (doctor) {
    const sum = doctorSummary(doctor);
    if (sum.notOk > 0) {
      rows.push(helpRow({ href: '#/ecosistema', iconName: 'pulse', title: `${plural(sum.notOk, 'aviso', 'avisos')} en el ecosistema`, text: `${sum.first.name}: ${sum.first.details}` }));
    }
  } else if (source('doctor').status === 'loading') {
    waiting += 1;
  }
  return { rows, waiting };
}

function renderHelp() {
  const card = $('#home-help');
  const { rows, waiting } = helpRows();
  // Sin nada relevante y sin nada pendiente, el bloque entero se oculta.
  card.hidden = rows.length === 0 && waiting === 0;
  paint('#home-help-body', rows.length > 0 ? `<div class="row-list">${rows.join('')}</div>` : stateLoading('Buscando sugerencias…'));
}

// ---- Pintado general --------------------------------------------------------------------------------------

// focusKey recuerda el control con foco dentro de Inicio para recuperarlo si se repinta.
function focusKey() {
  const el = document.activeElement;
  const root = $('#inicio-content');
  if (!el || !root || !root.contains(el) || !el.hasAttribute('data-action')) return null;
  return { action: el.getAttribute('data-action'), name: el.getAttribute('data-name') };
}

function restoreFocus(key) {
  if (!key || (document.activeElement && document.activeElement !== document.body)) return;
  const target = $all('#inicio-content [data-action]').find(el => el.getAttribute('data-action') === key.action && el.getAttribute('data-name') === key.name && !el.disabled);
  if (target) target.focus();
}

function renderBlocks() {
  const key = focusKey();
  renderHeadline();
  renderSyncNotice();
  paint('#home-kpis', [...incrementTiles(), healthTile()].join(''));
  renderAttention();
  renderFlow();
  renderHelp();
  restoreFocus(key);
}

export function renderInicio() {
  const ws = state.workspace;
  const hero = $('#zero-config-hero');
  const content = $('#inicio-content');
  if (!ws) return;

  const eyebrow = $('#inicio-eyebrow');
  const today = new Date().toLocaleDateString('es-ES', { weekday: 'long', day: 'numeric', month: 'long' });
  eyebrow.textContent = today.charAt(0).toUpperCase() + today.slice(1);

  if (ws.is_configured) {
    hero.hidden = true;
    content.hidden = false;
    renderBlocks();
    return;
  }
  hero.hidden = false;
  content.hidden = true;
  $('#zero-config-msg').innerHTML = `No se encontró un archivo <code>axiom.yaml</code> en <code>${escapeHtml(ws.root)}</code>. Puedes inicializarlo en 1 clic con detección automática del stack:`;
  const tech = $('#zero-config-tech');
  tech.innerHTML = '';
  if (ws.detected_tech) {
    const dt = ws.detected_tech;
    const labels = [`Tecnología: ${(dt.primary_language || 'genérico').toUpperCase()}`, ...(dt.frameworks || [])];
    tech.innerHTML = labels.map(l => `<span class="badge badge-accent">${escapeHtml(l)}</span>`).join('');
  }
}

// ---- Acciones ---------------------------------------------------------------------------------------------------

async function runContinue(name, hadFocus) {
  if (busy.has(name)) return;
  busy.add(name);
  renderBlocks();
  const result = await continueIncrement(name);
  renderConsole($('#home-console'), { title: `Continuar con Axiom · ${name}`, state: result.ok ? 'ok' : 'error', text: result.text });
  busy.delete(name);
  // loadIncrements actualiza la pantalla de Incrementos y, con ella, la caché de Inicio.
  await Promise.all([loadIncrements(), refreshOverview(['increments'])]);
  renderBlocks();
  // El botón estuvo deshabilitado y el navegador soltó el foco: se devuelve al terminar.
  if (hadFocus) restoreFocus({ action: 'continue', name });
}

export function initInicio(reload) {
  reloadAll = reload;
  $('#home-console-slot').innerHTML = consoleBox('home-console', 'Continuar con Axiom');
  on('overview', renderInicio);

  $('#inicio-content').addEventListener('click', (event) => {
    const btn = event.target.closest('[data-action]');
    if (!btn || btn.disabled) return;
    switch (btn.getAttribute('data-action')) {
      case 'continue':
        runContinue(btn.getAttribute('data-name'), document.activeElement === btn);
        break;
      case 'pull':
        pullSpecs(btn);
        break;
      case 'retry':
        refreshOverview(['increments']);
        break;
      default:
    }
  });

  const init = $('#btn-init-project');
  init.addEventListener('click', async () => {
    const restore = setBusy(init, 'Inicializando…');
    try {
      await api('/api/projects/init', { body: { path: state.workspace ? state.workspace.root : '' } });
      await reloadAll();
    } catch (err) {
      toast(`Error al inicializar el proyecto: ${err.message}`, 'bad');
    } finally {
      restore();
    }
  });
}
