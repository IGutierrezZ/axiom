// Specs vivas: catálogo de dominios (con filtro y métricas) y visor de requisitos y
// escenarios. Cada dominio es una ruta (#/specs/<dominio>), así que atrás y adelante
// funcionan. Según D3 no se pintan versiones del dominio ni deltas de incrementos: el
// servidor no los expone.

import { api, escapeHtml } from '../core/api.js';
import { $, setBusy, stateEmpty, stateError, stateLoading, toast } from '../core/components.js';
import { formatDateTime, formatDay, formatNumber, plural } from '../core/format.js';
import { icon } from '../core/icons.js';
import { refreshOverview } from '../core/overview.js';
import { hrefFor } from '../core/router.js';
import { renderSpec } from '../core/spec.js';

const ctx = {
  catalog: null,
  catalogError: '',
  filter: '',
  domain: null,            // dominio de la ruta activa
  detail: { domain: null, status: 'idle', data: null, error: '', external: null },
  raw: false,
  token: 0,                // descarta respuestas de detalle obsoletas
  focusViewer: false,      // mover el foco al visor tras elegir un dominio
  sync: null               // {kind, html}: resultado de la última sincronización
};

// ---- Resumen, avisos y lista ------------------------------------------------------------

function renderSummary() {
  const el = $('#specs-summary');
  const c = ctx.catalog;
  if (!c) {
    el.textContent = ctx.catalogError ? 'No se pudo leer el catálogo.' : 'Cargando el catálogo…';
    return;
  }
  if (c.external_baseline && Array.isArray(c.external_baseline.documents)) {
    el.textContent = `Baseline externa del proyecto · ${plural(c.external_baseline.documents.length, 'documento declarado', 'documentos declarados')}`;
    return;
  }
  const parts = [
    plural((c.specs || []).length, 'dominio', 'dominios'),
    plural(c.total_requirements || 0, 'requisito', 'requisitos'),
    plural(c.total_scenarios || 0, 'escenario', 'escenarios')
  ];
  const sync = formatDateTime(c.last_sync);
  if (sync) parts.push(`sincronizado el ${sync}`);
  el.textContent = parts.join(' · ');
}

function renderNotice() {
  const el = $('#specs-notice');
  const out = [];
  const c = ctx.catalog;
  // Con baseline del proyecto declarada el catálogo no se indexa: se informa de que es
  // externa y de dónde está, para que un catálogo vacío no se interprete como ausencia
  // de especificación (REQ-24.25).
  const externalBaseline = c && c.external_baseline;
  if (externalBaseline && Array.isArray(externalBaseline.documents)) {
    const items = externalBaseline.documents.map(doc =>
      `<li><strong>${escapeHtml(doc.domain)}</strong>: <code>${escapeHtml(doc.path)}</code></li>`
    ).join('');
    out.push(`<div class="alert alert-warn external-baseline-notice">${icon('info', 18)}<div class="grow"><p>La baseline de especificaciones es externa (del proyecto): sus documentos no se indexan aquí. Dominios y rutas declarados:</p><ul class="kn-notice-list">${items}</ul></div></div>`);
  }
  if (c && Array.isArray(c.warnings) && c.warnings.length) {
    out.push(`<div class="alert alert-warn">${icon('alert', 18)}<div class="grow"><strong>Avisos del catálogo</strong><ul class="kn-notice-list">${c.warnings.map(w => `<li>${escapeHtml(w)}</li>`).join('')}</ul></div></div>`);
  }
  if (ctx.sync) {
    const name = { ok: 'check', warn: 'alert', bad: 'alert' }[ctx.sync.kind] || 'info';
    out.push(`<div class="alert alert-${ctx.sync.kind}"${ctx.sync.kind === 'bad' ? ' role="alert"' : ''}>${icon(name, 18)}<div class="grow">${ctx.sync.html}</div></div>`);
  }
  el.innerHTML = out.join('');
}

function visibleDomains() {
  const specs = ((ctx.catalog && ctx.catalog.specs) || []).slice().sort((a, b) => String(a.domain).localeCompare(String(b.domain), 'es'));
  const q = ctx.filter.trim().toLowerCase();
  if (!q) return specs;
  return specs.filter(d => [d.domain, d.title, d.description].some(v => String(v || '').toLowerCase().includes(q)));
}

function renderList() {
  const list = $('#livingdoc-domains-list');
  const count = $('#specs-count');
  if (ctx.catalogError && !ctx.catalog) {
    count.textContent = '';
    list.innerHTML = stateError(ctx.catalogError, 'Error cargando el catálogo');
    return;
  }
  if (!ctx.catalog) {
    list.innerHTML = stateLoading('Cargando catálogo…');
    return;
  }
  const total = (ctx.catalog.specs || []).length;
  if (total === 0) {
    count.textContent = '';
    list.innerHTML = ctx.catalog.external_baseline
      ? stateEmpty('Sin dominios indexados', 'La baseline es externa al proyecto: sus documentos se enumeran arriba.', 'book')
      : stateEmpty('No hay especificaciones vivas', 'No hay ningún spec en openspec/specs/. Usa «Sincronizar catálogo» o la CLI para generarlas.', 'book');
    return;
  }
  const domains = visibleDomains();
  count.textContent = ctx.filter.trim() ? `${formatNumber(domains.length)} de ${plural(total, 'dominio', 'dominios')}` : plural(total, 'dominio', 'dominios');
  if (domains.length === 0) {
    list.innerHTML = stateEmpty('Ningún dominio coincide', `No hay dominios que coincidan con «${ctx.filter.trim()}».`, 'search');
    return;
  }
  list.innerHTML = `<ul class="kn-domains">${domains.map((d) => {
    const reqs = d.requirements ? d.requirements.length : 0;
    const showTitle = d.title && d.title !== d.domain;
    return `<li><a class="kn-domain" href="${escapeHtml(hrefFor('specs', { domain: d.domain }))}" data-domain="${escapeHtml(d.domain)}"${d.domain === ctx.domain ? ' aria-current="page"' : ''}>
      <span class="kn-domain-top"><span class="mono kn-domain-name">${escapeHtml(d.domain)}</span><span class="mono kn-domain-count" aria-label="${escapeHtml(plural(reqs, 'requisito', 'requisitos'))}">${formatNumber(reqs)}</span></span>
      ${showTitle ? `<span class="kn-domain-title">${escapeHtml(d.title)}</span>` : ''}
      <span class="kn-domain-meta">${escapeHtml(plural(reqs, 'requisito', 'requisitos'))} · ${escapeHtml(plural(d.total_scenarios || 0, 'escenario', 'escenarios'))}</span>
    </a></li>`;
  }).join('')}</ul>`;
}

// markCurrent actualiza aria-current sin repintar la lista (conserva el foco del teclado).
function markCurrent() {
  document.querySelectorAll('#livingdoc-domains-list a[data-domain]').forEach((a) => {
    if (a.getAttribute('data-domain') === ctx.domain) a.setAttribute('aria-current', 'page');
    else a.removeAttribute('aria-current');
  });
}

export async function loadLivingDocs() {
  try {
    // Los rechazos de la API (p. ej. 409 con baseline del proyecto) llegan en
    // err.message: api() extrae el campo `error` que nombra la baseline y la continuación.
    ctx.catalog = await api('/api/archive/specs');
    ctx.catalogError = '';
  } catch (err) {
    ctx.catalogError = err.message;
    if (!ctx.catalog) ctx.catalog = null;
  }
  renderSummary();
  renderNotice();
  renderList();
  // «Actualizar» con un dominio ya abierto: se vuelve a leer su contenido.
  if (ctx.domain && ctx.detail.domain === ctx.domain && ctx.detail.status === 'ready') loadDetail(ctx.domain);
}

// ---- Visor -----------------------------------------------------------------------------------

// shortPath deja la ruta relativa a openspec/ cuando el servidor la devuelve absoluta.
function shortPath(p) {
  const norm = String(p).split('\\').join('/');
  const i = norm.lastIndexOf('/openspec/');
  return i >= 0 ? norm.slice(i + 1) : norm;
}

function viewerShell(title, inner) {
  return `<div class="kn-viewer-head"><h2 id="livingdoc-detail-title" class="kn-viewer-title mono" tabindex="-1">${escapeHtml(title)}</h2></div>${inner}`;
}

function renderViewer() {
  const el = $('#livingdoc-viewer');
  const d = ctx.detail;
  el.removeAttribute('aria-busy');
  if (!ctx.domain) {
    el.innerHTML = `<div class="state"><span class="state-icon">${icon('book', 24)}</span><h2 class="state-title" id="livingdoc-detail-title">Visor de especificación viva</h2><span>Elige un dominio de la lista para ver sus requisitos y escenarios.</span></div>`;
    return;
  }
  if (d.status === 'loading') {
    el.setAttribute('aria-busy', 'true');
    el.innerHTML = viewerShell(ctx.domain, `<div class="kn-pad">${stateLoading('Cargando especificación…')}</div>`);
    return;
  }
  if (d.status === 'error') {
    const body = d.external
      ? `<div class="alert alert-warn">${icon('info', 18)}<div class="grow"><p>${escapeHtml(d.error)}</p>${d.path ? `<p class="kn-hint">Documento declarado: <code>${escapeHtml(d.path)}</code></p>` : ''}</div></div>`
      : stateError(d.error, 'No se pudo cargar la especificación');
    el.innerHTML = viewerShell(ctx.domain, `<div class="kn-pad">${body}<p class="kn-hint"><a href="${escapeHtml(hrefFor('specs'))}">Volver a la lista de dominios</a></p></div>`);
    return;
  }
  const entry = d.data.entry || {};
  const content = d.data.content || '';
  const reqs = entry.requirements ? entry.requirements.length : 0;
  const modified = formatDay(entry.last_modified);
  const meta = [plural(reqs, 'requisito', 'requisitos'), plural(entry.total_scenarios || 0, 'escenario', 'escenarios')];
  if (modified) meta.push(`actualizado el ${modified}`);
  let body;
  if (!content.trim()) body = stateEmpty('Especificación vacía', 'El documento existe, pero no tiene contenido.', 'file');
  else if (ctx.raw) body = `<pre class="md-raw" tabindex="0" aria-label="Especificación en Markdown">${escapeHtml(content)}</pre>`;
  else body = `<div class="spec-doc">${renderSpec(content)}</div>`;
  el.innerHTML = `<div class="kn-viewer-head">
      <h2 id="livingdoc-detail-title" class="kn-viewer-title mono" tabindex="-1">${escapeHtml(entry.domain || ctx.domain)}</h2>
      <span class="kn-meta">${escapeHtml(meta.join(' · '))}</span>
      <span class="grow"></span>
      <div class="seg-group seg-inline" role="group" aria-label="Formato del documento">
        <button type="button" class="seg seg-wide" data-format="view" aria-pressed="${!ctx.raw}">Vista</button>
        <button type="button" class="seg seg-wide" data-format="raw" aria-pressed="${ctx.raw}">Markdown</button>
      </div>
    </div>
    ${entry.title && entry.title !== entry.domain ? `<p class="kn-viewer-sub">${escapeHtml(entry.title)}</p>` : ''}
    ${entry.file_path ? `<p class="kn-path mono">${escapeHtml(shortPath(entry.file_path))}</p>` : ''}
    <div class="kn-viewer-body">${body}</div>`;
}

async function loadDetail(domain) {
  const token = ++ctx.token;
  const refresh = ctx.detail.domain === domain && ctx.detail.status === 'ready';
  if (!refresh) {
    ctx.detail = { domain, status: 'loading', data: null, error: '', external: null };
    renderViewer();
  }
  try {
    const data = await api(`/api/archive/specs/${encodeURIComponent(domain)}`);
    if (token !== ctx.token) return;
    ctx.detail = { domain, status: 'ready', data: data || {}, error: '', external: null };
  } catch (err) {
    if (token !== ctx.token) return;
    const data = err.data || {};
    ctx.detail = { domain, status: 'error', data: null, error: err.message, external: data.external_baseline ? true : null, path: data.path || '' };
  }
  renderViewer();
  applyFocus();
}

// applyFocus pasa el foco al título del visor tras elegir un dominio con el teclado o el ratón.
function applyFocus() {
  if (!ctx.focusViewer) return;
  ctx.focusViewer = false;
  const heading = $('#livingdoc-detail-title');
  if (heading) heading.focus();
}

// enterSpecs se llama al entrar en #/specs y en #/specs/<dominio>, también con atrás y adelante.
export function enterSpecs(route) {
  const domain = route && route.domain ? route.domain : null;
  const changed = domain !== ctx.domain;
  ctx.domain = domain;
  markCurrent();
  if (!domain) {
    ctx.token += 1;
    ctx.detail = { domain: null, status: 'idle', data: null, error: '', external: null };
    renderViewer();
    return;
  }
  if (changed) ctx.raw = false;
  if (ctx.detail.domain !== domain || ctx.detail.status === 'error') {
    loadDetail(domain);
  } else {
    renderViewer();
    applyFocus();
  }
}

// ---- Montaje -----------------------------------------------------------------------------------

export function initLivingDocs() {
  renderViewer();

  $('#specs-filter').addEventListener('input', (event) => {
    ctx.filter = event.target.value;
    renderList();
  });

  $('#livingdoc-domains-list').addEventListener('click', (event) => {
    const link = event.target.closest('a[data-domain]');
    // Un clic normal navega por hash; el foco pasa al visor cuando termine de cargar.
    if (link && link.getAttribute('data-domain') !== ctx.domain && !event.ctrlKey && !event.metaKey && !event.shiftKey) ctx.focusViewer = true;
  });

  $('#livingdoc-viewer').addEventListener('click', (event) => {
    const format = event.target.closest('[data-format]');
    if (!format) return;
    ctx.raw = format.getAttribute('data-format') === 'raw';
    renderViewer();
    const again = $(`#livingdoc-viewer [data-format="${format.getAttribute('data-format')}"]`);
    if (again) again.focus();
  });

  const sync = $('#btn-sync-livingdoc');
  sync.addEventListener('click', async () => {
    const restore = setBusy(sync, 'Sincronizando…');
    try {
      const report = await api('/api/archive/sync', { method: 'POST' });
      const summary = report
        ? `Catálogo sincronizado: ${plural(report.specs_count || 0, 'dominio', 'dominios')}, ${plural(report.requirements_count || 0, 'requisito', 'requisitos')} y ${plural(report.scenarios_count || 0, 'escenario', 'escenarios')}.`
        : 'Catálogo sincronizado.';
      const warnings = report && Array.isArray(report.warnings) ? report.warnings : [];
      ctx.sync = warnings.length
        ? { kind: 'warn', html: `<strong>${escapeHtml(summary)}</strong><ul class="kn-notice-list">${warnings.map(w => `<li>${escapeHtml(w)}</li>`).join('')}</ul>` }
        : null;
      toast(summary, warnings.length ? 'warn' : 'ok');
      await Promise.all([loadLivingDocs(), refreshOverview(['specs'])]);
    } catch (err) {
      ctx.sync = { kind: 'bad', html: `<strong>Fallo al sincronizar.</strong> ${escapeHtml(err.message)}` };
      toast(`Fallo al sincronizar: ${err.message}`, 'bad');
      renderNotice();
    } finally {
      restore();
    }
  });
}
