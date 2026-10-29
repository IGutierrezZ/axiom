// Specs vivas: catálogo de dominios y visor. Provisional hasta T5.

import { api, escapeHtml } from '../core/api.js';
import { $, setBusy, toast } from '../core/components.js';
import { icon } from '../core/icons.js';

let currentDomain = null;

export async function loadLivingDocs() {
  const list = $('#livingdoc-domains-list');
  try {
    // Los rechazos de la API (p. ej. 409 con baseline del proyecto) llegan en
    // err.message: api() extrae el campo `error` que nombra la baseline y la continuación.
    const catalog = await api('/api/archive/specs');
    const specs = catalog.specs || [];
    $('#metric-living-domains').textContent = specs.length;
    $('#metric-living-reqs').textContent = catalog.total_requirements || 0;
    if (catalog.last_sync) $('#metric-living-sync').textContent = new Date(catalog.last_sync).toLocaleTimeString('es-ES');

    // Con baseline del proyecto declarada el catálogo no se indexa: se informa de
    // que es externa y de dónde está, para que un catálogo vacío no se
    // interprete como ausencia de especificación (REQ-24.25).
    const externalBaseline = catalog.external_baseline;
    if (externalBaseline && Array.isArray(externalBaseline.documents)) {
      const items = externalBaseline.documents.map(doc =>
        `<li><strong>${escapeHtml(doc.domain)}</strong>: <code>${escapeHtml(doc.path)}</code></li>`
      ).join('');
      list.innerHTML = `<div class="alert alert-warn external-baseline-notice">${icon('info', 18)}<div class="grow"><p>La baseline de especificaciones es externa (del proyecto): sus documentos no se indexan aquí. Dominios y rutas declarados:</p><ul style="margin: 8px 0 0 18px">${items}</ul></div></div>`;
      return;
    }

    if (specs.length === 0) {
      list.innerHTML = '<div class="state">No hay especificaciones vivas en openspec/specs/. Usa «Sincronizar catálogo» o la CLI para generarlas.</div>';
      return;
    }

    list.innerHTML = specs.map(d => `<button type="button" class="livingdoc-domain-item" data-domain="${escapeHtml(d.domain)}" aria-pressed="${d.domain === currentDomain}" style="text-align:left;width:100%">
      <span class="livingdoc-domain-header"><span class="livingdoc-domain-title">${escapeHtml(d.title || d.domain)}</span><span class="badge">${d.total_scenarios || 0} BDD</span></span>
      <span class="livingdoc-domain-reqs">${d.requirements ? d.requirements.length : 0} requisitos activos · ${escapeHtml(d.domain)}</span>
    </button>`).join('');

    if (!currentDomain && specs.length > 0) {
      currentDomain = specs[0].domain;
      const first = list.querySelector('.livingdoc-domain-item');
      if (first) first.setAttribute('aria-pressed', 'true');
      loadLivingDocDetail(currentDomain);
    }
  } catch (err) {
    list.innerHTML = `<div class="error-banner">Error cargando el catálogo: ${escapeHtml(err.message)}</div>`;
  }
}

async function loadLivingDocDetail(domain) {
  const viewer = $('#livingdoc-viewer');
  viewer.textContent = 'Cargando especificación…';
  try {
    const data = await api(`/api/archive/specs/${encodeURIComponent(domain)}`);
    $('#livingdoc-detail-title').textContent = data.entry ? (data.entry.title || data.entry.domain) : domain;
    const ver = $('#livingdoc-detail-version');
    if (data.entry) {
      ver.textContent = `${data.entry.total_scenarios || 0} escenarios BDD`;
      ver.hidden = false;
    }
    viewer.textContent = data.content || '// Especificación vacía';
  } catch (err) {
    viewer.innerHTML = `<div class="error-banner">Error: ${escapeHtml(err.message)}</div>`;
  }
}

export function initLivingDocs() {
  $('#livingdoc-domains-list').addEventListener('click', (event) => {
    const item = event.target.closest('.livingdoc-domain-item');
    if (!item) return;
    currentDomain = item.getAttribute('data-domain');
    $('#livingdoc-domains-list').querySelectorAll('.livingdoc-domain-item').forEach(i => i.setAttribute('aria-pressed', String(i === item)));
    loadLivingDocDetail(currentDomain);
  });

  const sync = $('#btn-sync-livingdoc');
  sync.addEventListener('click', async () => {
    const restore = setBusy(sync, 'Sincronizando…');
    try {
      await api('/api/archive/sync', { method: 'POST' });
      await loadLivingDocs();
      toast('Catálogo sincronizado.', 'ok');
    } catch (err) {
      toast(`Fallo al sincronizar: ${err.message}`, 'bad');
    } finally {
      restore();
    }
  });
}
