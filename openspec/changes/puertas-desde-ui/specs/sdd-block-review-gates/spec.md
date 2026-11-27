# Delta para sdd-block-review-gates

## ADDED Requirements

### Requirement: El Dashboard Local es un Canal Humano Equivalente a `axiom sdd gate record` (prov:GAT:1)

El dashboard local DEBE poder registrar la decisión de una puerta como canal humano equivalente a `axiom sdd gate record`, usando el mismo núcleo de registro: misma validación de argumentos y de `--commit-review`, misma guarda de rama (contexto `archive` para `integration`), mismas reglas de cesión y de evidencia, mismo digest y escritura bajo cerrojo. Un registro hecho desde el dashboard DEBE ser indistinguible en `gates.yaml` del hecho con la CLI con los mismos datos, DEBE emitir el evento de integración de la aprobación una sola vez y NO DEBE alterar ni el contrato `gate-record/v1` ni RF-GAT-14. La doctrina de los agentes NO cambia: siguen registrando solo con `axiom sdd gate record`.

#### Scenario: Registros indistinguibles

- **DADO** dos copias del mismo cambio con la misma puerta pendiente
- **CUANDO** una se decide con la CLI y otra desde el dashboard con los mismos datos
- **ENTONCES** los registros de `gates.yaml` son idénticos salvo la marca de tiempo

#### Scenario: Evento de integración único

- **DADO** una aprobación de `spec` desde el dashboard
- **CUANDO** se registra
- **ENTONCES** se despacha un único evento de integración

#### Scenario: Misma guarda de rama

- **DADO** una rama distinta de la autorizada para el cambio
- **CUANDO** el dashboard intenta registrar
- **ENTONCES** se rechaza con la misma negativa que la CLI y no se escribe

#### Scenario: Contrato de la CLI intacto

- **DADO** la salida de `axiom sdd gate record` antes y después del cambio
- **CUANDO** se compara con los mismos datos
- **ENTONCES** es idéntica
