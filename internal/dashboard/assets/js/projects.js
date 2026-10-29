// Selector de proyecto (popover accesible), explorador de carpetas locales y
// modal de añadir / inicializar proyecto.

import { api, escapeHtml } from './core/api.js';
import { $, openDialog, closeDialog, toast, setBusy } from './core/components.js';
import { icon } from './core/icons.js';
import { state, set } from './core/state.js';

let onSwitched = async () => {};

// ---- Popover del selector de proyecto ---------------------------------------

const trigger = () => $('#project-trigger');
const popover = () => $('#project-popover');

function popoverItems() {
  return Array.from(popover().querySelectorAll('.popover-item'));
}

export function openPopover() {
  const pop = popover();
  pop.hidden = false;
  trigger().setAttribute('aria-expanded', 'true');
  const items = popoverItems();
  const current = items.find(i => i.getAttribute('aria-current') === 'true') || items[0];
  if (current) current.focus();
}

export function closePopover({ restoreFocus = false } = {}) {
  const pop = popover();
  if (pop.hidden) return;
  pop.hidden = true;
  trigger().setAttribute('aria-expanded', 'false');
  if (restoreFocus) trigger().focus();
}

function renderProjectList() {
  const list = $('#project-list');
  const data = state.projects;
  const projects = (data && data.projects) || [];
  if (projects.length === 0) {
    list.innerHTML = '<span class="popover-label" style="text-transform:none;letter-spacing:0">No hay proyectos registrados.</span>';
    return;
  }
  list.innerHTML = projects.map(p => {
    const active = p.id === data.active_workspace || p.path === data.active_workspace;
    return `<button type="button" class="popover-item" data-project-id="${escapeHtml(p.id)}" aria-current="${active ? 'true' : 'false'}">
      ${icon('folder', 16)}<span class="grow">${escapeHtml(p.name)}</span>${p.is_configured ? '' : '<span class="badge badge-warn">Sin configurar</span>'}${active ? icon('check', 16) : ''}
    </button>`;
  }).join('');
}

export async function loadProjects() {
  try {
    const data = await api('/api/projects');
    set('projects', data);
    renderProjectList();
  } catch (err) {
    console.warn('No se pudo cargar la lista de proyectos:', err);
  }
}

async function switchProject(id) {
  try {
    await api('/api/projects/switch', { body: { id } });
    closePopover({ restoreFocus: true });
    await loadProjects();
    await onSwitched();
  } catch (err) {
    toast(`Error al cambiar de proyecto: ${err.message}`, 'bad');
  }
}

function initPopover() {
  const trig = trigger();
  const pop = popover();

  trig.addEventListener('click', () => {
    if (pop.hidden) openPopover();
    else closePopover({ restoreFocus: true });
  });

  pop.addEventListener('click', (event) => {
    const item = event.target.closest('[data-project-id]');
    if (item) switchProject(item.getAttribute('data-project-id'));
  });

  // Teclado: Escape cierra y devuelve el foco; flechas recorren los elementos.
  pop.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      closePopover({ restoreFocus: true });
      return;
    }
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      const items = popoverItems();
      const idx = items.indexOf(document.activeElement);
      const next = event.key === 'ArrowDown' ? (idx + 1) % items.length : (idx - 1 + items.length) % items.length;
      items[next].focus();
    }
  });
  trig.addEventListener('keydown', (event) => {
    if (event.key === 'ArrowDown' && pop.hidden) {
      event.preventDefault();
      openPopover();
    }
  });

  // Se cierra al salir el foco del bloque o al hacer clic fuera.
  $('#project-switch').addEventListener('focusout', (event) => {
    if (!pop.hidden && !$('#project-switch').contains(event.relatedTarget) && event.relatedTarget !== null) {
      closePopover();
    }
  });
  document.addEventListener('click', (event) => {
    if (!pop.hidden && !$('#project-switch').contains(event.target)) closePopover();
  });
}

// ---- Explorador de carpetas locales ----------------------------------------

const folderPickerState = {
  currentPath: '',
  parentPath: '',
  selectedPath: '',
  directories: [],
  drives: [],
  homeDir: '',
  workspaceDir: '',
  separator: '\\',
  onSelectCallback: null,
  baseDirContext: ''
};

const fp = {};

function bindFolderPickerElements() {
  Object.assign(fp, {
    dialog: $('#modal-folder-picker'),
    drives: $('#fp-drives-container'),
    breadcrumbs: $('#fp-breadcrumbs'),
    pathInput: $('#fp-path-input'),
    toggleInput: $('#fp-btn-toggle-input'),
    filter: $('#fp-filter-input'),
    list: $('#fp-folder-list'),
    selected: $('#fp-selected-path-display'),
    count: $('#fp-count-info')
  });
}

function setSelected(path) {
  folderPickerState.selectedPath = path;
  fp.selected.textContent = path;
}

export async function openFolderPicker(options = {}) {
  const { initialPath, baseDirContext, onSelect } = options;
  folderPickerState.onSelectCallback = onSelect || null;
  folderPickerState.baseDirContext = baseDirContext || '';
  folderPickerState.selectedPath = initialPath || '';
  fp.selected.textContent = initialPath || 'Cargando…';
  fp.filter.value = '';
  hidePathInput();
  openDialog(fp.dialog);
  await navigateToDirectory(initialPath || '');
}

function hidePathInput() {
  fp.pathInput.classList.add('hidden');
  fp.breadcrumbs.classList.remove('hidden');
  fp.toggleInput.innerHTML = icon('edit', 14);
  fp.toggleInput.setAttribute('aria-label', 'Editar la ruta a mano');
}

function showPathInput() {
  fp.pathInput.value = folderPickerState.currentPath;
  fp.pathInput.classList.remove('hidden');
  fp.breadcrumbs.classList.add('hidden');
  fp.toggleInput.innerHTML = icon('check', 14);
  fp.toggleInput.setAttribute('aria-label', 'Terminar de editar la ruta');
  fp.pathInput.focus();
  fp.pathInput.select();
}

async function navigateToDirectory(targetPath) {
  fp.list.innerHTML = '<div class="loading-state">Cargando carpetas…</div>';
  try {
    const url = '/api/fs/directories' + (targetPath ? '?path=' + encodeURIComponent(targetPath) : '');
    const data = await api(url);
    folderPickerState.currentPath = data.current_path;
    folderPickerState.parentPath = data.parent_path;
    folderPickerState.drives = data.drives || [];
    folderPickerState.homeDir = data.home_dir || '';
    folderPickerState.workspaceDir = data.workspace_dir || '';
    folderPickerState.separator = data.separator || '\\';
    folderPickerState.directories = data.directories || [];
    if (!folderPickerState.selectedPath) folderPickerState.selectedPath = data.current_path;
    fp.selected.textContent = folderPickerState.selectedPath || data.current_path;
    hidePathInput();
    renderFolderPickerUI();
  } catch (err) {
    fp.list.innerHTML = `<div class="error-banner" style="margin: 12px">Error: ${escapeHtml(err.message)}</div>`;
  }
}

function goTo(path) {
  setSelected(path);
  navigateToDirectory(path);
}

function renderFolderPickerUI() {
  fp.drives.innerHTML = '';
  folderPickerState.drives.forEach(drv => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'fp-chip';
    const isCurrent = folderPickerState.currentPath.toLowerCase().startsWith(drv.toLowerCase());
    btn.setAttribute('aria-pressed', String(isCurrent));
    btn.textContent = drv;
    btn.title = `Ir a la unidad ${drv}`;
    btn.addEventListener('click', () => goTo(drv));
    fp.drives.appendChild(btn);
  });
  renderBreadcrumbs();
  renderFolderItems();
}

function renderBreadcrumbs() {
  fp.breadcrumbs.innerHTML = '';
  const p = folderPickerState.currentPath;
  const sep = folderPickerState.separator;
  fp.pathInput.value = p;

  let parts = [];
  let prefix = '';
  if (p.includes(':')) {
    const driveMatch = p.match(/^([A-Za-z]:[\\/]?)/);
    if (driveMatch) {
      prefix = driveMatch[1];
      parts = p.slice(prefix.length).split(/[\\/]+/).filter(Boolean);
    } else {
      parts = p.split(/[\\/]+/).filter(Boolean);
    }
  } else {
    if (p.startsWith('/')) prefix = '/';
    parts = p.split(/[\\/]+/).filter(Boolean);
  }

  const makeCrumb = (label, path, current) => {
    const crumb = document.createElement('button');
    crumb.type = 'button';
    crumb.className = 'fp-crumb';
    if (current) crumb.setAttribute('aria-current', 'true');
    crumb.textContent = label;
    crumb.title = `Ir a ${path}`;
    crumb.addEventListener('click', () => goTo(path));
    return crumb;
  };

  if (prefix) fp.breadcrumbs.appendChild(makeCrumb(prefix, prefix, parts.length === 0));

  let acc = prefix;
  parts.forEach((seg, idx) => {
    const sepSpan = document.createElement('span');
    sepSpan.className = 'fp-crumb-sep';
    sepSpan.setAttribute('aria-hidden', 'true');
    sepSpan.textContent = '›';
    fp.breadcrumbs.appendChild(sepSpan);
    if (!acc.endsWith(sep) && !acc.endsWith('/') && !acc.endsWith('\\')) acc += sep;
    acc += seg;
    fp.breadcrumbs.appendChild(makeCrumb(seg, acc, idx === parts.length - 1));
  });
}

function renderFolderItems() {
  fp.list.innerHTML = '';
  const filterText = fp.filter.value.trim().toLowerCase();
  const filtered = folderPickerState.directories.filter(d => !filterText || d.name.toLowerCase().includes(filterText));
  fp.count.textContent = `${filtered.length} carpetas encontradas`;

  if (folderPickerState.parentPath) {
    const up = document.createElement('button');
    up.type = 'button';
    up.className = 'fp-item';
    up.innerHTML = `<span class="fp-item-name">${icon('folder', 16)}<span><strong>..</strong> (directorio superior)</span></span><span class="fp-item-action">Subir ${icon('arrowUp', 14)}</span>`;
    up.addEventListener('click', () => goTo(folderPickerState.parentPath));
    fp.list.appendChild(up);
  }

  if (filtered.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'empty-state';
    empty.textContent = filterText ? 'No hay carpetas que coincidan con la búsqueda.' : 'No hay subcarpetas en este directorio.';
    fp.list.appendChild(empty);
    return;
  }

  filtered.forEach(dir => {
    const item = document.createElement('div');
    item.className = 'fp-item' + (dir.hidden ? ' hidden-dir' : '');
    item.setAttribute('role', 'option');
    item.setAttribute('aria-selected', String(dir.path === folderPickerState.selectedPath));
    item.tabIndex = 0;
    item.innerHTML = `<span class="fp-item-name" title="${escapeHtml(dir.path)}">${icon('folder', 16)}<span>${escapeHtml(dir.name)}</span></span>
      <button type="button" class="fp-item-action btn btn-ghost btn-xs" aria-label="Abrir ${escapeHtml(dir.name)}">Abrir ${icon('arrowRight', 14)}</button>`;

    const select = () => {
      fp.list.querySelectorAll('[role="option"]').forEach(el => el.setAttribute('aria-selected', 'false'));
      item.setAttribute('aria-selected', 'true');
      setSelected(dir.path);
    };
    item.addEventListener('click', select);
    item.addEventListener('dblclick', () => goTo(dir.path));
    item.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' && event.target === item) { event.preventDefault(); select(); }
      if (event.key === ' ' && event.target === item) { event.preventDefault(); select(); }
      if (event.key === 'ArrowRight' && event.target === item) { event.preventDefault(); goTo(dir.path); }
    });
    item.querySelector('.fp-item-action').addEventListener('click', (event) => {
      event.stopPropagation();
      goTo(dir.path);
    });
    fp.list.appendChild(item);
  });
}

function initFolderPicker() {
  bindFolderPickerElements();
  fp.filter.addEventListener('input', renderFolderItems);
  $('#fp-btn-up').addEventListener('click', () => {
    if (folderPickerState.parentPath) goTo(folderPickerState.parentPath);
  });
  $('#fp-btn-refresh').addEventListener('click', () => navigateToDirectory(folderPickerState.currentPath));
  fp.toggleInput.addEventListener('click', () => {
    if (fp.pathInput.classList.contains('hidden')) showPathInput();
    else hidePathInput();
  });
  fp.pathInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
      const p = fp.pathInput.value.trim();
      if (p) goTo(p);
    } else if (event.key === 'Escape' && !fp.pathInput.classList.contains('hidden')) {
      // Escape cierra solo el campo de ruta, no el diálogo.
      event.stopPropagation();
      event.preventDefault();
      hidePathInput();
      fp.toggleInput.focus();
    }
  });
  $('#fp-btn-home').addEventListener('click', () => {
    if (folderPickerState.homeDir) goTo(folderPickerState.homeDir);
  });
  $('#fp-btn-workspace').addEventListener('click', () => {
    if (folderPickerState.workspaceDir) goTo(folderPickerState.workspaceDir);
  });

  const nativeBtn = $('#fp-btn-native-so');
  nativeBtn.addEventListener('click', async () => {
    const restore = setBusy(nativeBtn, 'Abriendo…');
    try {
      const data = await api('/api/fs/native-picker', {
        body: { initial_path: folderPickerState.selectedPath || folderPickerState.currentPath }
      });
      if (data && data.path && !data.canceled) {
        setSelected(data.path);
        await navigateToDirectory(data.path);
      }
    } catch (err) {
      console.warn('Fallo en el selector nativo:', err);
      toast('No se pudo abrir el explorador del sistema.', 'warn');
    } finally {
      restore();
    }
  });

  $('#fp-btn-confirm').addEventListener('click', () => {
    const chosen = folderPickerState.selectedPath || folderPickerState.currentPath;
    if (!chosen) {
      toast('Selecciona una carpeta.', 'warn');
      return;
    }
    if (folderPickerState.onSelectCallback) folderPickerState.onSelectCallback(chosen);
    closeDialog(fp.dialog);
  });
}

// ---- Añadir / inicializar proyecto -----------------------------------------

function baseName(path) {
  const parts = path.replace(/[\\/]+$/, '').split(/[\\/]/);
  return parts[parts.length - 1];
}

function addRoleRow(key = '', name = '', repos = '', nonBlocking = false) {
  const container = $('#roles-builder-container');
  const card = document.createElement('div');
  card.className = 'role-builder-card';
  card.innerHTML = `
    <button type="button" class="btn btn-ghost btn-icon btn-sm btn-remove-role" aria-label="Quitar este rol">${icon('x', 14)}</button>
    <div class="field-grid">
      <div class="field"><label>Identificador (ej. web, core)<input type="text" class="form-input role-key-input" value="${escapeHtml(key)}" placeholder="web"></label></div>
      <div class="field"><label>Nombre legible<input type="text" class="form-input role-name-input" value="${escapeHtml(name)}" placeholder="Frontend Web UI"></label></div>
    </div>
    <div class="field">
      <span class="field-label">Rutas / repositorios (separadas por coma)</span>
      <div class="field-row">
        <input type="text" class="form-input role-repos-input" value="${escapeHtml(repos)}" placeholder="src/web, ." aria-label="Rutas de repositorios del rol">
        <button type="button" class="btn btn-sm btn-browse-role-repo" aria-label="Examinar una carpeta local" title="Examinar una carpeta local">${icon('folder', 14)}</button>
      </div>
    </div>
    <label class="check"><input type="checkbox" class="role-nonblocking-input" ${nonBlocking ? 'checked' : ''}><span>No bloqueante al archivar (advisory / deuda diferida acumulativa)</span></label>`;

  card.querySelector('.btn-remove-role').addEventListener('click', () => card.remove());

  const reposInput = card.querySelector('.role-repos-input');
  card.querySelector('.btn-browse-role-repo').addEventListener('click', () => {
    const projRoot = $('#input-project-path').value.trim();
    openFolderPicker({
      initialPath: projRoot || '',
      baseDirContext: projRoot || '',
      onSelect: (selectedPath) => {
        let finalPath = selectedPath;
        if (projRoot) {
          const cleanRoot = projRoot.replace(/[\\/]+$/, '');
          if (selectedPath.toLowerCase().startsWith(cleanRoot.toLowerCase())) {
            finalPath = selectedPath.slice(cleanRoot.length).replace(/^[\\/]+/, '') || '.';
          }
        }
        const current = reposInput.value.trim();
        if (!current || current === '.') {
          reposInput.value = finalPath;
        } else {
          const arr = current.split(',').map(s => s.trim()).filter(Boolean);
          if (!arr.includes(finalPath)) arr.push(finalPath);
          reposInput.value = arr.join(', ');
        }
      }
    });
  });
  container.appendChild(card);
}

function showError(id, message) {
  const box = document.getElementById(id);
  if (!box) return;
  box.hidden = !message;
  box.textContent = message || '';
}

function initAddProject() {
  const dialog = $('#modal-add-project');
  const inputPath = $('#input-project-path');
  const inputName = $('#input-project-name');

  const fillFromPicker = (selectedPath) => {
    inputPath.value = selectedPath;
    if (!inputName.value.trim()) {
      const base = baseName(selectedPath);
      if (base) inputName.value = base;
    }
  };

  const openAdd = () => {
    inputPath.value = '';
    inputName.value = '';
    $('#roles-builder-container').innerHTML = '';
    showError('add-project-error', '');
    openDialog(dialog, '#input-project-path');
  };

  $('#btn-add-project').addEventListener('click', () => { closePopover(); openAdd(); });
  $('#btn-add-role-row').addEventListener('click', () => addRoleRow());

  $('#btn-browse-project-path').addEventListener('click', () => {
    openFolderPicker({ initialPath: inputPath.value.trim(), onSelect: fillFromPicker });
  });

  // Explorar desde el popover: elige la carpeta y abre el modal ya rellenado.
  $('#btn-browse-header-project').addEventListener('click', () => {
    closePopover();
    openFolderPicker({
      onSelect: (selectedPath) => {
        fillFromPicker(selectedPath);
        showError('add-project-error', '');
        openDialog(dialog, '#input-project-name');
      }
    });
  });

  const submit = $('#btn-submit-add-project');
  submit.addEventListener('click', async () => {
    const pathVal = inputPath.value.trim();
    if (!pathVal) {
      showError('add-project-error', 'Debes indicar la ruta del proyecto en disco.');
      inputPath.focus();
      return;
    }
    const roles = [];
    $('#roles-builder-container').querySelectorAll('.role-builder-card').forEach(card => {
      const key = card.querySelector('.role-key-input').value.trim();
      const name = card.querySelector('.role-name-input').value.trim();
      const reposStr = card.querySelector('.role-repos-input').value.trim();
      const nonBlocking = card.querySelector('.role-nonblocking-input').checked;
      const repositories = reposStr ? reposStr.split(',').map(s => s.trim()).filter(Boolean) : ['.'];
      if (key || name) {
        roles.push({ key: key || name.toLowerCase().replace(/\s+/g, '-'), name: name || key, repositories, non_blocking: nonBlocking });
      }
    });

    const restore = setBusy(submit, 'Procesando…');
    try {
      await api('/api/projects/init', {
        body: { path: pathVal, name: inputName.value.trim(), topology: $('#select-project-topology').value || 'monorepo-embedded', roles }
      });
      closeDialog(dialog);
      await loadProjects();
      await onSwitched();
    } catch (err) {
      showError('add-project-error', `Error: ${err.message}`);
    } finally {
      restore();
    }
  });
}

// initProjects enlaza el popover, el explorador y el modal de alta.
// onSwitchedFn se llama tras cambiar o añadir un proyecto para recargar los datos.
export function initProjects(onSwitchedFn) {
  onSwitched = onSwitchedFn;
  initPopover();
  initFolderPicker();
  initAddProject();
}
