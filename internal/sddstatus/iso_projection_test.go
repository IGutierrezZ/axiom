package sddstatus

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// sealISOChange sella en continuous el cambio con la instantánea dada (nil, un
// kickoff anterior a este cambio). continuous no tiene compuertas, así que el
// archivado solo depende de las reglas ISO.
func sealISOChange(t *testing.T, changeRoot, name string, snapshot *kickoff.SDDSnapshot) {
	t.Helper()
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: name, SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		SDD:       snapshot,
	})
	if err != nil || !ok {
		t.Fatalf("sealISOChange: kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
}

// isoOf decodifica la clave `iso` del status v2, o nil si no existe.
func isoOf(t *testing.T, status Status) *ISOProjection {
	t.Helper()
	fields := statusJSON(t, status)
	if _, present := fields["iso"]; !present {
		return nil
	}
	projection := decodeField[ISOProjection](t, fields, "iso")
	return &projection
}

// REQ-29.1 y REQ-29.2 (design D7, D8): la proyección `iso` solo aparece con un
// bloque `sdd:` o con un cambio ISO; un `sdd:` inválido se proyecta `invalid`
// con su `reason` aunque no haya cambio; la instantánea manda sobre axiom.yaml.
func TestISOProjection(t *testing.T) {
	isoSnapshot := &kickoff.SDDSnapshot{ISO: true}
	tests := []struct {
		name       string
		blocks     string
		snapshot   *kickoff.SDDSnapshot
		sealed     bool
		change     string
		wantAbsent bool
		want       ISOProjection
		wantNote   string // fragmento que debe nombrar alguna nota
	}{
		{name: "sin bloque y cambio no ISO no proyecta nada", sealed: true, change: "demo", wantAbsent: true},
		{name: "sin bloque y sin sellar no proyecta nada", change: "demo", wantAbsent: true},
		{
			name: "proyecto ISO y cambio ISO", blocks: "sdd:\n  iso: true\n", sealed: true, snapshot: isoSnapshot, change: "demo",
			want: ISOProjection{Project: "iso", Change: "iso"},
		},
		{
			name: "sin cambio seleccionado se proyecta el valor del proyecto", blocks: "sdd:\n  iso: true\n",
			want: ISOProjection{Project: "iso"},
		},
		{
			name: "bloque con iso falso", blocks: "sdd:\n  iso: false\n", sealed: true, change: "demo",
			want: ISOProjection{Project: "not_iso", Change: "not_iso"},
		},
		{
			name: "el valor sellado manda aunque el proyecto ya no declare iso", sealed: true, snapshot: isoSnapshot, change: "demo",
			blocks:   "sdd:\n  iso: false\n",
			want:     ISOProjection{Project: "not_iso", Change: "iso"},
			wantNote: "(sdd.iso: true) difiere de la declaración vigente del proyecto (sdd.iso: false)",
		},
		{
			name: "un cambio sellado sin instantánea es no ISO", blocks: "sdd:\n  iso: true\n", sealed: true, change: "demo",
			want:     ISOProjection{Project: "iso", Change: "not_iso"},
			wantNote: "(sdd.iso: false) difiere de la declaración vigente del proyecto (sdd.iso: true)",
		},
		{
			name: "cambio ISO sin bloque en axiom.yaml", sealed: true, snapshot: isoSnapshot, change: "demo",
			want:     ISOProjection{Project: "not_iso", Change: "iso"},
			wantNote: "(sdd.iso: true) difiere",
		},
		{
			name: "un sdd inválido se proyecta inválido con cambio", blocks: "sdd:\n  iso: quizás\n", sealed: true, change: "demo",
			want:     ISOProjection{Project: "invalid", Change: "not_iso"},
			wantNote: "La declaración sdd: de axiom.yaml no es válida",
		},
		{
			name: "un sdd inválido se proyecta inválido sin cambio", blocks: "sdd:\n  iso: quizás\n",
			want:     ISOProjection{Project: "invalid"},
			wantNote: "mientras tanto ningún cambio puede archivarse",
		},
		{
			name: "la contradicción con workflow.lane se proyecta inválida", blocks: "sdd:\n  iso: true\nworkflow:\n  lane: odd\n",
			want:     ISOProjection{Project: "invalid"},
			wantNote: "workflow.lane",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ws workspacetest.Workspace
			if tt.change == "" {
				// Sin ningún cambio activo, el status no selecciona ninguno.
				ws = workspacetest.Embedded(t, ".")
				workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+tt.blocks)
			} else {
				ws = embeddedWithDeclaration(t, tt.blocks)
			}
			if tt.sealed {
				sealISOChange(t, filepath.Join(ws.Master, "openspec", "changes", "demo"), "demo", tt.snapshot)
			}
			status := resolveFrom(t, ws.Master, tt.change)
			got := isoOf(t, status)
			if tt.wantAbsent {
				if got != nil {
					t.Fatalf("iso = %+v, se esperaba la clave ausente", got)
				}
				return
			}
			if got == nil {
				t.Fatal("iso ausente, se esperaba la proyección")
			}
			if got.Project != tt.want.Project || got.Change != tt.want.Change || got.Matrix != nil {
				t.Errorf("iso = %+v, se esperaba %+v", *got, tt.want)
			}
			if (got.Project == "invalid") != (got.Reason != "") {
				t.Errorf("iso.reason = %q con project %q: solo se proyecta con un proyecto inválido", got.Reason, got.Project)
			}
			notes := strings.Join(status.Notes, "\n")
			if tt.wantNote != "" && !strings.Contains(notes, tt.wantNote) {
				t.Errorf("las notas no nombran %q: %v", tt.wantNote, status.Notes)
			}
			if tt.wantNote == "" && strings.Contains(notes, "difiere de la declaración vigente") {
				t.Errorf("no se esperaba nota de discrepancia: %v", status.Notes)
			}
			if strings.Contains(strings.Join(status.BlockedReasons, "\n"), "iso_declaration_invalid") && status.NextRecommended != "resolve-blockers" {
				t.Errorf("un motivo de archivado solo va a blockedReasons con la ruta archive: ruta %q", status.NextRecommended)
			}
		})
	}
}

// El contrato v1 interno y el v2 público proyectan lo mismo.
func TestISOProjectionV1AndV2Agree(t *testing.T) {
	ws := embeddedWithDeclaration(t, "sdd:\n  iso: true\n")
	status := resolveFrom(t, ws.Master, "demo")
	if status.ISO == nil || status.ISO.Project != "iso" || status.ISO.Change != "not_iso" {
		t.Fatalf("status.ISO = %+v", status.ISO)
	}
	if got := isoOf(t, status); got == nil || got.Project != status.ISO.Project || got.Change != status.ISO.Change {
		t.Fatalf("v2 iso = %+v, v1 iso = %+v", got, status.ISO)
	}
}

// REQ-29.2 y D8: una declaración inválida estrecha el archivado y solo pasa a
// blockedReasons cuando la ruta sería archive; con trabajo pendiente la ruta no
// cambia y el motivo queda en notes.
func TestDecorateISO(t *testing.T) {
	t.Run("ruta archive con declaración inválida", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: quizás\n")
		seedReadyChange(t, ws.Master, "demo", "- [x] 1.1 Hecho\n")
		status := resolveFrom(t, ws.Master, "demo")

		if status.Dependencies.Archive != DependencyBlocked {
			t.Errorf("dependencies.archive = %q, se esperaba blocked", status.Dependencies.Archive)
		}
		if status.NextRecommended != "resolve-blockers" {
			t.Errorf("nextRecommended = %q, se esperaba resolve-blockers (la ruta sería archive)", status.NextRecommended)
		}
		if len(reasonsWith(status.BlockedReasons, "blocked(iso_declaration_invalid):")) != 1 {
			t.Errorf("blockedReasons = %v, se esperaba un motivo iso_declaration_invalid", status.BlockedReasons)
		}
		if reason := reasonsWith(status.BlockedReasons, "iso_declaration_invalid"); len(reason) == 1 && !strings.Contains(reason[0], "axiom workspace validate --path") {
			t.Errorf("el motivo no nombra la continuación: %q", reason[0])
		}
		if len(reasonsWith(status.Notes, "mientras tanto ningún cambio puede archivarse")) != 1 {
			t.Errorf("notes = %v, se esperaba la nota de D8", status.Notes)
		}
	})

	t.Run("con trabajo pendiente la ruta no cambia", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: quizás\n")
		seedReadyChange(t, ws.Master, "demo", "- [ ] 1.1 Pendiente\n")
		status := resolveFrom(t, ws.Master, "demo")

		if status.NextRecommended != "apply" {
			t.Errorf("nextRecommended = %q, se esperaba apply: una errata de axiom.yaml no para la implementación", status.NextRecommended)
		}
		if status.Dependencies.Archive != DependencyBlocked {
			t.Errorf("dependencies.archive = %q, se esperaba blocked", status.Dependencies.Archive)
		}
		if len(status.BlockedReasons) != 0 {
			t.Errorf("blockedReasons = %v, apply no admite motivos", status.BlockedReasons)
		}
		if len(reasonsWith(status.Notes, "mientras tanto ningún cambio puede archivarse")) != 1 {
			t.Errorf("notes = %v, se esperaba la nota de D8", status.Notes)
		}
	})

	t.Run("sin cambio seleccionado no se estrecha nada", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: quizás\n")
		status := resolveFrom(t, ws.Master, "")
		if status.ChangeName != nil {
			t.Fatalf("changeName = %q, se esperaba ninguno", *status.ChangeName)
		}
		if got := isoOf(t, status); got == nil || got.Project != "invalid" || got.Reason == "" || got.Change != "" {
			t.Fatalf("iso = %+v, se esperaba invalid con reason y sin change", got)
		}
	})

	t.Run("almacén sin ficheros con un cambio ISO", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: true\n")
		changeRoot := seedReadyChange(t, ws.Master, "demo", "- [x] 1.1 Hecho\n")
		declareArtifactStore(t, ws.Master, "none")
		sealISOChange(t, changeRoot, "demo", &kickoff.SDDSnapshot{ISO: true})
		status := resolveFrom(t, ws.Master, "demo")

		if status.Dependencies.Archive != DependencyBlocked {
			t.Errorf("dependencies.archive = %q, se esperaba blocked", status.Dependencies.Archive)
		}
		reasons := reasonsWith(status.BlockedReasons, "blocked(iso_store_unsupported):")
		if len(reasons) != 1 || !strings.Contains(reasons[0], "artifact_store: openspec") {
			t.Errorf("blockedReasons = %v, se esperaba iso_store_unsupported con su continuación", status.BlockedReasons)
		}
		if len(reasonsWith(status.Notes, "blocked(iso_store_unsupported):")) != 1 {
			t.Errorf("notes = %v, se esperaba el motivo también en notes", status.Notes)
		}
	})

	t.Run("almacén sin ficheros con un cambio no ISO no se estrecha", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: true\n")
		changeRoot := seedReadyChange(t, ws.Master, "demo", "- [x] 1.1 Hecho\n")
		declareArtifactStore(t, ws.Master, "none")
		sealISOChange(t, changeRoot, "demo", nil)
		status := resolveFrom(t, ws.Master, "demo")
		if len(reasonsWith(status.BlockedReasons, "iso_store_unsupported")) != 0 {
			t.Errorf("un cambio no ISO no se bloquea por el almacén: %v", status.BlockedReasons)
		}
	})

	t.Run("las instrucciones de fase conservan la línea de integración", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+declThreeDocuments+"sdd:\n  iso: quizás\n")
		seedReadyChange(t, ws.Master, "demo", "- [x] 1.1 Hecho\n")
		status, err := Resolve(ResolveOptions{CWD: ws.Master, ChangeName: "demo", IncludeInstructions: true})
		if err != nil {
			t.Fatal(err)
		}
		if status.PhaseInstructions == nil {
			t.Fatal("se esperaban instrucciones de fase")
		}
		if len(reasonsWith(status.PhaseInstructions.Archive, "axiom sdd integrate prepare")) == 0 {
			t.Errorf("phaseInstructions.archive = %v, debe conservar la línea de la integración", status.PhaseInstructions.Archive)
		}
	})
}
