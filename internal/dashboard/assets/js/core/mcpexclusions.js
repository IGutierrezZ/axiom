// Lógica pura de las exclusiones de servidores MCP del proyecto abierto: normalizar la
// lista, añadir un nombre a mano, armar el cuerpo del POST y redactar el informe en
// castellano. Sin DOM, para probarla con Node.

export const AGENT = 'claude-code';

const STATUS_LABEL = {
  written: 'escrito',
  unchanged: 'sin cambios',
  skipped: 'omitido',
  failed: 'falló'
};

// normalizeNames recorta, descarta vacíos y repetidos y ordena.
export function normalizeNames(list) {
  const seen = new Set();
  for (const raw of list || []) {
    const name = String(raw ?? '').trim();
    if (name) seen.add(name);
  }
  return [...seen].sort((a, b) => a.localeCompare(b, 'es'));
}

// addName añade un nombre escrito a mano a los candidatos y lo deja marcado. Devuelve
// los candidatos y la selección nuevos, y el nombre añadido ('' si estaba vacío).
export function addName(candidates, selected, raw) {
  const name = String(raw ?? '').trim();
  if (!name) return { candidates: normalizeNames(candidates), selected: normalizeNames(selected), added: '' };
  return {
    candidates: normalizeNames([...(candidates || []), name]),
    selected: normalizeNames([...(selected || []), name]),
    added: name
  };
}

// buildBody arma el cuerpo del POST: sustituye las exclusiones del agente y conserva las
// de los demás agentes tal como están guardadas.
export function buildBody(current, selected, agent = AGENT) {
  const exclusions = {};
  for (const [id, names] of Object.entries(current || {})) {
    if (id !== agent) exclusions[id] = normalizeNames(names);
  }
  exclusions[agent] = normalizeNames(selected);
  return { exclusions };
}

export function statusLabel(status) {
  return STATUS_LABEL[status] || String(status || '');
}

// reportLines redacta el informe de una aplicación: una línea por directorio y otra por
// los agentes que no se pudieron aplicar.
export function reportLines(report) {
  if (!report) return [];
  const lines = (report.targets || []).map(target => {
    const reason = target.reason ? ` (${target.reason})` : '';
    return `${target.dir}: ${statusLabel(target.status)}${reason}`;
  });
  if ((report.unsupported || []).length) {
    lines.push(`Sin soporte, no se aplicaron: ${report.unsupported.join(', ')}.`);
  }
  return lines;
}

// hasFailures dice si algún directorio falló al aplicar.
export function hasFailures(report) {
  return (report?.targets || []).some(target => target.status === 'failed');
}
