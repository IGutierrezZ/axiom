// Carga diferida de una pantalla: la primera entrada dispara la carga y, a partir
// de ahí, «Actualizar» solo la repite si la pantalla ya se cargó alguna vez.
// Evita pedir al arrancar datos caros que quizá no se miren (p. ej. los símbolos de Código).

export function lazyLoader(load) {
  let started = false;
  return {
    enter() {
      if (started) return Promise.resolve();
      started = true;
      return load();
    },
    refresh() {
      return started ? load() : Promise.resolve();
    },
    get started() {
      return started;
    }
  };
}
