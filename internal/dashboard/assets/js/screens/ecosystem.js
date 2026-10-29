// Ecosistema: Doctor, operaciones (sync, upgrade, respaldos), topología y modelos.
// Provisional hasta T6.

import { api, escapeHtml } from '../core/api.js';
import { $, renderConsole, confirmDialog, promptDialog, setBusy, toast } from '../core/components.js';
import { icon } from '../core/icons.js';
import { state } from '../core/state.js';

function showConsoleOutput(title, text, stateName = '') {
  renderConsole($('#eco-action-output'), { title, text, state: stateName });
}

// Presenta el resultado de ambas fases de la cadena upgrade->sync (REQ-22.4),
// incluida la omisión de sync con su motivo y la instrucción de reinicio
// cuando el binario en ejecución fue reemplazado. No inventa estados que el
// DTO no declara: lee exactamente phases.upgrade y phases.sync.
export function formatEcosystemUpgradeSequence(data) {
  if (!data || !data.phases) {
    return ((data && data.output) || []).join('\n') || (data && data.message) || '';
  }
  const upgrade = data.phases.upgrade || {};
  const sync = data.phases.sync || {};
  const lines = [];

  let upgradeLine = 'Fase upgrade: ' + (upgrade.status || 'desconocido');
  if (upgrade.restart_required) upgradeLine += ' — binario axiom reemplazado';
  lines.push(upgradeLine);
  if (upgrade.manual_hint) lines.push('Actualización manual requerida: ' + upgrade.manual_hint);
  if (upgrade.error) lines.push('Error de upgrade: ' + upgrade.error);
  if (upgrade.output && upgrade.output.length) lines.push(...upgrade.output);

  if (sync.executed) {
    lines.push('Fase sync: ejecutada ' + (sync.success ? 'con éxito' : 'con errores'));
    if (sync.output && sync.output.length) lines.push(...sync.output);
    if (sync.error) lines.push('Error de sync: ' + sync.error);
  } else {
    const skipReasons = {
      'restart-required': 'omitida — reinicia axiom para continuar; después ejecuta sync',
      'upgrade-failed': 'omitida — la fase upgrade falló'
    };
    lines.push('Fase sync: ' + (skipReasons[sync.skipped_reason] || ('omitida (' + (sync.skipped_reason || '') + ')')));
  }
  return lines.join('\n');
}

// ---- Topología (alimentada por /api/workspace) -----------------------------

export function renderTopology() {
  const ws = state.workspace;
  if (!ws) return;
  $('#top-ws-name').textContent = ws.name;
  $('#top-ws-topology').textContent = ws.topology;
  $('#top-ws-specs').textContent = ws.specs_repository || 'openspec/';
  const health = $('#top-ws-health');
  if (ws.compliant) {
    health.innerHTML = `<span class="text-ok">${icon('check', 18)} Conforme</span>`;
  } else {
    health.innerHTML = `<span class="text-bad">${icon('x', 18)} ${ws.is_configured ? 'No conforme' : 'Sin configurar'}</span>`;
  }

  const tbody = $('#topology-roles-tbody');
  const roles = ws.roles ? Object.entries(ws.roles) : [];
  if (roles.length === 0) {
    tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No hay roles declarados.</td></tr>';
    return;
  }
  tbody.innerHTML = roles.map(([id, r]) => `<tr>
    <td><code>${escapeHtml(id)}</code></td>
    <td><strong>${escapeHtml(r.name || id)}</strong></td>
    <td><span class="badge badge-${escapeHtml(r.gate_policy)}">${escapeHtml(r.gate_policy || 'blocking')}</span></td>
    <td>${escapeHtml((r.tech || []).join(', ') || 'N/D')}</td>
    <td><code>${escapeHtml((r.repositories || []).join(', ') || '.')}</code></td>
  </tr>`).join('');
}

// ---- Doctor, respaldos y modelos -------------------------------------------

async function loadDoctor() {
  const tbody = $('#eco-doctor-tbody');
  const badge = $('#eco-doctor-badge');
  try {
    const data = await api('/api/ecosystem/doctor');
    badge.innerHTML = data.healthy ? `${icon('check', 13)}Sistema saludable` : `${icon('alert', 13)}Advertencias detectadas`;
    badge.className = `badge ${data.healthy ? 'badge-ok' : 'badge-warn'}`;
    if (!data.checks || data.checks.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No se recibieron diagnósticos.</td></tr>';
      return;
    }
    tbody.innerHTML = data.checks.map(c => {
      let status = `<span class="badge badge-ok">${icon('check', 13)}Saludable</span>`;
      if (c.status === 'warning') status = `<span class="badge badge-warn">${icon('alert', 13)}Advertencia</span>`;
      else if (c.status === 'error') status = `<span class="badge badge-bad">${icon('x', 13)}Error</span>`;
      return `<tr>
        <td><strong>${escapeHtml(c.name)}</strong></td>
        <td><span class="badge badge-phase">${escapeHtml(c.category)}</span></td>
        <td>${status}</td>
        <td>${escapeHtml(c.details)}</td>
        <td>${c.recommendation ? escapeHtml(c.recommendation) : '<span class="dim">—</span>'}</td>
      </tr>`;
    }).join('');
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="5" class="error-banner">Error cargando el diagnóstico: ${escapeHtml(err.message)}</td></tr>`;
  }
}

async function loadBackups() {
  const tbody = $('#eco-backups-tbody');
  try {
    const data = (await api('/api/ecosystem/backups')) || [];
    $('#eco-backups-count').textContent = `${data.length} respaldos`;
    if (data.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No hay respaldos registrados en ~/.axiom/backups/</td></tr>';
      return;
    }
    tbody.innerHTML = data.map(b => `<tr>
      <td><code>${escapeHtml(b.name)}</code>${b.pinned ? ' <span class="badge">Fijado</span>' : ''}</td>
      <td>${escapeHtml(b.created)}</td>
      <td>${escapeHtml(b.description || 'Sin descripción')}</td>
      <td><span class="badge">${(b.files || []).length} archivos</span></td>
      <td><button type="button" class="btn btn-sm btn-restore-backup" data-id="${escapeHtml(b.name)}">Restaurar</button></td>
    </tr>`).join('');
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="5" class="error-banner">Error cargando los respaldos: ${escapeHtml(err.message)}</td></tr>`;
  }
}

async function loadModels() {
  const tbody = $('#eco-models-tbody');
  try {
    const data = await api('/api/ecosystem/models');
    $('#eco-persona-badge').textContent = `Persona: ${data.active_persona || 'axiom'}`;
    if (!data.assignments || data.assignments.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="empty-state">No se detectaron asignaciones explícitas de modelos en opencode.json. Se usan los presets predeterminados de Axiom.</td></tr>';
      return;
    }
    tbody.innerHTML = data.assignments.map(m => `<tr>
      <td><strong>${escapeHtml(m.agent)}</strong></td>
      <td><span class="badge">${escapeHtml(m.role)}</span></td>
      <td><code>${escapeHtml(m.model)}</code></td>
      <td>${m.reasoning ? `<span class="badge">${escapeHtml(m.reasoning)}</span>` : '<span class="dim">estándar</span>'}</td>
    </tr>`).join('');
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="4" class="error-banner">Error cargando los modelos: ${escapeHtml(err.message)}</td></tr>`;
  }
}

export async function loadEcosystem() {
  await Promise.all([loadDoctor(), loadBackups(), loadModels()]);
}

// ---- Operaciones -------------------------------------------------------------

export function initEcosystem() {
  $('#btn-refresh-ecosystem').addEventListener('click', () => loadEcosystem());

  const sync = $('#btn-eco-sync');
  sync.addEventListener('click', async () => {
    const restore = setBusy(sync, 'Sincronizando…');
    showConsoleOutput('Sincronizando workspace', 'Ejecutando axiom sync --scope=workspace…');
    try {
      const data = await api('/api/ecosystem/sync', { body: { scope: 'workspace' } });
      showConsoleOutput('Resultado de la sincronización', (data.output || []).join('\n') || data.message, data.success ? 'ok' : 'error');
    } catch (err) {
      const data = err.data || {};
      showConsoleOutput('Error en la sincronización', ((data.output || []).join('\n') || '') + (data.output ? '\n' : '') + err.message, 'error');
    } finally {
      restore();
    }
  });

  const upgrade = $('#btn-eco-upgrade');
  upgrade.addEventListener('click', async () => {
    const channel = $('#select-eco-channel').value || 'stable';
    const restore = setBusy(upgrade, 'Actualizando…');
    showConsoleOutput(`Actualizando herramientas (${channel})`, `Ejecutando la comprobación y la actualización en el canal ${channel}…`);
    try {
      const data = await api('/api/ecosystem/upgrade', { body: { channel } });
      showConsoleOutput('Resultado de la actualización', formatEcosystemUpgradeSequence(data), data.success ? 'ok' : 'error');
    } catch (err) {
      // El upgrade puede responder 500 con las fases en el cuerpo: se presentan igualmente.
      const text = err.data && err.data.phases ? formatEcosystemUpgradeSequence(err.data) : err.message;
      showConsoleOutput('Error en la actualización', text, 'error');
    } finally {
      restore();
    }
  });

  const backup = $('#btn-eco-create-backup');
  backup.addEventListener('click', async () => {
    const desc = await promptDialog({
      title: 'Crear respaldo',
      message: 'Descripción para el nuevo respaldo (opcional):',
      confirmLabel: 'Crear respaldo',
      value: 'Respaldo manual desde la interfaz web'
    });
    if (desc === null) return;
    const restore = setBusy(backup, 'Creando…');
    try {
      const data = await api('/api/ecosystem/backups/create', { body: { description: desc } });
      showConsoleOutput('Respaldo creado', `Snapshot creado con éxito con ID: ${data.name}`, 'ok');
      await loadBackups();
    } catch (err) {
      toast(`Error al crear el respaldo: ${err.message}`, 'bad');
    } finally {
      restore();
    }
  });

  $('#eco-backups-tbody').addEventListener('click', async (event) => {
    const btn = event.target.closest('.btn-restore-backup');
    if (!btn) return;
    const id = btn.getAttribute('data-id');
    const ok = await confirmDialog({
      title: 'Restaurar respaldo',
      message: `¿Restaurar el respaldo ${id}? Los archivos actuales serán sustituidos.`,
      confirmLabel: 'Restaurar',
      danger: true
    });
    if (!ok) return;
    const restore = setBusy(btn, 'Restaurando…');
    try {
      const data = await api('/api/ecosystem/backups/restore', { body: { name: id } });
      showConsoleOutput(`Restauración de ${id}`, (data.output || []).join('\n') || data.message, 'ok');
      toast(data.message || 'Respaldo restaurado correctamente', 'ok');
    } catch (err) {
      toast(`Fallo al restaurar: ${err.message}`, 'bad');
    } finally {
      restore();
    }
  });
}
