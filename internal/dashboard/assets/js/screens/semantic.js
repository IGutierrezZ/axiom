// Código y grafo: conectores semánticos, agentes y MCPs, símbolos (con búsqueda y filtro
// de tipo resueltos en el servidor: parámetros query y kind), dependencias entre paquetes
// y reindexación de CodeGraph.

import { api, escapeHtml } from '../core/api.js';
import { $, badge, renderConsole, setBusy, stateEmpty, stateError, stateLoading, statusPill } from '../core/components.js';
import { formatNumber, plural } from '../core/format.js';
import { icon } from '../core/icons.js';
import { refreshOverview } from '../core/overview.js';

const CONNECTOR_LABEL = {
  codegraph: 'CodeGraph (grafo de conocimiento)',
  serena: 'Serena MCP (LSP y Tree-sitter)',
  'native-ast': 'AST nativo de Go (autónomo)'
};
const MODE_LABEL = { auto: 'automático', codegraph: 'CodeGraph', serena: 'Serena MCP', 'native-ast': 'AST nativo de Go' };
const KIND_LABEL = { struct: 'struct', interface: 'interface', func: 'func', method: 'method', type: 'type' };

// Filas de símbolos que se pintan como máximo; el recuento total va siempre en texto.
const MAX_ROWS = 300;

const ctx = { kind: '', query: '', symbolsToken: 0, symbols: null, deps: null, internalOnly: false, debounce: null };

function yesNo(flag, yes, no) {
  return flag ? statusPill('ok', yes) : badge('neutral', no, 'x');
}

// ---- Estado de conectores ------------------------------------------------------------------

function connectorCard(title, installed, configured, note) {
  return `<div class="kn-card">
    <h3 class="kn-card-label">${escapeHtml(title)}</h3>
    <div class="kn-pills">${yesNo(installed, 'Instalado (CLI en PATH)', 'No está en PATH')}${yesNo(configured, 'Configurado en el workspace', 'Sin configurar en el workspace')}</div>
    ${note ? `<p class="kn-hint">${note}</p>` : ''}
  </div>`;
}

function renderAgents(agents) {
  const box = $('#semantic-agents-list');
  if (!agents || agents.length === 0) {
    box.innerHTML = stateEmpty('Sin agentes con MCP', 'No se detectaron agentes con configuración MCP.', 'users');
    return;
  }
  box.innerHTML = `<ul class="kn-agents" aria-label="Agentes de desarrollo">${agents.map(ag => `<li class="kn-agent">
    <div class="kn-agent-head">
      <h3 class="kn-agent-name">${escapeHtml(ag.agent_name)}</h3>
      ${badge('neutral', ag.scope === 'workspace' ? 'Workspace' : 'Global / usuario', ag.scope === 'workspace' ? 'folder' : 'globe')}
      ${ag.configured ? statusPill('ok', 'Configurado') : badge('neutral', 'No detectado', 'x')}
    </div>
    ${ag.config_path ? `<p class="kn-path mono">${escapeHtml(ag.config_path)}</p>` : ''}
    ${ag.details ? `<p class="kn-agent-details">${escapeHtml(ag.details)}</p>` : ''}
  </li>`).join('')}</ul>`;
}

export async function loadSemanticStatus() {
  const box = $('#semantic-status-cards');
  try {
    const data = await api('/api/semantic/status');
    const active = data.active_connector;
    const mode = data.configured_connector || 'auto';
    const nativeReady = !!data.native_ast_ready;
    const warnings = Array.isArray(data.warnings) ? data.warnings : [];
    box.innerHTML = `
      <div class="kn-card kn-card-main">
        <h3 class="kn-card-label">Conector activo</h3>
        <p class="kn-active"><span class="kn-active-dot" aria-hidden="true"></span>${escapeHtml(CONNECTOR_LABEL[active] || active || 'Sin conector')}</p>
        <p class="kn-hint">Modo configurado: ${escapeHtml(MODE_LABEL[mode] || mode)} · ${plural(data.total_symbols || 0, 'símbolo', 'símbolos')} en ${plural(data.total_packages || 0, 'paquete', 'paquetes')}</p>
        <div class="kn-pills">${nativeReady ? statusPill('ok', 'AST nativo listo') : badge('warn', 'AST nativo no disponible', 'alert')}</div>
      </div>
      ${connectorCard('CodeGraph', data.codegraph_installed, data.codegraph_configured, 'Grafo de conocimiento del código.')}
      ${connectorCard('Serena MCP', data.serena_installed, data.serena_configured, data.serena_configured ? '' : 'Se configura con «Sincronizar workspace» en <a href="#/ecosistema">Ecosistema</a>.')}
      ${warnings.length ? `<div class="alert alert-warn kn-status-warn">${icon('alert', 18)}<div class="grow"><strong>Avisos de configuración</strong><ul class="kn-notice-list">${warnings.map(w => `<li>${escapeHtml(w)}</li>`).join('')}</ul></div></div>` : ''}`;
    renderAgents(data.agents);
  } catch (err) {
    box.innerHTML = stateError(err.message, 'Error cargando el estado semántico');
    $('#semantic-agents-list').innerHTML = stateError(err.message, 'No se pudo cargar la lista de agentes');
  }
}

// ---- Símbolos ------------------------------------------------------------------------------------

function renderSymbols(symbols) {
  const box = $('#semantic-symbols-container');
  const count = $('#semantic-symbols-count');
  count.textContent = plural(symbols.length, 'resultado', 'resultados');
  if (symbols.length === 0) {
    box.innerHTML = stateEmpty('Sin símbolos', 'No se encontraron símbolos que coincidan con los filtros.', 'search');
    return;
  }
  const shown = symbols.slice(0, MAX_ROWS);
  const rows = shown.map(sym => `<tr>
    <td class="mono kn-sym-name">${escapeHtml(sym.name)}</td>
    <td><span class="kn-kind kn-kind-${escapeHtml(KIND_LABEL[sym.kind] ? sym.kind : 'type')}">${escapeHtml(sym.kind)}</span></td>
    <td class="mono kn-pkg">${escapeHtml(sym.package)}</td>
    <td class="mono kn-sig">${escapeHtml(sym.signature || '–')}</td>
    <td class="mono kn-loc">${escapeHtml(sym.file_path)}${sym.line_number ? `:${escapeHtml(sym.line_number)}` : ''}</td>
  </tr>`).join('');
  const more = symbols.length > shown.length
    ? `<p class="kn-hint" role="status">Se muestran los primeros ${formatNumber(shown.length)} de ${formatNumber(symbols.length)}. Afina la búsqueda o el tipo para verlos todos.</p>`
    : '';
  box.innerHTML = `<div class="table-wrap kn-table-scroll" role="region" aria-label="Símbolos" tabindex="0">
    <table class="data-table"><thead><tr><th scope="col">Símbolo</th><th scope="col">Tipo</th><th scope="col">Paquete</th><th scope="col">Signatura</th><th scope="col">Ubicación</th></tr></thead><tbody>${rows}</tbody></table>
  </div>${more}`;
}

export async function loadSemanticSymbols(query = ctx.query, kind = ctx.kind) {
  const box = $('#semantic-symbols-container');
  const token = ++ctx.symbolsToken;
  if (ctx.symbols === null) box.innerHTML = stateLoading('Buscando símbolos en el workspace…');
  else box.setAttribute('aria-busy', 'true');
  try {
    const params = new URLSearchParams();
    if (query) params.append('query', query);
    if (kind) params.append('kind', kind);
    const qs = params.toString();
    const symbols = (await api(`/api/semantic/symbols${qs ? `?${qs}` : ''}`)) || [];
    if (token !== ctx.symbolsToken) return;
    ctx.symbols = symbols;
    box.removeAttribute('aria-busy');
    renderSymbols(symbols);
  } catch (err) {
    if (token !== ctx.symbolsToken) return;
    box.removeAttribute('aria-busy');
    $('#semantic-symbols-count').textContent = 'Error';
    box.innerHTML = stateError(err.message, 'Error buscando símbolos');
  }
}

// ---- Dependencias --------------------------------------------------------------------------------

// groupDeps agrupa las relaciones por paquete origen sin repetir destinos.
export function groupDeps(deps) {
  const map = new Map();
  for (const d of deps || []) {
    if (!map.has(d.source_package)) map.set(d.source_package, { internal: new Set(), external: new Set() });
    map.get(d.source_package)[d.is_internal ? 'internal' : 'external'].add(d.target_package);
  }
  return Array.from(map, ([source, t]) => ({ source, internal: [...t.internal].sort(), external: [...t.external].sort() }))
    .sort((a, b) => a.source.localeCompare(b.source));
}

function chips(list) {
  return list.map(p => `<span class="kn-chip mono">${escapeHtml(p)}</span>`).join('');
}

function renderDeps() {
  const box = $('#semantic-dependencies-container');
  if (!ctx.deps) return;
  let groups = groupDeps(ctx.deps);
  if (ctx.internalOnly) groups = groups.filter(g => g.internal.length > 0);
  if (groups.length === 0) {
    box.innerHTML = stateEmpty('Sin dependencias', ctx.internalOnly ? 'No hay dependencias internas entre paquetes.' : 'No se registraron dependencias entre paquetes.', 'graph');
    return;
  }
  const links = groups.reduce((n, g) => n + g.internal.length + (ctx.internalOnly ? 0 : g.external.length), 0);
  box.innerHTML = `<p class="kn-hint" role="status">${escapeHtml(plural(groups.length, 'paquete', 'paquetes'))} · ${escapeHtml(plural(links, 'relación', 'relaciones'))}${ctx.internalOnly ? ' internas' : ''}</p>
    <ul class="kn-deps kn-scroll" aria-label="Dependencias por paquete">${groups.map(g => `<li class="kn-dep">
      <p class="kn-dep-src mono">${escapeHtml(g.source)}</p>
      ${g.internal.length ? `<div class="kn-dep-row"><span class="kn-dep-label">Internas</span><span class="kn-chips">${chips(g.internal)}</span></div>` : ''}
      ${!ctx.internalOnly && g.external.length ? `<div class="kn-dep-row"><span class="kn-dep-label">Externas</span><span class="kn-chips">${chips(g.external)}</span></div>` : ''}
    </li>`).join('')}</ul>`;
}

export async function loadSemanticDependencies() {
  const box = $('#semantic-dependencies-container');
  try {
    ctx.deps = (await api('/api/semantic/dependencies')) || [];
    renderDeps();
  } catch (err) {
    box.innerHTML = stateError(err.message, 'Error calculando las dependencias');
  }
}

export async function loadSemanticData() {
  await Promise.all([loadSemanticStatus(), loadSemanticSymbols(), loadSemanticDependencies()]);
}

// ---- Reindexar -------------------------------------------------------------------------------------

function feedback(kind, html, busy = false) {
  const box = $('#reindex-feedback');
  const lead = busy ? '<span class="spinner" aria-hidden="true"></span>' : icon({ ok: 'check', warn: 'alert', bad: 'alert' }[kind] || 'info', 18);
  box.className = `alert alert-${kind}`;
  box.setAttribute('role', kind === 'bad' ? 'alert' : 'status');
  box.innerHTML = `${lead}<div class="grow">${html}</div>`;
}

async function reindex() {
  const btn = $('#btn-reindex-codegraph');
  const consoleBox = $('#reindex-console');
  const restore = setBusy(btn, 'Reindexando…');
  consoleBox.hidden = true;
  feedback('neutral', 'Ejecutando la reindexación de CodeGraph en el workspace. Puede tardar un rato; no cierres esta pestaña.', true);
  try {
    const data = await api('/api/semantic/reindex', { method: 'POST' });
    const ok = data.success !== false;
    const parts = [];
    if (data.connector) parts.push(`Conector: ${escapeHtml(data.connector)}`);
    if (data.duration) parts.push(`Duración: ${escapeHtml(data.duration)}`);
    feedback(ok ? 'ok' : 'warn', `<strong>${escapeHtml(data.message || 'Reindexación completada.')}</strong>${parts.length ? ` ${parts.join(' · ')}` : ''}`);
    if (data.output) renderConsole(consoleBox, { title: 'Salida de la reindexación', text: data.output, state: ok ? 'ok' : 'error' });
    await Promise.all([loadSemanticData(), refreshOverview(['semantic'])]);
  } catch (err) {
    console.error(err);
    feedback('bad', `<strong>La reindexación ha fallado.</strong> ${escapeHtml(err.message)}`);
    // Con un fallo, el servidor devuelve igualmente el resultado con la salida del proceso.
    const output = err.data && err.data.output;
    if (output) renderConsole(consoleBox, { title: 'Salida de la reindexación', text: output, state: 'error' });
  } finally {
    restore();
  }
}

// ---- Montaje -----------------------------------------------------------------------------------------

export function initSemantic() {
  $('#btn-reindex-codegraph').addEventListener('click', reindex);

  const search = $('#semantic-search-input');
  search.addEventListener('input', () => {
    clearTimeout(ctx.debounce);
    ctx.debounce = setTimeout(() => {
      ctx.query = search.value.trim();
      loadSemanticSymbols();
    }, 250);
  });

  const filters = $('#semantic-kind-filters');
  filters.addEventListener('click', (event) => {
    const btn = event.target.closest('.chip');
    if (!btn) return;
    filters.querySelectorAll('.chip').forEach(b => b.setAttribute('aria-pressed', String(b === btn)));
    ctx.kind = btn.getAttribute('data-kind') || '';
    ctx.query = search.value.trim();
    loadSemanticSymbols();
  });

  const toggle = $('#deps-internal-toggle');
  toggle.addEventListener('click', () => {
    ctx.internalOnly = !ctx.internalOnly;
    toggle.setAttribute('aria-pressed', String(ctx.internalOnly));
    renderDeps();
  });
}
