# Delta para `workspace-topology`

## MODIFIED Requirements

### Requirement: Catálogo Único de Claves de `axiom.yaml` Derivado del Esquema, Filtrado por Topología e ISO (RF-TOP-21)

El sistema DEBE ofrecer un catálogo de las claves de `axiom.yaml` que se pueden consultar o editar, como única fuente para cualquier pantalla o cliente. Cada entrada DEBE declarar su ruta, etiqueta, descripción, tipo, valores permitidos (si el vocabulario es cerrado), si es editable, el efecto descrito de editarla y las topologías en las que aplica. Los valores permitidos de los vocabularios cerrados (`git.promotion` y `repositories[].promotion`, `workflow.lane`, `sdd.explainer_html`, `roles[].gate_policy`) DEBEN derivarse de las constantes de su paquete dueño (las del esquema de `workspace` y, para `roles[].gate_policy`, las de `multirole`), expuestas como funciones que usan a la vez el validador y el catálogo; el catálogo NO DEBE duplicar esas listas. Una clave nueva de un vocabulario cerrado DEBE aparecer en el catálogo sin editar el catálogo.

Claves editables: `workspace.name`, `git.integration_branch`, `git.branch_prefix`, `git.main_branch`, `git.promotion`, `git.branch_template`, `workflow.lane`, `delivery.split_hint_lines`, `sdd.iso`, `sdd.id_pattern`, `sdd.explainer_html`; y, solo en `multirepo`, `repositories[].integration_branch`, `repositories[].branch_prefix`, `repositories[].promotion`, `repositories[].branch_template`, `acceptance_tests.enabled`, `acceptance_tests.environments`, `roles[].gate_policy` (con `blocking`, `deferred` y `optional`) y `roles[].branch_template`. Las tres claves `branch_template` son de tipo cadena, sin vocabulario cerrado, y una escritura con un valor que incumple RF-TOP-24 (por ejemplo, por `PUT /api/settings`) DEBE rechazarse nombrando la clave, sin modificar `axiom.yaml`. `sdd.iso` y `sdd.id_pattern` DEBEN llevar un efecto descrito explícito.

Claves de integración editables: `integrations[<plugin>].enabled` y `integrations[<plugin>].<clave>` por cada campo de configuración (`ConfigField`) que declara el manifiesto del plugin. El catálogo DEBE derivar de ese manifiesto el tipo, si es obligatoria, los valores permitidos (enumeración), el formato y los mínimos y máximos de cada clave, sin repetir literales del plugin, de modo que un campo nuevo del manifiesto aparezca sin editar el catálogo. Una prueba de coherencia DEBE fallar si el catálogo declara un valor o un límite que el decodificador estricto de `integrations` rechaza, o al revés. El PAT y cualquier secreto NO DEBEN ser claves del catálogo; el catálogo solo depende del manifiesto del plugin, no de su adaptador.

(Previously: el catálogo excluía `integrations.*`.)

Claves de solo lectura (con `editable: false`): `technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology` y `workspace.specs_repository`. `artifact_store` NO DEBE estar en el catálogo como editable.

El catálogo NO DEBE contener claves de `integrations` que el manifiesto del plugin no declare, ni controles de RDD, ni `commit_review` ni `docs_publication` (no son claves de `axiom.yaml`). Las claves documentadas al registrarlas, `sdd.explainer_html`, `acceptance_tests.*` y `delivery.split_hint_lines`, DEBEN estar registradas con su tipo y su vocabulario.

El catálogo para un proyecto concreto DEBE filtrarse: en `monorepo-embedded` NO DEBE incluir `roles`, `gate_policy`, `repositories` ni `acceptance_tests`; en un proyecto ISO (`sdd.iso: true`) los valores permitidos de `workflow.lane` NO DEBEN incluir `odd`.

#### Scenario: Los valores permitidos salen de las constantes del esquema
- **DADO** el catálogo construido
- **CUANDO** se leen los valores permitidos de `git.promotion`, `workflow.lane`, `sdd.explainer_html` y `roles[].gate_policy`
- **ENTONCES** son exactamente los de las constantes de su paquete dueño (`branch|release|ask`, `odd|sdd|ask`, `ask|always|never`, `blocking|deferred|optional`)
- **Y** una prueba de coherencia falla si el catálogo declara un valor que el decodificador rechaza o al revés

#### Scenario: Claves por topología en monorepo
- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se consulta el catálogo
- **ENTONCES** incluye `workspace.name`, `git.*` (con `git.branch_template`), `workflow.lane`, `delivery.split_hint_lines` y `sdd.*`
- **Y** no incluye `roles`, `gate_policy`, `repositories` (ni `repositories[].branch_template`), `roles[].branch_template` ni `acceptance_tests`

#### Scenario: Claves por topología en multirepo
- **DADO** un proyecto `multirepo`
- **CUANDO** se consulta el catálogo
- **ENTONCES** incluye `repositories[].{integration_branch,branch_prefix,promotion,branch_template}`, `acceptance_tests.{enabled,environments}`, `roles[].gate_policy` con `optional` entre sus valores y `roles[].branch_template`

#### Scenario: Claves de solo lectura
- **DADO** el catálogo de cualquier topología
- **CUANDO** se leen `technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology` y `workspace.specs_repository`
- **ENTONCES** cada una figura con `editable: false`
- **Y** una escritura que las incluya se rechaza como clave no editable

#### Scenario: Exclusiones del catálogo
- **DADO** el catálogo de cualquier proyecto
- **CUANDO** se buscan `artifact_store`, `rdd`, `commit_review` y `docs_publication`, y cualquier clave de PAT o de secreto
- **ENTONCES** ninguna aparece como entrada editable

#### Scenario: Claves registradas de cambios anteriores
- **DADO** el catálogo
- **CUANDO** se consultan `sdd.explainer_html`, `acceptance_tests.enabled`, `acceptance_tests.environments` y `delivery.split_hint_lines`
- **ENTONCES** cada una tiene tipo, descripción, efecto y, la primera, su vocabulario `ask|always|never`; `split_hint_lines` es entero mayor que cero

#### Scenario: ISO no ofrece `odd`
- **DADO** un proyecto con `sdd.iso: true`
- **CUANDO** se consulta el catálogo
- **ENTONCES** los valores permitidos de `workflow.lane` son `sdd` y `ask`



#### Scenario: El catálogo incluye las claves de integración derivadas del manifiesto
- **DADO** el plugin `azuredevops` con su manifiesto
- **CUANDO** se consulta el catálogo
- **ENTONCES** incluye `integrations[azuredevops].enabled` y una entrada `integrations[azuredevops].<clave>` por cada campo del manifiesto, con tipo, obligatoriedad, vocabulario, formato y límites tomados de él
- **Y** ninguna entrada es un PAT ni un secreto

#### Scenario: Un campo nuevo del manifiesto aparece sin editar el catálogo
- **DADO** un manifiesto de prueba con un campo de enumeración añadido
- **CUANDO** se construye el catálogo
- **ENTONCES** la clave aparece con exactamente los valores del manifiesto

#### Scenario: Una prueba de coherencia falla ante una divergencia con el decodificador
- **DADO** un valor o un límite del catálogo que el decodificador estricto de `integrations` rechaza, o al revés
- **CUANDO** se ejecuta la prueba de coherencia
- **ENTONCES** la prueba falla

#### Scenario: Las tres claves `branch_template` figuran en el catálogo
- **DADO** el catálogo de un proyecto `multirepo`
- **CUANDO** se consultan `git.branch_template`, `repositories[].branch_template` y `roles[].branch_template`
- **ENTONCES** cada una es editable, de tipo cadena, con descripción y efecto descritos, sin vocabulario cerrado, y aplica a `multirepo` (la de `git:` también a `monorepo-embedded`)

#### Scenario: Una plantilla inválida enviada como ajuste se rechaza
- **DADO** el dashboard y `PUT /api/settings` con `git.branch_template: "{prefix}/{ticket}"`
- **CUANDO** se procesa la escritura
- **ENTONCES** se rechaza nombrando `git.branch_template` y la variable desconocida
- **Y** `axiom.yaml` no se modifica

#### Scenario: Una plantilla inválida de repositorio o de rol se rechaza con la ruta del elemento
- **DADO** el dashboard de un `multirepo` y `PUT /api/settings` con `repositories[].branch_template: "feature/{ref"` para `repo-backend`, y por separado `roles[].branch_template: "{y"` para `backend`
- **CUANDO** se procesa cada escritura
- **ENTONCES** cada una se rechaza nombrando su clave y la ruta del elemento (`repo-backend` o el rol `backend`)
- **Y** `axiom.yaml` no se modifica

#### Scenario: Una plantilla válida de repositorio o de rol enviada como ajuste se escribe
- **DADO** el dashboard de un `multirepo` y `PUT /api/settings` con `repositories[].branch_template: "feature/{ref}-{slug}"` para `repo-backend` y `roles[].branch_template: "feature/TASK-{task_ref}"` para `backend`
- **CUANDO** se procesa la escritura
- **ENTONCES** `axiom.yaml` contiene ambas plantillas en la entrada y en el rol indicados y vuelve a cargarse como `COMPLIANT`

#### Scenario: Una plantilla con `{role}` en la rama con `openspec/` se rechaza por la API
- **DADO** el dashboard de un `monorepo-embedded` y `PUT /api/settings` con `git.branch_template: "{prefix}/{role}/{change}"`
- **CUANDO** se procesa la escritura
- **ENTONCES** se rechaza nombrando `git.branch_template` y la rama con `openspec/`
- **Y** `axiom.yaml` no se modifica

#### Scenario: Una plantilla de rol enviada en `monorepo-embedded` no está disponible
- **DADO** el dashboard de un `monorepo-embedded` y `PUT /api/settings` con `roles[].branch_template`
- **CUANDO** se procesa la escritura
- **ENTONCES** se rechaza porque la clave no aplica a esa topología (no figura en su catálogo)
- **Y** `axiom.yaml` no se modifica

#### Scenario: Una clave de integración enviada como ajuste se valida con el decodificador estricto
- **DADO** el dashboard y `PUT /api/settings` con una clave de `integrations[azuredevops]` de valor inválido para el manifiesto
- **CUANDO** se procesa la escritura
- **ENTONCES** se rechaza nombrando la clave y `axiom.yaml` no se modifica

#### Scenario: Una plantilla válida enviada como ajuste se escribe
- **DADO** el dashboard y `PUT /api/settings` con `git.branch_template: "{prefix}/{ref}-{slug}"`
- **CUANDO** se procesa la escritura
- **ENTONCES** `axiom.yaml` contiene `git.branch_template: "{prefix}/{ref}-{slug}"` y vuelve a cargarse como `COMPLIANT`

### Requirement: Escritor de Round-Trip de `axiom.yaml` que Conserva Comentarios y Orden y Revalida el Documento Completo (RF-TOP-22)

El sistema DEBE ofrecer un escritor que aplique un parche de claves del catálogo sobre `axiom.yaml` operando sobre el árbol YAML, de modo que el resto del documento conserve sus comentarios (de cabecera, de línea y de pie), el orden de las claves y los bloques que no se tocan. El escritor DEBE aplicar solo claves editables del catálogo y rechazar cualquier otra con un error tipado. Esto incluye el bloque `integrations`: el escritor edita `integrations.<plugin>` sobre el mismo árbol, con las mismas garantías que los demás bloques, y NO existe un escritor aparte para él.

(Previously: el escritor no editaba el bloque `integrations`.)

El parche es un mapa de claves del catálogo a valores, con `nil` para retirar. Las claves por elemento se direccionan como `repositories[<path>].<campo>` (el selector se compara con la ruta normalizada, `\` a `/` y `path.Clean`), `roles[<id>].<campo>` e `integrations[<plugin>].<clave>` (el identificador del plugin puede contener guiones y se interpreta íntegro). Todos los cambios del parche se aplican sobre el mismo árbol y se validan juntos, como un lote atómico de todo o nada: si uno falla, no se escribe ninguno. La comprobación de tipo y de vocabulario de cada clave se hace contra el vocabulario sin filtrar por ISO; la contradicción ISO la decide la validación del documento resultante, de modo que desactivar ISO y elegir `odd` en el mismo lote es válido.

Retirar la última clave de `integrations.<plugin>` borra la entrada del plugin y, al quedar vacío, el bloque `integrations`. Un valor `null` para una clave DEBE retirarla del fichero (el motor vuelve a su valor por defecto); si un bloque queda vacío, DEBE borrarse también. Al retirar el último campo de ajuste de una entrada de `repositories`, la entrada se borra, y al quedar vacía la secuencia, el bloque. `workspace.name` es obligatoria y no admite `null`. El escritor DEBE rechazar editar un documento que contenga anclas, alias o claves de mezcla `<<`, más de un documento YAML (reescribirlo borraría los siguientes) o una clave repetida en el mapeo que se edita, con un error tipado en castellano que nombra la construcción y su línea, sin escribir.

El round-trip DEBE conservar el formato del fichero, incluidas las líneas en blanco entre bloques, los finales de línea CRLF y el BOM UTF-8 si el original los tenía: un parche sin efecto NO DEBE alterar el fichero ni un solo byte, y uno con efecto solo cambia las líneas afectadas.

Antes de escribir, el escritor DEBE revalidar el documento completo ya serializado con los mismos decodificadores estrictos y la misma validación del motor (`workspace.ParseConfig`, que incluye `readDeclarationBlocks`, `decodeGit`, `decodeWorkflow` y `checkISOWorkflow`), y las comprobaciones del motor contra el disco (`Validate`, `ValidateDeclarationPaths` y `ValidateRepositoryPolicies`). Un parche válido por bloque que resulte inválido en conjunto DEBE rechazarse. Cualquier error de `ParseConfig` bloquea. De las comprobaciones contra el disco solo bloquean los errores que INTRODUCE el parche, es decir, los presentes en el documento nuevo y ausentes en el original; los errores que el fichero ya tenía en disco se devuelven como avisos (`warnings`) y no impiden guardar. Si la validación bloquea, el fichero NO DEBE modificarse y el error del validador DEBE llegar al llamante en castellano. En particular, `workflow.lane: odd` con `sdd.iso: true` (resultante del parche o ya presente) DEBE rechazarse con la contradicción ISO.

Los errores del escritor DEBEN ser tipados, con los códigos `invalid_request`, `config_missing`, `revision_mismatch`, `unsupported_document`, `unknown_key`, `key_not_editable`, `key_not_applicable`, `invalid_value`, `validation_failed` y `write_failed`, y mensaje en castellano; `validation_failed` lleva una violación por línea.

La escritura DEBE ser atómica: fichero temporal en el mismo directorio y `rename`; un fallo no deja un fichero a medias ni un temporal. El fallo de E/S es `write_failed` y su mensaje DEBE indicar si el reemplazo llegó a producirse. La ruta del fichero DEBE obtenerse de `workspace.Locate(...).ConfigPath`, de modo que en `multirepo` se escriba el `axiom.yaml` del repositorio de specs apuntado y nunca uno inventado en la raíz del código. Lo escrito queda sin confirmar en el árbol de trabajo del repositorio que contiene el fichero: el escritor no ejecuta git.

#### Scenario: El round-trip conserva comentarios y orden (golden)
- **DADO** un `axiom.yaml` comentado de referencia con comentarios de cabecera, de línea y de pie, y bloques en un orden concreto
- **CUANDO** se cambia `workflow.lane` de `ask` a `sdd`
- **ENTONCES** el fichero resultante coincide con el golden: solo cambia esa línea
- **Y** los comentarios, el orden, el sangrado y los bloques ajenos permanecen idénticos

#### Scenario: `null` borra la clave
- **DADO** un `axiom.yaml` con `git.main_branch: trunk` y otras claves en `git`
- **CUANDO** se aplica `git.main_branch: null`
- **ENTONCES** la clave desaparece y las demás claves de `git` se conservan
- **Y** el motor reporta su valor por defecto

#### Scenario: Un bloque que queda vacío se borra
- **DADO** un `axiom.yaml` con `delivery:` que solo contiene `split_hint_lines`
- **CUANDO** se aplica `delivery.split_hint_lines: null`
- **ENTONCES** el bloque `delivery:` desaparece por completo del fichero

#### Scenario: Se rechazan anclas, alias y claves de mezcla
- **DADO** un `axiom.yaml` que usa un ancla `&base`, un alias `*base` o una clave `<<`
- **CUANDO** se intenta aplicar cualquier parche
- **ENTONCES** el escritor devuelve un error tipado en castellano que nombra la construcción no admitida
- **Y** el fichero no se modifica

#### Scenario: Se rechaza un fichero con varios documentos YAML
- **DADO** un `axiom.yaml` con dos documentos YAML
- **CUANDO** se aplica cualquier parche
- **ENTONCES** se rechaza con `unsupported_document` sin escribir
- **Y** ambos documentos permanecen en el fichero

#### Scenario: El lote es atómico, todo o nada
- **DADO** un parche con un cambio válido y otro inválido
- **CUANDO** se aplica
- **ENTONCES** se rechaza entero y el fichero no cambia

#### Scenario: Se conservan líneas en blanco, CRLF y BOM
- **DADO** un `axiom.yaml` con líneas en blanco entre bloques, finales `\r\n` y BOM UTF-8
- **CUANDO** se aplica un parche que cambia una sola línea
- **ENTONCES** solo cambia esa línea y se conservan las líneas en blanco, los finales de línea y el BOM
- **Y** un parche sin efecto no escribe y deja el fichero byte a byte idéntico

#### Scenario: Un error previo del motor no bloquea y vuelve como aviso
- **DADO** un `axiom.yaml` que ya incumple una comprobación del motor contra el disco (por ejemplo, un rol especializado en `monorepo-embedded`)
- **CUANDO** se aplica un parche válido que no toca ese error
- **ENTONCES** el parche se escribe
- **Y** el error previo se devuelve como aviso

#### Scenario: Un error nuevo del motor bloquea
- **DADO** un `axiom.yaml` válido
- **CUANDO** el parche introduce una entrada de `repositories` hacia una ruta inexistente
- **ENTONCES** se rechaza con `validation_failed` sin escribir

#### Scenario: Valor inválido rechazado con el error del validador
- **DADO** un `axiom.yaml` válido
- **CUANDO** se aplica `git.promotion: siempre`
- **ENTONCES** el escritor devuelve el error del validador del motor en castellano
- **Y** el fichero conserva su contenido y su revisión

#### Scenario: Un parche válido por bloque pero inválido en conjunto se rechaza
- **DADO** un proyecto con `sdd.iso: true` y `workflow.lane: sdd`
- **CUANDO** se aplica `workflow.lane: odd`
- **ENTONCES** se rechaza con la contradicción ISO sin escribir
- **Y** aplicar `sdd.iso: true` sobre un proyecto con `workflow.lane: odd` se rechaza del mismo modo

#### Scenario: Clave no editable rechazada
- **DADO** un parche que incluye `workspace.topology`
- **CUANDO** se aplica
- **ENTONCES** se rechaza con un error tipado de clave no editable
- **Y** el fichero no cambia

#### Scenario: Se edita `integrations` conservando comentarios y orden
- **DADO** un `axiom.yaml` comentado con un bloque `integrations.azuredevops`
- **CUANDO** se aplica `integrations[azuredevops].<clave>` con un valor válido
- **ENTONCES** solo cambia esa línea y los comentarios, el orden y los bloques ajenos permanecen idénticos

#### Scenario: Se activa un plugin que no estaba en el fichero
- **DADO** un `axiom.yaml` sin bloque `integrations`
- **CUANDO** se aplica `integrations[azuredevops].enabled: true` junto con los campos obligatorios en el mismo lote
- **ENTONCES** se crea el bloque y el documento resultante vuelve a cargarse como `COMPLIANT`

#### Scenario: Retirar la última clave borra la entrada y el bloque
- **DADO** un `axiom.yaml` cuyo bloque `integrations` solo tiene un plugin con una clave
- **CUANDO** se aplica `null` a esa clave
- **ENTONCES** desaparecen la entrada del plugin y el bloque `integrations`

#### Scenario: Un bloque `integrations` con ancla, alias o `<<` se rechaza
- **DADO** un `axiom.yaml` cuyo bloque `integrations` usa un ancla, un alias o `<<`
- **CUANDO** se aplica cualquier parche
- **ENTONCES** se rechaza con `unsupported_document` sin escribir

#### Scenario: Un lote de integración inválido en conjunto se rechaza
- **DADO** un parche válido por clave pero que deja el plugin activo sin un campo obligatorio
- **CUANDO** se aplica
- **ENTONCES** se rechaza con `validation_failed` y el fichero no cambia

#### Scenario: Un plugin con guion en su identificador se direcciona entero
- **DADO** un plugin cuyo identificador contiene un guion
- **CUANDO** se aplica `integrations[<plugin>].<clave>`
- **ENTONCES** la clave se escribe en la entrada de ese plugin

#### Scenario: Escritura atómica
- **DADO** un parche válido
- **CUANDO** se escribe
- **ENTONCES** el contenido nuevo se escribe en un temporal del mismo directorio y se renombra sobre `axiom.yaml`
- **Y** si el temporal no puede escribirse o renombrarse, `axiom.yaml` conserva su contenido anterior y no queda ningún temporal

#### Scenario: Multirepo escribe el fichero del repositorio de specs
- **DADO** un `multirepo` cuyo `axiom.yaml` del código apunta al de specs
- **CUANDO** se aplica un parche
- **ENTONCES** se escribe la ruta que devuelve `Locate().ConfigPath`
- **Y** no se crea ni modifica ningún `axiom.yaml` en el repositorio de código
