# Delta for axiom-skills-index-governance

## ADDED Requirements

### Requirement: Saneamiento del Preset Predeterminado de Skills (REQ-46.9)

El catálogo de skills distribuido en el preset básico de Axiom DEBE contener únicamente skills de producto para usuarios, segregando las skills de colaboración interna del repositorio (`contributorSkills`).
(Trasladado desde REQ-18.5 de `rdd-decoupling-v3-stability`, retirada con RDD (C-01-e).)

#### Scenario: Inspección del preset básico de skills

- **DADO** el catálogo de skills en `internal/components/skills`
- **CUANDO** se consulta el listado del preset por defecto
- **ENTONCES** contiene únicamente skills fundacionales de producto y excluye las de contribución interna
