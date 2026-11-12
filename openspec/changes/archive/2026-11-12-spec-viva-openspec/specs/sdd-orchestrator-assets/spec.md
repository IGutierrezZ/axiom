# Delta para sdd-orchestrator-assets

> Parte del estado vigente: el resto de assets, skills y la proyección de las instrucciones del orquestador NO cambian. Esta delta (a) quita de las skills `sdd-*` y de la convención compartida toda referencia a la baseline ajena y (b) añade la guía de adopción, siempre con paridad entre runtimes.
> Coordinación con `carril-sdd-versatil`: su REQ-29.17 añade la guía ISO a los mismos assets; quien llega segundo rebasa. El nombre del fichero `_shared/project-baseline-convention.md` lo citan las instrucciones del orquestador (secciones 5 y 6); cualquier renombrado regenera los goldens y se hace con la herramienta de regeneración, nunca a mano.

## ADDED Requirements

### Requirement: Skills y Convención Compartida sin Baseline Ajena (REQ-30.28)

Las skills `sdd-*` y los ficheros compartidos (`_shared/`) distribuidos por Axiom NO DEBEN contener ramas, prosa ni escenarios que describan `baseline:`, `baseline.documents`, los modos de integración `compose`, `directed` o `register`, `axiom sdd integrate` ni `axiom sdd archive-compose --document` o `--in-place`. La skill `sdd-spec` DEBE leer la spec viva de `openspec/specs/{dominio}/spec.md` como única fuente del comportamiento vigente. La convención compartida del proyecto (`_shared/project-baseline-convention.md`) DEBE quedar reducida al contexto técnico (declaración, lectura, delta de contexto e integración con `axiom archive context`) y a la regla de «Sin impacto». La skill `sdd-archive` DEBE prescribir `axiom archive context check|prepare|verify|show` para el contexto mantenido. La prosa que se proyecta a los runtimes se regenera con la herramienta de goldens y permanece en paridad entre runtimes.

#### Scenario: Ninguna skill describe la baseline ajena
- **DADO** los textos de las skills `sdd-*` y de `_shared/` distribuidos
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno contiene `baseline.documents`, `integration: directed`, `axiom sdd integrate` ni `--in-place`

#### Scenario: `sdd-spec` lee `openspec/specs/`
- **DADO** el texto de la skill `sdd-spec`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica leer `openspec/specs/{dominio}/spec.md` para conocer el comportamiento vigente
- **Y** no condiciona la lectura a ninguna proyección `baseline`

#### Scenario: La convención compartida solo trata el contexto técnico
- **DADO** el texto de `_shared/project-baseline-convention.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** describe el contexto técnico, el delta de contexto y `axiom archive context`
- **Y** no describe documentos de baseline ni perfiles de requisito

#### Scenario: `sdd-archive` prescribe el verbo de contexto
- **DADO** el texto de la skill `sdd-archive`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** prescribe `axiom archive context` para la integración del contexto mantenido

#### Scenario: Paridad entre runtimes
- **DADO** dos runtimes soportados
- **CUANDO** se renderizan las skills y las instrucciones
- **ENTONCES** el contenido normativo es idéntico entre ambos y los goldens coinciden con la regeneración

---

### Requirement: Skill o Referencia de Adopción de OpenSpec (REQ-30.31)

Axiom DEBE distribuir una skill o una referencia compartida de adopción que describa, en un texto idéntico para todos los runtimes, el flujo de migración a OpenSpec: ejecutar `axiom archive adopt scan --change <cambio>`; redactar la delta de adopción con solo `## ADDED Requirements`, un identificador `(REQ-n.m)` de una serie nueva, al menos un escenario y una línea `Origen:` por requisito; listar con motivo las unidades descartadas en `Descartado:`; ejecutar `axiom archive adopt check --change <cambio>` hasta que no reporte fallos; y someter la compuerta `spec` a la aprobación humana. El texto NO DEBE afirmar que Go juzga la fidelidad semántica de la migración.

#### Scenario: La guía de adopción describe el flujo completo
- **DADO** el texto de la skill o referencia de adopción
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** cita `adopt scan`, `adopt check`, `Origen:`, `Descartado:`, la serie `REQ-n.m` y la compuerta `spec`

#### Scenario: La guía no atribuye a Go la fidelidad semántica
- **DADO** el mismo texto
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que la fidelidad la aprueba un humano en la compuerta `spec`

#### Scenario: Paridad entre runtimes
- **DADO** dos runtimes soportados
- **CUANDO** se renderiza la guía de adopción
- **ENTONCES** el contenido normativo es idéntico entre ambos
