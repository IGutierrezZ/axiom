// Servidores MCP excluidos en el proyecto abierto (Ecosistema): casillas para los MCP de
// usuario de Claude Code (solo sus nombres), un campo para añadir otro nombre (por ejemplo
// un conector de claude.ai) y «Guardar», que envía la lista a
// POST /api/projects/active/mcp-exclusions. El servidor la guarda en el Hub y la aplica a
// la configuración local del agente en el proyecto y en sus worktrees.
//
// La lógica (normalizar, añadir, cuerpo, informe) vive en core/mcpexclusions.js, sin DOM
// y probada con Node; aquí solo se pinta y se cablean los eventos.

import { api, escapeHtml } from '../core/api.js';
import { $, setBusy, stateEmpty, stateError, toast } from '../core/components.js';
import { AGENT, addName, buildBody, hasFailures, normalizeNames, reportLines } from '../core/mcpexclusions.js';

// Último estado recibido del servidor, y la selección y los candidatos en pantalla.
let current = {};
let candidates = [];
let selected = [];

function checkboxHtml(name, index) {
  const id = `mcp-excl-${index}`;
  const checked = selected.includes(name) ? ' checked' : '';
  return `<label class="eco-mcp-item" for="${id}"><input type="checkbox" id="${id}" data-mcp-name="${escapeHtml(name)}"${checked}> <span class="mono">${escapeHtml(name)}</span></label>`;
}

function renderList() {
  const list = $('#mcp-excl-list');
  list.innerHTML = candidates.length
    ? candidates.map(checkboxHtml).join('')
    : '<p class="eco-note">No se han encontrado servidores MCP de usuario. Añade a mano el nombre de uno.</p>';
}

function render(data) {
  current = data.exclusions || {};
  selected = normalizeNames(current[AGENT]);
  candidates = normalizeNames(data.candidates);
  renderList();
  const dirs = (data.dirs || []).map(dir => `<code>${escapeHtml(dir)}</code>`).join(' ');
  $('#mcp-excl-dirs').innerHTML = dirs ? `Se aplica en: ${dirs}.` : '';
}

function showStatus(message, tone = '') {
  const status = $('#mcp-excl-status');
  status.textContent = message;
  status.className = `field-hint${tone ? ` text-${tone}` : ''}`;
}

function showReport(report) {
  const lines = reportLines(report);
  $('#mcp-excl-report').innerHTML = lines.length
    ? `<ul class="eco-note">${lines.map(line => `<li>${escapeHtml(line)}</li>`).join('')}</ul>`
    : '';
}

export async function loadMcpExclusions() {
  const stateBox = $('#eco-mcp-state');
  const form = $('#form-mcp-exclusions');
  try {
    render(await api('/api/projects/active/mcp-exclusions'));
    stateBox.innerHTML = '';
    form.hidden = false;
  } catch (err) {
    console.error(err);
    form.hidden = true;
    stateBox.innerHTML = err.status === 409
      ? stateEmpty('El proyecto activo no está registrado en el Hub', err.message, 'folder')
      : stateError(err.message, 'No se pudieron leer las exclusiones de servidores MCP');
  }
}

function addTypedName() {
  const input = $('#input-mcp-excl-name');
  const result = addName(candidates, selected, input.value);
  if (!result.added) return;
  candidates = result.candidates;
  selected = result.selected;
  input.value = '';
  renderList();
}

async function saveMcpExclusions() {
  const restore = setBusy($('#btn-save-mcp-exclusions'), 'Guardando…');
  showStatus('');
  try {
    const data = await api('/api/projects/active/mcp-exclusions', {
      method: 'POST',
      body: buildBody(current, selected)
    });
    render(data);
    showReport(data.report);
    if (hasFailures(data.report)) {
      showStatus('Guardado en tu Hub, pero no se pudo aplicar en algún directorio.', 'bad');
    } else {
      showStatus('Guardado en tu Hub y aplicado. El cambio rige en las sesiones nuevas.', 'ok');
    }
    toast('Exclusiones de servidores MCP guardadas');
  } catch (err) {
    showStatus(`No se pudieron guardar las exclusiones: ${err.message}`, 'bad');
    toast('No se pudieron guardar las exclusiones de servidores MCP', 'bad');
  } finally {
    restore();
  }
}

export function initMcpExclusions() {
  $('#form-mcp-exclusions').addEventListener('submit', event => {
    event.preventDefault();
    saveMcpExclusions();
  });
  $('#btn-add-mcp-excl').addEventListener('click', addTypedName);
  $('#input-mcp-excl-name').addEventListener('keydown', event => {
    if (event.key === 'Enter') {
      event.preventDefault();
      addTypedName();
    }
  });
  $('#mcp-excl-list').addEventListener('change', event => {
    const box = event.target.closest('[data-mcp-name]');
    if (!box) return;
    const name = box.dataset.mcpName;
    selected = normalizeNames(box.checked ? [...selected, name] : selected.filter(n => n !== name));
  });
}
