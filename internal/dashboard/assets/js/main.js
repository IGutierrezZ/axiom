// Punto de entrada: arranca tema, estructura, router y pantallas.
// Módulos ES sin paso de compilación, embebidos con go:embed.

import { api } from './core/api.js';
import { bindCopyButtons, $ } from './core/components.js';
import { hydrateIcons } from './core/icons.js';
import { startRouter } from './core/router.js';
import { set, on } from './core/state.js';
import { initTheme } from './core/theme.js';
import { initProjects, loadProjects } from './projects.js';
import { initShell, setActiveNav, renderBreadcrumb, routeLabel, closeMenu, checkSpecsSync } from './shell.js';
import { initDetail, enterDetail } from './screens/detalle.js';
import { initEcosystem, loadEcosystem, renderTopology } from './screens/ecosystem.js';
import { initIncrements, loadIncrements, updateRoleFilterButtons } from './screens/incrementos.js';
import { initInicio, renderInicio } from './screens/inicio.js';
import { initLivingDocs, loadLivingDocs } from './screens/livingdoc.js';
import { initSemantic, loadSemanticData } from './screens/semantic.js';
import { initSkills, loadSkills, loadSkillsInbox } from './screens/skills.js';

// Cada ruta apunta a la vista (<section class="view">) que la muestra.
const VIEW_BY_ROUTE = {
  inicio: 'view-inicio',
  incrementos: 'view-incrementos',
  incremento: 'view-incremento',
  specs: 'view-specs',
  skills: 'view-skills',
  codigo: 'view-codigo',
  ecosistema: 'view-ecosistema'
};

// Ruta activa, para repintar la miga cuando llega el nombre del proyecto.
let currentRoute = null;

async function loadWorkspace() {
  try {
    const ws = await api('/api/workspace');
    set('workspace', ws);
  } catch (err) {
    console.error(err);
    $('#ws-name').textContent = 'Error al conectar';
  }
}

async function loadAllData() {
  await Promise.all([
    loadWorkspace(),
    loadProjects(),
    loadIncrements(),
    loadSkills(),
    loadSkillsInbox(),
    loadSemanticData(),
    loadLivingDocs(),
    loadEcosystem(),
    checkSpecsSync()
  ]);
}

function showView(route, { initial }) {
  const target = VIEW_BY_ROUTE[route.name];
  document.querySelectorAll('.view[id^="view-"]').forEach(v => { v.hidden = v.id !== target; });
  setActiveNav(route);
  renderBreadcrumb(route);
  closeMenu();

  const title = route.name === 'incremento' ? route.change : routeLabel(route);
  document.title = `${title} · Axiom`;

  if (route.name === 'incremento') enterDetail(route);

  // Tras navegar (no en la carga inicial) el foco pasa al título de la vista,
  // para que los lectores de pantalla anuncien el cambio.
  if (!initial) {
    const heading = document.querySelector(`#${target} h1`);
    if (heading) heading.focus({ preventScroll: false });
  }
}

function boot() {
  initTheme();
  hydrateIcons();
  bindCopyButtons(document);

  initShell({ onRefresh: loadAllData, reloadWorkspace: loadWorkspace });
  initProjects(loadAllData);
  initInicio(loadAllData);
  initIncrements();
  initDetail();
  initSkills();
  initSemantic();
  initLivingDocs();
  initEcosystem();

  // Las pantallas que dependen del workspace se repintan cuando llega.
  on('workspace', renderInicio);
  on('workspace', renderTopology);
  on('workspace', updateRoleFilterButtons);
  on('workspace', () => {
    // La miga muestra el nombre del proyecto.
    const route = currentRoute;
    if (route) renderBreadcrumb(route);
  });

  startRouter((route, opts) => {
    currentRoute = route;
    showView(route, opts);
  });

  loadAllData();
}

boot();
