// Inicio: marcador de posición hasta T4 y aviso zero-config.
// Si el proyecto no está configurado (is_configured === false), #zero-config-hero
// sustituye al contenido de Inicio.

import { api, escapeHtml } from '../core/api.js';
import { $, toast, setBusy } from '../core/components.js';
import { state } from '../core/state.js';

let reloadAll = async () => {};

export function renderInicio() {
  const ws = state.workspace;
  const hero = $('#zero-config-hero');
  const content = $('#inicio-content');
  if (!ws) return;

  const eyebrow = $('#inicio-eyebrow');
  const today = new Date().toLocaleDateString('es-ES', { weekday: 'long', day: 'numeric', month: 'long' });
  eyebrow.textContent = today.charAt(0).toUpperCase() + today.slice(1);

  if (ws.is_configured) {
    hero.hidden = true;
    content.hidden = false;
    return;
  }
  hero.hidden = false;
  content.hidden = true;
  $('#zero-config-msg').innerHTML = `No se encontró un archivo <code>axiom.yaml</code> en <code>${escapeHtml(ws.root)}</code>. Puedes inicializarlo en 1 clic con detección automática del stack:`;
  const tech = $('#zero-config-tech');
  tech.innerHTML = '';
  if (ws.detected_tech) {
    const dt = ws.detected_tech;
    const labels = [`Tecnología: ${(dt.primary_language || 'genérico').toUpperCase()}`, ...(dt.frameworks || [])];
    tech.innerHTML = labels.map(l => `<span class="badge badge-accent">${escapeHtml(l)}</span>`).join('');
  }
}

export function initInicio(reload) {
  reloadAll = reload;
  const btn = $('#btn-init-project');
  btn.addEventListener('click', async () => {
    const restore = setBusy(btn, 'Inicializando…');
    try {
      await api('/api/projects/init', { body: { path: state.workspace ? state.workspace.root : '' } });
      await reloadAll();
    } catch (err) {
      toast(`Error al inicializar el proyecto: ${err.message}`, 'bad');
    } finally {
      restore();
    }
  });
}
