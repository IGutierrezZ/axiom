# Explainer HTML (`resumen-spec.html` and `resumen-design.html`)

Shared contract for `sdd-spec` and `sdd-design`. It applies ONLY when `axiom sdd status` projects `governance.kickoff.explainerHtml: yes` and the artifact store is `openspec` or `hybrid`. With `engram` or `none`, write no HTML and say so in the result.

The HTML is a reading aid for non-technical people. It never replaces the md: what gets approved is the md. A stale or missing HTML never blocks a phase, a gate or archive.

## Contract

- Written in Spanish (castellano), neutral and professional register, for a non-technical audience.
- Sections: what changes, for whom, flow before and after, an inline SVG diagram, and a glossary.
- Every block carries `data-axiom-ref="<ID>"` and shows that ID: the final requirement ID, the provisional `prov:<series>:<n>` ID, or the decision title. Cite only IDs that exist in the md; add nothing that is not in the md.
- Carries the literal notice: "Este resumen no sustituye al documento: lo que se aprueba es el md."
- Self-contained: inline HTML, CSS and SVG; images only as `data:` URIs.
- No `<script>` tag, no `on*` event handlers, no external resources: no `http://`, `https://` or `//` URL in `src`, `href`, `xlink:href`, `srcset` or CSS `url(...)`, and no `@import`.
- `<meta name="axiom-source-digest" content="<expectedDigest>">` with the literal `expectedDigest` that `axiom sdd status` projects for this kind (`explainer.spec` or `explainer.design`). Never compute a digest yourself.
- Smaller than 5 MiB.

## Procedure

1. Write the md and persist it as the phase says.
2. Run `axiom sdd status --cwd "<workspaceRoot>" <change> --json` and copy `explainer.<kind>.expectedDigest` into the `<meta>`.
3. Write `<changeRoot>/resumen-<kind>.html` following this contract.
4. Read the status again and fix the HTML until `explainer.<kind>.state` is `fresh`.
5. If the HTML cannot be written or breaks the contract, finish the phase anyway with the md, warn in the result, and leave the state `missing` or `stale`: it blocks nothing.

Regenerate the HTML whenever the md changes: after a rejection or correction that reopens the phase, when the previous HTML is absent, and after a manual edit of the md.

## Example md

```markdown
### RF-EJ-01: Aviso de caducidad

El sistema MUST avisar a la persona usuaria siete días antes de que caduque su acceso.

#### Scenario: Acceso a punto de caducar

- GIVEN un acceso que caduca en siete días
- WHEN llega la fecha de aviso
- THEN la persona usuaria recibe un mensaje

### RF-EJ-02: Renovación

El sistema SHOULD permitir renovar el acceso desde el propio aviso.

#### Scenario: Renovar desde el aviso

- GIVEN un aviso de caducidad recibido
- WHEN la persona pulsa renovar
- THEN el acceso se amplía un año
```

## Example skeleton

```html
<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="axiom-source-digest" content="sha256:0000000000000000000000000000000000000000000000000000000000000000">
<title>Resumen de la spec: caducidad de accesos</title>
<style>
body { font-family: sans-serif; max-width: 48rem; margin: 2rem auto; line-height: 1.5; }
section { border-left: 4px solid #2b6cb0; padding-left: 1rem; margin-bottom: 1.5rem; }
.aviso { background: #fffbea; padding: 0.75rem; }
</style>
</head>
<body>
<h1>Caducidad de accesos</h1>
<p class="aviso">Este resumen no sustituye al documento: lo que se aprueba es el md.</p>
<section data-axiom-ref="RF-EJ-01">
<h2>RF-EJ-01: Aviso de caducidad</h2>
<p>Antes de que un acceso caduque, la persona usuaria recibe un mensaje siete días antes.</p>
</section>
<section data-axiom-ref="RF-EJ-02">
<h2>RF-EJ-02: Renovación</h2>
<p>Desde ese mismo mensaje se puede renovar el acceso.</p>
</section>
<svg width="320" height="60" role="img" aria-label="Flujo: aviso, renovación">
<rect x="5" y="10" width="120" height="40" fill="#bee3f8"/>
<text x="15" y="35">Aviso</text>
<rect x="195" y="10" width="120" height="40" fill="#c6f6d5"/>
<text x="205" y="35">Renovación</text>
<line x1="125" y1="30" x2="195" y2="30" stroke="#333"/>
</svg>
<h2>Glosario</h2>
<dl><dt>Acceso</dt><dd>Permiso de una persona para usar el sistema.</dd></dl>
</body>
</html>
```
