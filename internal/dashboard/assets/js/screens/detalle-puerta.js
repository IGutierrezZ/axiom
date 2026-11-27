// Panel de puerta del detalle del incremento: muestra la parada abierta de un cambio `checkpointed`
// (GET /api/increments/gate/stop) y permite decidirla desde el navegador con las mismas tres salidas
// que la CLI: iterar (rechazar con motivo), aprobar y ceder, aprobar y continuar. Las salidas que
// publican piden UN diálogo de confirmación que avisa de la operación remota y encadenan el registro
// y la publicación de los documentos del cambio. El push de código nunca se ejecuta: solo se muestra
// como texto. Todo texto proveniente del servidor se inserta escapado.

import { api, escapeHtml } from '../core/api.js';
import { confirmDialog, promptDialog, toast } from '../core/components.js';

const COMMIT_REVIEW_LABEL = {
  notify: 'Avisar de cada commit y seguir (notify)',
  stop: 'Parar tras cada commit (stop)',
  'stop-before-commit': 'Parar antes de cada commit (stop-before-commit)',
};
const EVIDENCE_KINDS = [['pr_merged', 'Pull request fusionada'], ['deployment', 'Despliegue'], ['attestation', 'Atestación']];

// Resultado de la última decisión por cambio: sobrevive al repintado que provoca onChanged.
const results = new Map();

function refusalOf(err) {
  const d = (err && err.data) || {};
  return {
    code: d.code || '', label: d.label || '', error: d.error || d.message || (err && err.message) || '',
    continuation: d.continuation || '',
  };
}

function refusalText(r) {
  return [r.label, r.error, r.continuation].filter(Boolean).join(' · ');
}

function listHtml(items) {
  const rows = (items || []).filter(Boolean);
  if (!rows.length) return '';
  return `<ul class="gate-list">${rows.map(i => `<li>${escapeHtml(i)}</li>`).join('')}</ul>`;
}

function commandsHtml(title, commands) {
  const rows = (commands || []).filter(Boolean);
  if (!rows.length) return '';
  return `<p class="ix-more">${escapeHtml(title)}</p><ul class="gate-list">${rows.map(c => `<li><code>${escapeHtml(c)}</code></li>`).join('')}</ul>`;
}

function noticesHtml(notices) {
  const rows = (notices || []).filter(n => n && n.text);
  if (!rows.length) return '';
  const line = n => (n.source ? [n.source, n.text].join(': ') : String(n.text));
  return `<ul class="gate-list" aria-label="Avisos">${rows.map(n => `<li>${escapeHtml(line(n))}</li>`).join('')}</ul>`;
}

function resultHtml(res) {
  const rec = res.record || {};
  let html = `<p class="ix-more" role="status">Decisión registrada: ${escapeHtml(rec.gate || '')} · ${escapeHtml(rec.decision || '')}</p>`;
  if (rec.handedOff) {
    html += `<p class="ix-more" role="status">Fase cedida${rec.handedOffTo ? ` a ${escapeHtml(rec.handedOffTo)}` : ''}.</p>`;
  }
  if (rec.lastRoleClosed) {
    html += '<p class="ix-more" role="status">Se ha cerrado el último rol bloqueante.</p>';
  }
  const pending = rec.pendingNonBlockingRoles || [];
  if (pending.length) {
    html += '<p class="ix-more">Roles no bloqueantes con la puerta abierta:</p>'
      + listHtml(pending.map(p => `${p.role} (${p.policy}, ${p.status})`));
  }
  if (res.published) {
    const p = res.published;
    const where = p.commit ? `${p.commit} en ${p.remote}/${p.branch}` : `${p.remote || ''}/${p.branch || ''}`;
    html += `<p class="ix-more" role="status">Publicación: ${escapeHtml(p.status || '')} (${escapeHtml(where)})</p>`
      + listHtml((p.uncommittedDocs || []).map(d => `Documento sin publicar: ${d}`));
  }
  const resume = (res.published && res.published.resume) || (rec.delivery && rec.delivery.resume) || [];
  if (rec.handedOff) html += commandsHtml('Quien reanude la fase ejecuta:', resume);
  return html + noticesHtml(res.notices);
}

function unpublishedHtml(u) {
  if (!u) return '';
  return '<div class="gate-unpublished" role="alert"><h3>Decisión aprobada sin publicar</h3>'
    + `<p class="ix-more">La decisión quedó registrada, pero los documentos no se han publicado en ${escapeHtml(u.remote)}/${escapeHtml(u.branch)}.</p>`
    + `<p class="ix-more">${u.label ? `<strong>${escapeHtml(u.label)}</strong> ` : ''}<code>${escapeHtml(u.code)}</code> ${escapeHtml(u.error)}`
    + `${u.continuation ? ` <span>${escapeHtml(u.continuation)}</span>` : ''}</p>`
    + '<button type="button" class="btn" data-gate-retry>Reintentar la publicación</button></div>';
}

function publicationHtml(pub) {
  if (!pub || pub.state !== 'code_unpublished') return '';
  // El push de código es de la persona: se muestra como texto y jamás se ejecuta desde aquí.
  return '<div class="gate-form"><p class="ix-more">Hay commits de código sin publicar. Súbelos tú desde tu terminal:</p>'
    + (pub.pushCommand ? `<p><code>${escapeHtml(pub.pushCommand)}</code></p>` : '') + '</div>';
}

function commitReviewHtml(stop) {
  if (stop.gate !== 'tasks') return '';
  const cr = stop.commitReview || {};
  if (cr.offered === false) return '';
  const modes = (cr.modes && cr.modes.length) ? cr.modes : ['notify', 'stop', 'stop-before-commit'];
  const def = modes.includes(cr.default) ? cr.default : 'notify';
  const radios = modes.map(m => `<label class="gate-radio"><input type="radio" name="commitReview" value="${escapeHtml(m)}"${m === def ? ' checked' : ''}> `
    + `${escapeHtml(COMMIT_REVIEW_LABEL[m] || m)}</label>`).join('');
  return `<fieldset class="gate-form"><legend>Revisión de commits durante la implementación</legend>${radios}</fieldset>`;
}

function evidenceHtml(stop) {
  if (stop.gate !== 'integration') return '';
  const kinds = EVIDENCE_KINDS.map(([v, l]) => `<option value="${escapeHtml(v)}"${v === 'pr_merged' ? ' selected' : ''}>${escapeHtml(l)}</option>`).join('');
  return '<fieldset class="gate-form" data-gate-evidence><legend>Evidencia de la fusión</legend>'
    + `<label>Tipo <select name="kind" aria-label="Tipo de evidencia" required>${kinds}</select></label>`
    + '<label>Commit <input type="text" name="commit" class="form-input" autocomplete="off" spellcheck="false" placeholder="commit fusionado"></label>'
    + '<label>Rama base <input type="text" name="baseRef" class="form-input" autocomplete="off" spellcheck="false" placeholder="rama base"></label>'
    + '<label>Texto <input type="text" name="text" class="form-input" autocomplete="off" placeholder="texto de la evidencia"></label>'
    + '</fieldset>';
}

function stopHtml(stop) {
  const pub = stop.publication;
  let html = `<h2 class="gate-title">${escapeHtml(stop.headline || `Parada: ${stop.gate}`)}</h2>`;
  if (stop.reason) html += `<p class="ix-more">${escapeHtml(stop.reason)}</p>`;
  if (stop.value) html += `<p class="ix-more">${escapeHtml(stop.value)}</p>`;
  html += listHtml(stop.criteria) + listHtml(stop.evidence);
  if (stop.handOffUnavailable) html += `<p class="ix-more">Ceder no disponible: ${escapeHtml(stop.handOffUnavailable)}</p>`;
  if (pub && stop.gate !== 'integration') {
    html += `<p class="ix-more" role="status">Remoto: ${escapeHtml(pub.remote)} · Rama: ${escapeHtml(pub.branch)}</p>`;
  } else if (stop.publicationUnavailable) {
    html += `<p class="ix-more">Publicación no disponible: ${escapeHtml(stop.publicationUnavailable)}</p>`;
  }
  html += commitReviewHtml(stop) + evidenceHtml(stop);
  const buttons = (stop.choices || []).map((choice, i) => '<li>'
    + `<button type="button" class="btn${choice.answer === 'rejected' ? '' : ' btn-primary'}" data-gate-choice="${i}">${escapeHtml(choice.label)}</button> `
    + `<span class="gate-choice-effect">${escapeHtml(choice.effect)}</span></li>`).join('');
  return `${html}<ul class="gate-choices">${buttons}</ul>`;
}

// pendingOf reconstruye «aprobada sin publicar» desde datos persistentes (publication.state), sin
// depender del estado en memoria: solo en un cambio checkpointed con la publicación pendiente.
function pendingOf(view) {
  const d = view.data;
  const pub = d && d.publication;
  if (!d || d.executionStyle !== "checkpointed" || !pub || pub.state !== "pending") return null;
  return { remote: pub.remote || "", branch: pub.branch || "", code: "publication_pending", label: "", error: "Publicación pendiente de los documentos del cambio.", continuation: (pub.reasons || []).join(" · ") };
}

function unavailableHtml(pub) {
  if (!pub || pub.state !== "unavailable" || !pub.unavailable) return "";
  return `<p class="ix-more" role="status">Publicación no disponible: <code>${escapeHtml(pub.unavailable)}</code></p>`;
}

function paint(view) {
  const data = view.data;
  const stop = data && data.stop;
  const res = results.get(view.change);
  let html = '';
  if (stop) html += stopHtml(stop);
  html += unpublishedHtml((res && res.unpublished) || pendingOf(view));
  html += unavailableHtml(data && data.publication);
  html += publicationHtml(data && data.publication);
  if (res && res.record && res.record.gate) html += resultHtml(res);
  view.section.innerHTML = html;
  view.section.hidden = !html;
}

function commitReviewValue(section) {
  const picked = section.querySelector('input[name="commitReview"]:checked');
  return picked ? picked.value : 'notify';
}

function evidenceValue(section) {
  const box = section.querySelector('[data-gate-evidence]');
  if (!box) return null;
  const val = name => (box.querySelector(`[name="${name}"]`).value || '').trim();
  const evidence = { kind: val('kind') || 'pr_merged' };
  ['commit', 'baseRef', 'text'].forEach(k => { if (val(k)) evidence[k] = val(k); });
  return evidence;
}

// exitFlow es el camino de las salidas que aprueban: UN diálogo, el registro y, solo si el registro
// devuelve el destino, la publicación.
async function exitFlow(view, choice) {
  const stop = view.data.stop;
  const pub = stop.publication;
  const lines = [`Se registrará la aprobación de la puerta «${stop.gate}».`];
  if (stop.gate === 'integration') {
    lines.push('Esta aprobación no publica nada.');
  } else if (pub) {
    lines.push('Operación remota: se hará commit de los documentos del cambio y push, sin forzar.',
      `Remoto: ${pub.remote}`, `Rama: ${pub.branch}`);
  } else {
    lines.push('No hay destino de publicación disponible: no se publicará nada.');
  }
  const ok = await confirmDialog({ title: choice.label, message: lines.join('\n'), confirmLabel: 'Aprobar' });
  if (!ok) return;
  const body = { change: view.change, gate: stop.gate, digest: stop.digest, decision: 'approved', confirm: true };
  if (choice.answer === 'hand_off') body.handOff = true;
  if (stop.gate === 'tasks') body.commitReview = commitReviewValue(view.section);
  const evidence = evidenceValue(view.section);
  if (evidence) body.evidence = evidence;
  const recorded = await recordDecision(view, body);
  if (!recorded) return;
  const res = { record: recorded.record, notices: recorded.notices, published: null, unpublished: null };
  results.set(view.change, res);
  if (stop.gate !== 'integration' && res.record.publish) await publishGate(view, res, res.record.publish);
  view.onChanged();
  await load(view);
}

// recordDecision envía el registro. Si falla no se publica nada y se vuelve a consultar la parada.
async function recordDecision(view, body) {
  try {
    const res = await api('/api/increments/gate', { body });
    return { record: res.record || {}, notices: res.notices || [] };
  } catch (err) {
    toast(refusalText(refusalOf(err)), 'bad');
    await load(view);
    return null;
  }
}

// publishGate publica los documentos del cambio en la rama que la pantalla mostró. Se pide desde
// aquí y solo tras un diálogo aceptado.
async function publishGate(view, res, target) {
  try {
    res.published = await api('/api/increments/gate/publish', { body: { change: view.change, branch: target.branch, confirm: true } });
    res.unpublished = null;
    res.notices = (res.notices || []).concat(res.published.notices || []);
    toast(`Publicación: ${res.published.status}`);
  } catch (err) {
    res.unpublished = Object.assign(refusalOf(err), { remote: target.remote, branch: target.branch });
    toast(refusalText(res.unpublished), 'bad');
  }
}

async function retryFlow(view) {
  let res = results.get(view.change);
  const u = (res && res.unpublished) || pendingOf(view);
  if (!u) return;
  if (!res) res = { record: {}, notices: [], published: null, unpublished: null };
  const ok = await confirmDialog({
    title: 'Reintentar la publicación',
    message: `Operación remota: commit de los documentos del cambio y push, sin forzar.\nRemoto: ${u.remote}\nRama: ${u.branch}`,
    confirmLabel: 'Publicar',
  });
  if (!ok) return;
  await publishGate(view, res, { remote: u.remote, branch: u.branch });
  results.set(view.change, res);
  view.onChanged();
  await load(view);
}

async function iterateFlow(view, choice) {
  const stop = view.data.stop;
  const reason = await promptDialog({
    title: choice.label, message: choice.effect, confirmLabel: 'Registrar', placeholder: 'Motivo del rechazo (obligatorio)',
  });
  if (reason === null) return;
  if (!reason.trim()) {
    toast('El motivo es obligatorio para iterar', 'warn');
    return;
  }
  const body = { change: view.change, gate: stop.gate, digest: stop.digest, decision: 'rejected', reason: reason.trim(), confirm: true };
  const recorded = await recordDecision(view, body);
  if (!recorded) return;
  results.set(view.change, { record: recorded.record, notices: recorded.notices, published: null, unpublished: null });
  view.onChanged();
  await load(view);
}

function bind(view) {
  view.section.onclick = async ev => {
    if (ev.target.closest('[data-gate-retry]')) { await retryFlow(view); return; }
    const btn = ev.target.closest('[data-gate-choice]');
    if (!btn || btn.disabled) return;
    const stop = view.data && view.data.stop;
    const choice = stop && stop.choices[Number(btn.dataset.gateChoice)];
    if (!choice) return;
    btn.disabled = true;
    try {
      if (choice.answer === 'rejected') await iterateFlow(view, choice);
      else await exitFlow(view, choice);
    } finally { btn.disabled = false; }
  };
}

async function load(view) {
  try {
    view.data = await api(`/api/increments/gate/stop?change=${encodeURIComponent(view.change)}`);
  } catch (_) {
    // Pasarela ausente (503) o parada ilegible: el panel queda oculto y el detalle sigue igual.
    view.data = null;
  }
  if (!view.isCurrent()) return;
  paint(view);
}

// renderGate pinta la parada del cambio en `section` (oculta si no hay parada o no hay pasarela).
export async function renderGate(section, { change, onChanged = () => {}, isCurrent = () => true }) {
  const view = { section, change, onChanged, isCurrent, data: null };
  bind(view);
  await load(view);
}
