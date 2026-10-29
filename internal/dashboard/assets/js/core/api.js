// Cliente de la API del dashboard: fetch + JSON + normalización de errores.

// ApiError normaliza cualquier fallo (red, HTTP no 2xx, cuerpo ilegible).
// Conserva el estado HTTP y el cuerpo ya decodificado, porque algunos
// endpoints devuelven `output` o `message` útiles también cuando fallan.
export class ApiError extends Error {
  constructor(message, status, data) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

function errorMessage(data, status, statusText) {
  if (data && typeof data === 'object') {
    if (data.error) return String(data.error);
    if (data.message) return String(data.message);
  }
  return statusText || `Error HTTP ${status}`;
}

// api ejecuta la petición y devuelve el JSON decodificado (o null si no hay cuerpo).
// Con opts.body serializa JSON y usa POST por defecto.
export async function api(path, opts = {}) {
  const init = { method: opts.method || 'GET', headers: {} };
  if (opts.body !== undefined) {
    if (!opts.method) init.method = 'POST';
    init.headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify(opts.body);
  }

  let res;
  try {
    res = await fetch(path, init);
  } catch (err) {
    throw new ApiError(`No se pudo contactar con el servidor: ${err.message}`, 0, null);
  }

  let data = null;
  const text = await res.text();
  if (text) {
    try {
      data = JSON.parse(text);
    } catch (_) {
      data = { message: text };
    }
  }
  if (!res.ok) {
    throw new ApiError(errorMessage(data, res.status, res.statusText), res.status, data);
  }
  return data;
}

// escapeHtml escapa todo dato de la API antes de interpolarlo en HTML.
export function escapeHtml(value) {
  if (value === null || value === undefined) return '';
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}
