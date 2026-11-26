# Delta para Hub Multiproyecto (multi-project-hub)

> **Cambio:** `autofetch-specs` (C-05-d). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> `RF-HUB-08` no cambia: el registro de un proyecto no pregunta por el autofetch. La tabla de requisitos y las referencias cruzadas de la capacidad se actualizan al integrar.

## ADDED Requirements

### Requirement: Comando `axiom autofetch` para Activar, Desactivar y Consultar la Preferencia (RF-HUB-13)

El sistema MUST exponer la preferencia `autofetch` de RF-HUB-07 mediante el verbo propio `axiom autofetch` con los subcomandos `enable`, `disable` y `status`, todos con `--cwd`, `--agent` y `--json` (`run` lo define `change-branch-lifecycle`, RF-BRA-39). `enable` MUST guardar `autofetch: true` en la entrada del proyecto del hub y MUST ser la única vía de encenderlo desde la CLI; `disable` MUST apagarlo y dejar la entrada sin la clave. `enable` y `disable` MUST ser idempotentes. `status` MUST informar del estado efectivo (`enabled` o `disabled`), sin red y sin escribir. Ninguno de los tres MUST modificar `axiom.yaml` ni ningún fichero versionado.

Encender el interruptor MUST ser la autorización explícita y persistente del `fetch` para ese proyecto. El registro de un proyecto (`axiom init`, `axiom project add` y el registro del dashboard, RF-HUB-08) MUST NOT preguntar por el autofetch ni encenderlo: un proyecto recién registrado queda con el interruptor apagado. `enable` sobre un proyecto no registrado MUST rechazarse con un mensaje en español que indique cómo registrarlo, sin escribir el hub; `disable` sobre un proyecto no registrado MUST no hacer nada, informarlo y terminar sin error. Con el hub ilegible, `enable` y `disable` MUST rechazarse sin modificarlo y `status` MUST informar de `disabled` con un aviso. Con un proyecto no registrado, `status` MUST informar de `disabled` con aviso.

#### Scenario: `enable` persiste la preferencia en el hub y no toca `axiom.yaml`

- **DADO** un proyecto registrado con el interruptor apagado
- **CUANDO** se ejecuta `axiom autofetch enable`
- **ENTONCES** `workspaces.json` contiene `autofetch: true` en la entrada de ese proyecto
- **Y** `axiom.yaml` y los demás ficheros versionados no se modifican
- **Y** `axiom autofetch status` informa de `enabled`

#### Scenario: `disable` apaga y retira la clave

- **DADO** un proyecto registrado con el interruptor encendido
- **CUANDO** se ejecuta `axiom autofetch disable`
- **ENTONCES** la entrada del proyecto en `workspaces.json` ya no contiene la clave `autofetch`
- **Y** `axiom autofetch status` informa de `disabled`

#### Scenario: `enable` y `disable` son idempotentes

- **DADO** un proyecto con el interruptor ya encendido
- **CUANDO** se ejecuta `axiom autofetch enable` por segunda vez
- **ENTONCES** `workspaces.json` queda byte a byte igual y la orden termina con código 0

#### Scenario: `status` no hace red ni escribe

- **DADO** un proyecto registrado y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom autofetch status --json`
- **ENTONCES** el observador no registra ninguna invocación de git remota
- **Y** el hub no se modifica

#### Scenario: Registrar un proyecto no pregunta ni enciende el autofetch

- **DADO** una terminal interactiva y un proyecto sin registrar
- **CUANDO** se ejecuta `axiom project add <ruta>`
- **ENTONCES** la CLI no formula ninguna pregunta sobre el autofetch
- **Y** el proyecto queda registrado con el interruptor apagado

#### Scenario: `enable` en un proyecto no registrado se rechaza

- **DADO** un proyecto que no figura en `workspaces.json`
- **CUANDO** se ejecuta `axiom autofetch enable`
- **ENTONCES** se rechaza con un mensaje en español que indica cómo registrar el proyecto
- **Y** `workspaces.json` no se crea ni se modifica

#### Scenario: `disable` en un proyecto no registrado no hace nada

- **DADO** un proyecto que no figura en `workspaces.json`
- **CUANDO** se ejecuta `axiom autofetch disable`
- **ENTONCES** la orden informa de que no hay nada que apagar y termina con código 0
- **Y** `workspaces.json` no se crea ni se modifica

#### Scenario: Hub ilegible equivale a apagado con aviso

- **DADO** un `workspaces.json` ilegible
- **CUANDO** se ejecuta `axiom autofetch status --json`
- **ENTONCES** el estado es `disabled` y la salida lleva un aviso sobre el hub ilegible
- **Y** `axiom autofetch enable` se rechaza sin modificar el fichero

## MODIFIED Requirements

### Requirement: Preferencias Personales de Worktree por Proyecto en el Hub (RF-HUB-07)

Además de los campos de RF-HUB-01, cada proyecto registrado en `~/.axiom/workspaces.json` PUEDE almacenar tres preferencias personales opcionales: el **modo de trabajo** (`worktree`, `in_place` o `ask`), la **raíz de worktrees** (ruta absoluta) y el **autofetch** (booleano de dos estados, apagado por defecto). Las preferencias son personales y NO DEBEN escribirse nunca en `axiom.yaml` ni en ningún fichero versionado, ni leerse de ellos. Sin preferencias guardadas para un proyecto, estén o no registrado, el modo efectivo DEBE ser `worktree`, la raíz efectiva DEBE ser `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt` y el autofetch efectivo DEBE ser apagado. Un modo distinto de los tres valores, o una raíz que no sea una ruta absoluta, DEBEN rechazarse con un mensaje en español, sin modificar el registro. El autofetch apagado DEBE representarse con la ausencia de la clave `autofetch` en la entrada; encendido, con `autofetch: true`. Un proyecto no registrado o un hub ilegible DEBEN dar autofetch apagado, sin hacer red por esa causa. Avisan `axiom autofetch status` y `axiom autofetch run` invocado a mano (sin `--agent`); `enable` rechaza y `disable` sobre un proyecto no registrado no hace nada y lo informa sin error (RF-HUB-13); el hook de sesión (`run --agent`) y `axiom branch ensure` se comportan exactamente como con el interruptor apagado, sin aviso, para no añadir ruido a cada arranque ni alterar la salida de `ensure`. Registrar de nuevo un proyecto ya registrado (RF-HUB-01, registro idempotente) NO DEBE borrar ni restablecer sus preferencias; desregistrarlo SÍ DEBE eliminarlas junto con su entrada. Las preferencias NO DEBEN cambiar la versión del esquema (`"1.0"`): un binario anterior las ignora. La edición del autofetch es el comando `axiom autofetch` de RF-HUB-13.
(Previously: solo existían las preferencias de modo de trabajo y raíz de worktrees; el autofetch no existía)

#### Scenario: Sin preferencias el modo efectivo es `worktree`
- **DADO** un proyecto registrado sin preferencias guardadas
- **CUANDO** se consulta su modo y su raíz de worktrees efectivos
- **ENTONCES** el modo es `worktree`
- **Y** la raíz es `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`

#### Scenario: Proyecto no registrado también recibe los valores por defecto
- **DADO** un proyecto que no está en `workspaces.json`
- **CUANDO** `axiom branch ensure` consulta su modo de trabajo
- **ENTONCES** el modo efectivo es `worktree`

#### Scenario: Persistencia de las preferencias
- **DADO** un proyecto registrado
- **CUANDO** se guarda el modo `ask` y la raíz `D:\worktrees\axiom`
- **ENTONCES** `workspaces.json` contiene ambas preferencias en la entrada de ese proyecto
- **Y** `axiom.yaml` no se modifica

#### Scenario: Modo inválido rechazado
- **DADO** un proyecto registrado
- **CUANDO** se intenta guardar el modo `siempre`
- **ENTONCES** se rechaza con un mensaje en español que lista los tres modos válidos
- **Y** el registro no se modifica

#### Scenario: Raíz relativa rechazada
- **DADO** un proyecto registrado
- **CUANDO** se intenta guardar la raíz `../wt`
- **ENTONCES** se rechaza indicando que la raíz debe ser una ruta absoluta

#### Scenario: Re-registrar conserva las preferencias
- **DADO** un proyecto registrado con el modo `in_place` y el autofetch encendido
- **CUANDO** se ejecuta de nuevo `Register(path, name, topology)` para ese proyecto
- **ENTONCES** el modo sigue siendo `in_place`
- **Y** el autofetch sigue encendido

#### Scenario: Desregistrar elimina las preferencias
- **DADO** un proyecto registrado con preferencias, incluido el autofetch encendido
- **CUANDO** se ejecuta `Unregister`
- **ENTONCES** la entrada y sus preferencias desaparecen de `workspaces.json`
- **Y** el autofetch efectivo del proyecto vuelve a ser apagado

#### Scenario: Un binario anterior ignora las preferencias
- **DADO** un `workspaces.json` con entradas que incluyen las preferencias, también `autofetch: true`
- **CUANDO** lo carga un binario anterior a esta capacidad
- **ENTONCES** carga el registro sin error y conserva el resto de campos

#### Scenario: El autofetch está apagado por defecto y la clave está ausente
- **DADO** un proyecto recién registrado
- **CUANDO** se consulta el autofetch efectivo y se lee su entrada de `workspaces.json`
- **ENTONCES** el autofetch es apagado
- **Y** la entrada no contiene la clave `autofetch`

#### Scenario: El autofetch nunca se lee de `axiom.yaml`
- **DADO** un proyecto no registrado cuyo `axiom.yaml` contiene una clave `autofetch: true`
- **CUANDO** se consulta el autofetch efectivo
- **ENTONCES** el autofetch es apagado y la consulta incluye un aviso de proyecto no registrado

#### Scenario: Proyecto no registrado o hub ilegible dan apagado con aviso
- **DADO** un proyecto que no figura en el hub, o un `workspaces.json` ilegible
- **CUANDO** se ejecutan `axiom autofetch status` y `axiom autofetch run` a mano
- **ENTONCES** el autofetch es apagado y ambas órdenes emiten un aviso
- **Y** ninguna realiza operación de red por esa consulta

#### Scenario: Hook de sesión y `ensure` no avisan con el hub no resoluble
- **DADO** un proyecto que no figura en el hub, o un `workspaces.json` ilegible
- **CUANDO** se ejecutan `axiom autofetch run --agent claude-code` y `axiom branch ensure --change x --json`
- **ENTONCES** se comportan como con el interruptor apagado: sin red por esta causa, sin aviso y con la salida de `ensure` sin cambios
