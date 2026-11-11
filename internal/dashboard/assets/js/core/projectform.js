// Lógica pura del formulario de alta de proyecto: normaliza los roles que el
// usuario teclea y valida en cliente lo que se puede saber sin tocar el disco.
// El servidor repite estas reglas (Initializer.ValidateInit en internal/hub) y añade las que dependen de
// la configuración generada; aquí solo se adelanta el aviso para no esperar a la
// respuesta. Sin acceso al DOM, para poder probarla con Node.

// splitList convierte «a, b ,, c» en ['a', 'b', 'c'].
export function splitList(text) {
  return String(text ?? '')
    .split(',')
    .map(s => s.trim())
    .filter(Boolean);
}

// roleKeyFor devuelve la clave con la que se enviará el rol: la que se tecleó o,
// si falta, la que se deriva del nombre (minúsculas y guiones).
export function roleKeyFor(card) {
  const key = String(card?.key ?? '').trim();
  if (key) return key;
  return String(card?.name ?? '').trim().toLowerCase().replace(/\s+/g, '-');
}

// repoPathProblem explica por qué una ruta de repositorio no es admisible, o
// devuelve '' si lo es: debe ser relativa y quedar dentro del proyecto.
export function repoPathProblem(repo) {
  const path = String(repo ?? '').trim();
  if (!path) return '';
  if (/^[\\/]/.test(path) || /^[A-Za-z]:/.test(path)) {
    return 'es una ruta absoluta: debe ser relativa a la carpeta del proyecto';
  }
  // Se normaliza antes de decidir si sale: «src/../../fuera» sale aunque empiece dentro.
  const parts = [];
  for (const part of path.split(/[\\/]+/)) {
    if (!part || part === '.') continue;
    if (part === '..') {
      if (!parts.length) return 'sale de la carpeta del proyecto: debe quedar dentro de ella';
      parts.pop();
    } else {
      parts.push(part);
    }
  }
  return '';
}

// buildRoles convierte las tarjetas del formulario en el cuerpo `roles` de la
// petición y reúne las violaciones. Una tarjeta del todo vacía se ignora; una
// con repositorios o tecnologías pero sin identificador ni nombre es un error.
//
// card: { key, name, repos, tech, nonBlocking } con los textos tal como están en
// los campos. Devuelve { roles, violations }.
export function buildRoles(cards) {
  const roles = [];
  const violations = [];
  const seen = new Map();

  for (const card of cards || []) {
    const key = roleKeyFor(card);
    const name = String(card?.name ?? '').trim();
    const repos = splitList(card?.repos);
    const tech = splitList(card?.tech);

    if (!key) {
      if (repos.length || tech.length) {
        violations.push('Hay un rol sin identificador ni nombre: rellena al menos uno de los dos o quítalo.');
      }
      continue;
    }

    const folded = key.toLowerCase();
    if (seen.has(folded)) {
      const first = seen.get(folded);
      violations.push(first === key
        ? `El identificador de rol «${key}» está repetido.`
        : `Los identificadores «${first}» y «${key}» solo se distinguen por las mayúsculas.`);
      continue;
    }
    seen.set(folded, key);

    const repositories = repos.length ? repos : ['.'];
    for (const repo of repositories) {
      const problem = repoPathProblem(repo);
      if (problem) violations.push(`El repositorio «${repo}» del rol «${key}» ${problem}.`);
    }

    const role = { key, name: name || key, repositories, non_blocking: !!card?.nonBlocking };
    if (tech.length) role.tech = tech;
    roles.push(role);
  }
  return { roles, violations };
}

// specsRepositoryProblem explica por qué el repositorio de especificaciones de un
// alta multirepo no es admisible, o devuelve '' si lo es: obligatorio, relativo a
// la carpeta del proyecto, dentro de ella y distinto de ella.
export function specsRepositoryProblem(specs) {
  const value = String(specs ?? '').trim();
  if (!value) return 'Indica el repositorio de especificaciones: en multirepo es obligatorio.';
  const normalized = value.split(/[\\/]+/).filter(part => part && part !== '.');
  if (!normalized.length) {
    return 'El repositorio de especificaciones no puede ser la propia carpeta del proyecto: debe ser un repositorio aparte dentro de ella.';
  }
  const problem = repoPathProblem(value);
  return problem ? `El repositorio de especificaciones «${value}» ${problem}.` : '';
}

// validateProjectForm valida el formulario completo antes de enviarlo y devuelve
// { roles, specsRepository, violations }; con violaciones no debe hacerse la
// petición. specsRepository solo se rellena en multirepo.
export function validateProjectForm({ path, cards, topology, specsRepository }) {
  const violations = [];
  if (!String(path ?? '').trim()) {
    violations.push('Debes indicar la ruta del proyecto en disco.');
  }
  let specs = '';
  if (topology === 'multirepo') {
    const problem = specsRepositoryProblem(specsRepository);
    if (problem) violations.push(problem);
    else specs = String(specsRepository).trim();
  }
  const built = buildRoles(cards);
  violations.push(...built.violations);
  return { roles: built.roles, specsRepository: specs, violations };
}

// violationsFromError extrae la lista de violaciones de un error de la API. El
// servidor responde 400 con `violations` cuando la configuración no es válida;
// ante cualquier otro fallo devuelve el mensaje como única entrada.
export function violationsFromError(err) {
  const list = err?.data?.violations;
  if (Array.isArray(list) && list.length) return list.map(String);
  return [`Error: ${err?.message ?? 'fallo desconocido'}`];
}

// adapterOptions normaliza el catálogo de /api/adapters para pintarlo: los que no
// admiten el modo workspace salen deshabilitados, con el motivo.
export function adapterOptions(list) {
  return (Array.isArray(list) ? list : []).map(item => ({
    id: String(item.id),
    name: String(item.name || item.id),
    disabled: !item.workspace,
    reason: item.workspace ? '' : String(item.reason || 'no admite el modo workspace')
  }));
}

// adapterWarning devuelve el aviso a mostrar tras el alta cuando el proyecto se creó
// pero la instalación de los adaptadores falló (fallo parcial), o '' si todo fue bien.
export function adapterWarning(response) {
  const outcome = response?.adapters;
  if (!outcome?.error) return '';
  return `El proyecto se ha creado, pero no se pudieron instalar los adaptadores (${outcome.error}). Puedes instalarlos más tarde con «axiom install --scope workspace».`;
}
