# Delta para Integración de SDD en la CLI `axiom` (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** delta sobre la capacidad viva `axiom-sdd-cli-integration`.
> **Idioma:** castellano de España; identificadores y claves JSON en inglés.

Cambios de este delta:

- REQ-27.2 (nuevo): `axiom sdd status` proyecta el carril efectivo del proyecto y su origen, en JSON v1 y v2, con o sin cambio seleccionado. REQ-13.1 no cambia: el campo es aditivo y opcional, como `baseline` y `technicalContext` (REQ-24.19 de `spec-baseline-declaration`).
- REQ-27.2b (nuevo): `axiom sdd publish` figura en el contrato del binario como subcomando de `sdd` (el comportamiento vive en `sdd-docs-publication`).

## ADDED Requirements

### Requirement: `axiom sdd status` Proyecta el Carril Efectivo del Proyecto (REQ-27.2)

`axiom sdd status` DEBE incluir, en el JSON v1 y en el v2 y con o sin cambio seleccionado, un campo opcional `workflow` con el carril efectivo (`lane`: `odd`, `sdd` o `ask`) y su origen (`source`), con los tokens estables en inglés `declared`, `default`, `no_workspace` e `invalid`. El origen `declared` indica que `axiom.yaml` declara `workflow.lane`; `default` que `axiom.yaml` es válido y no lo declara (carril `sdd`); `no_workspace` que no hay `axiom.yaml` (repositorio no gobernado por Axiom); e `invalid` que el bloque `workflow` no es válido. Con `no_workspace` o `invalid` el carril proyectado DEBE ser `ask`, nunca `sdd`: la ausencia de configuración o una configuración inválida NO DEBEN activar SDD automático. El campo DEBE estar presente con `changeName: null` cuando no hay cambio seleccionado, con el mismo criterio que `baseline` y `technicalContext`, y NO DEBE alterar ningún otro campo del contrato ni de las rutas proyectadas. Un consumidor anterior que ignore los campos desconocidos DEBE seguir funcionando. El contrato publicado (`sdd-status-contract.md`) DEBE documentar el campo y la regla de fallo cerrado hacia `ask`.

#### Scenario: Carril declarado con cambio seleccionado

- **DADO** un `axiom.yaml` con `workflow: {lane: odd}` y un cambio SDD activo
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` en el formato v2
- **ENTONCES** el JSON contiene `workflow` con `lane: odd` y `source: declared`

#### Scenario: Carril por defecto sin bloque

- **DADO** un `axiom.yaml` válido sin `workflow:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `workflow` indica `lane: sdd` y `source: default`

#### Scenario: Sin cambio seleccionado el campo sigue presente

- **DADO** un `axiom.yaml` con `workflow: {lane: ask}` y ningún cambio seleccionado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `changeName` es `null`
- **Y** `workflow` indica `lane: ask` y `source: declared`

#### Scenario: Presente en el formato v1 y en el v2

- **DADO** el mismo proyecto con `workflow: {lane: sdd}`
- **CUANDO** se ejecuta `axiom sdd status --json` en el formato v1 y en el v2
- **ENTONCES** ambos contienen `workflow` con los mismos valores de `lane` y `source`

#### Scenario: Sin `axiom.yaml` el carril proyectado es `ask`

- **DADO** un repositorio sin `axiom.yaml`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `workflow` indica `lane: ask` y `source: no_workspace`

#### Scenario: Bloque inválido proyecta `ask` y su origen

- **DADO** un `axiom.yaml` con `workflow: {lane: agil}`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `workflow` indica `lane: ask` y `source: invalid`
- **Y** el resto del JSON se proyecta como hasta ahora

#### Scenario: El resto del contrato no cambia

- **DADO** un cambio con `baseline`, `technicalContext` y `governance` proyectados
- **CUANDO** se ejecuta `axiom sdd status --json` antes y después de añadir `workflow` a `axiom.yaml`
- **ENTONCES** los demás campos son idénticos

#### Scenario: El contrato publicado documenta el campo

- **DADO** `sdd-status-contract.md`
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** documenta `workflow`, sus tres valores de `lane`, los cuatro orígenes y que `no_workspace` e `invalid` proyectan `ask`

---

### Requirement: Subcomando `axiom sdd publish` en el Contrato del Binario (REQ-27.2b)

El binario `axiom` DEBE soportar la invocación `axiom sdd publish --change <nombre> [--cwd <ruta>] [--json]` como subcomando de `axiom sdd`, con el comportamiento de la capacidad `sdd-docs-publication`. La ayuda y la lista de subcomandos de `axiom sdd` DEBEN incluirlo.

#### Scenario: La ayuda de `sdd` lista `publish`

- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd --help`
- **ENTONCES** la salida lista `publish` junto a los subcomandos ya existentes, que siguen listados

#### Scenario: `publish` sin cambio se rechaza con ayuda

- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd publish` sin `--change`
- **ENTONCES** termina con código de salida distinto de cero y un mensaje que nombra `--change`
