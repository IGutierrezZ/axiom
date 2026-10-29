// Reglas y piezas de interfaz compartidas por Incrementos y por el detalle:
// fases SDD, tipo, bandera de estado única, barra de progreso, minipaso y la
// derivación del «Siguiente paso» (D5: botón solo para lo que ejecuta Axiom).

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

export function phaseLabel(phase) {
  return PHASES[phaseIndex(phase)].label;
}

export function isArchived(inc) {
  return !!inc && inc.type === 'archived';
}

// ---- Tipo ---------------------------------------------------------------------

// typeInfo: feature → Feature (acento), fix o is_bug → Bug (--bad), refactor →
// neutro, architecture → Arquitectura (--warn). La API hoy solo devuelve
// feature y fix; los otros dos ya están previstos en el diseño.
export function typeInfo(inc) {
  if (inc && (inc.is_bug || inc.change_type === 'fix')) return { id: 'bug', label: 'Bug', iconName: 'bug' };
  switch (inc && inc.change_type) {
    case 'refactor': return { id: 'refactor', label: 'Refactor', iconName: 'refresh' };
    case 'architecture': return { id: 'architecture', label: 'Arquitectura', iconName: 'cube' };
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

// ---- Siguiente paso (D5) ---------------------------------------------------------------

// Órdenes reales: los comandos de barra de las fases de agente están en
// internal/assets/claude/commands (sdd-continue, sdd-apply, sdd-verify,
// sdd-archive); la acción de Axiom es `axiom sdd continue` (botón del servidor).
export const AGENT_COMMANDS = {
  planning: (name) => `/sdd-continue ${name}`,
  apply: (name) => `/sdd-apply ${name}`,
  verify: (name) => `/sdd-verify ${name}`,
  archive: (name) => `/sdd-archive ${name}`
};

const REMINDER_TEXT = 'Esta fase la ejecuta un agente. Lánzala desde tu IDE o CLI favorito con esta orden; el dashboard no la ejecuta.';

function reminder(title, command) {
  return { title, text: REMINDER_TEXT, command };
}

const PLANNING_PHASES = ['explore', 'spec', 'design', 'tasks'];

// planningStep: qué falta en las fases de planificación (todas de agente).
function planningStep(s, name) {
  const cmd = AGENT_COMMANDS.planning(name);
  switch (s.phase) {
    case 'explore':
      return { id: 'propose', title: 'Falta la propuesta', text: 'Todavía no hay proposal.md. Explorar la idea y redactar la propuesta es una fase de agente.', canContinue: true, reminder: reminder('Recordatorio: lanza la fase de propuesta', cmd) };
    case 'spec':
      return { id: 'spec', title: 'Falta la especificación', text: 'Hay propuesta pero no spec.md. Redactar requisitos y escenarios es una fase de agente.', canContinue: true, reminder: reminder('Recordatorio: lanza la fase spec', cmd) };
    case 'design':
      return { id: 'design', title: 'Listo para el diseño', text: 'La especificación está completa; falta design.md.', canContinue: true, reminder: reminder('Recordatorio: lanza la fase design', cmd) };
    default:
      return { id: 'tasks', title: 'Falta el plan de tareas', text: 'Hay diseño pero no tasks.md. Desglosar el trabajo en tareas es una fase de agente.', canContinue: true, reminder: reminder('Recordatorio: lanza la fase tasks', cmd) };
  }
}

// deriveNextStep resume qué toca en un incremento a partir de las banderas del
// resumen y de la barrera. Devuelve {id, title, text, canContinue, reminder}.
// canContinue activa el botón «Continuar con Axiom» (orden de Axiom, no de agente).
export function deriveNextStep(detail) {
  const s = detail.summary;
  const name = s.name;
  if (isArchived(s) || s.phase === 'archive') {
    return { id: 'archived', title: 'Incremento archivado', text: 'Este cambio ya está cerrado: sus requisitos pasaron a las specs vivas.', canContinue: false, reminder: null };
  }
  const barrier = detail.barrier_report;
  if (s.ready_for_global_verify) {
    return {
      id: 'verify', title: 'Toca la verificación global',
      text: 'Todos los roles han terminado sus tareas. Falta la fase verify, que valida la implementación frente a spec, design y tareas.',
      canContinue: true, reminder: reminder('Recordatorio: lanza la fase verify', AGENT_COMMANDS.verify(name))
    };
  }
  // Las fases de planificación mandan sobre la espera de roles: sin tasks.md todos los roles figuran pendientes.
  if (PLANNING_PHASES.includes(s.phase)) return planningStep(s, name);
  if (s.waiting_roles) {
    const roles = (s.pending_roles || []).join(', ');
    const blockers = barrier && !barrier.satisfied && barrier.blockers && barrier.blockers.length ? ` ${barrier.blockers[0]}` : '';
    return {
      id: 'waiting', title: roles ? `Esperando a: ${roles}` : 'Esperando a roles',
      text: `Estos roles aún tienen tareas pendientes o sin verificar, así que la barrera no se abre.${blockers}`,
      canContinue: true, reminder: reminder('Recordatorio: continúa la fase apply', AGENT_COMMANDS.apply(name))
    };
  }
  if (s.ready_for_archive) {
    return {
      id: 'archive', title: 'Listo para archivar',
      text: 'Existe verify-report.md. Falta archivar el cambio para integrar sus requisitos en las specs vivas.',
      canContinue: true, reminder: reminder('Recordatorio: lanza la fase archive', AGENT_COMMANDS.archive(name))
    };
  }
  const progress = s.tasks_total > 0 ? ` Llevas ${s.tasks_completed} de ${s.tasks_total} tareas (${s.progress_pct} %).` : '';
  return { id: 'apply', title: 'Implementación en curso', text: `El plan de tareas ya existe y se está aplicando.${progress}`, canContinue: true, reminder: reminder('Recordatorio: continúa la fase apply', AGENT_COMMANDS.apply(name)) };
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
  return steps;
}
