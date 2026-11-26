// Integraciones (#/integraciones, prov:WEB:3 a WEB:7): configura desde el dashboard los plugins
// de integración con herramientas externas (D-10).
//
// - La pantalla es genérica: pinta un bloque por manifiesto de GET /api/integrations y no conoce
//   ningún plugin. Los controles salen del catálogo de ajustes (controlHtml/collectChanges de
//   core/settings.js), de modo que tipo, vocabulario, obligatoriedad y límites son los del servidor.
// - Guardar es un único PUT /api/settings con la revisión leída. Un 409 ofrece recargar y
//   conserva lo tecleado; con writable:false los controles se deshabilitan con el motivo.
// - El PAT es de solo entrada: type="password", autocomplete="off", se vacía tras guardar y no
//   entra nunca en el lote de ajustes ni en el almacenamiento del navegador. Se envía solo a
//   POST /api/integrations/secret.
// - Probar conexión y Descubrir operan con lo guardado: se desactivan con cambios sin guardar.
// - Todo texto se inserta con escapeHtml o textContent; los enlaces externos, con safeExternalURL.

import { api, escapeHtml } from '../core/api.js';
import { $, confirmDialog, setBusy, stateEmpty, stateError, toast } from '../core/components.js';
import {
  candidatesHtml, datalistHtml, discoveryHtml, discoveryOptions, errorBoxHtml, identitiesHtml,
  isFreeText, noStoreGuidance, notFoundInTracker, secretRowHtml, sourceView
} from '../core/integrations.js';
import { collectChanges, controlHtml, errorView, settingId } from '../core/settings.js';

let current = null; // último GET /api/integrations pintado
const discoveries = new Map(); // plugin → opciones descubiertas (sugerencias), solo en memoria

function announce(text) {
  const region = $('#integrations-announce');
  if (region) region.textContent = text;
}

function statusLine(html) {
  const box = $('#integrations-status');
  if (box) box.innerHTML = html;
}

function allEntries() {
  return current ? current.plugins.flatMap(p => p.entries) : [];
}

function typedValues() {
  const values = {};
  document.querySelectorAll('#form-integrations [data-key]').forEach(el => { values[el.dataset.key] = el.value; });
  return values;
}

function hasUnsavedChanges() {
  return current !== null && Object.keys(collectChanges(allEntries(), typedValues())).length > 0;
}

// refreshActionState desactiva «Probar conexión» y «Descubrir» con cambios sin guardar: operan
// con lo guardado en axiom.yaml, nunca con lo tecleado.
function refreshActionState() {
  const dirty = hasUnsavedChanges();
  document.querySelectorAll('[data-connect], [data-discover]').forEach(btn => { btn.disabled = dirty; });
  const hint = $('#integrations-dirty');
  if (hint) hint.hidden = !dirty;
}

// ---- Pintado -----------------------------------------------------------------------------

function documentNotes(data) {
  const parts = [];
  if (!data.writable) {
    parts.push(`<div class="alert alert-warn" role="status"><p><strong>Este axiom.yaml no se puede editar desde aquí.</strong> ${escapeHtml(data.writeBlockedReason || '')}</p><p>Los controles quedan deshabilitados: edítalo a mano. El PAT sí se puede guardar, porque no vive en axiom.yaml.</p></div>`);
  }
  if (Array.isArray(data.documentErrors) && data.documentErrors.length) {
    parts.push(`<div class="alert alert-warn" role="status"><p>El documento tiene errores de validación.</p><ul class="settings-violations">${data.documentErrors.map(e => `<li>${escapeHtml(e)}</li>`).join('')}</ul></div>`);
  }
  if (data.workItems) {
    parts.push(data.workItems.applicable
      ? `<p class="eco-note">El alta de cambios desde un work item está disponible (${escapeHtml(data.workItems.plugin || '')}).</p>`
      : `<p class="eco-note">Alta desde un work item no disponible${data.workItems.reason ? `: ${escapeHtml(data.workItems.reason)}` : '.'}</p>`);
  }
  return parts.join('');
}

function pluginHtml(plugin, index, writable) {
  const m = plugin.manifest;
  const id = `int-${index}`;
  const fields = plugin.entries.map((entry, i) => controlHtml(entry, `${settingId(entry.key)}-${index}-${i}`, { disabled: !writable })).join('');
  const secrets = (m.secrets || []).map((spec, i) => {
    const status = (plugin.secrets || []).find(s => s.name === spec.name) || { name: spec.name, source: 'missing', envVar: '' };
    return secretRowHtml(m.id, spec, status, { input: `${id}-secret-${i}`, note: `${id}-secret-${i}-note` });
  }).join('');
  const state = plugin.enabled
    ? '<span class="badge badge-ok">Activo</span>'
    : `<span class="badge badge-neutral">${plugin.declared ? 'Desactivado' : 'Sin configurar'}</span>`;
  const identityError = plugin.identityError ? `<p class="alert alert-warn" role="status">${escapeHtml(plugin.identityError)}</p>` : '';
  return `<section class="card card-flush int-plugin" data-plugin-block data-plugin="${escapeHtml(m.id)}" aria-labelledby="${id}-h">
    <div class="card-head"><h2 class="card-title-h" id="${id}-h">${escapeHtml(m.title || m.id)}</h2><span class="badge badge-mono">${escapeHtml(m.id)}</span> ${state}</div>
    <div class="eco-pad">
      <p class="eco-note">Desactivar el plugin (<code>enabled: false</code>) no borra las operaciones pendientes de la outbox: quedan pendientes (<code>plugin_not_configured</code>) hasta que lo actives de nuevo.</p>
      <h3 class="eco-sub">Configuración del proyecto</h3>
      <div class="field-grid">${fields || '<p class="dim">Este plugin no declara configuración.</p>'}</div>
      <h3 class="eco-sub">Credenciales (tuyas, compartidas por todos tus proyectos)</h3>
      ${secrets || '<p class="dim">Este plugin no usa credenciales.</p>'}
      <h3 class="eco-sub">Identidad en el tracker</h3>
      ${identityError}<div data-identities>${identitiesHtml(plugin.identities)}</div>
      <div class="eco-wt-actions">
        <button type="button" class="btn btn-sm" data-connect>Probar conexión</button>
        <button type="button" class="btn btn-sm" data-discover>Descubrir</button>
        <span class="eco-note">Operan con lo guardado en axiom.yaml.</span>
      </div>
      <div data-plugin-status role="status" aria-live="polite"></div>
      <div data-discovery-box></div>
    </div>
  </section>`;
}

function renderScreen(data) {
  current = data;
  $('#integrations-config-path').textContent = data.configPath || 'axiom.yaml';
  if (!Array.isArray(data.plugins) || data.plugins.length === 0) {
    $('#integrations-body').innerHTML = `${documentNotes(data)}${stateEmpty('No hay plugins de integración en este binario', 'El catálogo de plugins compilados está vacío.', 'info')}`;
    return;
  }
  const plugins = data.plugins.map((p, i) => pluginHtml(p, i, data.writable)).join('');
  const save = data.writable
    ? '<div class="eco-wt-actions"><button type="submit" id="btn-save-integrations" class="btn btn-primary">Guardar configuración</button><span class="eco-note">Se escribe en axiom.yaml y queda sin confirmar en git.</span></div>'
    : '';
  $('#integrations-body').innerHTML = `${documentNotes(data)}
    <p class="eco-note">Las plantillas de rama no se editan aquí: están en <a href="#/ajustes">Ajustes del proyecto</a>.</p>
    <p id="integrations-dirty" class="alert alert-warn" role="status" hidden>Hay cambios sin guardar: guárdalos para poder probar la conexión o descubrir.</p>
    <form id="form-integrations" novalidate>${plugins}${save}</form>
    <div id="integrations-status" role="status" aria-live="polite"></div>`;
  for (const [plugin, options] of discoveries) applyDiscovery(plugin, options, null);
  refreshActionState();
}

// ---- Carga -------------------------------------------------------------------------------

// loadIntegrations lee el estado. Con keepTyped, lo tecleado que difiera de lo recién leído se
// vuelve a poner en su control (un 409 ofrece recargar sin perder lo escrito).
export async function loadIntegrations({ keepTyped = false } = {}) {
  const body = $('#integrations-body');
  const typed = keepTyped ? typedValues() : null;
  try {
    renderScreen(await api('/api/integrations'));
    if (typed) {
      document.querySelectorAll('#form-integrations [data-key]').forEach(el => {
        if (Object.prototype.hasOwnProperty.call(typed, el.dataset.key)) el.value = typed[el.dataset.key];
      });
      refreshActionState();
    }
  } catch (err) {
    console.error(err);
    current = null;
    body.innerHTML = err.status === 404
      ? stateEmpty('Este proyecto no tiene axiom.yaml', err.message, 'file')
      : stateError(err.message, 'No se pudieron leer las integraciones');
  }
}

// ---- Guardar la configuración (un único PUT /api/settings) --------------------------------

async function saveConfig() {
  if (!current) return;
  const changes = collectChanges(allEntries(), typedValues());
  if (Object.keys(changes).length === 0) {
    statusLine('<p class="field-hint">No hay cambios que guardar.</p>');
    announce('No hay cambios que guardar');
    return;
  }
  const restore = setBusy($('#btn-save-integrations'), 'Guardando…');
  statusLine('');
  try {
    const result = await api('/api/settings', { method: 'PUT', body: { revision: current.revision, changes } });
    await loadIntegrations();
    statusLine(`<p class="field-hint text-ok">${result.changed ? 'Configuración guardada en axiom.yaml. Solo afecta a cambios nuevos.' : 'No había nada que cambiar en axiom.yaml.'}</p>`);
    announce('Configuración guardada');
    toast('Configuración de integraciones guardada');
  } catch (err) {
    // Sin repintar: lo tecleado se conserva. Un 409 (revision_mismatch) ofrece recargar.
    statusLine(errorView(err));
    announce('No se pudo guardar la configuración');
    toast('No se pudo guardar la configuración', 'bad');
  } finally {
    restore();
  }
}

// ---- PAT: guardar y eliminar --------------------------------------------------------------

function showSecretResult(row, status, shadowed) {
  const view = sourceView(status, { shadowed });
  const badge = row.querySelector('[data-secret-source]');
  badge.className = `badge badge-${view.tone}`;
  badge.textContent = view.label;
  row.querySelector('[data-secret-note]').textContent = view.note;
}

function secretError(row, err) {
  const data = (err && err.data) || {};
  const note = row.querySelector('[data-secret-note]');
  note.textContent = data.code === 'no_writable_store' ? noStoreGuidance(data.envVar) : (err.message || 'No se pudo completar la operación');
  toast('No se pudo completar la operación con el PAT', 'bad');
}

async function saveSecret(button) {
  const row = button.closest('[data-secret-row]');
  const input = row.querySelector('[data-secret-input]');
  if (input.value.trim() === '') {
    row.querySelector('[data-secret-note]').textContent = 'Escribe el PAT antes de guardarlo.';
    return;
  }
  const restore = setBusy(button, 'Guardando…');
  try {
    const result = await api('/api/integrations/secret', { body: { plugin: row.dataset.plugin, name: row.dataset.secret, value: input.value } });
    showSecretResult(row, result, result.shadowedByEnv);
    toast('PAT guardado');
    announce('PAT guardado');
  } catch (err) {
    secretError(row, err);
  } finally {
    input.value = ''; // el PAT no se conserva en el campo, haya ido bien o mal
    restore();
  }
}

async function deleteSecret(button) {
  const row = button.closest('[data-secret-row]');
  const ok = await confirmDialog({
    title: 'Eliminar el PAT',
    message: `Vas a eliminar el PAT «${row.dataset.secret}» del almacén de tu equipo. Es tuyo y lo comparten todos tus proyectos: dejarán de poder usarlo hasta que lo guardes de nuevo o definas la variable de entorno.`,
    confirmLabel: 'Eliminar PAT',
    danger: true
  });
  if (!ok) return;
  const restore = setBusy(button, 'Eliminando…');
  try {
    const result = await api('/api/integrations/secret/delete', { body: { plugin: row.dataset.plugin, name: row.dataset.secret, confirm: true } });
    showSecretResult(row, result, result.shadowedByEnv);
    toast('PAT eliminado');
    announce('PAT eliminado');
  } catch (err) {
    secretError(row, err);
  } finally {
    restore();
  }
}

// ---- Probar conexión ----------------------------------------------------------------------

async function connect(block, user, button) {
  const box = block.querySelector('[data-plugin-status]');
  const restore = button ? setBusy(button, 'Probando…') : () => {};
  box.innerHTML = '';
  try {
    const body = user ? { plugin: block.dataset.plugin, user } : { plugin: block.dataset.plugin };
    const result = await api('/api/integrations/connect', { body });
    const who = result.identity.displayName || result.identity.id;
    const replaced = result.replaced ? ` Sustituye a ${escapeHtml(result.replaced.displayName || result.replaced.id)}.` : '';
    box.innerHTML = `<div class="alert alert-ok" role="status"><p>Conexión correcta. Identidad guardada: <strong>${escapeHtml(who)}</strong>.${replaced}</p></div>`;
    block.querySelector('[data-identities]').innerHTML = identitiesHtml([result.identity]);
    announce('Conexión correcta');
  } catch (err) {
    const data = err.data || {};
    if (Array.isArray(data.candidates) && data.candidates.length) {
      box.innerHTML = `${data.code === 'identity_user_unknown' ? errorBoxHtml(err) : ''}${candidatesHtml(block.dataset.plugin, data.candidates, `identity-${block.dataset.plugin}`)}`;
    } else {
      box.innerHTML = errorBoxHtml(err);
    }
    announce('La conexión no se pudo completar');
  } finally {
    restore();
    refreshActionState();
  }
}

// ---- Descubrir ----------------------------------------------------------------------------

// applyDiscovery pinta el panel de solo lectura y deja el datalist como sugerencia en los
// campos de texto libre. Un valor que no aparece se conserva y se marca.
function applyDiscovery(pluginId, options, discovery) {
  const block = document.querySelector(`[data-plugin-block][data-plugin="${CSS.escape(pluginId)}"]`);
  if (!block) return;
  const listId = `int-dl-${settingId(pluginId)}`;
  const box = block.querySelector('[data-discovery-box]');
  if (discovery) box.innerHTML = discoveryHtml(discovery, options);
  else if (!box.firstChild) return;
  const old = block.querySelector(`#${listId}`);
  if (old) old.remove();
  const holder = document.createElement('div');
  holder.innerHTML = datalistHtml(listId, options); // opciones escapadas con escapeHtml
  block.append(holder.firstElementChild);
  block.querySelectorAll('[data-not-found]').forEach(el => el.remove());
  const plugin = current && current.plugins.find(p => p.manifest.id === pluginId);
  for (const entry of plugin ? plugin.entries : []) {
    if (!isFreeText(entry)) continue;
    const input = Array.from(block.querySelectorAll('input[data-key]')).find(el => el.dataset.key === entry.key);
    if (!input) continue;
    input.setAttribute('list', listId);
    if (notFoundInTracker(input.value, options)) {
      const flag = document.createElement('span');
      flag.className = 'badge badge-warn';
      flag.dataset.notFound = '';
      flag.textContent = 'no encontrado en el tracker';
      input.insertAdjacentElement('afterend', flag);
    }
  }
}

async function discover(block, button) {
  const box = block.querySelector('[data-plugin-status]');
  const restore = setBusy(button, 'Descubriendo…');
  box.innerHTML = '';
  try {
    const discovery = await api('/api/integrations/discover', { body: { plugin: block.dataset.plugin } });
    const options = discoveryOptions(discovery);
    discoveries.set(block.dataset.plugin, options);
    applyDiscovery(block.dataset.plugin, options, discovery);
    announce('Descubrimiento completado');
  } catch (err) {
    box.innerHTML = errorBoxHtml(err);
    announce('El descubrimiento no se pudo completar');
  } finally {
    restore();
    refreshActionState();
  }
}

// ---- Entrada de la pantalla ---------------------------------------------------------------

export function initIntegrations() {
  const view = $('#view-integraciones');
  view.addEventListener('submit', event => {
    event.preventDefault();
    if (event.target.id === 'form-integrations') saveConfig();
  });
  view.addEventListener('input', event => {
    if (event.target.closest('#form-integrations [data-key]')) refreshActionState();
  });
  view.addEventListener('change', event => {
    if (event.target.closest('#form-integrations [data-key]')) refreshActionState();
  });
  view.addEventListener('click', event => {
    if (event.target.closest('[data-settings-reload]')) { loadIntegrations({ keepTyped: true }); return; }
    const save = event.target.closest('[data-secret-save]');
    if (save) { saveSecret(save); return; }
    const del = event.target.closest('[data-secret-delete]');
    if (del) { deleteSecret(del); return; }
    const block = event.target.closest('[data-plugin-block]');
    if (!block) return;
    const probe = event.target.closest('[data-connect]');
    if (probe) { connect(block, '', probe); return; }
    const find = event.target.closest('[data-discover]');
    if (find) { discover(block, find); return; }
    const confirmId = event.target.closest('[data-identity-confirm]');
    if (confirmId) {
      const chosen = block.querySelector('[data-identity-choice]:checked');
      if (chosen) connect(block, chosen.value, confirmId);
    }
  });
}
