// Reglas y piezas de interfaz compartidas por Incrementos y por el detalle:
// fases SDD, tipo, bandera de estado única, barra de progreso, minipaso y la
// orden de agente que toca en cada incremento.

import { escapeHtml } from './api.js';
import { icon } from './icons.js';

export const PHASES = [
  { id: 'explore', label: 'Explore' },
  { id: 'spec', label: 'Spec' },
  { id: 'design', label: 'Design' },
  { id: 'tasks', label: 'Tasks' },
  { id: 'apply', label: 'Apply' },
  { id: 'verify', label: 'Verify' },
  { id: 'archive', label: 'Archive' }
];

export function phaseIndex(phase) {
  const i = PHASES.findIndex(p => p.id === phase);
  return i < 0 ? 0 : i;
}

export function isArchived(inc) {
  return !!inc && inc.type === 'archived';
}

// ---- Tipo ---------------------------------------------------------------------

// typeInfo: feature → Feature (acento), fix o is_bug → Bug (--bad), refactor →
// neutro, architecture → Arquitectura (--warn). La API hoy solo devuelve
// feature y fix (SDD) y odd (las features ODD de los worktrees); los otros dos ya
// están previstos en el diseño.
export function typeInfo(inc) {
  if (inc && (inc.is_bug || inc.change_type === 'fix')) return { id: 'bug', label: 'Bug', iconName: 'bug' };
  switch (inc && inc.change_type) {
    case 'refactor': return { id: 'refactor', label: 'Refactor', iconName: 'refresh' };
    case 'architecture': return { id: 'architecture', label: 'Arquitectura', iconName: 'cube' };
    case 'odd': return { id: 'odd', label: 'ODD', iconName: 'list' };
    default: return { id: 'feature', label: 'Feature', iconName: 'sparkles' };
  }
}

// typeChip: siempre con icono y texto, nunca solo color.
export function typeChip(inc) {
  const t = typeInfo(inc);
  return `<span class="type-chip type-${t.id}">${icon(t.iconName, 13)}${escapeHtml(t.label)}</span>`;
}

// ---- Bandera de estado (una sola, por prioridad) -----------------------------------

// Prioridad: ready_for_global_verify > waiting_roles > ready_for_archive >
// pending_spec > ready_for_design.
export function statusFlag(inc) {
  if (!inc || isArchived(inc)) return null;
  if (inc.ready_for_global_verify) return { id: 'gverify', label: 'Verificación global pendiente', tone: 'accent', iconName: 'shield' };
  if (inc.waiting_roles) {
    const roles = (inc.pending_roles || []).join(', ');
    return { id: 'waiting', label: roles ? `Esperando a: ${roles}` : 'Esperando a roles', tone: 'warn', iconName: 'clock' };
  }
  if (inc.ready_for_archive) return { id: 'archive', label: 'Listo para archivar', tone: 'ok', iconName: 'archive' };
  if (inc.pending_spec) return { id: 'spec', label: 'Pendiente de spec', tone: 'accent', iconName: 'edit' };
  if (inc.ready_for_design) return { id: 'design', label: 'Listo para design', tone: 'accent', iconName: 'arrowRight' };
  return null;
}

export function flagBadge(inc) {
  const f = statusFlag(inc);
  if (!f) return '';
  return `<span class="badge badge-${f.tone}">${icon(f.iconName, 13)}${escapeHtml(f.label)}</span>`;
}

// ---- Fechas y progreso -------------------------------------------------------------

// formatDate muestra solo una fecha real (YYYY-MM-DD del nombre archivado).
export function formatDate(iso) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(iso || ''));
  if (!m) return '';
  const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
  if (Number.isNaN(d.getTime())) return '';
  try {
    return d.toLocaleDateString('es-ES', { day: 'numeric', month: 'short', year: 'numeric' });
  } catch (_) {
    return String(iso);
  }
}

// progressBar: verde al llegar al 100 %; el valor también va en texto y en aria.
export function progressBar(pct, label = 'Tareas completadas') {
  const v = Math.max(0, Math.min(100, Math.round(Number(pct) || 0)));
  return `<div class="progress${v >= 100 ? ' is-done' : ''}" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${v}" aria-label="${escapeHtml(label)}"><span style="width:${v}%"></span></div>`;
}

// miniStepper: 7 segmentos; la fase actual se anuncia con texto accesible.
export function miniStepper(inc) {
  const archived = isArchived(inc);
  const current = archived ? PHASES.length - 1 : phaseIndex(inc.phase);
  const segs = PHASES.map((p, i) => {
    const cls = archived || i < current ? 'is-done' : i === current ? 'is-current' : '';
    return `<span class="mini-seg ${cls}"></span>`;
  }).join('');
  const label = archived ? 'Archivado, 7 de 7 fases' : `Fase ${PHASES[current].label}, paso ${current + 1} de ${PHASES.length}`;
  return `<span class="mini-stepper" role="img" aria-label="${escapeHtml(label)}">${segs}</span><span class="mini-phase">${escapeHtml(PHASES[current].label)}</span>`;
}

// ---- Orden de agente ---------------------------------------------------------------------

// Órdenes reales: los comandos de barra de las fases de agente están en
// internal/assets/claude/commands (sdd-continue, sdd-apply, sdd-verify,
// sdd-archive). El dashboard no las ejecuta: las muestra para copiarlas.
export const AGENT_COMMANDS = {
  planning: (name) => `/sdd-continue ${name}`,
  apply: (name) => `/sdd-apply ${name}`,
  verify: (name) => `/sdd-verify ${name}`,
  archive: (name) => `/sdd-archive ${name}`
};

const PLANNING_PHASES = ['explore', 'spec', 'design', 'tasks'];

// nextCommand devuelve la orden de agente de la fase que toca, o '' si el
// incremento está archivado. Prioridad: verificación global, planificación (sin
// tasks.md todos los roles figuran pendientes), roles esperando, archivar y apply.
export function nextCommand(s) {
  if (!s || isArchived(s) || s.phase === 'archive') return '';
  const name = s.name;
  if (s.ready_for_global_verify) return AGENT_COMMANDS.verify(name);
  if (PLANNING_PHASES.includes(s.phase)) return AGENT_COMMANDS.planning(name);
  if (s.waiting_roles) return AGENT_COMMANDS.apply(name);
  if (s.ready_for_archive) return AGENT_COMMANDS.archive(name);
  return AGENT_COMMANDS.apply(name);
}

// ---- Pasos del detalle ----------------------------------------------------------------

// detailSteps deriva los 7 pasos a partir de los artefactos existentes. El primer
// paso sin hacer es el actual; si todo está hecho no hay paso actual.
export function detailSteps(detail) {
  const s = detail.summary;
  const archived = isArchived(s) || !!detail.has_archive;
  // summary.phase cubre lo que has_* no ve (p. ej. tasks.<rol>.md sin tasks.md):
  // cuántos pasos ya se han alcanzado según la fase.
  const reached = { explore: 0, spec: 1, design: 2, tasks: 3, apply: 4, verify: 6, archive: 7 }[s.phase] || 0;
  const steps = [
    { id: 'proposal', label: 'Propuesta', file: 'proposal.md', done: !!detail.has_proposal },
    { id: 'spec', label: 'Spec', file: 'spec.md', done: !!detail.has_spec },
    { id: 'design', label: 'Design', file: 'design.md', done: !!detail.has_design },
    { id: 'tasks', label: 'Tasks', file: 'tasks.md', done: !!detail.has_tasks },
    { id: 'apply', label: 'Apply', file: '', done: !!detail.has_verify || archived },
    { id: 'verify', label: 'Verify', file: 'verify-report.md', done: !!detail.has_verify || archived },
    { id: 'archive', label: 'Archive', file: 'archive-report.md', done: archived }
  ];
  steps.forEach((st, i) => { if (i < reached) st.done = true; });
  if (archived) steps.forEach(st => { st.done = true; });
  const current = steps.findIndex(st => !st.done);
  steps.forEach((st, i) => { st.state = st.done ? 'done' : i === current ? 'current' : 'todo'; });
  const apply = steps[4];
  if (apply.state === 'current' && s.tasks_total > 0) {
    apply.detail = `${s.tasks_completed}/${s.tasks_total} tareas`;
    apply.pct = s.progress_pct;
  }
  // Multirol: el paso Apply en curso desglosa el avance de cada rol y avisa
  // cuando la barrera está cerrada. Con un único rol el paso queda como estaba.
  const barrier = detail.barrier_report;
  const roles = barrier && Array.isArray(barrier.roles) ? barrier.roles : [];
  if (apply.state === 'current' && roles.length > 1) {
    apply.roles = roles.map(roleProgress);
    apply.waiting = barrier.satisfied === false;
    apply.blocker = apply.waiting && Array.isArray(barrier.blockers) && barrier.blockers.length ? String(barrier.blockers[0]) : '';
  }
  return steps;
}

// roleProgress resume un rol de la barrera para el paso Apply: nombre, porcentaje
// de tareas y estado en texto (verificado, sin tasks o completadas/total).
function roleProgress(r) {
  const t = r.tasks || {};
  const total = Number(t.total) || 0;
  const completed = Number(t.completed) || 0;
  let status;
  if (r.verify_done) status = { tone: 'ok', text: 'verificado' };
  else if (!r.tasks_found || total === 0) status = { tone: 'muted', text: 'sin tasks' };
  else status = { tone: completed >= total ? 'ok' : '', text: `${completed}/${total}` };
  return { id: r.role, label: r.name || r.role, pct: total > 0 ? Math.round((completed / total) * 100) : 0, status };
}

// ---- Atención (Inicio) ---------------------------------------------------------------

// Regla única: un incremento activo necesita atención cuando statusFlag() le
// pone alguna bandera (ready_for_global_verify, waiting_roles, ready_for_archive,
// pending_spec o ready_for_design). Inicio, el titular y la lista parten de aquí.
export function needsAttention(inc) {
  return statusFlag(inc) !== null;
}

// Orden de la lista: verificación global, roles, archivado y, por último, spec
// y design (mismo rango). A igual rango, por nombre.
const ATTENTION_RANK = { gverify: 0, waiting: 1, archive: 2, spec: 3, design: 3 };

// attentionItems devuelve [{inc, flag}] de los activos que necesitan atención, ya ordenados.
export function attentionItems(increments) {
  return (increments || [])
    .map(inc => ({ inc, flag: statusFlag(inc) }))
    .filter(item => item.flag !== null)
    .sort((a, b) => (ATTENTION_RANK[a.flag.id] - ATTENTION_RANK[b.flag.id]) || String(a.inc.name).localeCompare(String(b.inc.name), 'es'));
}

// incrementStats resume la lista para los indicadores y el flujo SDD de Inicio.
// Las features ODD (change_type: 'odd') no entran: Inicio resume el flujo SDD.
// byPhase cuenta solo activos (explore…verify); archived suma los archivados y
// los activos que ya están en la fase archive.
export function incrementStats(increments) {
  const all = (increments || []).filter(inc => !inc || inc.change_type !== 'odd');
  const active = all.filter(inc => !isArchived(inc));
  const archivedList = all.filter(isArchived);
  const byPhase = PHASES.filter(p => p.id !== 'archive').map(p => ({ id: p.id, label: p.label, count: 0, bugs: 0 }));
  let tasksDone = 0;
  let tasksTotal = 0;
  let archivedActive = 0;
  active.forEach(inc => {
    tasksDone += inc.tasks_completed || 0;
    tasksTotal += inc.tasks_total || 0;
    if (inc.phase === 'archive') {
      archivedActive += 1;
      return;
    }
    const cell = byPhase[phaseIndex(inc.phase)];
    cell.count += 1;
    if (typeInfo(inc).id === 'bug') cell.bugs += 1;
  });
  return {
    active: active.length,
    bugs: active.filter(inc => typeInfo(inc).id === 'bug').length,
    archived: archivedList.length + archivedActive,
    tasksDone,
    tasksTotal,
    pct: tasksTotal > 0 ? Math.round((tasksDone / tasksTotal) * 100) : 0,
    verifying: active.filter(inc => inc.phase === 'verify').length,
    readyForArchive: active.filter(inc => inc.ready_for_archive).length,
    byPhase
  };
}
