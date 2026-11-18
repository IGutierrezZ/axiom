# Propuesta: identidad git como actor de los artefactos versionados (C-07-a)

## Intención

Axiom no sabe quién hace cada cosa. Los dos ficheros versionados que dicen quién hizo qué registran hoy un literal fijo:

- `axiom sdd gate record` escribe `actor: cli` en `gates.yaml` cuando no se pasa `--actor` (`internal/cli/sdd_gate.go:100-103`).
- `axiom sdd kickoff seal` escribe `sealed_by: cli` en `kickoff.yaml` (`SealArgs.ToKickoff`, `internal/kickoff/args.go:375`); `seal` no admite `--actor`.

El usuario decidió que «lo que tengan configurado entiendo que será su identidad de git». Con ello se puede saber quién aprobó cada compuerta y quién selló cada cambio, y se deja la base que necesitan C-07-e (quién imputó horas) y C-07-h (vista por persona).

Éxito: con `git config user.name` y `user.email` definidos, `gate record` sin `--actor` escribe `actor: "Nombre <email>"` y `kickoff seal` escribe `sealed_by` con esa misma identidad; sin identidad git el valor sigue siendo `cli`, se avisa por stderr y nada se bloquea; los registros anteriores siguen siendo válidos.

## Alcance

### Dentro del alcance

1. **Resolución de la identidad git.**
   - Se lee solo la configuración de git (`git config --get user.name` y `user.email`: local, global e `includeIf` según el directorio), desde la raíz del workspace (`loc.WorkspaceRoot`), que es donde se escriben los artefactos. No se leen variables `GIT_AUTHOR_*` ni `git var`, que puede inventar `usuario@host`.
   - Formato: `Nombre <email>`.
   - Saneado: recorte de espacios, rechazo de caracteres de control (CR/LF), retirada de `<` y `>` del nombre y del email y tope de longitud. Un valor que queda vacío tras el saneado cuenta como ausente.
   - La resolución nunca falla: git ausente, directorio que no es repositorio, clave vacía o error de ejecución producen «sin identidad».
2. **Actor de compuerta.** `gate record` sin `--actor` registra la identidad resuelta. `--actor` explícito prevalece y no consulta git ni avisa.
3. **Sellador del kickoff.** `kickoff seal` sin `--infer` registra la identidad resuelta en `sealed_by`. `seal --infer` (retro-sellado) sigue escribiendo `inferred`, con su significado especial intacto (`ValidateForSeal`, `internal/kickoff/schema.go:149-155`).
4. **Reserva y aviso.** Sin identidad completa (falta `user.name`, falta `user.email` o faltan los dos) se conserva `cli` y se emite un aviso por stderr con el prefijo `Aviso: `, que dice qué falta y cómo resolverlo (configurar git o, en `gate record`, indicar `--actor`). El aviso va a un `io.Writer` propio, sustituible en pruebas; no reutiliza `sealBranchWarnings` ni `writerBranchWarnings`, ni ensucia el stdout JSON. Nunca se bloquea.
5. **Compatibilidad.** Los registros existentes (`cli`, `maintainer`, `release-manager`, `inferred`...) se siguen leyendo y validando igual, sin migración. El tipo de `GateRecord.Actor` y de `Kickoff.SealedBy` no cambia; solo se actualiza su documentación.
6. **Ayuda de los verbos.** El texto de `--actor` de `gate record` indica el nuevo valor por defecto.
7. **Spec viva.** Requisito ADDED en una capacidad nueva `change-actor-identity`.

### Fuera del alcance

- `integrations.yaml` y el registro de quién inició o cerró un rol o imputó horas: los aportan C-07-b y C-07-e reutilizando esta resolución.
- La identidad de Azure DevOps en `~/.axiom/` y su vínculo con la persona (C-07-c).
- La vista «quién está con qué» en la UI (C-07-h).
- Añadir `--actor` (o `--sealed-by`) a `kickoff seal`.
- Añadir un campo `actor` o `warnings` al JSON de `gate record` o de `seal`.
- `reviewAuditActor` de RDD (`internal/cli/review_audit_actor.go`) y los `--actor` de otros verbos (`sdd attempt grant`, `review *`, `techcontext`), con ledgers no versionados y semántica propia.
- Guías de agente y goldens: ninguna cita `--actor` de compuerta ni fija el valor `cli` de estos comandos.
- Reescribir `gates.yaml` o `kickoff.yaml` existentes o archivados.

## Capacidades

### Nuevas capacidades

- `change-actor-identity`: identidad registrada como actor en los artefactos versionados de un cambio (`actor` de `gates.yaml`, `sealed_by` de `kickoff.yaml`): fuente (configuración de git desde la raíz del workspace), formato `Nombre <email>`, saneado, precedencia de `--actor`, reserva `cli` con aviso sin bloqueo, `inferred` intacto y validez de los registros anteriores. Prefijo propuesto **REQ-42** (máximo actual REQ-41; ver Riesgos).

### Capacidades modificadas

Ninguna. Ninguna spec viva fija hoy el valor `cli`; `sealed_by` solo aparece como `inferred` en `sdd-lane-kickoff` y `sdd-preflight-configuration`, y ese comportamiento no cambia. No se toca `sdd-block-review-gates` para evitar el solape con C-06-c.

## Enfoque

Enfoque A de la exploración: un paquete hoja nuevo y la decisión en el adaptador CLI.

1. **Paquete `internal/gitidentity`** (hoja; importa solo `internal/gitcommon`). Lee `user.name` y `user.email` con `gitcommon.Run(ctx, dir, "config", "--get", ...)`, que ya aporta directorio absoluto, entorno saneado, `LC_ALL=C` y timeout. Expone una identidad resuelta (o ausente, con qué falta) y su formato de actor; el saneado y el formateo son funciones puras, comprobables sin git. Así lo reutilizan C-07-b/C-07-e sin duplicar y se respeta que solo `gitcommon.Run` lanza git.
2. **`gate record`.** En `runSDDGateRecord`, solo si `parsed.Actor == ""`, se resuelve la identidad desde `loc.WorkspaceRoot`; si falta, se usa `cli` y se avisa.
3. **`kickoff seal`.** `SealArgs.ToKickoff` sigue pura y conserva `SealedBy: "cli"` como valor de reserva (la usan tests y e2e). En `runSDDKickoffSeal`, en la rama sin `--infer` y tras `ToKickoff`, el adaptador sobrescribe `SealedBy` con la identidad resuelta, igual que ya completa `Change`; si falta, deja `cli` y avisa.
4. **Aviso.** Variable de paquete propia (`io.Writer = os.Stderr`) con texto en castellano y prefijo `Aviso: `.
5. **Pruebas herméticas.** Toda prueba que dependa de la identidad aísla git (`GIT_CONFIG_GLOBAL` a un fichero temporal y `GIT_CONFIG_NOSYSTEM=1`) y, cuando la necesita, crea un repositorio temporal con `git init` y `git config`. Se cubren: identidad completa en compuerta y en sellado, `--actor` explícito, identidad ausente y parcial (reserva `cli` y aviso), `--infer` intacto, round-trip YAML con un nombre no ASCII y con `#`/`:` y lectura de registros anteriores. El diseño decide si `internal/gitidentity` entra en la lista de `internal/gitcommon/architecture_test.go`.

Estimación: S (unas 150-250 líneas con pruebas), una sola PR (`single-pr`).

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/gitidentity/` | Nuevo | Lectura de `user.name`/`user.email` con `gitcommon.Run`, saneado y formato `Nombre <email>`, con sus pruebas |
| `internal/cli/sdd_gate.go` | Modificado | Actor por defecto con la identidad resuelta (`:100-103`), aviso y ayuda de `--actor` (`:48`) |
| `internal/cli/sdd_kickoff.go` | Modificado | Sobrescritura de `SealedBy` tras `ToKickoff` en la rama sin `--infer` (`:83`) y aviso |
| `internal/kickoff/args.go`, `types.go` | Modificado (solo documentación) | `cli` como valor de reserva en `ToKickoff` (`:375`); comentarios de `SealedBy` (`:146`) y `Actor` (`:191`) |
| `internal/cli/sdd_gate_test.go`, `sdd_kickoff_test.go`, `sdd_kickoff_branch_test.go` | Modificado | Casos nuevos con git aislado; los que afirman cero avisos de rama no cambian |
| `internal/gitcommon/architecture_test.go` | Posible modificación | Si el diseño incorpora el paquete nuevo a la lista escaneada |
| `openspec/changes/identidad-git-actor/specs/change-actor-identity/spec.md` | Nuevo | Spec delta de la capacidad nueva (ADDED) |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|------|------------|------------|
| Colisión del prefijo REQ-42 con otras hijas de la misma ola que lo reclamen a la vez | Media | Antes de archivar, confirmar en `main` el siguiente prefijo libre y renumerar si hace falta |
| Pruebas no herméticas: pasan en la máquina del desarrollador (con identidad global) y fallan en CI (sin identidad), o al revés | Media | Aislar siempre `GIT_CONFIG_GLOBAL` y `GIT_CONFIG_NOSYSTEM` en las pruebas que dependen de la identidad |
| El aviso de identidad aparece en pruebas existentes que comprueban cero avisos (CI sin identidad) | Media | Writer de aviso propio, distinto de `sealBranchWarnings` y `writerBranchWarnings` |
| Caracteres especiales o no ASCII en `user.name` (`#`, `:`, comillas, acentos) rompen el YAML o el round-trip | Baja | yaml.v3 escapa el escalar; prueba de round-trip con un nombre no ASCII y con `#`/`:` |
| En multirepo la identidad depende del repositorio (config local, `includeIf`) | Baja | Se resuelve desde `loc.WorkspaceRoot` (planning home, donde se escribe el artefacto) y el requisito lo fija |
| El email entra en artefactos versionados | Baja | Ya es público en los commits del mismo repositorio; el requisito lo deja anotado |
| Un binario anterior lee un `actor` con formato nuevo | Baja | El campo es un `string` sin validación de valor; lo lee igual |

## Plan de reversión

- Revertir la PR única: el actor y el sellador vuelven al literal `cli`, desaparece el paquete `internal/gitidentity` y su aviso.
- No hay migración de datos: los `gates.yaml` y `kickoff.yaml` escritos con `Nombre <email>` siguen siendo válidos para el binario revertido, porque ningún código compara ni valida el valor del actor.
- La spec delta de la capacidad nueva se retira con la reversión; ninguna spec viva existente cambia.

## Dependencias

- Ninguna previa. Usa `internal/gitcommon.Run`, ya presente.
- Bloquea a C-07-e (registro de quién imputó) y a C-07-h (vista por persona); C-07-b depende de forma blanda.

## Criterios de éxito

- [ ] Con `git config user.name`/`user.email` definidos, `axiom sdd gate record` sin `--actor` escribe `actor: "Nombre <email>"` en `gates.yaml`.
- [ ] Con esa identidad, `axiom sdd kickoff seal` (sin `--infer`) escribe `sealed_by` con `Nombre <email>`.
- [ ] `--actor` explícito en `gate record` se registra tal cual, sin consultar git ni avisar.
- [ ] Sin identidad git, o con solo nombre o solo email, se registra `cli`, se emite un único aviso `Aviso: ...` por stderr que dice qué falta, y el comando termina con éxito; el stdout JSON no cambia.
- [ ] `kickoff seal --infer` sigue escribiendo `sealed_by: inferred` y conserva su exención en `ValidateForSeal`.
- [ ] Los `gates.yaml` y `kickoff.yaml` existentes con `cli`, `maintainer` u otros valores se leen y validan sin cambios.
- [ ] Un `user.name` con acentos y `#`/`:` hace round-trip por YAML sin pérdida.
- [ ] `go test ./internal/gitidentity/... ./internal/kickoff/... ./internal/cli/... ./internal/gitcommon/...` en verde, y `go test ./...` sin regresiones de goldens.
