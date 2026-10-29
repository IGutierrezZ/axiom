// Lógica pura de Ecosistema (sin DOM): traducción de los valores de la API,
// orden del Doctor, resumen de salud, tamaños y la presentación de la cadena
// upgrade -> sync. Se prueba con Node (ver platform_assets_test.go).

// ---- Traducción al castellano ------------------------------------------------------------

// Los valores que la API declara; cualquier otro se muestra tal cual llega.
export const STATUS_TEXT = { ok: 'Correcto', warning: 'Aviso', error: 'Error' };
export const CATEGORY_TEXT = { agent: 'agente', tool: 'herramienta', environment: 'entorno' };

// statusView devuelve el texto, el tono (ok | warn | bad | neutral) y el icono de un estado.
export function statusView(status) {
  switch (status) {
    case 'ok': return { text: STATUS_TEXT.ok, tone: 'ok', icon: 'check' };
    case 'warning': return { text: STATUS_TEXT.warning, tone: 'warn', icon: 'alert' };
    case 'error': return { text: STATUS_TEXT.error, tone: 'bad', icon: 'x' };
    default: return { text: String(status || 'Desconocido'), tone: 'neutral', icon: 'info' };
  }
}

export function categoryText(category) {
  return CATEGORY_TEXT[category] || String(category || '');
}

// ---- Doctor ------------------------------------------------------------------------------

const RANK = { error: 0, warning: 1, ok: 3 };

// sortChecks pone primero los errores, luego los avisos, después los estados que
// la interfaz no conoce y al final los correctos. Es estable: respeta el orden original.
export function sortChecks(checks) {
  const list = Array.isArray(checks) ? checks : [];
  return list
    .map((check, index) => ({ check, index, rank: RANK[check && check.status] === undefined ? 2 : RANK[check.status] }))
    .sort((a, b) => a.rank - b.rank || a.index - b.index)
    .map(item => item.check);
}

// healthSummary: cuenta correctos, avisos, errores y otros, y decide el tono del banner.
export function healthSummary(report) {
  const checks = report && Array.isArray(report.checks) ? report.checks : [];
  const count = status => checks.filter(c => c.status === status).length;
  const ok = count('ok');
  const warnings = count('warning');
  const errors = count('error');
  const other = checks.length - ok - warnings - errors;
  const healthy = !!(report && report.healthy);
  let tone = 'ok';
  let title = 'Ecosistema saludable';
  if (errors > 0) {
    tone = 'bad';
    title = 'Ecosistema con errores';
  } else if (!healthy || warnings > 0 || other > 0) {
    tone = 'warn';
    title = 'Ecosistema con avisos';
  }
  return { total: checks.length, ok, warnings, errors, other, healthy, tone, title };
}

// ---- Tamaños ---------------------------------------------------------------------------

// formatBytes: «1,2 MB». Devuelve '' si el servidor no informa del tamaño (0 o ausente).
export function formatBytes(bytes) {
  const n = Number(bytes);
  if (!Number.isFinite(n) || n <= 0) return '';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = n;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const digits = unit === 0 ? 0 : 1;
  let text;
  try {
    text = value.toLocaleString('es-ES', { minimumFractionDigits: digits, maximumFractionDigits: digits });
  } catch (_) {
    text = value.toFixed(digits);
  }
  return `${text} ${units[unit]}`;
}

// ---- Cadena upgrade -> sync ------------------------------------------------------------

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

// ---- Última operación ------------------------------------------------------------------

export const OPERATION_LABEL = { sync: 'sync', upgrade: 'upgrade → sync', backup: 'respaldo', restore: 'restauración' };

// operationOutcome decide el estado visible de una operación terminada. Un upgrade que
// dejó el binario reemplazado acabó bien, pero queda pendiente reiniciar axiom.
export function operationOutcome(kind, data, failed) {
  if (failed) return { tone: 'bad', text: 'con errores' };
  if (kind === 'upgrade' && data && data.phases && data.phases.upgrade && data.phases.upgrade.restart_required) {
    return { tone: 'warn', text: 'reinicio pendiente' };
  }
  if (data && data.success === false) return { tone: 'bad', text: 'con errores' };
  return { tone: 'ok', text: 'correcta' };
}
