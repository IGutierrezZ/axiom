// Punto de entrada: arranca tema, estructura, router y pantallas.
// Módulos ES sin paso de compilación, embebidos con go:embed.

import { api } from './core/api.js';
import { bindCopyButtons, $ } from './core/components.js';
import { hydrateIcons } from './core/icons.js';
import { startRouter } from './core/router.js';
import { refreshOverview } from './core/overview.js';
import { set, on } from './core/state.js';
import { initTheme } from './core/theme.js';
import { initProjects, loadProjects } from './projects.js';
import { initShell, setActiveNav, renderBreadcrumb, routeLabel, closeMenu } from './shell.js';
import { initDetail, enterDetail, refreshDetail } from './screens/detalle.js';
import { initEcosystem, loadEcosystem } from './screens/ecosystem.js';
import { initIncrements, loadIncrements, renderIncrements } from './screens/incrementos.js';
import { initInicio, renderInicio } from './screens/inicio.js';
import { enterSpecs, initLivingDocs, loadLivingDocs } from './screens/livingdoc.js';
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
// Última vista mostrada: el foco pasa al título solo al cambiar de vista o de incremento,
// no al pasar de pestaña dentro del mismo incremento (el teclado debe quedarse en la pestaña).
let shownKey = null;

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
    // Inicio y la barra lateral: las rutas que otras pantallas ya piden en este
    // mismo instante se comparten (api() agrupa los GET simultáneos).
    refreshOverview()
  ]);
  // Actualizar también refresca el incremento abierto, si lo hay.
  if (currentRoute && currentRoute.name === 'incremento') refreshDetail();
}

function showView(route, { initial }) {
  const target = VIEW_BY_ROUTE[route.name];
  document.querySelectorAll('.view[id^="view-"]').forEach(v => { v.hidden = v.id !== target; });
  setActiveNav(route);
  renderBreadcrumb(route);
  closeMenu();

  const title = route.name === 'incremento' ? route.change : route.name === 'specs' && route.domain ? route.domain : routeLabel(route);
  document.title = `${title} · Axiom`;

  if (route.name === 'incremento') enterDetail(route);
  // El visor de specs vivas es una ruta (#/specs/<dominio>): también con atrás y adelante.
  if (route.name === 'specs') enterSpecs(route);

  // Tras navegar (no en la carga inicial) el foco pasa al título de la vista,
  // para que los lectores de pantalla anuncien el cambio. Cambiar de dominio dentro de
  // Specs vivas no cuenta: el foco se queda en la lista (o pasa al visor).
  const key = `${route.name}:${route.change || ''}`;
  if (!initial && key !== shownKey) {
    const heading = document.querySelector(`#${target} h1`);
    if (heading) heading.focus({ preventScroll: false });
  }
  shownKey = key;
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
  on('workspace', renderIncrements);
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
