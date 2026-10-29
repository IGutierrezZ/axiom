// Código y grafo: conectores semánticos, agentes, símbolos y dependencias.
// Provisional hasta T5.

import { api, escapeHtml } from '../core/api.js';
import { $, setBusy } from '../core/components.js';
import { icon } from '../core/icons.js';

let currentKind = '';
let searchTimeout = null;

function errorBanner(prefix, err) {
  return `<div class="error-banner">${escapeHtml(prefix)}: ${escapeHtml(err.message)}</div>`;
}

export async function loadSemanticStatus() {
  const container = $('#semantic-status-cards');
  const agents = $('#semantic-agents-list');
  try {
    const data = await api('/api/semantic/status');

    let connectorClass = 'badge-connector-ast';
    let connectorLabel = 'AST nativo Go (autónomo)';
    if (data.active_connector === 'serena') {
      connectorClass = 'badge-connector-serena';
      connectorLabel = 'Serena MCP (LSP/Tree-sitter)';
    } else if (data.active_connector === 'codegraph') {
      connectorClass = 'badge-connector-codegraph';
      connectorLabel = 'CodeGraph Knowledge Graph';
    }

    const installed = (flag) => flag
      ? `<span class="badge badge-ok">${icon('check', 13)}CLI en PATH</span>`
      : `<span class="badge badge-bad">${icon('x', 13)}No está en PATH</span>`;
    const configured = (flag) => flag
      ? `<span class="badge badge-ok">${icon('check', 13)}En el workspace</span>`
      : '<span class="badge badge-neutral">No está en el workspace</span>';

    container.innerHTML = `
      <div class="stat-card"><div class="stat-label">Conector semántico activo</div><div class="stat-value" style="margin-top:6px"><span class="badge-connector ${connectorClass}">${connectorLabel}</span></div></div>
      <div class="stat-card"><div class="stat-label">Modo configurado</div><div class="stat-value">${escapeHtml(data.configured_connector || 'auto')}</div></div>
      <div class="stat-card"><div class="stat-label">Símbolos indexados</div><div class="stat-value">${data.total_symbols || 0}</div></div>
      <div class="stat-card"><div class="stat-label">Paquetes analizados</div><div class="stat-value">${data.total_packages || 0}</div></div>
      <div class="stat-card"><div class="stat-label">Diagnóstico de CodeGraph</div><div style="display:flex;flex-direction:column;gap:6px;margin-top:6px">${installed(data.codegraph_installed)}${configured(data.codegraph_configured)}</div></div>
      <div class="stat-card"><div class="stat-label">Diagnóstico de Serena MCP</div><div style="display:flex;flex-direction:column;gap:6px;margin-top:6px">${installed(data.serena_installed)}${configured(data.serena_configured)}</div></div>`;

    if (data.warnings && data.warnings.length > 0) {
      const warn = document.createElement('div');
      warn.className = 'alert alert-warn';
      warn.style.gridColumn = '1 / -1';
      warn.innerHTML = `${icon('alert', 18)}<div class="grow"><strong>Aviso de configuración</strong><br>${data.warnings.map(w => '• ' + escapeHtml(w)).join('<br>')}</div>`;
      container.appendChild(warn);
    }

    if (!data.agents || data.agents.length === 0) {
      agents.innerHTML = '<div class="state">No se detectaron agentes con configuración MCP.</div>';
    } else {
      agents.innerHTML = data.agents.map(ag => `<div class="agent-item">
        <h4>
          <span style="display:inline-flex;align-items:center;gap:8px;flex-wrap:wrap">
            <span>${escapeHtml(ag.agent_name)}</span>
            <span class="badge badge-neutral">${ag.scope === 'workspace' ? icon('folder', 13) + 'Workspace' : icon('globe', 13) + 'Global / usuario'}</span>
          </span>
          ${ag.configured ? `<span class="badge badge-ok">${icon('check', 13)}Configurado</span>` : '<span class="badge badge-neutral">No detectado</span>'}
        </h4>
        <p><strong>Config:</strong> <code>${escapeHtml(ag.config_path)}</code></p>
        <p style="margin-top:4px">${escapeHtml(ag.details)}</p>
      </div>`).join('');
    }
  } catch (err) {
    container.innerHTML = errorBanner('Error cargando el estado semántico', err);
  }
}

export async function loadSemanticSymbols(query, kind) {
  const tbody = $('#semantic-symbols-body');
  tbody.innerHTML = '<tr><td colspan="5" class="loading-state">Buscando símbolos en el workspace…</td></tr>';
  try {
    const params = new URLSearchParams();
    if (query) params.append('query', query);
    if (kind) params.append('kind', kind);
    const symbols = await api(`/api/semantic/symbols?${params.toString()}`);
    if (!symbols || symbols.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No se encontraron símbolos que coincidan con los filtros.</td></tr>';
      return;
    }
    const kindClass = { struct: 'badge-kind-struct', interface: 'badge-kind-interface', func: 'badge-kind-func', method: 'badge-kind-method' };
    tbody.innerHTML = symbols.map(sym => `<tr>
      <td><strong><code>${escapeHtml(sym.name)}</code></strong></td>
      <td><span class="badge-kind ${kindClass[sym.kind] || 'badge-kind-type'}">${escapeHtml(sym.kind)}</span></td>
      <td><code>${escapeHtml(sym.package)}</code></td>
      <td><code class="muted" style="font-size:12px">${escapeHtml(sym.signature || '-')}</code></td>
      <td><span class="dim" style="font-size:12px">${escapeHtml(sym.file_path)}:${sym.line_number}</span></td>
    </tr>`).join('');
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="5" class="error-banner">Error: ${escapeHtml(err.message)}</td></tr>`;
  }
}

export async function loadSemanticDependencies() {
  const container = $('#semantic-dependencies-container');
  try {
    const deps = await api('/api/semantic/dependencies');
    if (!deps || deps.length === 0) {
      container.innerHTML = '<div class="state">No se registraron dependencias entre paquetes.</div>';
      return;
    }
    container.innerHTML = deps.map(dep => `<div class="dep-chip ${dep.is_internal ? 'internal' : ''}">
      <strong><code>${escapeHtml(dep.source_package)}</code></strong>
      <span class="dim-icon">${icon('arrowRight', 14)}</span>
      <code>${escapeHtml(dep.target_package)}</code>
      ${dep.is_internal ? '<span class="badge badge-accent">Interno</span>' : ''}
    </div>`).join('');
  } catch (err) {
    container.innerHTML = errorBanner('Error calculando las dependencias', err);
  }
}

export async function loadSemanticData() {
  await Promise.all([
    loadSemanticStatus(),
    loadSemanticSymbols($('#semantic-search-input').value.trim(), currentKind),
    loadSemanticDependencies()
  ]);
}

export function initSemantic() {
  $('#btn-refresh-semantic').addEventListener('click', () => loadSemanticData());

  const reindex = $('#btn-reindex-codegraph');
  const feedback = $('#reindex-feedback');
  reindex.addEventListener('click', async () => {
    const restore = setBusy(reindex, 'Reindexando…');
    feedback.classList.remove('hidden');
    feedback.textContent = 'Ejecutando la reindexación semántica de CodeGraph en el workspace…';
    try {
      const data = await api('/api/semantic/reindex', { method: 'POST' });
      feedback.innerHTML = `${icon('check', 14)} <strong>${escapeHtml(data.message || 'Reindexación completada')}</strong> (${escapeHtml(data.duration || '')})`;
      await loadSemanticData();
    } catch (err) {
      console.error(err);
      feedback.innerHTML = `<span class="text-danger">Error: ${escapeHtml(err.message)}</span>`;
    } finally {
      restore();
    }
  });

  const search = $('#semantic-search-input');
  search.addEventListener('input', () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => loadSemanticSymbols(search.value.trim(), currentKind), 250);
  });

  const filters = $('#semantic-kind-filters');
  filters.addEventListener('click', (event) => {
    const btn = event.target.closest('.filter-btn');
    if (!btn) return;
    filters.querySelectorAll('.filter-btn').forEach(b => b.setAttribute('aria-pressed', String(b === btn)));
    currentKind = btn.getAttribute('data-kind') || '';
    loadSemanticSymbols(search.value.trim(), currentKind);
  });
}
