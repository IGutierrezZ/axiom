// Ramas y worktrees (INC-25): la lógica de las cuatro funciones que el dashboard
// anterior ofrecía y que el rediseño conserva, separada del DOM para poder
// ejecutarla en las pruebas.
//
//   1. Avisos del listado de incrementos (cabecera X-Axiom-Notes) e insignias de
//      «pendiente de archivar» y de rama del worktree.
//   2. Preferencias personales de worktree (modo y raíz): textos y valores por defecto.
//   3. Preferencias que se envían al registrar un proyecto.
//   4. Creación de un incremento con el reintento tras una decisión del servidor.
//
// La política de ramas efectiva y los ajustes por repositorio ya no se pintan aquí: se ven y se
// editan en Ajustes del proyecto (screens/ajustes.js, a partir del catálogo de /api/settings).
//
// Todo dato del servidor se escapa aquí antes de interpolarlo en HTML.

import { escapeHtml } from './api.js';
import { icon } from './icons.js';

// ---- 1. Avisos del listado e insignias ---------------------------------------------

// parseNotes decodifica la cabecera X-Axiom-Notes: un array JSON de frases (cambios
// pendientes de archivar, worktrees desaparecidos, falta de la referencia de
// integración). No son errores ni bloquean nada. Sin cabecera, o con una cabecera
// que no es un array de textos, no hay aviso.
export function parseNotes(header) {
  if (!header) return [];
  let parsed;
  try {
    parsed = JSON.parse(header);
  } catch (_) {
    return [];
  }
  if (!Array.isArray(parsed)) return [];
  return parsed.filter(note => typeof note === 'string' && note.trim() !== '');
}

// pendingArchiveBadge marca un cambio activo ya fusionado en la rama de integración
// que aún no se ha archivado. Es informativo; el título dice qué orden lo archiva.
export function pendingArchiveBadge(inc) {
  if (!inc || !inc.pendingArchive) return '';
  const hint = `Fusionado en la rama de integración y sin archivar. Para archivarlo: axiom branch ensure --change ${inc.name} --archive`;
  return `<span class="badge badge-warn" title="${escapeHtml(hint)}">${icon('archive', 13)}Pendiente de archivar</span>`;
}

// branchBadge muestra la rama de un cambio que vive en un worktree registrado.
export function branchBadge(inc) {
  if (!inc || !inc.branch) return '';
  const where = inc.worktreePath ? `Worktree: ${inc.worktreePath}` : 'Rama del cambio';
  return `<span class="badge badge-neutral badge-mono" title="${escapeHtml(where)}">${icon('graph', 13)}${escapeHtml(inc.branch)}</span>`;
}

// ---- 2 y 3. Preferencias personales de worktree ------------------------------------

// defaultWorktreeRoot es la raíz de worktrees por defecto de un proyecto: la ruta del
// proyecto con el sufijo -wt. Devuelve '' si todavía no hay una ruta de la que calcularla.
export function defaultWorktreeRoot(projectPath) {
  const clean = String(projectPath || '').trim().replace(/[\\/]+$/, '');
  if (!clean || /[\\/:]$/.test(clean) || !/[\\/]/.test(clean)) return '';
  return `${clean}-wt`;
}

// registrationPrefs devuelve los campos de worktree que se envían al registrar un
// proyecto. `worktree` es el modo por defecto: elegirlo no guarda nada, de modo que
// volver a vincular un proyecto ya registrado no pisa el modo que ya tenía. La raíz
// vacía tampoco se envía.
export function registrationPrefs(mode, root) {
  const prefs = {};
  const chosenMode = String(mode || '').trim();
  const chosenRoot = String(root || '').trim();
  if (chosenMode && chosenMode !== 'worktree') prefs.worktree_mode = chosenMode;
  if (chosenRoot) prefs.worktree_root = chosenRoot;
  return prefs;
}

// effectivePrefsText resume lo que rige (con los valores por defecto aplicados).
export function effectivePrefsText(effective) {
  if (!effective || !effective.mode || !effective.root) return '';
  return `Vigentes: modo ${effective.mode}, raíz ${effective.root}`;
}

// ---- 4. Crear un incremento con decisión -------------------------------------------

// decisionOf devuelve la decisión que el servidor pide ({ question, choices }) si el
// error es un 409 `decision_required` con respuestas utilizables; si no, null.
export function decisionOf(err) {
  const data = err && err.status === 409 ? err.data : null;
  if (!data || data.code !== 'decision_required' || !data.decision) return null;
  const choices = Array.isArray(data.decision.choices) ? data.decision.choices.filter(choice => choice && choice.answer) : [];
  if (choices.length === 0) return null;
  return { question: String(data.decision.question || ''), choices };
}

// refusalError añade al mensaje de un rechazo 409 la continuación que el servidor propone.
function refusalError(err) {
  const continuation = err && err.data && err.data.continuation;
  return continuation ? new Error(`${err.message} Continuación: ${continuation}`) : err;
}

function ensureCreated(data) {
  if (data && data.success === false) throw new Error(data.error || 'Error al crear el incremento');
  return data;
}

// createWithDecision crea un incremento con `post(payload)`. Con el modo de trabajo en
// `ask` y sin respuesta, el servidor contesta 409 con la decisión: se presenta con
// `ask(decision)` (devuelve la respuesta elegida o null si se cancela) y la petición se
// repite una sola vez con `mode` igual a esa respuesta. Cualquier otro rechazo 409
// trae su error y su continuación, que se muestran juntos.
export async function createWithDecision(post, ask, payload) {
  let decision;
  try {
    return ensureCreated(await post(payload));
  } catch (err) {
    decision = decisionOf(err);
    if (!decision) throw refusalError(err);
  }
  const answer = await ask(decision);
  if (!answer) throw new Error('Creación cancelada: no elegiste cómo trabajar el cambio.');
  try {
    return ensureCreated(await post({ ...payload, mode: answer }));
  } catch (err) {
    if (decisionOf(err)) throw new Error('El servidor volvió a pedir la decisión tras responderla.');
    throw refusalError(err);
  }
}

// ---- 5. Alta desde un work item (prov:WEB:9) ---------------------------------------

// WorkItemStartError es un fallo del alta desde un work item. Si `change` no está vacío, la fase 1
// ya dejó su staging: `actions` dice qué ofrece la interfaz («retry», «discard»). Todos los textos se
// pintan con textContent: pueden derivar del tracker.
export class WorkItemStartError extends Error {
  constructor(message, change, actions) {
    super(message);
    this.name = 'WorkItemStartError';
    this.change = change || '';
    this.actions = change ? actions : [];
  }
}

// workItemFailure traduce el rechazo del servidor: mensaje, motivo, continuación y variable de
// entorno, juntos y como texto plano.
export function workItemFailure(err) {
  const data = (err && err.data) || {};
  const parts = [(err && err.message) || data.error || 'El alta desde el work item ha fallado.'];
  if (data.reason) parts.push(`Motivo: ${data.reason}`);
  if (data.continuation) parts.push(`Continuación: ${data.continuation}`);
  if (data.envVar) parts.push(`Variable de entorno: ${data.envVar}`);
  const actions = Array.isArray(data.actions) ? data.actions.filter(a => a === 'retry' || a === 'discard') : [];
  return new WorkItemStartError(parts.join(' '), typeof data.change === 'string' ? data.change : '', actions);
}

// startFromWorkItem da el alta con `post(payload)`. Un 409 `decision_required` se presenta con
// `ask(decision)` (el selector de modo del alta manual) y se repite UNA vez con `{change, mode}`,
// sin repetir la fase 1. Cancelar la decisión, o cualquier fallo posterior a la fase 1, deja el
// staging: el error ofrece reintentar con ese `change` o descartarlo.
export async function startFromWorkItem(post, ask, payload) {
  try {
    return await post(payload);
  } catch (err) {
    const decision = decisionOf(err);
    const change = err && err.data && typeof err.data.change === 'string' ? err.data.change : '';
    if (!decision || !change) throw workItemFailure(err);
    const answer = await ask(decision);
    if (!answer) {
      throw new WorkItemStartError('Alta cancelada: no elegiste cómo trabajar el cambio. El work item ya está preparado.', change, ['retry', 'discard']);
    }
    try {
      return await post({ change, mode: answer, intent: payload.intent, type: payload.type });
    } catch (second) {
      if (decisionOf(second)) {
        throw new WorkItemStartError('El servidor volvió a pedir la decisión tras responderla.', change, ['retry', 'discard']);
      }
      throw workItemFailure(second);
    }
  }
}
