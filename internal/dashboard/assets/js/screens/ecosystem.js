// Ecosistema (#/ecosistema): salud de Axiom (Doctor), operaciones de plataforma
// (sync, upgrade, respaldo), topología, modelos y respaldos.
//
// - El Doctor sale de la caché de Inicio (overview.js): no se consulta dos veces y
//   «Volver a diagnosticar» la actualiza, igual que la insignia de la barra lateral.
// - Cada bloque carga y falla por separado: uno que falle no vacía la página.
// - Las operaciones deshabilitan sus botones mientras corren y dejan su resultado
//   en «Última operación» (solo las hechas en esta sesión de la página).
// - Restaurar un respaldo pide confirmación explícita antes de llamar a la API.
// - Ramas y worktrees (INC-25): la topología muestra la política de ramas efectiva en
//   solo lectura (se declara en axiom.yaml) y una tarjeta propia edita las preferencias
//   personales de worktree del proyecto activo, que el servidor guarda en el Hub.

import { api, escapeHtml } from '../core/api.js';
import { branchPolicyRows, effectivePrefsText, repoPolicyHtml } from '../core/branches.js';
import { $, badge, confirmDialog, promptDialog, setBusy, stateEmpty, stateError, stateLoading, statusPill, tableWrap, toast } from '../core/components.js';
import {
  OPERATION_LABEL, categoryText, formatBytes, formatEcosystemUpgradeSequence, healthSummary, operationOutcome, sortChecks, statusView
} from '../core/ecosystem.js';
import { formatDateTime, plural } from '../core/format.js';
import { icon } from '../core/icons.js';
import { refreshOverview } from '../core/overview.js';
import { on, state } from '../core/state.js';
import { loadProjects } from '../projects.js';

let busy = false;        // una operación (o restauración) en curso: bloquea las demás
let diagnosing = false;  // «Volver a diagnosticar» en curso
let shownDoctor = null;  // última entrada de la caché pintada, para no repintar sin cambios
let backups = [];

const lockable = () => document.querySelectorAll('#view-ecosistema [data-eco-lock]');

function setLocked(locked) {
  busy = locked;
  lockable().forEach(el => { el.disabled = locked; });
}

// ---- Salud y Doctor (desde la caché) -------------------------------------------------------

const BANNER_ICON = { ok: 'shield', warn: 'alert', bad: 'x' };

function bannerButton() {
  return `<button type="button" class="btn" id="btn-eco-diagnose"${diagnosing ? ' disabled aria-busy="true"' : ''}>${icon('refresh', 16)}${diagnosing ? 'Diagnosticando…' : 'Volver a diagnosticar'}</button>`;
}

function banner(entry) {
  if (!entry.data) {
    if (entry.status === 'error') {
      return `<div class="eco-banner is-bad" role="alert"><span class="eco-banner-icon" aria-hidden="true">${icon('x', 20)}</span><div class="eco-banner-text"><span class="eco-banner-title">No se pudo ejecutar el diagnóstico</span><span class="eco-banner-sub">${escapeHtml(entry.error)}</span></div>${bannerButton()}</div>`;
    }
    return `<div class="eco-banner" role="status"><span class="eco-banner-icon" aria-hidden="true"><span class="spinner"></span></span><div class="eco-banner-text"><span class="eco-banner-title">Comprobando el ecosistema…</span><span class="eco-banner-sub">Axiom Doctor está revisando agentes, herramientas y entorno.</span></div></div>`;
  }
  const sum = healthSummary(entry.data);
  const parts = [sum.total === 0 ? 'Sin comprobaciones' : `${sum.ok} de ${sum.total} comprobaciones correctas`];
  if (sum.warnings > 0) parts.push(plural(sum.warnings, 'aviso', 'avisos'));
  if (sum.errors > 0) parts.push(plural(sum.errors, 'error', 'errores'));
  if (sum.other > 0) parts.push(plural(sum.other, 'estado desconocido', 'estados desconocidos'));
  const when = formatDateTime(entry.data.timestamp);
  if (when) parts.push(`diagnóstico: ${when}`);
  const stale = entry.status === 'error' ? `<span class="eco-banner-sub text-bad" role="alert">No se pudo actualizar el diagnóstico: ${escapeHtml(entry.error)}</span>` : '';
  return `<div class="eco-banner is-${sum.tone}" role="status"><span class="eco-banner-icon" aria-hidden="true">${icon(BANNER_ICON[sum.tone], 20)}</span><div class="eco-banner-text"><span class="eco-banner-title">${escapeHtml(sum.title)}</span><span class="eco-banner-sub">${escapeHtml(parts.join(' · '))}</span>${stale}</div>${bannerButton()}</div>`;
}

function doctorRows(checks) {
  return checks.map(c => {
    const view = statusView(c.status);
    return `<tr class="${c.status === 'warning' ? 'is-warn' : c.status === 'error' ? 'is-bad' : ''}">
      <th scope="row" class="mono eco-name">${escapeHtml(c.name)}</th>
      <td>${escapeHtml(categoryText(c.category)) || '<span class="dim">—</span>'}</td>
      <td>${statusPill(view.tone, view.text)}</td>
      <td class="eco-details">${escapeHtml(c.details)}</td>
      <td>${c.recommendation ? escapeHtml(c.recommendation) : '<span class="dim">—</span>'}</td>
    </tr>`;
  }).join('');
}

function doctorBody(entry) {
  if (!entry.data) {
    return entry.status === 'error' ? stateError(entry.error, 'No se pudo cargar el diagnóstico') : stateLoading('Cargando el diagnóstico del sistema…');
  }
  const checks = sortChecks(entry.data.checks);
  if (checks.length === 0) return stateEmpty('Sin diagnósticos', 'Axiom Doctor no devolvió ninguna comprobación.', 'pulse');
  return tableWrap(`<table class="data-table eco-doctor"><thead><tr><th>Componente</th><th>Categoría</th><th>Estado</th><th>Detalle</th><th>Recomendación</th></tr></thead><tbody>${doctorRows(checks)}</tbody></table>`, 'Comprobaciones de Axiom Doctor');
}

function renderDoctor(force = false) {
  const entry = state.overview.doctor;
  if (!entry || (entry === shownDoctor && force !== true)) return;
  shownDoctor = entry;
  $('#eco-health').innerHTML = banner(entry);
  $('#eco-doctor-body').innerHTML = doctorBody(entry);
  const total = entry.data && Array.isArray(entry.data.checks) ? entry.data.checks.length : 0;
  $('#eco-doctor-count').textContent = entry.data ? plural(total, 'comprobación', 'comprobaciones') : '';
}

async function diagnose() {
  if (diagnosing) return;
  diagnosing = true;
  renderDoctor(true);
  try {
    await refreshOverview(['doctor']);
  } finally {
    diagnosing = false;
    renderDoctor(true);
  }
}

// ---- Topología ---------------------------------------------------------------------------

function roleItem(id, role) {
  const policy = role.gate_policy || 'blocking';
  const tech = (role.tech || []).join(', ');
  const repos = (role.repositories || []).join(', ') || '.';
  const name = role.name && role.name !== id ? `<span class="eco-role-name">${escapeHtml(role.name)}</span>` : '';
  return `<li class="eco-role">
    <div class="eco-role-head"><span class="mono eco-role-id">${escapeHtml(id)}</span>${name}<span class="eco-policy is-${escapeHtml(policy)}">${escapeHtml(policy)}</span></div>
    <div class="eco-role-meta"><span class="dim">Repositorios</span><code>${escapeHtml(repos)}</code>${tech ? `<span class="dim">Tecnologías</span><span>${escapeHtml(tech)}</span>` : ''}</div>
  </li>`;
}

function topologyHtml(ws) {
  const status = ws.compliant ? statusPill('ok', 'Conforme') : statusPill('warn', ws.is_configured ? 'No conforme' : 'Sin configurar');
  const note = !ws.compliant && ws.message ? `<p class="eco-note">${escapeHtml(ws.message)}</p>` : '';
  const roles = ws.roles ? Object.entries(ws.roles).sort(([a], [b]) => a.localeCompare(b)) : [];
  const list = roles.length === 0
    ? stateEmpty('No hay roles declarados', 'Declara roles en axiom.yaml para ver aquí su política y sus repositorios.', 'users')
    : `<ul class="eco-roles" aria-label="Roles y repositorios">${roles.map(([id, r]) => roleItem(id, r)).join('')}</ul>`;
  return `<div class="eco-pad">
    <dl class="eco-dl">
      <dt>Workspace</dt><dd class="mono">${escapeHtml(ws.name || '—')}</dd>
      <dt>Topología</dt><dd class="mono">${escapeHtml(ws.topology || '—')}</dd>
      <dt>Repo de specs</dt><dd class="mono">${escapeHtml(ws.specs_repository || 'openspec/')}</dd>
      <dt>Estado</dt><dd>${status}</dd>
      ${branchPolicyRows(ws)}
    </dl>${note}
    <h3 class="eco-sub">Roles</h3>${list}${repoPolicyHtml(ws)}
  </div>`;
}

async function loadTopology() {
  const body = $('#eco-topology-body');
  try {
    body.innerHTML = topologyHtml(await api('/api/workspace'));
  } catch (err) {
    console.error(err);
    body.innerHTML = `<div class="eco-pad">${stateError(err.message, 'No se pudo cargar la topología')}</div>`;
  }
}

// ---- Modelos -----------------------------------------------------------------------------

async function loadModels() {
  const body = $('#eco-models-body');
  try {
    const data = (await api('/api/ecosystem/models')) || {};
    $('#eco-persona').textContent = data.active_persona || '—';
    const list = Array.isArray(data.assignments) ? data.assignments.slice() : [];
    if (list.length === 0) {
      body.innerHTML = `<div class="eco-pad">${stateEmpty('Sin asignaciones explícitas', 'No se detectaron asignaciones de modelos en opencode.json. Se usan los presets predeterminados de Axiom.', 'cube')}</div>`;
      return;
    }
    list.sort((a, b) => String(a.agent).localeCompare(String(b.agent)) || String(a.role).localeCompare(String(b.role)));
    const rows = list.map(m => `<tr>
      <td class="mono">${escapeHtml(m.agent)}</td>
      <td class="mono">${escapeHtml(m.role)}</td>
      <td class="mono">${escapeHtml(m.model)}</td>
      <td>${m.reasoning ? escapeHtml(m.reasoning) : '<span class="dim">estándar</span>'}</td>
    </tr>`).join('');
    body.innerHTML = tableWrap(`<table class="data-table"><thead><tr><th>Agente</th><th>Rol</th><th>Modelo</th><th>Razonamiento</th></tr></thead><tbody>${rows}</tbody></table>`, 'Modelos asignados por agente y rol');
  } catch (err) {
    console.error(err);
    body.innerHTML = `<div class="eco-pad">${stateError(err.message, 'No se pudieron cargar los modelos')}</div>`;
  }
}

// ---- Respaldos ---------------------------------------------------------------------------

function backupItem(b) {
  const when = formatDateTime(b.created);
  const size = formatBytes(b.size_bytes);
  const files = Array.isArray(b.files) && b.files.length > 0 ? plural(b.files.length, 'archivo', 'archivos') : '';
  const meta = [when, size, files].filter(Boolean).map(escapeHtml);
  return `<li class="eco-backup">
    <div class="eco-backup-main">
      <span class="eco-backup-title">${escapeHtml(b.description || 'Sin descripción')}${b.pinned ? ` ${badge('accent', 'Fijado', 'pin')}` : ''}</span>
      <span class="eco-backup-meta">${meta.join(' · ')}${meta.length ? ' · ' : ''}<code title="Identificador del respaldo">${escapeHtml(b.name)}</code></span>
    </div>
    <button type="button" class="btn btn-sm" data-restore="${escapeHtml(b.name)}" data-eco-lock${busy ? ' disabled' : ''} aria-label="Restaurar el respaldo ${escapeHtml(b.name)}">Restaurar</button>
  </li>`;
}

async function loadBackups() {
  const body = $('#eco-backups-body');
  try {
    backups = ((await api('/api/ecosystem/backups')) || []).slice();
    backups.sort((a, b) => String(b.created).localeCompare(String(a.created)));
    $('#eco-backups-count').textContent = plural(backups.length, 'respaldo', 'respaldos');
    body.innerHTML = backups.length === 0
      ? `<div class="eco-pad">${stateEmpty('Sin respaldos', 'Aún no hay respaldos en ~/.axiom/backups. Crea uno antes de hacer cambios grandes.', 'archive')}</div>`
      : `<ul class="eco-backups" aria-label="Respaldos disponibles">${backups.map(backupItem).join('')}</ul>`;
  } catch (err) {
    console.error(err);
    $('#eco-backups-count').textContent = '';
    body.innerHTML = `<div class="eco-pad">${stateError(err.message, 'No se pudieron cargar los respaldos')}</div>`;
  }
}

// ---- Preferencias personales de worktree -------------------------------------------------

function showWorktreePrefsStatus(message, tone = '') {
  const status = $('#worktree-prefs-status');
  status.textContent = message;
  status.className = `field-hint${tone ? ` text-${tone}` : ''}`;
}

function renderWorktreePrefs(data) {
  const saved = data.worktree_prefs || {};
  const effective = data.effective_worktree_prefs || {};
  const root = $('#input-worktree-prefs-root');
  $('#select-worktree-prefs-mode').value = saved.mode || '';
  root.value = saved.root || '';
  if (effective.root) root.placeholder = `por defecto: ${effective.root}`;
  $('#worktree-prefs-effective').textContent = effectivePrefsText(effective);
}

// loadWorktreePrefs lee las preferencias del proyecto activo. Un proyecto que no está
// en el Hub (404) no tiene dónde guardarlas: se dice con la orden que lo registra, que
// ya viene en el mensaje del servidor, y no se ofrece el formulario.
async function loadWorktreePrefs() {
  const stateBox = $('#eco-worktree-state');
  const form = $('#form-worktree-prefs');
  try {
    renderWorktreePrefs(await api('/api/projects/active/worktree-prefs'));
    stateBox.innerHTML = '';
    form.hidden = false;
  } catch (err) {
    console.error(err);
    form.hidden = true;
    stateBox.innerHTML = err.status === 404
      ? stateEmpty('El proyecto activo no está registrado en el Hub', err.message, 'folder')
      : stateError(err.message, 'No se pudieron leer las preferencias de worktree');
  }
}

// saveWorktreePrefs envía ambas preferencias (un campo vacío retira la preferencia).
// Guardar es un botón porque lo ejecuta el servidor (D5); después se refresca el
// selector de proyecto, que lleva las mismas preferencias.
async function saveWorktreePrefs() {
  const restore = setBusy($('#btn-save-worktree-prefs'), 'Guardando…');
  showWorktreePrefsStatus('');
  try {
    const data = await api('/api/projects/active/worktree-prefs', {
      method: 'PUT',
      body: { mode: $('#select-worktree-prefs-mode').value, root: $('#input-worktree-prefs-root').value.trim() }
    });
    renderWorktreePrefs(data);
    showWorktreePrefsStatus('Preferencias guardadas en tu Hub.', 'ok');
    toast('Preferencias de worktree guardadas');
    await loadProjects();
  } catch (err) {
    showWorktreePrefsStatus(`No se pudieron guardar las preferencias: ${err.message}`, 'bad');
    toast('No se pudieron guardar las preferencias de worktree', 'bad');
  } finally {
    restore();
  }
}

export async function loadEcosystem() {
  // El Doctor lo refresca la caché (overview.js) junto con Actualizar.
  await Promise.all([loadTopology(), loadModels(), loadBackups(), loadWorktreePrefs()]);
}

// ---- Última operación ----------------------------------------------------------------------

function renderLast({ kind, at, running = false, outcome = null, text = '' }) {
  const box = $('#eco-last');
  box.hidden = false;
  const pill = running ? statusPill('neutral', 'en curso') : statusPill(outcome.tone, outcome.text);
  const tone = running ? '' : outcome.tone === 'ok' ? 'ok' : outcome.tone === 'bad' ? 'error' : '';
  box.querySelector('[data-role="summary"]').innerHTML = `<span class="eco-last-title">Última operación</span><code>${escapeHtml(OPERATION_LABEL[kind])}</code>${pill}<span class="eco-last-time">${escapeHtml(formatDateTime(at))}</span>`;
  const out = box.querySelector('[data-role="out"]');
  out.textContent = text;
  if (tone) out.setAttribute('data-state', tone);
  else out.removeAttribute('data-state');
  box.open = true;
}

function announce(text) {
  const live = $('#eco-announce');
  if (live) live.textContent = text;
}

// runOperation ejecuta una operación del servidor: bloquea el resto, muestra el
// progreso, deja el resultado en «Última operación» y refresca lo que pudo cambiar.
async function runOperation({ kind, button, busyLabel, progress, run, present, presentError, refresh }) {
  if (busy) return;
  setLocked(true);
  const restore = setBusy(button, busyLabel);
  const at = new Date().toISOString();
  renderLast({ kind, at, running: true, text: progress });
  announce(`${busyLabel} ${progress}`);
  let outcome;
  let text;
  try {
    const data = await run();
    outcome = operationOutcome(kind, data, false);
    text = present(data);
  } catch (err) {
    outcome = operationOutcome(kind, err.data, true);
    text = presentError(err);
  }
  restore();
  setLocked(false);
  renderLast({ kind, at, outcome, text });
  announce(`Operación ${OPERATION_LABEL[kind]} terminada: ${outcome.text}.`);
  await refresh();
  return outcome;
}

const refreshAfterChange = () => Promise.all([loadBackups(), refreshOverview(['doctor'])]);

function runSync(button) {
  return runOperation({
    kind: 'sync', button, busyLabel: 'Sincronizando…', progress: 'Ejecutando axiom sync --scope=workspace…',
    run: () => api('/api/ecosystem/sync', { body: { scope: 'workspace' } }),
    present: data => (data.output || []).join('\n') || data.message || 'Sincronización terminada.',
    presentError: err => [((err.data && err.data.output) || []).join('\n'), err.message].filter(Boolean).join('\n'),
    refresh: refreshAfterChange
  });
}

function runUpgrade(button) {
  const channel = $('#select-eco-channel').value || 'stable';
  return runOperation({
    kind: 'upgrade', button, busyLabel: 'Actualizando…', progress: `Comprobando y actualizando herramientas en el canal ${channel}…`,
    run: () => api('/api/ecosystem/upgrade', { body: { channel } }),
    present: data => formatEcosystemUpgradeSequence(data),
    // El upgrade puede responder 500 con las fases en el cuerpo: se presentan igualmente.
    presentError: err => (err.data && err.data.phases ? formatEcosystemUpgradeSequence(err.data) : err.message),
    refresh: refreshAfterChange
  });
}

async function runBackup(button) {
  if (busy) return;
  const description = await promptDialog({
    title: 'Crear respaldo',
    message: 'Se guardará un snapshot de la configuración de Axiom en ~/.axiom/backups. Descripción (opcional):',
    confirmLabel: 'Crear respaldo',
    placeholder: 'Antes de actualizar las herramientas'
  });
  if (description === null) return;
  const outcome = await runOperation({
    kind: 'backup', button, busyLabel: 'Creando…', progress: 'Creando el respaldo…',
    run: () => api('/api/ecosystem/backups/create', { body: { description: description.trim() } }),
    present: data => `Respaldo creado con el identificador ${data.name}${data.description ? `\nDescripción: ${data.description}` : ''}`,
    presentError: err => `No se pudo crear el respaldo: ${err.message}`,
    refresh: loadBackups
  });
  if (outcome) toast(outcome.tone === 'bad' ? 'No se pudo crear el respaldo' : 'Respaldo creado', outcome.tone === 'bad' ? 'bad' : 'ok');
}

// restoreMessage dice qué pasará: se sustituyen archivos y lo posterior se pierde.
function restoreMessage(b) {
  const when = formatDateTime(b.created);
  const files = Array.isArray(b.files) && b.files.length > 0 ? ` (${plural(b.files.length, 'archivo', 'archivos')})` : '';
  return `Vas a restaurar el respaldo ${b.name}${when ? `, creado el ${when}` : ''}. Los archivos actuales que cubre${files} se sustituirán por los guardados en él y se perderán los cambios hechos en ellos desde entonces. Esta acción no se deshace desde aquí: si quieres conservar el estado actual, crea antes un respaldo.`;
}

async function restoreBackup(button) {
  const name = button.getAttribute('data-restore');
  const b = backups.find(item => item.name === name) || { name };
  const ok = await confirmDialog({ title: 'Restaurar respaldo', message: restoreMessage(b), confirmLabel: 'Restaurar respaldo', danger: true });
  if (!ok) return;
  const outcome = await runOperation({
    kind: 'restore', button, busyLabel: 'Restaurando…', progress: `Restaurando el respaldo ${name}…`,
    run: () => api('/api/ecosystem/backups/restore', { body: { name } }),
    present: data => (data.output || []).join('\n') || data.message || `Respaldo ${name} restaurado.`,
    presentError: err => `No se pudo restaurar el respaldo ${name}: ${err.message}`,
    refresh: refreshAfterChange
  });
  if (outcome) toast(outcome.tone === 'bad' ? 'No se pudo restaurar el respaldo' : 'Respaldo restaurado', outcome.tone === 'bad' ? 'bad' : 'ok');
}

export function initEcosystem() {
  $('#btn-eco-sync').addEventListener('click', event => runSync(event.currentTarget));
  $('#btn-eco-upgrade').addEventListener('click', event => runUpgrade(event.currentTarget));
  $('#btn-eco-create-backup').addEventListener('click', event => runBackup(event.currentTarget));
  $('#form-worktree-prefs').addEventListener('submit', event => {
    event.preventDefault();
    saveWorktreePrefs();
  });
  $('#eco-backups-body').addEventListener('click', event => {
    const button = event.target.closest('[data-restore]');
    if (button && !busy) restoreBackup(button);
  });
  $('#eco-health').addEventListener('click', event => {
    if (event.target.closest('#btn-eco-diagnose')) diagnose();
  });
  on('overview', () => renderDoctor());
  renderDoctor();
}
