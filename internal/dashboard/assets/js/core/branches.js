// Ramas y worktrees (INC-25): la lógica de las cinco funciones que el dashboard
// anterior ofrecía y que el rediseño conserva, separada del DOM para poder
// ejecutarla en las pruebas.
//
//   1. Avisos del listado de incrementos (cabecera X-Axiom-Notes) e insignias de
//      «pendiente de archivar» y de rama del worktree.
//   2. Política de ramas efectiva, en solo lectura (rama de integración y prefijo,
//      más los ajustes por repositorio).
//   3. Preferencias personales de worktree (modo y raíz): textos y valores por defecto.
//   4. Preferencias que se envían al registrar un proyecto.
//   5. Creación de un incremento con el reintento tras una decisión del servidor.
//
// Todo dato del servidor se escapa aquí antes de interpolarlo en HTML.

import { escapeHtml } from './api.js';
import { tableWrap } from './components.js';
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

// ---- 2. Política de ramas (solo lectura) -------------------------------------------

// branchPolicyRows devuelve los pares <dt>/<dd> de la política efectiva del workspace,
// para la lista de datos de la tarjeta Topología. Se declara en axiom.yaml: aquí no se edita.
export function branchPolicyRows(ws) {
  const policy = (ws && ws.branchPolicy) || {};
  const row = (label, value) => `<dt>${label}</dt><dd class="mono">${value ? `<code>${escapeHtml(value)}</code>` : '—'}</dd>`;
  return row('Rama de integración', policy.integrationBranch) + row('Prefijo de ramas de cambio', policy.branchPrefix);
}

// repoPolicyHtml es la tabla de los ajustes por repositorio (`repositories:` de
// axiom.yaml). Un hueco significa que el repositorio hereda el valor del workspace.
// Sin entradas no hay tabla.
export function repoPolicyHtml(ws) {
  const repos = ws && Array.isArray(ws.repositories) ? ws.repositories : [];
  if (repos.length === 0) return '';
  const cell = value => (value ? `<code>${escapeHtml(value)}</code>` : '<span class="dim">hereda del workspace</span>');
  const rows = repos.map(repo => `<tr>
    <th scope="row"><code>${escapeHtml(repo.path)}</code></th>
    <td>${cell(repo.integrationBranch)}</td>
    <td>${cell(repo.branchPrefix)}</td>
  </tr>`).join('');
  const table = `<table class="data-table eco-repo-policy"><thead><tr><th scope="col">Repositorio</th><th scope="col">Rama de integración</th><th scope="col">Prefijo de ramas</th></tr></thead><tbody>${rows}</tbody></table>`;
  return `<h3 class="eco-sub">Ajustes de ramas por repositorio</h3>${tableWrap(table, 'Ajustes de ramas por repositorio')}`;
}

// ---- 3 y 4. Preferencias personales de worktree ------------------------------------

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

// ---- 5. Crear un incremento con decisión -------------------------------------------

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
