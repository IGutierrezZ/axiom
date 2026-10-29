// Estado compartido entre pantallas (datos de la API ya cargados).

export const state = {
  workspace: null,   // GET /api/workspace
  increments: [],    // GET /api/increments
  projects: null,    // GET /api/projects
  overview: {}       // fuentes de Inicio y de la barra lateral (core/overview.js)
};

const listeners = new Map();

// on registra un oyente para una clave ('workspace', 'increments', 'projects', 'overview').
export function on(key, fn) {
  if (!listeners.has(key)) listeners.set(key, new Set());
  listeners.get(key).add(fn);
}

// set guarda el valor y avisa a los oyentes.
export function set(key, value) {
  state[key] = value;
  (listeners.get(key) || []).forEach(fn => fn(value));
}
