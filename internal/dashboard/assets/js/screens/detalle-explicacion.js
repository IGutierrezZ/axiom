// Pestaña «Explicación» del detalle: muestra el HTML explicativo de la spec y del diseño en un
// iframe de origen opaco (sandbox vacío, sin scripts). Solo lectura: lo que se aprueba es el md.

import { escapeHtml } from '../core/api.js';
import { stateEmpty, stateError } from '../core/components.js';

const TITLES = { spec: 'Resumen de la spec', design: 'Resumen del diseño' };

function frame(change, kind) {
  const src = `/api/explainer?change=${encodeURIComponent(change)}&kind=${kind}`;
  return `<iframe class="explainer-frame" sandbox="" referrerpolicy="no-referrer" loading="lazy" title="${escapeHtml(TITLES[kind])}" src="${src}"></iframe>`;
}

function itemHtml(change, item) {
  const title = TITLES[item.kind] || item.kind;
  let body;
  if (item.error) {
    body = stateError(item.error, 'No se pudo calcular el resumen');
  } else if (!item.state) {
    body = stateEmpty('Todavía no hay documento de origen', 'El resumen se generará con él.');
  } else if (item.state === 'missing') {
    body = stateEmpty('No hay explicación generada');
  } else {
    const warning = item.state === 'stale'
      ? '<p class="ix-more" role="status">Este resumen está desactualizado respecto del md. Lo que se aprueba es el md.</p>'
      : '';
    body = warning + frame(change, item.kind);
  }
  return `<section class="explainer-item"><h3>${escapeHtml(title)}</h3>${body}</section>`;
}

// renderExplainer pinta la pestaña a partir de detail.explainer; sin activación no crea ningún iframe.
export function renderExplainer(panel, { change, detail }) {
  const ex = detail.explainer;
  if (!ex || !ex.enabled) {
    panel.innerHTML = stateEmpty('La explicación no está activada para este cambio.');
    return;
  }
  if (ex.error) {
    panel.innerHTML = stateError(`No se pudo calcular la explicación: ${ex.error}`);
    return;
  }
  const items = ex.items || [];
  panel.innerHTML = items.length
    ? items.map(item => itemHtml(change, item)).join('')
    : stateEmpty('Todavía no hay documento de origen', 'El resumen se generará con él.');
}
