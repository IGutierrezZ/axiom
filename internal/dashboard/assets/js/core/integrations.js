// Pantalla «Integraciones» (#/integraciones, prov:WEB:7): la lógica pura, separada del DOM.
//
// - Nada aquí conoce un plugin concreto: todo sale del manifiesto que devuelve
//   GET /api/integrations.
// - Todo texto del manifiesto, de axiom.yaml y del tracker se escapa con escapeHtml antes de
//   interpolarlo. Un enlace externo nunca se interpola en una plantilla: se crea con
//   a.href = … y a.textContent = … tras pasar safeExternalURL (D-09).
// - El valor de un PAT nunca pasa por este módulo: solo se habla de su procedencia.

import { escapeHtml } from './api.js';

// Tope de opciones por lista de sugerencias (D-10).
export const MAX_DATALIST_OPTIONS = 500;

// safeExternalURL: solo http o https con host, sin credenciales ni blancos ni caracteres de
// control (equivalente en el cliente del `safeExternalURL` del servidor).
export function safeExternalURL(raw) {
  if (typeof raw !== 'string' || raw === '') return false;
  if (/[\s\u0000-\u001f\u007f]/.test(raw)) return false;
  let url;
  try {
    url = new URL(raw);
  } catch (_) {
    return false;
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') return false;
  if (!url.hostname) return false;
  if (url.username || url.password) return false;
  return true;
}

// externalLink crea un <a> seguro, o null si la URL no es segura. Nunca usa innerHTML.
export function externalLink(href, label) {
  if (!safeExternalURL(href)) return null;
  const a = document.createElement('a');
  a.href = href;
  a.textContent = label;
  a.target = '_blank';
  a.rel = 'noopener noreferrer';
  return a;
}

// workItemsApplicable dice si el alta puede ofrecer el campo del work item: solo con la respuesta
// de GET /api/integrations y `workItems.applicable === true`. Cualquier otra cosa, no.
export function workItemsApplicable(data) {
  return Boolean(data && data.workItems && data.workItems.applicable === true);
}

const SOURCE_LABELS = {
  env: 'Variable de entorno',
  keyring: 'Almacén del sistema',
  missing: 'Sin configurar'
};

// sourceView resume la procedencia de un secreto: etiqueta, tono del distintivo y la nota que
// explica qué hacer. `status` es un SecretStatusDTO; `shadowed` viene de la respuesta de guardar.
export function sourceView(status, { shadowed = false } = {}) {
  const source = status && status.source;
  const label = SOURCE_LABELS[source] || 'Desconocida';
  const tone = source === 'missing' ? 'warn' : source === 'env' || source === 'keyring' ? 'ok' : 'neutral';
  const envVar = (status && status.envVar) || '';
  let note = '';
  if (source === 'env') {
    note = `Se lee de la variable de entorno ${envVar}. ${shadowed ? 'Esa variable tapa el valor guardado en el almacén: para cambiarlo, modifica la variable.' : 'Tiene prioridad sobre el almacén del sistema.'}`;
  } else if (source === 'missing') {
    note = `Guarda el PAT aquí o define la variable de entorno ${envVar}.`;
  }
  return { source, label, tone, note, envVar };
}

// noStoreGuidance es el texto cuando no hay almacén escribible (422 no_writable_store).
export function noStoreGuidance(envVar) {
  return `Este equipo no tiene un almacén de secretos escribible. Define la variable de entorno ${envVar || 'indicada en el manifiesto'} y reinicia el dashboard.`;
}

// secretRowHtml pinta un secreto del manifiesto: procedencia, campo de entrada y acciones. El
// campo del PAT es de solo entrada: sin data-key (no entra en el lote de ajustes), password y
// sin autocompletado; nunca se rellena con un valor.
export function secretRowHtml(pluginId, spec, status, ids) {
  const view = sourceView(status);
  const inputId = escapeHtml(ids.input);
  const noteId = escapeHtml(ids.note);
  const required = spec.required ? ' <span class="badge badge-neutral">obligatorio</span>' : '';
  return `<div class="field settings-field int-secret" data-secret-row data-plugin="${escapeHtml(pluginId)}" data-secret="${escapeHtml(spec.name)}">
    <label for="${inputId}">${escapeHtml(spec.name)}${required}</label>
    <p class="field-hint">${escapeHtml(spec.description || '')}</p>
    <p class="int-source"><span class="badge badge-${view.tone}" data-secret-source>${escapeHtml(view.label)}</span> <span class="eco-note" id="${noteId}" data-secret-note>${escapeHtml(view.note)}</span></p>
    <div class="int-secret-entry">
      <input type="password" id="${inputId}" class="form-input" data-secret-input autocomplete="off" spellcheck="false" aria-describedby="${noteId}" placeholder="Pega aquí el PAT (solo se envía al guardarlo)">
      <button type="button" class="btn btn-primary btn-sm" data-secret-save>Guardar PAT</button>
      <button type="button" class="btn btn-danger btn-sm" data-secret-delete>Eliminar PAT</button>
    </div>
  </div>`;
}

// candidatesHtml pinta la elección de identidad (409 identity_selection_required o 422
// identity_user_unknown): un radio por candidato. El id y el nombre van escapados.
export function candidatesHtml(pluginId, candidates, groupId) {
  const list = Array.isArray(candidates) ? candidates : [];
  if (list.length === 0) return '';
  const items = list.map((c, i) => `<li><label><input type="radio" name="${escapeHtml(groupId)}" value="${escapeHtml(c.id)}" data-identity-choice${i === 0 ? ' checked' : ''}> ${escapeHtml(c.displayName || c.id)}</label></li>`).join('');
  return `<div class="alert alert-warn" role="status" data-identity-picker data-plugin="${escapeHtml(pluginId)}"><p>No se pudo identificarte sola: elige quién eres en el tracker.</p><ul class="int-candidates">${items}</ul><button type="button" class="btn btn-primary btn-sm" data-identity-confirm>Usar esta identidad</button></div>`;
}

// identitiesHtml muestra la identidad persistida de un plugin.
export function identitiesHtml(identities) {
  const list = Array.isArray(identities) ? identities : [];
  if (list.length === 0) return '<p class="eco-note">Sin identidad resuelta todavía. «Probar conexión» la resuelve y la guarda en tu equipo.</p>';
  return `<ul class="int-identities">${list.map(i => `<li><strong>${escapeHtml(i.displayName || i.id)}</strong> <span class="eco-note">${escapeHtml(i.account || '')}${i.resolvedAt ? ` · ${escapeHtml(i.resolvedAt)}` : ''}</span></li>`).join('')}</ul>`;
}

// discoveryOptions reúne los nombres de tipo y de estado descubiertos, sin duplicados y con el
// tope de la lista. Son sugerencias, no restricciones.
export function discoveryOptions(discovery) {
  const names = [];
  const seen = new Set();
  for (const type of (discovery && discovery.item_types) || []) {
    for (const name of [type.name, ...((type.states || []).map(s => s.name))]) {
      if (name && !seen.has(name)) { seen.add(name); names.push(name); }
    }
  }
  return { names: names.slice(0, MAX_DATALIST_OPTIONS), truncated: names.length > MAX_DATALIST_OPTIONS, total: names.length, all: seen };
}

// datalistHtml: la lista de sugerencias de un plugin.
export function datalistHtml(id, options) {
  const opts = options.names.map(n => `<option value="${escapeHtml(n)}"></option>`).join('');
  return `<datalist id="${escapeHtml(id)}">${opts}</datalist>`;
}

// discoveryHtml es el panel de solo lectura del descubrimiento: tipos con sus estados,
// iteraciones y miembros. Nada de aquí es editable.
export function discoveryHtml(discovery, options = discoveryOptions(discovery)) {
  const d = discovery || {};
  const scope = Object.entries(d.scope || {}).filter(([, v]) => v)
    .map(([k, v]) => `<li><span class="eco-note">${escapeHtml(k)}:</span> <code>${escapeHtml(v)}</code></li>`).join('');
  const types = (d.item_types || []).map(t => {
    const states = (t.states || []).map(s => `<li>${escapeHtml(s.name)}${s.category ? ` <span class="badge badge-neutral">${escapeHtml(s.category)}</span>` : ''}</li>`).join('');
    return `<li><strong>${escapeHtml(t.name)}</strong><ul class="int-states">${states || '<li class="dim">sin estados</li>'}</ul></li>`;
  }).join('');
  const iterations = (d.iterations || []).map(i => `<li>${escapeHtml(i.name)}${i.time_frame ? ` <span class="badge badge-neutral">${escapeHtml(i.time_frame)}</span>` : ''}${i.start_date ? ` <span class="eco-note">${escapeHtml(i.start_date)} → ${escapeHtml(i.finish_date || '')}</span>` : ''}</li>`).join('');
  const members = (d.members || []).map(m => `<li>${escapeHtml(m.display_name || m.id)}</li>`).join('');
  const trimmed = options.truncated ? `<p class="eco-note">Las sugerencias se recortan a ${MAX_DATALIST_OPTIONS} de ${options.total} nombres.</p>` : '';
  return `<div class="int-discovery" data-discovery><h4 class="eco-sub">Lo que ofrece el tracker (solo lectura)</h4><ul class="int-scope">${scope}</ul>
    <details open><summary>Tipos y estados (${(d.item_types || []).length})</summary><ul>${types || '<li class="dim">ninguno</li>'}</ul></details>
    <details><summary>Iteraciones (${(d.iterations || []).length})</summary><ul>${iterations || '<li class="dim">ninguna</li>'}</ul></details>
    <details><summary>Miembros (${(d.members || []).length})</summary><ul>${members || '<li class="dim">ninguno</li>'}</ul></details>
    ${trimmed}<p class="eco-note">Los nombres descubiertos se ofrecen como sugerencias en los campos de texto; un valor que no aparece aquí se conserva y se marca.</p></div>`;
}

// isFreeText: campo de texto libre sin formato propio (las sugerencias solo van ahí).
export function isFreeText(entry) {
  return Boolean(entry) && entry.editable && entry.type === 'string' && !(entry.allowed && entry.allowed.length) && !entry.format;
}

// notFoundInTracker: un valor configurado que no figura entre lo descubierto.
export function notFoundInTracker(value, options) {
  const v = value === null || value === undefined ? '' : String(value).trim();
  return v !== '' && !options.all.has(v);
}

// errorBoxHtml presenta un error de /api/integrations* con su guía; si trae envVar la nombra.
export function errorBoxHtml(err) {
  const data = (err && err.data) || {};
  const message = (err && err.message) || data.error || 'La operación ha fallado';
  const guide = data.code === 'no_writable_store'
    ? `<p>${escapeHtml(noStoreGuidance(data.envVar))}</p>`
    : (data.envVar ? `<p class="eco-note">Variable de entorno: ${escapeHtml(data.envVar)}</p>` : '');
  return `<div class="alert alert-bad" role="alert"><p>${escapeHtml(message)}</p>${guide}</div>`;
}
