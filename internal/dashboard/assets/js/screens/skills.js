// Skills: buzón de propuestas (human-in-the-loop) y skills activas. Provisional hasta T5.

import { api, escapeHtml } from '../core/api.js';
import { $, openDialog, confirmDialog, setBusy, toast } from '../core/components.js';
import { icon } from '../core/icons.js';

export async function loadSkills() {
  const container = $('#skills-container');
  try {
    const skills = (await api('/api/skills')) || [];
    if (skills.length === 0) {
      container.innerHTML = '<div class="state">No se encontraron skills locales.</div>';
      return;
    }
    container.innerHTML = skills.map(sk => `<div class="skill-card">
      <div>
        <div class="card-header"><div class="card-title">${escapeHtml(sk.name)}</div><span class="badge badge-phase">Skill</span></div>
        <p class="muted" style="font-size:13px">${escapeHtml(sk.description || 'Sin descripción')}</p>
      </div>
      <div class="card-footer"><span class="dim" style="font-size:12px"><code>${escapeHtml(sk.path)}</code></span></div>
    </div>`).join('');
  } catch (err) {
    console.error(err);
    container.innerHTML = `<div class="state state-error" role="alert">${icon('alert', 24)}<span class="state-title">No se pudo cargar el catálogo de skills</span><span>${escapeHtml(err.message)}</span></div>`;
  }
}

let proposals = [];

export async function loadSkillsInbox() {
  const container = $('#inbox-container');
  const count = $('#inbox-count');
  try {
    proposals = (await api('/api/skills/inbox')) || [];
    count.textContent = `${proposals.length} pendiente${proposals.length === 1 ? '' : 's'}`;
    if (proposals.length === 0) {
      container.innerHTML = '<div class="state">No hay propuestas pendientes en el buzón. Pulsa «Escanear tecnologías y minar» para detectar directrices.</div>';
      return;
    }
    container.innerHTML = proposals.map((p, idx) => {
      const midudev = p.origin === 'midudev';
      return `<div class="inbox-card">
        <div>
          <div class="card-header">
            <div class="card-title">${escapeHtml(p.name)}</div>
            <span class="badge ${midudev ? 'badge-origin-midudev' : 'badge-origin-mined'}">${midudev ? 'midudev (auditado)' : 'minería local'}</span>
          </div>
          <div class="card-badges" style="margin: 8px 0">
            ${p.verified ? `<span class="badge badge-ok" title="Hash verificado contra el registro oficial">${icon('shield', 13)}SHA-256 verificado</span>` : ''}
            ${p.role ? `<span class="badge badge-neutral">Rol: ${escapeHtml(p.role)}</span>` : ''}
          </div>
          <p class="muted" style="font-size:13px;margin: 8px 0">${escapeHtml(p.justification || 'Directriz recomendada')}</p>
        </div>
        <div>
          <button type="button" class="btn btn-sm btn-block btn-preview-skill" data-idx="${idx}">${icon('search', 14)}Ver contenido SKILL.md</button>
          <div class="inbox-actions">
            <button type="button" class="btn btn-sm btn-primary btn-approve" data-name="${escapeHtml(p.name)}">${icon('check', 14)}Aprobar e instalar</button>
            <button type="button" class="btn btn-sm btn-danger btn-reject" data-name="${escapeHtml(p.name)}">${icon('x', 14)}Descartar</button>
          </div>
        </div>
      </div>`;
    }).join('');
  } catch (err) {
    console.error(err);
    container.innerHTML = `<div class="state state-error" role="alert">${icon('alert', 24)}<span class="state-title">No se pudo consultar el buzón</span><span>${escapeHtml(err.message)}</span></div>`;
  }
}

async function scanSkills() {
  const btn = $('#btn-scan-skills');
  const feedback = $('#scan-feedback');
  const restore = setBusy(btn, 'Escaneando y minando…');
  feedback.classList.remove('hidden');
  feedback.textContent = 'Analizando el stack tecnológico, las dependencias y los patrones idiomáticos del repositorio…';
  try {
    const report = await api('/api/skills/scan', { body: { offline: false } });
    const techs = report.detected_technologies ? report.detected_technologies.join(', ') : 'tecnologías';
    const added = report.skills_proposed ? report.skills_proposed.length : 0;
    feedback.innerHTML = `${icon('check', 14)} <strong>Escaneo completado:</strong> detectadas ${escapeHtml(techs)}. ${added} nuevas propuestas añadidas al buzón (total pendientes: ${report.total_in_inbox || 0}).`;
    await Promise.all([loadSkillsInbox(), loadSkills()]);
  } catch (err) {
    console.error(err);
    feedback.innerHTML = `<span class="text-danger">Error durante el escaneo: ${escapeHtml(err.message)}</span>`;
  } finally {
    restore();
  }
}

async function approve(name) {
  const ok = await confirmDialog({ title: 'Aprobar skill', message: `¿Aprobar e instalar formalmente la skill «${name}» en skills/?`, confirmLabel: 'Aprobar e instalar' });
  if (!ok) return;
  try {
    await api('/api/skills/approve', { body: { name } });
    await Promise.all([loadSkillsInbox(), loadSkills()]);
  } catch (err) {
    toast(`Error aprobando la skill: ${err.message}`, 'bad');
  }
}

async function reject(name) {
  const ok = await confirmDialog({ title: 'Descartar propuesta', message: `¿Descartar la propuesta de skill «${name}» del buzón?`, confirmLabel: 'Descartar', danger: true });
  if (!ok) return;
  try {
    await api('/api/skills/reject', { body: { name } });
    await loadSkillsInbox();
  } catch (err) {
    toast(`Error descartando la propuesta: ${err.message}`, 'bad');
  }
}

function showPreview(p) {
  $('#preview-title').textContent = `Previsualización: ${p.name}`;
  $('#preview-sub').textContent = `Origen: ${p.origin}`;
  $('#preview-content').innerHTML = `
    <div class="alert">
      <div class="grow">
        <p><strong>Justificación:</strong> ${escapeHtml(p.justification)}</p>
        <p><strong>Fuente:</strong> <code>${escapeHtml(p.source || 'n/a')}</code></p>
        <p><strong>Integridad SHA-256:</strong> ${p.verified ? `<span class="text-ok">${icon('check', 13)} Verificado</span>` : `<span class="text-bad">${icon('x', 13)} Sin verificación</span>`}</p>
      </div>
    </div>
    <h3 class="subsection-title">Contenido SKILL.md</h3>
    <pre class="preview-pre">${escapeHtml(p.skill_md || 'Sin contenido')}</pre>`;
  openDialog($('#modal-preview'));
}

export function initSkills() {
  $('#btn-scan-skills').addEventListener('click', scanSkills);
  $('#inbox-container').addEventListener('click', (event) => {
    const preview = event.target.closest('.btn-preview-skill');
    if (preview) return showPreview(proposals[Number(preview.getAttribute('data-idx'))]);
    const approveBtn = event.target.closest('.btn-approve');
    if (approveBtn) return approve(approveBtn.getAttribute('data-name'));
    const rejectBtn = event.target.closest('.btn-reject');
    if (rejectBtn) return reject(rejectBtn.getAttribute('data-name'));
  });
}
