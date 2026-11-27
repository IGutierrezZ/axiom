// Ajustes del proyecto (#/ajustes, C-06-e): la lógica pura de la pantalla, separada del DOM
// para poder ejecutarla en las pruebas con Node. La pantalla no conoce ninguna clave de
// axiom.yaml: todo sale del catálogo que devuelve GET /api/settings (tipo, vocabulario
// cerrado, si es editable, efecto y valor por defecto).
//
// Todo texto del servidor se escapa aquí antes de interpolarlo en HTML.

import { escapeHtml } from './api.js';

// Etiqueta y orden de las secciones del catálogo. Una sección que el servidor añada y no
// figure aquí se pinta con su nombre y al final.
const SECTION_LABELS = {
  workspace: 'Proyecto',
  git: 'Política de ramas',
  repositories: 'Ajustes por repositorio',
  workflow: 'Carril de trabajo',
  delivery: 'Entrega',
  sdd: 'SDD',
  acceptance_tests: 'Pruebas de aceptación',
  roles: 'Roles',
  technical_context: 'Contexto técnico',
  governance: 'Gobierno',
  integrations: 'Integraciones'
};
const SECTION_ORDER = Object.keys(SECTION_LABELS);

// groupBySection agrupa las entradas por sección conservando el orden del catálogo dentro de
// cada una y el orden de SECTION_ORDER entre ellas.
export function groupBySection(entries) {
  const groups = new Map();
  for (const entry of Array.isArray(entries) ? entries : []) {
    if (!groups.has(entry.section)) groups.set(entry.section, []);
    groups.get(entry.section).push(entry);
  }
  const rank = section => {
    const i = SECTION_ORDER.indexOf(section);
    return i < 0 ? SECTION_ORDER.length : i;
  };
  return [...groups.keys()]
    .sort((a, b) => rank(a) - rank(b))
    .map(section => ({ section, label: SECTION_LABELS[section] || section, entries: groups.get(section) }));
}

// settingId deriva un id de DOM de la clave: una clave con rutas o corchetes no es un id válido.
export function settingId(key) {
  return `set-${String(key).replace(/[^A-Za-z0-9_-]/g, '_')}`;
}

// editableValue es el valor inicial de una entrada tal como lo muestra su control (texto).
function shownValue(entry) {
  const v = entry.value;
  if (v === null || v === undefined) return '';
  if (Array.isArray(v)) return v.join('\n');
  return String(v);
}

function option(value, label, selected) {
  return `<option value="${escapeHtml(value)}"${selected ? ' selected' : ''}>${escapeHtml(label)}</option>`;
}

function defaultLabel(entry) {
  return entry.default ? `Por defecto (${entry.default})` : 'Por defecto';
}

// controlHtml devuelve el control de una entrada editable, con su etiqueta, su descripción,
// el efecto de editarla y su valor por defecto. El vocabulario cerrado es un select con
// exactamente los valores permitidos (más «Por defecto», que retira la clave, cuando no es
// obligatoria, y el valor escrito si no es uno de los permitidos, para poder corregirlo).
export function controlHtml(entry, id = settingId(entry.key), { disabled = false } = {}) {
  const key = escapeHtml(entry.key);
  const off = disabled || !entry.editable ? ' disabled' : '';
  const hintId = `${id}-hint`;
  const attrs = `id="${escapeHtml(id)}" data-key="${key}" data-type="${escapeHtml(entry.type)}" aria-describedby="${escapeHtml(hintId)}"${off}`;
  const current = shownValue(entry);
  let control;
  switch (entry.type) {
    case 'enum': {
      const allowed = Array.isArray(entry.allowed) ? entry.allowed : [];
      const parts = [];
      if (!entry.required) parts.push(option('', defaultLabel(entry), current === ''));
      for (const value of allowed) parts.push(option(value, value, current === value));
      if (current !== '' && !allowed.includes(current)) parts.push(option(current, `${current} (no válido)`, true));
      control = `<select ${attrs} class="form-select">${parts.join('')}</select>`;
      break;
    }
    case 'bool': {
      const value = entry.value === true ? 'true' : entry.value === false ? 'false' : current;
      control = `<select ${attrs} class="form-select">${option('', defaultLabel(entry), value === '')}${option('true', 'Sí', value === 'true')}${option('false', 'No', value === 'false')}${value && value !== 'true' && value !== 'false' ? option(value, `${value} (no válido)`, true) : ''}</select>`;
      break;
    }
    case 'int':
      control = `<input type="number" min="${Number.isInteger(entry.min) ? entry.min : 1}"${Number.isInteger(entry.max) ? ` max="${entry.max}"` : ''} step="1" ${attrs} class="form-input" value="${escapeHtml(current)}">`;
      break;
    case 'stringList':
      control = `<textarea rows="3" ${attrs} class="form-textarea">${escapeHtml(current)}</textarea>`;
      break;
    default:
      control = `<input type="text" ${attrs} class="form-input" value="${escapeHtml(current)}" autocomplete="off">`;
  }
  const hints = [
    entry.description ? `<span>${escapeHtml(entry.description)}</span>` : '',
    entry.effect ? `<span>${escapeHtml(entry.effect)}</span>` : '',
    entry.type === 'stringList' ? '<span>Un elemento por línea.</span>' : '',
    entry.default && entry.type !== 'enum' && entry.type !== 'bool' ? `<span>Por defecto: <code>${escapeHtml(entry.default)}</code>.</span>` : ''
  ].filter(Boolean).join(' ');
  return `<div class="field settings-field"><label for="${escapeHtml(id)}">${escapeHtml(entry.label || entry.key)} <code class="settings-key">${key}</code></label>${control}<p class="field-hint" id="${escapeHtml(hintId)}">${hints}</p></div>`;
}

// editorHref construye el enlace «Abrir en el editor» (VS Code) de una ruta absoluta del
// sistema operativo: esquema fijo, `\` pasa a `/`, se quita la `/` inicial y cada segmento
// se codifica por separado, de modo que `#`, `?`, `&`, los espacios y los acentos no
// truncan ni alteran el enlace. La unidad de Windows (`C:`) se conserva tal cual. Con una
// línea (entero positivo) el editor abre el fichero en ella: `…/axiom.yaml:16`.
export function editorHref(configPath, line) {
  if (!configPath || typeof configPath !== 'string') return '';
  const segments = configPath.replace(/\\/g, '/').replace(/^\/+/, '').split('/');
  const encoded = segments.map((segment, i) => (i === 0 && /^[A-Za-z]:$/.test(segment) ? segment : encodeURIComponent(segment)));
  const at = Number.isInteger(line) && line > 0 ? `:${line}` : '';
  return `vscode://file/${encoded.join('/')}${at}`;
}

// pathActionsHtml: el enlace «Abrir en el editor» (en la línea, si se conoce) y el botón
// «Copiar ruta» de una ruta.
export function pathActionsHtml(configPath, line) {
  const href = editorHref(configPath, line);
  if (!href) return '';
  return `<span class="settings-path-actions"><a class="link-btn" href="${escapeHtml(href)}">Abrir en el editor</a> <button type="button" class="link-btn" data-copy-path="${escapeHtml(configPath)}">Copiar ruta</button></span>`;
}

// readOnlyHtml devuelve una clave de solo lectura con su valor, el motivo y cómo editarla
// a mano: el enlace al editor, en la línea de la clave si está declarada, y el botón para
// copiar la ruta de axiom.yaml.
export function readOnlyHtml(entry, configPath) {
  const v = entry.value;
  let shown;
  if (v === null || v === undefined) shown = '<span class="dim">sin declarar</span>';
  else if (entry.type === 'yaml') shown = `<pre class="console-out" tabindex="0">${escapeHtml(String(v))}</pre>`;
  else if (Array.isArray(v)) shown = v.length ? `<code>${escapeHtml(v.join(', '))}</code>` : '<span class="dim">vacío</span>';
  else shown = `<code>${escapeHtml(String(v))}</code>`;
  const reason = entry.readOnlyReason ? `<span>${escapeHtml(entry.readOnlyReason)}</span>` : '';
  return `<div class="field settings-field settings-readonly"><span class="field-label">${escapeHtml(entry.label || entry.key)} <code class="settings-key">${escapeHtml(entry.key)}</code></span><div>${shown}</div><p class="field-hint">${reason} ${pathActionsHtml(configPath, entry.line)}</p></div>`;
}

// collectChanges compara lo que muestran los controles (values: clave → texto) con lo que
// declara el catálogo y devuelve solo lo que cambió: la clave con su valor tipado, o null si
// se vació (retira la clave y el motor vuelve a su valor por defecto).
export function collectChanges(entries, values) {
  const changes = {};
  for (const entry of Array.isArray(entries) ? entries : []) {
    if (!entry.editable || !Object.prototype.hasOwnProperty.call(values, entry.key)) continue;
    const raw = values[entry.key];
    const text = raw === null || raw === undefined ? '' : String(raw);
    if (entry.type === 'stringList') {
      const items = text.split(/\r?\n/).map(s => s.trim()).filter(Boolean);
      const before = Array.isArray(entry.value) ? entry.value : [];
      if (items.length === before.length && items.every((s, i) => s === before[i])) continue;
      changes[entry.key] = items.length ? items : null;
      continue;
    }
    const trimmed = text.trim();
    if (trimmed === shownValue(entry)) continue;
    if (trimmed === '') {
      if (entry.declared) changes[entry.key] = null;
      continue;
    }
    if (entry.type === 'bool') changes[entry.key] = trimmed === 'true' ? true : trimmed === 'false' ? false : trimmed;
    else if (entry.type === 'int') changes[entry.key] = /^-?\d+$/.test(trimmed) ? Number(trimmed) : trimmed;
    else changes[entry.key] = trimmed;
  }
  return changes;
}

// errorView presenta un error de PUT /api/settings (un ApiError con `data`): el mensaje, las
// violaciones del motor y, si el fichero cambió fuera de la pantalla (revision_mismatch), la
// acción de recargar. No altera lo tecleado: la pantalla no se vuelve a pintar.
export function errorView(err) {
  const data = (err && err.data) || {};
  const message = (err && err.message) || data.error || 'No se pudieron guardar los ajustes';
  const violations = Array.isArray(data.violations) && data.violations.length
    ? `<ul class="settings-violations">${data.violations.map(v => `<li>${escapeHtml(v)}</li>`).join('')}</ul>`
    : '';
  const reload = data.code === 'revision_mismatch'
    ? ' <button type="button" class="btn" data-settings-reload>Recargar</button>'
    : '';
  return `<div class="alert alert-bad" role="alert"><p>${escapeHtml(message)}${reload}</p>${violations}</div>`;
}
