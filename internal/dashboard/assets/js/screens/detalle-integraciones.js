// Pestaña «Integraciones» del detalle de un cambio (D-07): work item enlazado, Task por rol con su
// estimación, imputaciones y tiempo pendiente, outbox de solo lectura y actores por fase. No ofrece
// ninguna acción: no ejecuta ni reintenta la outbox ni modifica vínculos.
//
// Todo texto (identificadores, roles, actores, errores) procede de ficheros del cambio o del tracker
// y se inserta con textContent; el DOM se construye con nodos, nunca con HTML. Los enlaces
// externos se crean con externalLink (core/integrations.js), que vuelve a validar la URL.

import { api } from '../core/api.js';
import { stateEmpty, stateError, stateLoading } from '../core/components.js';
import { externalLink } from '../core/integrations.js';

const NO_TASK = 'sin Task';
const NO_ACTOR = 'sin registrar';
const NOT_SEALED = 'sin sellar';

// Todo texto externo se acota antes de mostrarlo y sin caracteres de control.
const MAX_TEXT = 200;
function plain(value) {
  const text = String(value == null ? '' : value).replace(/[\u0000-\u001f\u007f]/g, ' ');
  return text.length > MAX_TEXT ? `${text.slice(0, MAX_TEXT)}…` : text;
}

function el(tag, text, className) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text != null) node.textContent = plain(text);
  return node;
}

function section(title) {
  const card = el('section', null, 'card');
  const head = el('div', null, 'card-head');
  head.appendChild(el('h3', title, 'card-title-h'));
  card.appendChild(head);
  return card;
}

function table(headers, rows) {
  const wrap = el('div', null, 'table-wrap');
  const tbl = el('table', null, 'data-table');
  const headRow = document.createElement('tr');
  headers.forEach(h => {
    const th = el('th', h);
    th.scope = 'col';
    headRow.appendChild(th);
  });
  const thead = document.createElement('thead');
  thead.appendChild(headRow);
  tbl.appendChild(thead);
  const tbody = document.createElement('tbody');
  rows.forEach(cells => {
    const tr = document.createElement('tr');
    cells.forEach(cell => {
      const td = document.createElement('td');
      if (cell instanceof Node) td.appendChild(cell);
      else td.textContent = plain(cell);
      tr.appendChild(td);
    });
    tbody.appendChild(tr);
  });
  tbl.appendChild(tbody);
  wrap.appendChild(tbl);
  return wrap;
}

// linkCell muestra el identificador del elemento y, si la URL es segura, «Abrir en <plugin>».
function linkCell(link) {
  const cell = document.createElement('span');
  cell.appendChild(document.createTextNode(plain(link.id)));
  const anchor = externalLink(link.url, `Abrir en ${plain(link.pluginTitle || link.plugin)}`);
  if (anchor) {
    cell.appendChild(document.createTextNode(' · '));
    cell.appendChild(anchor);
  }
  return cell;
}

function minutesText(minutes) {
  const total = Number(minutes) || 0;
  const h = Math.floor(total / 60);
  const m = total % 60;
  return h ? `${h} h ${m} min` : `${m} min`;
}

function workItemsSection(dto) {
  const card = section('Work item enlazado');
  if (!dto.workItems.length) {
    card.appendChild(el('p', 'Sin work item enlazado.', 'ix-more'));
    return card;
  }
  card.appendChild(table(['Elemento', 'Plugin', 'Tipo'], dto.workItems.map(w => [linkCell(w), w.pluginTitle || w.plugin, w.kind])));
  return card;
}

function rolesSection(dto) {
  const card = section('Tasks por rol');
  if (!dto.roles.length) {
    card.appendChild(el('p', 'Sin Tasks ni imputaciones registradas.', 'ix-more'));
    return card;
  }
  const rows = dto.roles.map(r => {
    const task = r.task ? linkCell(r.task) : NO_TASK;
    const estimate = r.task && r.task.estimateHours != null ? `${r.task.estimateHours} h` : '—';
    const spent = (r.entries || []).reduce((sum, e) => sum + (Number(e.minutes) || 0), 0);
    const entries = (r.entries || []).length
      ? (r.entries || []).map(e => `${e.date}: ${minutesText(e.minutes)} (${e.actor || NO_ACTOR})`).join(' · ')
      : 'Sin imputaciones';
    const pending = r.pending ? `${minutesText(r.pending.minutes)} (${r.pending.date}, elemento ${r.pending.itemId})` : '—';
    return [r.role || 'Sin rol', task, estimate, spent ? minutesText(spent) : '—', entries, pending];
  });
  card.appendChild(table(['Rol', 'Task', 'Estimación', 'Imputado', 'Imputaciones', 'Pendiente de imputar'], rows));
  return card;
}

function outboxSection(dto) {
  const card = section('Outbox (solo lectura)');
  if (!dto.outbox.length) {
    card.appendChild(el('p', 'No hay operaciones pendientes en la outbox.', 'ix-more'));
    return card;
  }
  const rows = dto.outbox.map(o => [
    o.event, o.plugin, o.role || o.gate || '—', o.state, String(o.attempts), o.enqueuedAt, o.lastAttemptAt || '—', o.reason || '—',
  ]);
  card.appendChild(table(['Evento', 'Plugin', 'Rol o puerta', 'Estado', 'Intentos', 'Encolada', 'Último intento', 'Motivo'], rows));
  if (dto.outboxTruncated) {
    card.appendChild(el('p', `Se muestran las primeras ${dto.outbox.length} de ${dto.outboxTotal} operaciones.`, 'ix-more'));
  }
  return card;
}

function actorsSection(dto) {
  const card = section('Actores por fase');
  const sealed = dto.kickoff.sealed ? (dto.kickoff.sealedBy || NO_ACTOR) : NOT_SEALED;
  const rows = [['Kickoff sellado por', sealed, '—']];
  dto.gates.forEach(g => rows.push([`Puerta ${g.gate}: ${g.decision}`, g.actor || NO_ACTOR, g.recordedAt || '—']));
  card.appendChild(table(['Fase', 'Actor', 'Fecha'], rows));
  return card;
}

function errorsBlock(errors) {
  const box = el('div', null, 'state state-error');
  box.setAttribute('role', 'alert');
  box.appendChild(el('span', 'Algunos datos no se pudieron leer', 'state-title'));
  errors.forEach(message => box.appendChild(el('span', message)));
  return box;
}

function render(panel, dto) {
  panel.replaceChildren();
  if (dto.errors.length) panel.appendChild(errorsBlock(dto.errors));
  if (dto.empty) {
    const empty = el('p', 'Este cambio no tiene work item, Tasks ni operaciones de integración.', 'ix-more');
    empty.setAttribute('role', 'status');
    panel.appendChild(empty);
  } else {
    panel.appendChild(workItemsSection(dto));
    panel.appendChild(rolesSection(dto));
    panel.appendChild(outboxSection(dto));
  }
  panel.appendChild(actorsSection(dto));
}

// renderChangeIntegrations pide la pestaña bajo demanda. isCurrent descarta respuestas tardías.
export async function renderChangeIntegrations(panel, { change, isCurrent = () => true }) {
  panel.innerHTML = stateLoading('Cargando integraciones…');
  try {
    const dto = await api(`/api/integrations/change?name=${encodeURIComponent(change)}`);
    if (!isCurrent()) return;
    if (!dto) {
      panel.innerHTML = stateEmpty('Sin datos de integraciones');
      return;
    }
    render(panel, {
      ...dto,
      workItems: dto.workItems || [],
      roles: dto.roles || [],
      outbox: dto.outbox || [],
      gates: dto.gates || [],
      errors: dto.errors || [],
      kickoff: dto.kickoff || { sealed: false, sealedBy: '' },
    });
  } catch (err) {
    if (!isCurrent()) return;
    panel.innerHTML = stateError(err && err.message ? err.message : 'Error desconocido');
  }
}
