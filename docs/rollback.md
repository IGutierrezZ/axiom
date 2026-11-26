# Copias de seguridad y restauración

Axiom crea una copia de los ficheros de configuración que va a modificar antes de instalar, sincronizar o actualizar componentes. Axiom migra una vez las copias antiguas de `~/.gentle-ai/backups/` a la raíz de estado.

## Qué se guarda

Cada operación de escritura:

1. Calcula la huella de los ficheros incluidos en la copia.
2. Omite el respaldo si su contenido coincide con el último.
3. Crea una instantánea comprimida (`snapshot.tar.gz`).
4. Aplica la política de retención a las copias no fijadas.

Un `sync` que termina bien sin escribir ningún fichero («no changes needed») descarta la copia que acababa de crear: sería idéntica al estado actual y no sirve de punto de restauración, así que las sincronizaciones sin cambios no acumulan copias. Un `sync` que sí escribe conserva su copia previa.

Las copias de Axiom viven en `<raíz de estado>/backups/`, donde la raíz de estado es `~/.axiom` o, si defines `AXIOM_STATE_DIR`, esa carpeta. Con la variable definida, las copias hechas antes en `~/.axiom/backups/` dejan de listarse (siguen en disco).

El manifiesto registra el origen, la fecha, el número de ficheros, la huella y si una ruta existía antes de la operación. Las copias antiguas, anteriores al formato comprimido, conservan un directorio `files/`; Axiom también las admite al restaurar.

El alcance de un respaldo de sincronización depende de los agentes registrados como instalados por Axiom. No presupongas que incluye los directorios de agentes configurados manualmente fuera de Axiom.

## Retención

| Ajuste | Valor predeterminado | Comportamiento |
|--------|---------------------|----------------|
| Cantidad | 5 | Se conservan las cinco copias no fijadas más recientes en cada carpeta de respaldos |
| Copias fijadas | Sin límite | No se eliminan durante la poda automática |
| Duplicados | Omitidos | No se crea otra copia si la configuración no ha cambiado |
| Compresión | Activada | Las nuevas copias usan `tar.gz` |

## Restaurar desde la CLI

Lista las copias disponibles y restaura una por su identificador, o la más reciente con `latest`:

```sh
axiom restore --list
axiom restore <id>
axiom restore latest
```

`axiom restore` pide confirmación; `--yes` la omite.

## Semántica de restauración

- Si `existed=true`, restaura el fichero en su ruta original.
- Si `existed=false`, elimina el fichero creado por la operación respaldada.
- Las escrituras de cada fichero son atómicas; la restauración completa puede abarcar varios ficheros.
- Se admiten tanto instantáneas `tar.gz` como el formato histórico `files/`.
- La restauración revierte ficheros de configuración; no desinstala paquetes del sistema.

## Si falla una verificación

1. Lee qué comprobaciones han fallado y qué ficheros se han modificado.
2. Restaura la copia de Axiom apropiada con `axiom restore --list` y `axiom restore <id>`.
3. Corrige la causa externa y valida el plan antes de repetir la operación con `axiom install --dry-run` o `axiom sync --dry-run`.
4. Aplica de nuevo la operación solo cuando el plan sea el esperado.

## Límites

- Las instalaciones realizadas mediante `brew`, `apt-get`, `pacman` o `dnf` no se desinstalan al restaurar una copia de configuración. Usa el gestor de paquetes correspondiente si también quieres retirarlas.
