// Pestaña «Pruebas» del detalle: muestra la batería de pruebas de aceptación (pruebas.md) con su
// cobertura y sus hallazgos. Solo lectura: sin edición, ejecución ni marcas por entorno. Todo texto
// proveniente de pruebas.md se inserta escapado, nunca como HTML ni como Markdown.

import { escapeHtml } from '../core/api.js';
import { stateEmpty, stateError } from '../core/components.js';

const COLUMNS = [
  ['id', 'ID'], ['role', 'Rol'], ['type', 'Tipo'], ['priority', 'Prioridad'],
  ['preconditions', 'Precondiciones'], ['steps', 'Pasos'], ['expected', 'Resultado esperado'],
  ['cubre', 'Cubre'], ['environments', 'Entornos'],
];

function coverage(c) {
  const cov = c || {};
  return `${cov.covered || 0}/${cov.total || 0}`;
}

function summaryHtml(at) {
  return `<p class="ix-more" role="status">Escenarios ${escapeHtml(coverage(at.scenarios))} · `
    + `Requisitos ${escapeHtml(coverage(at.requirements))} · Casos ${escapeHtml(String(at.case_count || 0))}</p>`;
}

function findingsHtml(findings) {
  if (!findings.length) return '<p class="ix-more">Sin hallazgos.</p>';
  const items = findings.map(f => {
    const where = f.line ? `${f.file}:${f.line}` : f.file;
    return `<li><code>${escapeHtml(f.code)}</code> ${escapeHtml(where)}: ${escapeHtml(f.message)}</li>`;
  }).join('');
  return `<ul class="at-findings">${items}</ul>`;
}

function casesHtml(cases) {
  if (!cases.length) return stateEmpty('La tabla no tiene casos');
  const head = COLUMNS.map(([, label]) => `<th scope="col">${escapeHtml(label)}</th>`).join('');
  const rows = cases.map(c => `<tr>${COLUMNS.map(([key]) => `<td>${escapeHtml(c[key] || '')}</td>`).join('')}</tr>`).join('');
  return `<div class="table-wrap"><table class="data-table"><thead><tr>${head}</tr></thead><tbody>${rows}</tbody></table></div>`;
}

// renderAcceptanceTests pinta la pestaña a partir de detail.acceptance_tests (nil: no activo).
export function renderAcceptanceTests(panel, { detail }) {
  const at = detail.acceptance_tests;
  if (!at || (!at.active && !at.present && !at.error)) {
    panel.innerHTML = stateEmpty('Las pruebas de aceptación no están activadas para este proyecto.');
    return;
  }
  let html = '';
  if (at.error) html += stateError(at.error, 'No se pudo calcular la batería de pruebas');
  if (at.warning) html += `<p class="ix-more" role="status">${escapeHtml(at.warning)}</p>`;
  if (!at.present) {
    html += stateEmpty(at.active ? 'Todavía no hay batería de pruebas' : 'No hay batería de pruebas');
    if (at.findings && at.findings.length) html += findingsHtml(at.findings);
    panel.innerHTML = html;
    return;
  }
  if (!at.error) {
    html += summaryHtml(at) + findingsHtml(at.findings || []) + casesHtml(at.cases || []);
  }
  panel.innerHTML = html;
}
