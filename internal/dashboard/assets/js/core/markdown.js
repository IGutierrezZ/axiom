// Renderizador de Markdown pequeño y seguro para los artefactos del dashboard.
//
// Seguridad: todo texto se escapa ANTES de añadir etiquetas, de modo que el HTML
// crudo del documento se ve como texto y nunca se inyecta. Solo se generan
// enlaces http(s) y mailto; imágenes y el resto de esquemas se muestran como texto
// (sin recursos externos, el dashboard funciona sin red).
//
// Admite: encabezados, párrafos, listas anidadas, casillas de tareas, bloques de
// código, código en línea, negrita, cursiva, tachado, enlaces, citas, reglas,
// tablas simples y metadatos YAML iniciales.

import { escapeHtml } from './api.js';
import { icon } from './icons.js';

const FENCE = /^\s*(```|~~~)\s*([\w+-]*)\s*$/;
const HEADING = /^(#{1,6})\s+(.*?)\s*#*\s*$/;
const HRULE = /^\s*([-*_])(\s*\1){2,}\s*$/;
const ITEM = /^(\s*)([-*+]|\d+[.)])\s+(.*)$/;
const TASK = /^\[([ xX])\]\s+(.*)$/;
const QUOTE = /^\s*>\s?(.*)$/;
const TABLE_SEP = /^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$/;

// ---- En línea ----------------------------------------------------------------

// safeHref admite solo http(s) y mailto. Un esquema desconocido, #ancla o ruta
// relativa devuelve '' (se pinta como texto): una #ancla rompería el router por hash.
export function safeHref(url) {
  const u = String(url || '').trim();
  return /^(https?:\/\/|mailto:)[^\s]+$/i.test(u) ? u : '';
}

function renderLinks(escaped) {
  // Imágenes: solo el texto alternativo (no se cargan recursos).
  let out = escaped.replace(/!\[([^\]]*)\]\((?:[^()\s]|\([^()\s]*\))*[^)]*\)/g, (_, alt) => alt);
  out = out.replace(/\[([^\]]+)\]\(((?:[^()\s]|\([^()\s]*\))+)(?:\s+&quot;[^)]*?&quot;)?\)/g, (m, text, url) => {
    // La URL ya viene escapada; para validarla se compara tal cual (sin decodificar).
    const href = safeHref(url.replace(/&amp;/g, '&'));
    if (!href) return text;
    return `<a href="${escapeHtml(href)}" target="_blank" rel="noopener noreferrer">${text}</a>`;
  });
  return out;
}

export function renderInline(text) {
  const parts = String(text == null ? '' : text).split(/(`+[^`]*`+)/);
  return parts.map((part, i) => {
    if (i % 2 === 1) {
      const code = part.replace(/^`+/, '').replace(/`+$/, '');
      return `<code>${escapeHtml(code)}</code>`;
    }
    let s = escapeHtml(part);
    s = renderLinks(s);
    s = s.replace(/\*\*([^*]+?)\*\*/g, '<strong>$1</strong>');
    s = s.replace(/__([^_]+?)__/g, '<strong>$1</strong>');
    s = s.replace(/(^|[^*\w])\*([^*\s][^*]*?)\*(?!\*)/g, '$1<em>$2</em>');
    s = s.replace(/(^|[^_\w])_([^_\s][^_]*?)_(?![_\w])/g, '$1<em>$2</em>');
    s = s.replace(/~~([^~]+?)~~/g, '<del>$1</del>');
    return s;
  }).join('');
}

// ---- Bloques -----------------------------------------------------------------

function indentOf(line) {
  return line.match(/^[ \t]*/)[0].replace(/\t/g, '    ').length;
}

function isBlockStart(line, next) {
  return FENCE.test(line) || HEADING.test(line) || HRULE.test(line) || ITEM.test(line) || QUOTE.test(line) ||
    (line.includes('|') && next !== undefined && TABLE_SEP.test(next));
}

function splitRow(line) {
  let s = line.trim();
  if (s.startsWith('|')) s = s.slice(1);
  if (s.endsWith('|')) s = s.slice(0, -1);
  return s.split('|').map(c => c.trim());
}

function taskBox(done) {
  return done
    ? `<span class="md-check is-done" role="img" aria-label="Hecha">${icon('checkSquare', 18)}</span>`
    : `<span class="md-check" role="img" aria-label="Pendiente">${icon('square', 18)}</span>`;
}

// renderList convierte los elementos {indent, ordered, task, done, text} en listas anidadas.
function renderList(items) {
  const out = [];
  const stack = []; // {indent, tag, liOpen}
  const close = (frame) => {
    if (frame.liOpen) out.push('</li>');
    out.push(`</${frame.tag}>`);
  };
  items.forEach(item => {
    const tag = item.ordered ? 'ol' : 'ul';
    while (stack.length && item.indent < stack[stack.length - 1].indent) close(stack.pop());
    let top = stack[stack.length - 1];
    if (top && item.indent === top.indent && top.tag !== tag) {
      close(stack.pop());
      top = stack[stack.length - 1];
    }
    if (!top || item.indent > top.indent) {
      out.push(`<${tag}${item.task !== null && !item.ordered ? ' class="md-tasks"' : ''}>`);
      stack.push({ indent: item.indent, tag, liOpen: false });
    } else if (top.liOpen) {
      out.push('</li>');
      top.liOpen = false;
    }
    const frame = stack[stack.length - 1];
    if (item.task !== null) {
      out.push(`<li class="md-task${item.task ? ' is-done' : ''}">${taskBox(item.task)}<span class="md-task-text">${renderInline(item.text)}</span>`);
    } else {
      out.push(`<li>${renderInline(item.text)}`);
    }
    frame.liOpen = true;
  });
  while (stack.length) close(stack.pop());
  return out.join('');
}

// sectionStats calcula, por cada encabezado, las casillas hechas/totales de su sección.
function sectionStats(lines) {
  const heads = [];
  let fenced = false;
  lines.forEach((line, i) => {
    if (FENCE.test(line)) { fenced = !fenced; return; }
    if (fenced) return;
    const h = line.match(HEADING);
    if (h) heads.push({ line: i, level: h[1].length, done: 0, total: 0 });
  });
  const stats = new Map();
  fenced = false;
  const open = [];
  let cursor = 0;
  lines.forEach((line, i) => {
    if (FENCE.test(line)) { fenced = !fenced; return; }
    if (fenced) return;
    if (cursor < heads.length && heads[cursor].line === i) {
      const h = heads[cursor++];
      while (open.length && open[open.length - 1].level >= h.level) open.pop();
      open.push(h);
      stats.set(i, h);
      return;
    }
    const m = line.match(ITEM);
    if (!m) return;
    const t = m[3].match(TASK);
    if (!t) return;
    open.forEach(h => {
      h.total += 1;
      if (t[1] !== ' ') h.done += 1;
    });
  });
  return stats;
}

// renderMarkdown devuelve HTML seguro. Con opts.sectionStats cada encabezado con
// casillas muestra el avance «hechas/totales» de su sección (útil en tasks.md).
export function renderMarkdown(src, opts = {}) {
  let lines = String(src == null ? '' : src).replace(/\r\n?/g, '\n').split('\n');
  const html = [];

  // Metadatos YAML iniciales: se muestran aparte, escapados.
  if (lines[0] && lines[0].trim() === '---') {
    const end = lines.findIndex((l, i) => i > 0 && l.trim() === '---');
    if (end > 0) {
      html.push(`<details class="md-front"><summary>Metadatos</summary><pre class="md-code">${escapeHtml(lines.slice(1, end).join('\n'))}</pre></details>`);
      lines = lines.slice(end + 1);
    }
  }

  const stats = opts.sectionStats ? sectionStats(lines) : new Map();
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (!line.trim()) { i++; continue; }

    const fence = line.match(FENCE);
    if (fence) {
      const buf = [];
      i++;
      while (i < lines.length && !(FENCE.test(lines[i]) && lines[i].trim().startsWith(fence[1]))) buf.push(lines[i++]);
      i++; // cierre
      html.push(`<pre class="md-code"><code>${escapeHtml(buf.join('\n'))}</code></pre>`);
      continue;
    }

    const head = line.match(HEADING);
    if (head) {
      const level = Math.min(head[1].length + 2, 6); // el visor ya está bajo un h1/h2
      const st = stats.get(i);
      const badge = st && st.total > 0 ? `<span class="md-count${st.done === st.total ? ' is-done' : ''}">${st.done}/${st.total} hechas</span>` : '';
      html.push(`<h${level} class="md-h">${renderInline(head[2])}${badge}</h${level}>`);
      i++;
      continue;
    }

    if (HRULE.test(line)) { html.push('<hr class="md-hr">'); i++; continue; }

    if (QUOTE.test(line)) {
      const buf = [];
      while (i < lines.length && QUOTE.test(lines[i])) buf.push(lines[i++].match(QUOTE)[1]);
      html.push(`<blockquote class="md-quote">${buf.map(renderInline).join('<br>')}</blockquote>`);
      continue;
    }

    if (line.includes('|') && i + 1 < lines.length && TABLE_SEP.test(lines[i + 1])) {
      const header = splitRow(line);
      i += 2;
      const rows = [];
      while (i < lines.length && lines[i].trim() && lines[i].includes('|')) rows.push(splitRow(lines[i++]));
      html.push(`<div class="table-wrap md-table" role="region" aria-label="Tabla del documento" tabindex="0"><table class="data-table"><thead><tr>${header.map(c => `<th scope="col">${renderInline(c)}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${header.map((_, k) => `<td>${renderInline(r[k] || '')}</td>`).join('')}</tr>`).join('')}</tbody></table></div>`);
      continue;
    }

    if (ITEM.test(line)) {
      const items = [];
      while (i < lines.length) {
        const cur = lines[i];
        const m = cur.match(ITEM);
        if (m) {
          const t = m[3].match(TASK);
          items.push({
            indent: indentOf(m[1]),
            ordered: /\d/.test(m[2]),
            task: t ? t[1] !== ' ' : null,
            text: t ? t[2] : m[3]
          });
          i++;
        } else if (!cur.trim()) {
          let j = i;
          while (j < lines.length && !lines[j].trim()) j++;
          if (j < lines.length && (ITEM.test(lines[j]) || indentOf(lines[j]) >= 2)) i = j;
          else break;
        } else if (indentOf(cur) >= 2 && items.length) {
          items[items.length - 1].text += ` ${cur.trim()}`;
          i++;
        } else {
          break;
        }
      }
      html.push(renderList(items));
      continue;
    }

    const buf = [];
    while (i < lines.length && lines[i].trim() && (buf.length === 0 || !isBlockStart(lines[i], lines[i + 1]))) buf.push(lines[i++].trim());
    html.push(`<p>${renderInline(buf.join(' '))}</p>`);
  }
  return html.join('');
}

// ---- Extractos de texto --------------------------------------------------------

// plainText quita la sintaxis Markdown en línea y deja el texto legible.
export function plainText(text) {
  return String(text == null ? '' : text)
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/[`*_~]/g, '')
    .replace(/\s+/g, ' ')
    .trim();
}

// firstParagraph devuelve el primer párrafo de texto de un documento (sin metadatos,
// encabezados, listas, citas ni bloques de código), como texto plano y acotado.
export function firstParagraph(src, max = 280) {
  let lines = String(src == null ? '' : src).replace(/\r\n?/g, '\n').split('\n');
  if (lines[0] && lines[0].trim() === '---') {
    const end = lines.findIndex((l, i) => i > 0 && l.trim() === '---');
    if (end > 0) lines = lines.slice(end + 1);
  }
  const buf = [];
  let fenced = false;
  for (const line of lines) {
    if (FENCE.test(line)) { fenced = !fenced; continue; }
    if (fenced) continue;
    const t = line.trim();
    if (!t) {
      if (buf.length) break;
      continue;
    }
    if (HEADING.test(t) || HRULE.test(t) || ITEM.test(line) || QUOTE.test(t) || t.startsWith('|')) {
      if (buf.length) break;
      continue;
    }
    buf.push(t);
  }
  const text = plainText(buf.join(' '));
  return text.length > max ? `${text.slice(0, max - 1).trimEnd()}…` : text;
}
