# Delta para Dashboard Web Local (local-web-dashboard)

> **Cambio:** `bateria-pruebas` (C-06-c). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.
> RF-WEB-01 a RF-WEB-16 no cambian de texto. No se añade ninguna ruta HTTP.

## ADDED Requirements

### Requirement: Pestaña «Pruebas» de Solo Lectura en el Detalle del Incremento, sin Ruta Nueva (RF-WEB-17)

`GET /api/increments/<n>` DEBE incluir en el detalle de un incremento SDD un campo de la batería de pruebas con el contenido de `pruebas.md` y su cobertura, calculada con el mismo `acceptancetests.Check` que usan `axiom sdd tests check` y `axiom sdd status` (RF-KOF-20). El campo DEBE incluir los casos de la tabla, las cifras de escenarios M/M, requisitos M/M y número de casos, y los hallazgos. Sin `pruebas.md`, el campo DEBE indicar su ausencia y, si la función está activa, el aviso de RF-KOF-22. No DEBE añadirse ninguna ruta, y el endpoint de ficheros de RF-WEB-15 NO DEBE ampliar su lista blanca.

El detalle DEBE incluir una pestaña «Pruebas», de solo lectura: muestra la tabla de casos con sus columnas, la cobertura y los hallazgos; no ofrece edición, ejecución ni marcas por entorno ni ninguna acción de escritura. Todo texto proveniente de `pruebas.md` DEBE insertarse como texto, nunca como HTML. Con la función no activa y sin `pruebas.md`, la pestaña DEBE indicar que las pruebas de aceptación no están activadas para el proyecto. Los hallazgos nunca bloquean ninguna otra acción del detalle. Las pruebas de pines de pestañas y de assets, y el canario de secretos, DEBEN seguir en verde.

#### Scenario: El detalle devuelve la batería y su cobertura
- **DADO** un incremento con `pruebas.md` válido y su cobertura completa
- **CUANDO** se consulta `GET /api/increments/<n>`
- **ENTONCES** el detalle incluye los casos, los escenarios M/M, los requisitos M/M, el número de casos y cero hallazgos

#### Scenario: La cobertura coincide con la del verbo
- **DADO** un incremento con `pruebas.md` con hallazgos
- **CUANDO** se consulta el detalle y se ejecuta `axiom sdd tests check --json`
- **ENTONCES** cifras y hallazgos son iguales

#### Scenario: La pestaña es de solo lectura
- **DADO** un incremento con `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** se muestran la tabla, la cobertura y los hallazgos
- **Y** no hay controles de edición ni de ejecución

#### Scenario: Contenido hostil se muestra como texto
- **DADO** un `pruebas.md` con `<script>` en una celda
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** el texto aparece literal y no se interpreta como HTML

#### Scenario: Función no activada
- **DADO** un proyecto sin el bloque `acceptance_tests` y un incremento sin `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** indica que las pruebas de aceptación no están activadas

#### Scenario: Activa y sin fichero
- **DADO** la función activa y un incremento tras tasks sin `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** indica la ausencia y muestra el aviso de RF-KOF-22
- **Y** no bloquea ninguna otra pestaña

#### Scenario: Sin ruta nueva
- **DADO** el servidor del dashboard
- **CUANDO** se enumeran sus rutas
- **ENTONCES** no existe ninguna ruta añadida por este requisito y la lista blanca de RF-WEB-15 no cambia
