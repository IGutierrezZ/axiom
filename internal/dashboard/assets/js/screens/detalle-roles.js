// Pestaña «Roles y barrera» del detalle: banner de la barrera, bloqueos y avisos,
// tarjeta por rol y tareas diferidas. Usa el contrato de T1 (barrier_report del
// detalle o GET /api/roles?change=). «Migrar a acumulado» solo en roles `deferred`.

import { api, escapeHtml } from '../core/api.js';
import { button, confirmDialog, stateEmpty, stateLoading, statusPill, toast } from '../core/components.js';
import { icon } from '../core/icons.js';
import { progressBar } from '../core/increment.js';

const POLICIES = {
  blocking: { label: 'Bloqueante', hint: 'Debe completar sus tareas y verificar para abrir la barrera.' },
  deferred: { label: 'Diferido', hint: 'No bloquea la entrega; sus tareas pendientes pueden pasar a un incremento acumulativo.' },
  optional: { label: 'Opcional', hint: 'Trabajo accesorio que no condiciona la barrera.' }
};

function policyOf(role) {
  return POLICIES[role.gate_policy] || { label: role.gate_policy || 'Sin política', hint: '' };
}

// verdictPill: veredicto con icono y texto (nunca solo color).
function verdictPill(verdict) {
  if (verdict === 'pass' || verdict === 'pass_with_warnings') {
    return statusPill('ok', verdict === 'pass' ? 'Verificación superada' : 'Superada con avisos');
  }
  if (!verdict || verdict === 'missing') return statusPill('neutral', 'Sin verificar');
  return statusPill('bad', `Verificación no superada (${verdict})`);
}

function isSingleFullstack(roles) {
  return roles.length === 1 && roles[0].role === 'fullstack';
}

function bannerHtml(barrier, roles) {
  if (isSingleFullstack(roles)) {
    return `<div class="dt-banner is-accent" role="status"><span class="dt-banner-icon">${icon('sparkles', 20)}</span><div class="dt-banner-text"><span class="dt-banner-title">Rol unificado (fullstack): flujo continuo</span><span class="muted">Este incremento no se reparte en roles: no hay esperas entre agentes y el ciclo avanza de forma continua.</span></div></div>`;
  }
  const blocking = roles.filter(r => r.gate_policy === 'blocking');
  const passed = blocking.filter(r => r.compliant).length;
  const count = blocking.length > 0 ? `<span class="dt-banner-count">${passed} de ${blocking.length} roles obligatorios verificados</span>` : '';
  if (barrier.satisfied) {
    return `<div class="dt-banner is-ok" role="status"><span class="dt-banner-icon">${icon('check', 20)}</span><div class="dt-banner-text"><span class="dt-banner-title">Barrera superada</span><span class="muted">Todos los roles obligatorios han terminado y verificado sus tareas.</span></div>${count}</div>`;
  }
  return `<div class="dt-banner is-warn" role="status"><span class="dt-banner-icon">${icon('lock', 20)}</span><div class="dt-banner-text"><span class="dt-banner-title">Barrera bloqueada</span><span class="muted">Hay roles obligatorios con tareas pendientes o sin verificación superada.</span></div>${count}</div>`;
}

function issuesHtml(barrier) {
  const blockers = barrier.blockers || [];
  const warnings = barrier.warnings || [];
  if (blockers.length === 0 && warnings.length === 0) return '';
  const list = (items, iconName, cls, title) => items.length === 0 ? '' :
    `<section class="dt-issues ${cls}" aria-label="${title}"><h3 class="dt-issues-title">${icon(iconName, 16)}${title} (${items.length})</h3><ul>${items.map(t => `<li>${escapeHtml(t)}</li>`).join('')}</ul></section>`;
  return `<div class="dt-issues-grid">${list(blockers, 'x', 'is-bad', 'Bloqueos')}${list(warnings, 'alert', 'is-warn', 'Avisos')}</div>`;
}

function roleCardHtml(role) {
  const policy = policyOf(role);
  const tasks = role.tasks || { total: 0, completed: 0, pending: 0, percent: 0 };
  const pct = Math.round(tasks.percent || 0);
  const blockedHere = role.gate_policy === 'blocking' && !role.compliant;
  const title = role.name && role.name !== role.role
    ? `${escapeHtml(role.name)} <span class="mono dim">${escapeHtml(role.role)}</span>`
    : `<span class="mono">${escapeHtml(role.role)}</span>`;
  const taskBlock = role.tasks_found
    ? `<div class="dt-role-tasks"><div class="ix-row"><span>${tasks.completed}/${tasks.total} tareas</span><span class="mono">${pct}%</span></div>${progressBar(pct, `Tareas de ${role.role}`)}</div>`
    : '<p class="muted">Sin fichero de tareas para este rol.</p>';
  const repos = (role.repositories || []).length > 0
    ? role.repositories.map(r => `<code class="dt-tag">${escapeHtml(r)}</code>`).join('')
    : '<span class="muted">Todo el repositorio</span>';
  const deliverables = (role.deliverables || []).length > 0
    ? `<dt>Entregables</dt><dd><ul class="dt-plain">${role.deliverables.map(d => `<li>${escapeHtml(d)}</li>`).join('')}</ul></dd>`
    : '';
  const migrate = role.gate_policy === 'deferred'
    ? (tasks.pending > 0
      ? button({ label: 'Migrar a acumulado', iconName: 'arrowRight', size: 'sm', attrs: `data-migrate="${escapeHtml(role.role)}" data-pending="${tasks.pending}"` })
      : '<p class="muted">Sin tareas pendientes que migrar.</p>')
    : '';
  return `<article class="dt-role${blockedHere ? ' is-blocked' : ''}${role.gate_policy === 'deferred' ? ' is-deferred' : ''}">
    <header class="dt-role-head">
      <h4 class="dt-role-name">${title}</h4>
      <span class="badge badge-neutral" title="gate_policy: ${escapeHtml(role.gate_policy)}">${escapeHtml(policy.label)}</span>
    </header>
    <div>${verdictPill(role.verdict)}</div>
    ${taskBlock}
    <dl class="dt-dl">
      <dt>Apply</dt><dd>${role.apply_done ? 'Hecho' : 'Pendiente'}</dd>
      <dt>Verify</dt><dd>${role.verify_done ? 'Hecho' : 'Pendiente'}</dd>
      <dt>Repos</dt><dd class="dt-tags">${repos}</dd>
      ${deliverables}
    </dl>
    ${policy.hint ? `<p class="dt-hint">${escapeHtml(policy.hint)}</p>` : ''}
    ${migrate}
  </article>`;
}

function deferredHtml(tasks) {
  if (!tasks || tasks.length === 0) return '';
  return `<section class="card" aria-labelledby="dt-deferred-h"><h3 id="dt-deferred-h" class="card-title-h">Tareas de roles diferidos (${tasks.length})</h3><ul class="dt-deferred">${tasks.map(t => `<li><span class="mono dt-tag">${escapeHtml(t.role)}</span><span class="dt-deferred-text">${escapeHtml(t.task_text)}</span>${t.spec_ref ? `<code class="dim">${escapeHtml(t.spec_ref)}</code>` : ''}</li>`).join('')}</ul></section>`;
}

// barrierHtml compone la pestaña a partir de un informe ya cargado.
export function barrierHtml(barrier) {
  const roles = Array.isArray(barrier.roles) ? barrier.roles : [];
  if (roles.length === 0) {
    return `${bannerHtml(barrier, roles)}${stateEmpty('Sin roles específicos', 'Modo mono-rol: no se detectaron roles declarados para este incremento.', 'users')}`;
  }
  return `${bannerHtml(barrier, roles)}${issuesHtml(barrier)}
    <section aria-labelledby="dt-roles-h"><h3 id="dt-roles-h" class="card-title-h dt-roles-title">Avance por rol</h3><div class="dt-role-grid">${roles.map(roleCardHtml).join('')}</div></section>
    ${deferredHtml(barrier.deferred_tasks)}`;
}

// barrierSummary resume la barrera para la insignia de la pestaña (texto, no solo color).
export function barrierSummary(barrier) {
  if (!barrier || !Array.isArray(barrier.roles) || isSingleFullstack(barrier.roles) || barrier.roles.length === 0) return null;
  return barrier.satisfied ? { tone: 'ok', text: 'Superada' } : { tone: 'warn', text: 'Bloqueada' };
}

// renderRoles pinta la pestaña. Con barrier_report en el detalle no hace falta otra
// llamada; sin él se consulta /api/roles para mostrar el motivo que devuelva el servidor.
export async function renderRoles(panel, { detail, change, onMigrated }) {
  let barrier = detail.barrier_report;
  if (!barrier) {
    panel.innerHTML = stateLoading('Evaluando roles y barrera…');
    try {
      barrier = await api(`/api/roles?change=${encodeURIComponent(change)}`);
    } catch (err) {
      panel.innerHTML = stateEmpty('Sin informe de barrera', `La barrera se calcula a partir de design.md y de los roles del workspace. Motivo: ${err.message}`, 'lock');
      return;
    }
  }
  panel.innerHTML = `<div class="dt-roles">${barrierHtml(barrier)}</div>`;
  bindMigrate(panel, change, onMigrated);
}

function bindMigrate(panel, change, onMigrated) {
  panel.querySelectorAll('[data-migrate]').forEach(btn => {
    btn.addEventListener('click', async () => {
      const role = btn.getAttribute('data-migrate');
      const pending = btn.getAttribute('data-pending');
      const ok = await confirmDialog({
        title: 'Migrar tareas a acumulado',
        message: `Las ${pending} tareas pendientes del rol «${role}» se marcarán como migradas y se copiarán al incremento acumulativo cumulative-${role}. Este cambio no se deshace desde el dashboard.`,
        confirmLabel: 'Migrar tareas'
      });
      if (!ok) return;
      btn.disabled = true;
      btn.setAttribute('aria-busy', 'true');
      try {
        const data = await api('/api/increments/migrate-cumulative', { body: { change, role } });
        toast(`Se migraron ${data && data.migrated != null ? data.migrated : 0} tareas del rol ${role}.`, 'ok');
        await onMigrated();
      } catch (err) {
        btn.disabled = false;
        btn.removeAttribute('aria-busy');
        toast(`No se pudo migrar: ${err.message}`, 'bad');
      }
    });
  });
}
