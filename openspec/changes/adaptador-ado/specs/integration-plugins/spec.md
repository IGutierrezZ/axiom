# Delta para integration-plugins

## ADDED Requirements

### Requirement: Puertos Opcionales de Identidad y Descubrimiento (REQ-45.17)

El paquete `internal/integrations` DEBE definir dos puertos opcionales que un plugin implementa solo si los ofrece: la resolución de la identidad de la persona y el descubrimiento del proceso del tracker. Se DEBEN detectar por aserción de tipo sobre el `Plugin` registrado. El puerto `Plugin` (`Manifest`, `Deliver`) NO DEBE cambiar. Los verbos genéricos `axiom integrations connect` y `axiom integrations discover` DEBEN despachar por estos puertos. Cuando el plugin indicado con `--plugin` no ofrece el puerto correspondiente, el verbo DEBE terminar con un error tipado que nombra el plugin y el puerto que falta, sin efectos secundarios.

#### Scenario: Plugin que ofrece ambos puertos
- **DADO** un plugin de prueba que implementa los puertos de identidad y de descubrimiento
- **CUANDO** el verbo `connect` y el verbo `discover` lo invocan
- **ENTONCES** cada verbo detecta su puerto por aserción de tipo y lo invoca

#### Scenario: Plugin sin puerto de identidad
- **DADO** un plugin de prueba que solo implementa `Plugin`
- **CUANDO** se ejecuta `axiom integrations connect --plugin <doble>`
- **ENTONCES** termina con código distinto de cero y un error tipado que nombra el plugin y el puerto de identidad
- **Y** no se crea ningún fichero

#### Scenario: Plugin sin puerto de descubrimiento
- **DADO** un plugin de prueba que solo implementa `Plugin`
- **CUANDO** se ejecuta `axiom integrations discover --plugin <doble>`
- **ENTONCES** termina con código distinto de cero y un error tipado que nombra el plugin y el puerto de descubrimiento

#### Scenario: El puerto `Plugin` no cambia
- **DADO** un plugin que implementa solo `Manifest` y `Deliver`
- **CUANDO** se registra con `NewRegistry`
- **ENTONCES** el registro lo acepta sin exigir ningún puerto opcional

## MODIFIED Requirements

### Requirement: Registro Cerrado en Compilación (REQ-43.4)

`NewRegistry(plugins...)` DEBE construir un registro de plugins y rechazar con error tipado un identificador duplicado, un manifiesto inválido, un plugin cuyo identificador no está en el catálogo de manifiestos y un plugin cuyo manifiesto difiere de la entrada del catálogo (el código del plugin y el manifiesto publicado no pueden divergir). `DefaultRegistry()` DEBE devolver un registro que contiene exactamente el plugin `azure-devops` (el primer adaptador real, C-07-c), y el catálogo de manifiestos por defecto DEBE contener exactamente `azure-devops`. El catálogo de manifiestos que usa `workspace` para validar `integrations.<plugin>` DEBE derivarse de la misma lista cerrada y DEBE ofrecer una costura de paquete que permita a las pruebas registrar un plugin de prueba. La lista de plugins compilados DEBE ensamblarse sin que `internal/integrations` importe el paquete del adaptador (sin ciclo de importaciones), y una prueba de importaciones DEBE protegerlo. No existe carga dinámica de código.
(Previously: `DefaultRegistry()` devolvía un registro vacío y el catálogo por defecto no admitía ningún identificador; el primer adaptador real era futuro)

#### Scenario: Identificador duplicado
- **DADO** dos plugins con el mismo identificador
- **CUANDO** se llama a `NewRegistry` con ambos
- **ENTONCES** devuelve un error tipado de duplicado y ningún registro

#### Scenario: Plugin ausente del catálogo o distinto de él
- **DADO** un plugin válido cuyo manifiesto no está en el catálogo, o está con contenido distinto
- **CUANDO** se llama a `NewRegistry`
- **ENTONCES** devuelve un error tipado y ningún registro

#### Scenario: Registro por defecto con `azure-devops`
- **CUANDO** se llama a `DefaultRegistry()`
- **ENTONCES** el registro contiene exactamente el plugin `azure-devops` y ningún otro
- **Y** el catálogo de manifiestos por defecto admite exactamente el identificador `azure-devops` bajo `integrations:`

#### Scenario: Identificador fuera del catálogo por defecto
- **DADO** un `axiom.yaml` con `integrations.jira`
- **CUANDO** se carga el workspace con el catálogo por defecto
- **ENTONCES** la carga falla con un error tipado que nombra `azure-devops` como único plugin admitido

#### Scenario: Costura de prueba
- **DADO** una prueba que registra el doble en memoria mediante la costura del catálogo
- **CUANDO** se valida `integrations.<doble>` en un `axiom.yaml`
- **ENTONCES** el bloque se acepta
- **Y** al terminar la prueba el catálogo vuelve a su estado por defecto, que contiene `azure-devops`

#### Scenario: Sin ciclo de importaciones
- **DADO** los paquetes `internal/integrations` e `internal/integrations/azuredevops`
- **CUANDO** se comprueban las importaciones de `internal/integrations`, directas y transitivas
- **ENTONCES** no incluyen `internal/integrations/azuredevops`

### Requirement: Alta del Verbo `integrations` en la CLI y la Ayuda (REQ-43.18)

La CLI DEBE exponer la familia de primer nivel `axiom integrations` con exactamente los subcomandos `sync`, `emit`, `secret` (este con `set`, `status` y `delete`), `connect` y `discover`. La ayuda global y la ayuda de cada subcomando DEBEN describirlos y las pruebas de ayuda DEBEN cubrirlos. Todo subcomando acepta `--cwd`; los que operan sobre un cambio, `--change`.

(Previously: la familia tenía exactamente los subcomandos `sync`, `emit` y `secret`; `connect` y `discover` los añaden REQ-45.12 y REQ-45.13)

#### Scenario: Ayuda global
- **CUANDO** se ejecuta `axiom --help`
- **ENTONCES** el texto lista `integrations` con `sync`, `emit`, `secret`, `connect` y `discover`

#### Scenario: Subcomando desconocido
- **CUANDO** se ejecuta `axiom integrations inspect`
- **ENTONCES** termina con código distinto de cero y muestra la ayuda de la familia, que lista `sync`, `emit`, `secret`, `connect` y `discover`
