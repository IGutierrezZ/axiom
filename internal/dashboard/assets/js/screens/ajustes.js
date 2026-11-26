// Ajustes del proyecto (#/ajustes, C-06-e): edita desde el dashboard las claves de axiom.yaml
// que el catálogo declara editables, y las preferencias personales del Hub.
//
// - La pantalla se genera del catálogo que devuelve GET /api/settings: no conoce ninguna clave.
//   La lógica pura (agrupar, pintar controles, calcular el parche, enlace al editor, errores)
//   vive en core/settings.js, que se ejecuta en las pruebas con Node.
// - Guardar es un botón (D5): envía un lote `{revision, changes}` con solo lo que cambió. Si el
//   servidor lo rechaza (409 por una edición externa, 422 por la validación) se dice por qué y
//   NO se repinta el formulario, de modo que lo tecleado no se pierde.
// - Los cambios quedan sin confirmar en el árbol de trabajo: confirmarlos con git es decisión
//   de la persona. Editar solo afecta a cambios nuevos.
// - «Preferencias personales» (autofetch y modo/raíz de worktree) viven en el Hub, no en
//   axiom.yaml. Cada bloque carga y falla por separado: uno que falle no vacía la página.

import { api, escapeHtml } from '../core/api.js';
import { effectivePrefsText } from '../core/branches.js';
import { $, copyText, setBusy, stateEmpty, stateError, toast } from '../core/components.js';
import { collectChanges, controlHtml, errorView, groupBySection, pathActionsHtml, readOnlyHtml, settingId } from '../core/settings.js';
import { loadProjects } from '../projects.js';

let current = null; // último GET /api/settings pintado

function announce(text) {
  const region = $('#settings-announce');
  if (region) region.textContent = text;
}

function statusLine(html) {
  $('#settings-status').innerHTML = html;
}

// ---- Catálogo del proyecto --------------------------------------------------------------

function documentNotes(data) {
  const parts = [];
  if (!data.writable) {
    parts.push(`<div class="alert alert-warn" role="status"><p><strong>Este axiom.yaml no se puede editar desde aquí.</strong> ${escapeHtml(data.writeBlockedReason || '')}</p><p>Edítalo a mano: ${pathActionsHtml(data.configPath)}</p></div>`);
  }
  if (Array.isArray(data.documentErrors) && data.documentErrors.length) {
    parts.push(`<div class="alert alert-warn" role="status"><p>El documento tiene errores de validación${data.writable ? ': corrígelos desde aquí o a mano' : ''}.</p><ul class="settings-violations">${data.documentErrors.map(e => `<li>${escapeHtml(e)}</li>`).join('')}</ul></div>`);
  }
  return parts.join('');
}

function renderSettings(data) {
  current = data;
  $('#settings-config-path').textContent = data.configPath || 'axiom.yaml';
  const disabled = !data.writable;
  const ids = new Map(data.entries.map((e, i) => [e.key, `${settingId(e.key)}-${i}`]));
  const groups = groupBySection(data.entries).map(group => {
    const fields = group.entries.map(entry => (entry.editable
      ? controlHtml(entry, ids.get(entry.key), { disabled })
      : readOnlyHtml(entry, data.configPath))).join('');
    return `<fieldset class="settings-group"><legend class="eco-sub">${escapeHtml(group.label)}</legend><div class="field-grid">${fields}</div></fieldset>`;
  }).join('');
  const topology = `<p class="eco-note">Topología <code>${escapeHtml(data.topology)}</code>${data.iso ? ' · proyecto ISO' : ''}. Archivo: <code>${escapeHtml(data.configPath)}</code> ${pathActionsHtml(data.configPath)}</p>`;
  const save = data.writable
    ? '<div class="eco-wt-actions"><button type="submit" id="btn-save-settings" class="btn btn-primary">Guardar ajustes</button><span class="eco-note">Se escribe en axiom.yaml y queda sin confirmar en git.</span></div>'
    : '';
  $('#settings-body').innerHTML = `${documentNotes(data)}${topology}<form id="form-settings" novalidate>${groups}${save}</form><div id="settings-status" role="status" aria-live="polite"></div>`;
}

// loadCatalog lee el catálogo. Sin axiom.yaml (404 config_missing) se dice y no se ofrece nada.
async function loadCatalog() {
  const body = $('#settings-body');
  try {
    renderSettings(await api('/api/settings'));
  } catch (err) {
    console.error(err);
    current = null;
    body.innerHTML = err.status === 404
      ? stateEmpty('Este proyecto no tiene axiom.yaml', err.message, 'file')
      : stateError(err.message, 'No se pudieron leer los ajustes del proyecto');
  }
}

function typedValues() {
  const values = {};
  document.querySelectorAll('#form-settings [data-key]').forEach(el => { values[el.dataset.key] = el.value; });
  return values;
}

// saveSettings envía solo lo que cambió. Un fallo no repinta el formulario.
async function saveSettings() {
  if (!current) return;
  const changes = collectChanges(current.entries, typedValues());
  if (Object.keys(changes).length === 0) {
    statusLine('<p class="field-hint">No hay cambios que guardar.</p>');
    announce('No hay cambios que guardar');
    return;
  }
  const restore = setBusy($('#btn-save-settings'), 'Guardando…');
  statusLine('');
  try {
    const result = await api('/api/settings', { method: 'PUT', body: { revision: current.revision, changes } });
    renderSettings(result.settings);
    const warnings = Array.isArray(result.warnings) && result.warnings.length
      ? `<div class="alert alert-warn" role="status"><p>El proyecto ya tenía estos avisos de validación, que no impiden guardar:</p><ul class="settings-violations">${result.warnings.map(w => `<li>${escapeHtml(w)}</li>`).join('')}</ul></div>`
      : '';
    statusLine(`<p class="field-hint text-ok">${result.changed ? 'Ajustes guardados en axiom.yaml. Solo afectan a cambios nuevos.' : 'No había nada que cambiar en axiom.yaml.'}</p>${warnings}`);
    announce('Ajustes guardados');
    toast('Ajustes del proyecto guardados');
    loadProjects();
  } catch (err) {
    statusLine(errorView(err));
    announce('No se pudieron guardar los ajustes');
    toast('No se pudieron guardar los ajustes', 'bad');
  } finally {
    restore();
  }
}

// ---- Preferencias personales: autofetch --------------------------------------------------

async function loadPersonal() {
  const stateBox = $('#settings-personal-state');
  const form = $('#form-autofetch');
  try {
    const data = await api('/api/settings/personal');
    $('#input-autofetch').checked = Boolean(data.autofetch);
    stateBox.innerHTML = '';
    form.hidden = false;
  } catch (err) {
    console.error(err);
    form.hidden = true;
    stateBox.innerHTML = err.status === 404
      ? stateEmpty('El proyecto activo no está registrado en el Hub', err.message, 'folder')
      : stateError(err.message, 'No se pudieron leer las preferencias personales');
  }
}

function showAutofetchStatus(message, tone = '') {
  const status = $('#autofetch-status');
  status.textContent = message;
  status.className = `field-hint${tone ? ` text-${tone}` : ''}`;
}

async function saveAutofetch() {
  const restore = setBusy($('#btn-save-autofetch'), 'Guardando…');
  showAutofetchStatus('');
  try {
    const data = await api('/api/settings/personal', { method: 'PUT', body: { autofetch: $('#input-autofetch').checked } });
    $('#input-autofetch').checked = Boolean(data.autofetch);
    showAutofetchStatus('Preferencia guardada en tu Hub.', 'ok');
    toast('Autofetch guardado');
  } catch (err) {
    showAutofetchStatus(`No se pudo guardar el autofetch: ${err.message}`, 'bad');
    toast('No se pudo guardar el autofetch', 'bad');
  } finally {
    restore();
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

// ---- Entrada de la pantalla --------------------------------------------------------------

// loadSettings carga los tres bloques a la vez; cada uno falla por separado.
export async function loadSettings() {
  await Promise.all([loadCatalog(), loadPersonal(), loadWorktreePrefs()]);
}

async function copyPath(button) {
  const ok = await copyText(button.dataset.copyPath || '');
  toast(ok ? 'Ruta copiada al portapapeles' : 'No se pudo copiar la ruta', ok ? 'ok' : 'bad');
}

export function initSettings() {
  const view = $('#view-ajustes');
  view.addEventListener('submit', event => {
    event.preventDefault();
    if (event.target.id === 'form-settings') saveSettings();
    else if (event.target.id === 'form-autofetch') saveAutofetch();
    else if (event.target.id === 'form-worktree-prefs') saveWorktreePrefs();
  });
  view.addEventListener('click', event => {
    if (event.target.closest('[data-settings-reload]')) loadCatalog();
    const copy = event.target.closest('[data-copy-path]');
    if (copy) copyPath(copy);
  });
}
