// Componentes comunes: pequeñas funciones que devuelven HTML (con el estilo de
// style.css) o enlazan comportamiento. Todo dato interpolado se escapa aquí.

import { escapeHtml } from './api.js';
import { icon } from './icons.js';

// ---- Utilidades DOM -------------------------------------------------------

export function $(selector, root = document) {
  return root.querySelector(selector);
}

export function $all(selector, root = document) {
  return Array.from(root.querySelectorAll(selector));
}

export function htmlToElement(html) {
  const tpl = document.createElement('template');
  tpl.innerHTML = html.trim();
  return tpl.content.firstElementChild;
}

// ---- Botones, chips e insignias -------------------------------------------

// button devuelve un <button>. variant: primary | secondary | ghost | danger.
export function button({ label, iconName, variant = 'secondary', size = '', attrs = '' }) {
  const cls = ['btn', `btn-${variant}`, size ? `btn-${size}` : ''].filter(Boolean).join(' ');
  return `<button type="button" class="${cls}" ${attrs}>${iconName ? icon(iconName, 16) : ''}${escapeHtml(label)}</button>`;
}

// iconButton: botón de solo icono; el aria-label es obligatorio.
export function iconButton({ iconName, label, variant = 'secondary', size = '', attrs = '' }) {
  const cls = ['btn', `btn-${variant}`, 'btn-icon', size ? `btn-${size}` : ''].filter(Boolean).join(' ');
  return `<button type="button" class="${cls}" aria-label="${escapeHtml(label)}" title="${escapeHtml(label)}" ${attrs}>${icon(iconName, 16)}</button>`;
}

// chip: filtro conmutable con aria-pressed y contador opcional.
export function chip({ label, pressed = false, count = null, attrs = '' }) {
  const counter = count === null ? '' : `<span class="chip-count">${escapeHtml(count)}</span>`;
  return `<button type="button" class="chip" aria-pressed="${pressed ? 'true' : 'false'}" ${attrs}>${escapeHtml(label)}${counter}</button>`;
}

// badge: kind = ok | warn | bad | accent | neutral. Con iconName el estado no
// depende solo del color; el texto siempre acompaña.
export function badge(kind, text, iconName = '') {
  return `<span class="badge badge-${kind}">${iconName ? icon(iconName, 13) : ''}${escapeHtml(text)}</span>`;
}

// STATUS_ICON asocia cada tono a un icono distinto.
const STATUS_ICON = { ok: 'check', warn: 'alert', bad: 'x', accent: 'info', neutral: 'clock' };

// statusPill: insignia de estado con icono propio del tono más el texto.
export function statusPill(kind, text) {
  return badge(kind, text, STATUS_ICON[kind] || 'info');
}

// ---- Estados vacío / carga / error ---------------------------------------

export function stateLoading(text = 'Cargando…') {
  return `<div class="state" role="status"><div class="spinner" aria-hidden="true"></div><span>${escapeHtml(text)}</span></div>`;
}

export function stateEmpty(title, text = '', iconName = 'inbox') {
  return `<div class="state"><span class="state-icon">${icon(iconName, 24)}</span><span class="state-title">${escapeHtml(title)}</span>${text ? `<span>${escapeHtml(text)}</span>` : ''}</div>`;
}

export function stateError(message, title = 'No se pudo cargar') {
  return `<div class="state state-error" role="alert"><span class="state-icon">${icon('alert', 24)}</span><span class="state-title">${escapeHtml(title)}</span><span>${escapeHtml(message)}</span></div>`;
}

// tableWrap envuelve una tabla en una caja con desplazamiento horizontal propio.
export function tableWrap(tableHtml, label = '') {
  const aria = label ? ` role="region" aria-label="${escapeHtml(label)}" tabindex="0"` : '';
  return `<div class="table-wrap"${aria}>${tableHtml}</div>`;
}

// ---- Recordatorio (D5) -----------------------------------------------------

// reminder: texto que dice qué paso de agente toca y que se lance desde el IDE
// o la CLI favoritos, con la orden copiable. No es un botón de acción.
export function reminder({ title, text, command = '' }) {
  const cmd = command
    ? `<div class="reminder-cmd"><code>${escapeHtml(command)}</code>${iconButton({ iconName: 'copy', label: 'Copiar la orden', size: 'sm', attrs: `data-copy="${escapeHtml(command)}"` })}</div>`
    : '';
  return `<aside class="reminder" aria-label="Recordatorio">${icon('terminal', 18)}<div class="reminder-body"><span class="reminder-title">${escapeHtml(title)}</span><span class="reminder-text">${escapeHtml(text)}</span>${cmd}</div></aside>`;
}

// copyText copia al portapapeles con respaldo para contextos sin la API moderna.
export async function copyText(text) {
  try {
    if (navigator.clipboard && window.isSecureContext !== false) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch (_) {
    // Se prueba el respaldo.
  }
  try {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand('copy');
    ta.remove();
    return ok;
  } catch (_) {
    return false;
  }
}

// bindCopyButtons activa los [data-copy] dentro de root (delegación única).
export function bindCopyButtons(root = document) {
  root.addEventListener('click', async (event) => {
    const btn = event.target.closest('[data-copy]');
    if (!btn) return;
    const ok = await copyText(btn.getAttribute('data-copy'));
    toast(ok ? 'Orden copiada al portapapeles' : 'No se pudo copiar; selecciónala y cópiala a mano', ok ? 'ok' : 'warn');
  });
}

// ---- Avisos (sustituyen a alert) -------------------------------------------

export function toast(message, kind = 'ok') {
  const region = document.getElementById('toast-region');
  if (!region) return;
  const item = document.createElement('div');
  item.className = `toast toast-${kind}`;
  item.innerHTML = `${icon(kind === 'ok' ? 'check' : kind === 'bad' ? 'x' : 'alert', 16)}<span>${escapeHtml(message)}</span>`;
  region.appendChild(item);
  setTimeout(() => item.remove(), kind === 'ok' ? 4000 : 8000);
}

// ---- Diálogos modales (<dialog>) -------------------------------------------

// wireDialog enlaza el cierre por [data-close], por clic en el fondo y devuelve el
// elemento. Escape y el foco atrapado los aporta <dialog>.showModal().
export function wireDialog(dialog) {
  if (!dialog || dialog.dataset.wired === '1') return dialog;
  dialog.dataset.wired = '1';
  dialog.addEventListener('click', (event) => {
    if (event.target === dialog) dialog.close('backdrop');
    if (event.target.closest('[data-close]')) dialog.close('close');
  });
  return dialog;
}

export function openDialog(dialog, focusSelector = '') {
  if (!dialog) return;
  wireDialog(dialog);
  if (!dialog.open) dialog.showModal();
  const target = focusSelector ? dialog.querySelector(focusSelector) : null;
  if (target) target.focus();
}

export function closeDialog(dialog) {
  if (dialog && dialog.open) dialog.close();
}

// confirmDialog sustituye a confirm(): devuelve una promesa booleana.
export function confirmDialog({ title, message, confirmLabel = 'Confirmar', danger = false }) {
  return askDialog({ title, message, confirmLabel, danger, input: null }).then(r => r !== null);
}

// promptDialog sustituye a prompt(): devuelve el texto o null si se cancela.
export function promptDialog({ title, message, confirmLabel = 'Aceptar', value = '', placeholder = '' }) {
  return askDialog({ title, message, confirmLabel, danger: false, input: { value, placeholder } });
}

function askDialog({ title, message, confirmLabel, danger, input }) {
  const dialog = document.getElementById('dlg-confirm');
  if (!dialog) return Promise.resolve(window.confirm(message) ? '' : null);
  wireDialog(dialog);
  dialog.querySelector('[data-role="title"]').textContent = title;
  dialog.querySelector('[data-role="message"]').textContent = message;
  const field = dialog.querySelector('[data-role="input"]');
  const ok = dialog.querySelector('[data-role="ok"]');
  ok.textContent = confirmLabel;
  ok.className = `btn ${danger ? 'btn-danger' : 'btn-primary'}`;
  field.hidden = !input;
  if (input) {
    field.value = input.value;
    field.placeholder = input.placeholder;
  }
  const cancel = dialog.querySelector('[data-role="cancel"]');
  cancel.onclick = () => dialog.close('cancel');
  return new Promise((resolve) => {
    const finish = () => {
      dialog.removeEventListener('close', onClose);
      resolve(dialog.returnValue === 'ok' ? (input ? field.value : '') : null);
    };
    const onClose = () => finish();
    dialog.addEventListener('close', onClose);
    dialog.returnValue = '';
    dialog.showModal();
    (input ? field : dialog.querySelector('[data-role="cancel"]')).focus();
  });
}

// ---- Consola de ejecución --------------------------------------------------

// consoleBox devuelve el HTML de una consola; usa renderConsole para escribir.
export function consoleBox(id, title = 'Salida de ejecución') {
  return `<div class="console" id="${escapeHtml(id)}" hidden><div class="console-head"><span>${icon('terminal', 14)} <span data-role="title">${escapeHtml(title)}</span></span><button type="button" class="btn btn-ghost btn-xs" data-role="clear">Limpiar</button></div><pre class="console-out" data-role="out" tabindex="0"></pre></div>`;
}

// renderConsole escribe en una consola creada con consoleBox (state: ok | error | '').
export function renderConsole(box, { title, text, state = '' }) {
  if (!box) return;
  if (title) box.querySelector('[data-role="title"]').textContent = title;
  const out = box.querySelector('[data-role="out"]');
  out.textContent = text;
  if (state) out.setAttribute('data-state', state);
  else out.removeAttribute('data-state');
  box.hidden = false;
  const clear = box.querySelector('[data-role="clear"]');
  if (clear && !clear.dataset.bound) {
    clear.dataset.bound = '1';
    clear.addEventListener('click', () => { out.textContent = ''; box.hidden = true; });
  }
}

// ---- Estado ocupado de un botón -------------------------------------------

// setBusy deshabilita el botón y cambia su texto; devuelve la función que lo restaura.
export function setBusy(btn, busyLabel) {
  if (!btn) return () => {};
  const original = btn.innerHTML;
  btn.disabled = true;
  btn.setAttribute('aria-busy', 'true');
  btn.textContent = busyLabel;
  return () => {
    btn.disabled = false;
    btn.removeAttribute('aria-busy');
    btn.innerHTML = original;
  };
}

// initials: iniciales de dos letras para el bloque del selector de proyecto.
export function initials(name) {
  const words = String(name || '?').split(/[\s_\-./\\]+/).filter(Boolean);
  if (words.length === 0) return '?';
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return (words[0][0] + words[1][0]).toUpperCase();
}
