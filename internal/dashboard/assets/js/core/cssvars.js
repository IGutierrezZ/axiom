// Valores dinámicos de CSS sin estilos en línea. La CSP (style-src 'self') bloquea el
// atributo style escrito en HTML, pero no las asignaciones CSSOM desde JS
// (element.style.x y style.setProperty). Los módulos emiten data-* y este módulo los aplica.

const IX_COLS_RE = /^[0-9a-z().,% -]+$/;

// applyCssData aplica data-pct (ancho 0-100 %) y data-ix-cols (--ix-cols) bajo root.
export function applyCssData(root = document) {
  if (!root || !root.querySelectorAll) return;
  const nodes = [];
  if (root.matches && root.matches('[data-pct], [data-ix-cols]')) nodes.push(root);
  root.querySelectorAll('[data-pct], [data-ix-cols]').forEach(el => nodes.push(el));
  nodes.forEach(el => {
    if (el.hasAttribute('data-pct')) {
      const pct = Math.max(0, Math.min(100, Number(el.getAttribute('data-pct')) || 0));
      el.style.width = `${pct}%`;
    }
    if (el.hasAttribute('data-ix-cols')) {
      const cols = el.getAttribute('data-ix-cols');
      if (IX_COLS_RE.test(cols)) el.style.setProperty('--ix-cols', cols);
    }
  });
}

// observeCssData aplica los data-* de todo nodo que se inserte después (asíncrono; los
// módulos que pintan HTML pueden llamar además a applyCssData de forma síncrona).
export function observeCssData(root = document.body) {
  applyCssData(root);
  const obs = new MutationObserver(mutations => {
    for (const m of mutations) {
      if (m.type === 'attributes') applyCssData(m.target);
      else m.addedNodes.forEach(n => { if (n.nodeType === 1) applyCssData(n); });
    }
  });
  obs.observe(root, { childList: true, subtree: true, attributes: true, attributeFilter: ['data-pct', 'data-ix-cols'] });
  return obs;
}
