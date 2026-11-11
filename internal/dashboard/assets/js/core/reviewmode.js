// Lógica pura de la preferencia de revisión (RDD) del proyecto abierto: textos en
// castellano del modo efectivo y de qué fuente lo decide. Sin DOM, para probarla con Node.

// SOURCE_LABEL nombra la fuente que decide, tal como la devuelve /api/project/review-mode.
const SOURCE_LABEL = {
  clone_local: 'el «apagar» de este clon',
  project: 'tu preferencia para este proyecto',
  global: 'tu interruptor global',
  default: 'el valor por defecto (nadie lo ha activado)'
};

// sourceLabel devuelve el nombre de la fuente; una desconocida se muestra tal cual.
export function sourceLabel(source) {
  return SOURCE_LABEL[source] || String(source || SOURCE_LABEL.default);
}

// describeReviewMode resume el modo efectivo y quién lo decide.
export function describeReviewMode(dto) {
  if (!dto) return '';
  const state = dto.effective === 'on' ? 'encendida' : 'apagada';
  return `La revisión está ${state} en este proyecto: decide ${sourceLabel(dto.source)}.`;
}

// selectValue es lo que muestra el selector: la preferencia guardada para el proyecto o
// «inherit» si no hay ninguna.
export function selectValue(dto) {
  return dto?.project === 'on' || dto?.project === 'off' ? dto.project : 'inherit';
}

// overrideNote avisa de que el «apagar» de este clon gana a la preferencia del proyecto,
// o devuelve '' si no pasa.
export function overrideNote(dto) {
  if (dto?.clone_local === 'off' && (dto.project === 'on')) {
    return 'Este clon tiene la revisión apagada y eso gana a tu preferencia de proyecto. Se quita con «axiom review mode enable --scope clone».';
  }
  return '';
}
