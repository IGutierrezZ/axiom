// Visor de especificaciones vivas: convierte el Markdown de openspec/specs/<dominio>/spec.md
// en bloques estructurados.
//
// - «### Requirement: Título (REQ-x)» abre un requisito; «#### Scenario: Título» abre un
//   escenario dentro de él.
// - En un escenario, las líneas «- **DADO** …», «- **CUANDO** …», «- **ENTONCES** …» (y su
//   forma inglesa GIVEN/WHEN/THEN, más Y/AND/PERO/BUT) se pintan como pasos estructurados.
// - Todo lo demás se pinta con markdown.js, que lo escapa todo. Aquí también se escapa
//   cualquier texto de la API antes de añadir etiquetas.
// - DEBE / MUST (y sus negaciones) se resaltan solo en texto, nunca dentro de código ni enlaces.

import { escapeHtml } from './api.js';
import { renderInline, renderMarkdown } from './markdown.js';

const REQ_HEAD = /^###\s+(?:Requirement|Requisito)\s*:\s*(.+?)\s*#*\s*$/i;
const SCN_HEAD = /^####\s+(?:Scenario|Escenario)\s*:\s*(.+?)\s*#*\s*$/i;
const ANY_HEAD = /^(#{1,6})\s+\S/;
const FENCE = /^\s*(```|~~~)/;
const HRULE = /^\s*([-*_])(\s*\1){2,}\s*$/;
const KEYWORDS = 'DADO|CUANDO|ENTONCES|Y|PERO|GIVEN|WHEN|THEN|AND|BUT';
// Con negrita se admite cualquier mayúscula/minúscula; sin negrita solo las tres palabras
// clave principales y en mayúsculas (una «Y» suelta al inicio de una frase no es un paso).
const STEP_BOLD = new RegExp(`^\\s*[-*+]\\s+(?:\\*\\*|__)\\s*(${KEYWORDS})\\s*:?\\s*(?:\\*\\*|__)\\s*:?\\s*(.*)$`, 'i');
const STEP_PLAIN = /^\s*[-*+]\s+(DADO|CUANDO|ENTONCES|GIVEN|WHEN|THEN)(?::|\s)\s*(.*)$/;
const REQ_ID = /^(.*?)\s*\(((?:REQ|REQUISITO)[^)]*)\)\s*$/i;
const MODAL = /\b(NO DEBEN?|DEBEN?|MUST NOT|MUST|SHALL NOT|SHALL)\b/g;

const KIND = { DADO: 'given', GIVEN: 'given', CUANDO: 'when', WHEN: 'when', ENTONCES: 'then', THEN: 'then', Y: 'and', AND: 'and', PERO: 'and', BUT: 'and' };

// stepOf devuelve {kw, kind, text} si la línea es un paso Given/When/Then; si no, null.
function stepOf(line) {
  const m = STEP_BOLD.exec(line) || STEP_PLAIN.exec(line);
  if (!m) return null;
  const kw = m[1].toUpperCase();
  return { kw, kind: KIND[kw], text: m[2].trim() };
}

// parseSpec devuelve una lista de bloques:
//   {type:'md', text}
//   {type:'requirement', id, title, body, scenarios:[{title, steps:[{kw,kind,text}], extra}]}
// No interpreta ni escapa nada: eso lo hace renderSpec al pintar.
export function parseSpec(src) {
  const lines = String(src == null ? '' : src).replace(/\r\n?/g, '\n').split('\n');
  const blocks = [];
  let md = [];
  let req = null;
  let scn = null;
  let fence = null;

  const flushMd = () => {
    const text = md.join('\n');
    if (text.trim()) blocks.push({ type: 'md', text });
    md = [];
  };
  const closeReq = () => {
    if (!req) return;
    req.body = req.bodyLines.join('\n');
    delete req.bodyLines;
    req.scenarios.forEach(s => { s.extra = s.extraLines.join('\n'); delete s.extraLines; });
    blocks.push(req);
    req = null;
    scn = null;
  };
  const openReq = (title) => {
    flushMd();
    closeReq();
    const m = REQ_ID.exec(title);
    req = { type: 'requirement', id: m ? m[2].trim() : '', title: (m ? m[1] : title).trim(), bodyLines: [], scenarios: [] };
  };

  for (const line of lines) {
    // Dentro de un bloque de código no se interpreta nada.
    const f = FENCE.exec(line);
    if (fence) {
      (scn ? scn.extraLines : req ? req.bodyLines : md).push(line);
      if (f && line.trim().startsWith(fence)) fence = null;
      continue;
    }
    if (f) {
      fence = f[1];
      (scn ? scn.extraLines : req ? req.bodyLines : md).push(line);
      continue;
    }

    const reqHead = REQ_HEAD.exec(line);
    if (reqHead) { openReq(reqHead[1]); continue; }

    const scnHead = SCN_HEAD.exec(line);
    if (scnHead) {
      if (!req) openReq('');
      scn = { title: scnHead[1].trim(), steps: [], extraLines: [] };
      req.scenarios.push(scn);
      continue;
    }

    const head = ANY_HEAD.exec(line);
    if (head && head[1].length <= 3) {
      // Otro encabezado de nivel superior cierra el requisito.
      flushMd();
      closeReq();
      md.push(line);
      continue;
    }
    if (req && HRULE.test(line)) { closeReq(); continue; }

    if (scn) {
      const step = stepOf(line);
      if (step) { scn.steps.push(step); continue; }
      if (!line.trim()) continue;
      // Una línea con sangría que continúa el paso anterior.
      if (/^\s{2,}\S/.test(line) && scn.steps.length && !scn.extraLines.length && !/^\s*[-*+]\s/.test(line)) {
        scn.steps[scn.steps.length - 1].text += ` ${line.trim()}`;
        continue;
      }
      scn.extraLines.push(line);
      continue;
    }
    if (req) { req.bodyLines.push(line); continue; }
    md.push(line);
  }
  flushMd();
  closeReq();
  return blocks;
}

// emphasizeModals resalta DEBE / MUST en las partes de texto del HTML ya escapado. Se
// salta lo que ya va en negrita, en código o en un enlace, y nunca toca atributos.
export function emphasizeModals(html) {
  let skip = 0;
  // Un «**DEBE**» ya en negrita recibe el mismo resalte que el texto suelto.
  const bolded = String(html).replace(/<strong>(NO DEBEN?|DEBEN?|MUST NOT|MUST|SHALL NOT|SHALL)<\/strong>/g, '<strong class="spec-must">$1</strong>');
  return bolded.split(/(<[^>]+>)/).map((part) => {
    if (part.startsWith('<')) {
      const m = /^<(\/?)(code|pre|strong|a)\b/i.exec(part);
      if (m) skip = Math.max(0, skip + (m[1] ? -1 : 1));
      return part;
    }
    return skip > 0 ? part : part.replace(MODAL, '<strong class="spec-must">$1</strong>');
  }).join('');
}

function mdHtml(text) {
  return text.trim() ? `<div class="md">${emphasizeModals(renderMarkdown(text))}</div>` : '';
}

function renderScenario(s) {
  const steps = s.steps.map(st => `<li class="spec-step spec-step-${st.kind}"><span class="spec-kw mono">${escapeHtml(st.kw)}</span><span class="spec-step-text">${emphasizeModals(renderInline(st.text))}</span></li>`).join('');
  const label = s.title ? `Escenario: ${s.title}` : 'Escenario';
  return `<div class="spec-scn" role="group" aria-label="${escapeHtml(label)}">
    <p class="spec-scn-title"><span class="spec-scn-label">Escenario</span>${s.title ? `<span>${renderInline(s.title)}</span>` : ''}</p>
    ${steps ? `<ol class="spec-steps">${steps}</ol>` : ''}
    ${mdHtml(s.extra)}
  </div>`;
}

// renderSpec devuelve HTML seguro para el visor.
export function renderSpec(src) {
  return parseSpec(src).map((b) => {
    if (b.type === 'md') return mdHtml(b.text);
    const head = b.title || b.id
      ? `<header class="spec-req-head">${b.id ? `<span class="spec-req-id mono">${escapeHtml(b.id)}</span>` : ''}${b.title ? `<h3 class="spec-req-title">${renderInline(b.title)}</h3>` : ''}</header>`
      : '';
    return `<article class="spec-req">${head}${mdHtml(b.body)}${b.scenarios.map(renderScenario).join('')}</article>`;
  }).join('');
}

// specStats cuenta requisitos y escenarios de un documento (para comprobaciones y tests).
export function specStats(src) {
  const reqs = parseSpec(src).filter(b => b.type === 'requirement');
  return { requirements: reqs.length, scenarios: reqs.reduce((n, r) => n + r.scenarios.length, 0) };
}
