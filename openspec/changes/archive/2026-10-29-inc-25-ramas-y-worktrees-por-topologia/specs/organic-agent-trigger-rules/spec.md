# Delta para Organic Agent Routing Projection (INC-25)

> **Incremento:** `inc-25-ramas-y-worktrees-por-topologia` · **Tipo:** delta sobre la capacidad viva `organic-agent-trigger-rules`
> **Idioma:** Español (castellano peninsular)

Cambios de este delta (solo requerimientos nuevos; ningún requerimiento vigente cambia):

- REQ-25.40: el paso de implementación del protocolo ODD invoca `axiom branch ensure --lane odd` en lugar de «branch first when on the default branch».
- REQ-25.41: se prohíben `EnterWorktree` y `ExitWorktree` en los assets del producto, con un único punto de generación en la doctrina de enrutamiento.

## ADDED Requirements

### Requirement: El Protocolo ODD Prepara la Rama con `axiom branch ensure` (REQ-25.40)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN sustituir la regla «branch first when on the default branch» por la ejecución de `axiom branch ensure --lane odd --change <feature>` antes de la primera escritura del documento vivo `odd/tasks/<feature>.md` y de cualquier escritura de código, y DEBEN indicar que las órdenes posteriores usan como `--cwd` las rutas absolutas que devuelve su JSON. El cierre de cada tarea mantiene el commit por unidad de trabajo, ahora en la rama que `ensure` preparó. La regla NO DEBE alterar los demás pasos del protocolo ODD ni el carril SDD, NO DEBE exigir interacción con el usuario salvo el resultado `decision_required` del modo `ask`, y la proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado. Esta regla no activa SDD ni crea artefactos SDD.

#### Scenario: El paso de implementación invoca `ensure`
- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el protocolo ODD indica ejecutar `axiom branch ensure --lane odd --change <feature>` antes de la primera escritura
- **Y** el texto ya no contiene «branch first when on the default branch»

#### Scenario: Las rutas del resultado se usan como `--cwd`
- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de implementación
- **ENTONCES** indica usar las rutas absolutas del JSON de `ensure` como `--cwd` de las órdenes posteriores

#### Scenario: El resto del protocolo no cambia
- **DADO** las instrucciones renderizadas antes y después de este cambio
- **CUANDO** se comparan los demás pasos del protocolo ODD (autorizar, explorar, resolver incertidumbre, clasificar, registrar, implementar, cerrar) y el carril SDD
- **ENTONCES** son idénticos salvo la sustitución de la regla de rama

#### Scenario: El renderizado sigue siendo determinista
- **DADO** una sincronización de adaptador
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado

#### Scenario: El modo `ask` no añade una pregunta de gobernanza
- **DADO** las instrucciones renderizadas
- **CUANDO** se inspecciona (pin de texto) la regla
- **ENTONCES** la única interacción que prevé es presentar el resultado `decision_required` y repetir `ensure` con la respuesta

---

### Requirement: Prohibición de `EnterWorktree` y `ExitWorktree` en los Assets del Producto (REQ-25.41)

La doctrina de enrutamiento generada (punto único de generación) DEBE prohibir expresamente el uso de las primitivas `EnterWorktree` y `ExitWorktree` y DEBE prescribir el mecanismo común para todos los runtimes: el JSON de `axiom branch ensure` con rutas absolutas, y el uso de `--cwd`, `git -C` y rutas absolutas en lugar de cambiar el directorio de trabajo. Esa línea de prohibición es la **única** aparición permitida de los dos nombres en los assets del producto: ningún otro asset (skills, orquestadores por runtime, comandos, guías) DEBE contenerlos ni prescribir cualquier primitiva propia de un runtime para crear o cambiar de worktree. El texto generado DEBE ser idéntico para todos los adaptadores, de modo que el comportamiento no dependa de primitivas propias de uno de ellos.

#### Scenario: La doctrina generada contiene la prohibición
- **DADO** un adaptador registrado
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** contienen la prohibición de `EnterWorktree` y `ExitWorktree` y el mecanismo de rutas absolutas, `--cwd` y `git -C`

#### Scenario: Ningún otro asset contiene los nombres
- **DADO** el conjunto completo de assets del producto (skills, orquestadores de cada runtime, comandos y guías)
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** `EnterWorktree` y `ExitWorktree` solo aparecen en la línea de prohibición generada por la doctrina de enrutamiento

#### Scenario: La doctrina es idéntica en todos los adaptadores
- **DADO** dos adaptadores registrados distintos (por ejemplo Claude Code y OpenCode)
- **CUANDO** se comparan las líneas de prohibición y de mecanismo común de su doctrina renderizada
- **ENTONCES** son idénticas

#### Scenario: Los goldens fijan el texto
- **DADO** los goldens de doctrina y de assets regenerados
- **CUANDO** se ejecuta la batería de pruebas
- **ENTONCES** una modificación de la línea de prohibición falla por diferencia con el golden
