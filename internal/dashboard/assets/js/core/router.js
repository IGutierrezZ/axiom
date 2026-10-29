// Router por hash. Rutas:
//   #/inicio
//   #/incrementos
//   #/incrementos/<name>                     (pestaña Artefactos)
//   #/incrementos/<name>/{artefactos,roles,handoff}
//   #/specs
//   #/specs/<dominio>                        (visor de la especificación viva)
//   #/skills   #/codigo   #/ecosistema
// Por defecto #/inicio. Atrás y adelante funcionan porque cada ruta es una
// entrada del historial (location.hash).

export const SIMPLE_ROUTES = ['inicio', 'incrementos', 'specs', 'skills', 'codigo', 'ecosistema'];
export const INCREMENT_TABS = ['artefactos', 'roles', 'handoff'];

// parseHash convierte un hash en una ruta {name, change, tab} o {name:'specs', domain}.
// name es una de SIMPLE_ROUTES o 'incremento' (detalle). Lo desconocido cae en inicio.
export function parseHash(hash) {
  const raw = (hash || '').replace(/^#/, '');
  const parts = raw.split('/').filter(Boolean);
  if (parts.length === 0) return { name: 'inicio', canonical: true };

  const head = parts[0];
  if (head === 'incrementos' && parts.length >= 2) {
    let change;
    try {
      change = decodeURIComponent(parts[1]);
    } catch (_) {
      return { name: 'inicio', canonical: false };
    }
    const tab = INCREMENT_TABS.includes(parts[2]) ? parts[2] : 'artefactos';
    const canonical = parts.length <= 3 && (parts.length < 3 || INCREMENT_TABS.includes(parts[2]));
    return { name: 'incremento', change, tab, canonical };
  }
  if (head === 'specs' && parts.length >= 2) {
    let domain;
    try {
      domain = decodeURIComponent(parts[1]);
    } catch (_) {
      return { name: 'inicio', canonical: false };
    }
    return { name: 'specs', domain, canonical: parts.length === 2 };
  }
  if (SIMPLE_ROUTES.includes(head)) {
    return { name: head, canonical: parts.length === 1 };
  }
  return { name: 'inicio', canonical: false };
}

// hrefFor construye el hash de una ruta.
export function hrefFor(name, params = {}) {
  if (name === 'incremento') {
    const base = `#/incrementos/${encodeURIComponent(params.change)}`;
    return params.tab && params.tab !== 'artefactos' ? `${base}/${params.tab}` : base;
  }
  if (name === 'specs' && params.domain) return `#/specs/${encodeURIComponent(params.domain)}`;
  return `#/${name}`;
}

export function navigate(name, params) {
  const target = hrefFor(name, params);
  if (window.location.hash === target) {
    // Mismo hash: no hay hashchange; se fuerza el refresco de la vista.
    window.dispatchEvent(new HashChangeEvent('hashchange'));
  } else {
    window.location.hash = target;
  }
}

// startRouter llama a onRoute en la carga y en cada cambio de hash.
export function startRouter(onRoute) {
  let first = true;
  const handle = () => {
    const route = parseHash(window.location.hash);
    if (!route.canonical && window.location.hash) {
      // Normaliza rutas desconocidas sin añadir una entrada de historial.
      const fixed = hrefFor(route.name, route);
      window.history.replaceState(null, '', fixed);
    }
    onRoute(route, { initial: first });
    first = false;
  };
  window.addEventListener('hashchange', handle);
  handle();
}
