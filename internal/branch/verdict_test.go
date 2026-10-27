package branch

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Pruebas del veredicto único de rama (REQ-25.18, design D-10): `Decide` es puro y se
// prueba con tablas, sin git ni disco. Cubre los escenarios de la spec: el papel del
// repositorio, el contexto ordinario y el de archivado, los cuatro consultantes y las
// exenciones.

const verdictMaster = "/proyecto"

// monorepoInput es la consulta de un repositorio con rama de cambio (monorepo embebido) del
// cambio `x` con la política por defecto, parado en current.
func monorepoInput(current string) VerdictInput {
	return VerdictInput{
		Role:       RepoChangeBranch,
		Repo:       "proyecto",
		Change:     "x",
		Lane:       LaneSDD,
		Context:    ContextOrdinary,
		Consultant: ConsultantWrite,
		Expected:   ExpectedNames{Change: "feat/x", Archive: "feat/x-archive", Integration: "main"},
		Current:    current,
		Master:     verdictMaster,
	}
}

// withRoles añade las ramas de rol `feat/x-core` y `feat/x-qa` a la consulta.
func withRoles(in VerdictInput) VerdictInput {
	in.Expected.Roles = []string{"feat/x-core", "feat/x-qa"}
	return in
}

func wantVerdict(t *testing.T, got Verdict, kind VerdictKind) {
	t.Helper()
	if got.Kind != kind {
		t.Fatalf("veredicto %q (%s), se esperaba %q", got.Kind, got.Detail, kind)
	}
	if kind != VerdictWrongBranch && got.Continuation != "" {
		t.Errorf("solo un wrong_branch lleva continuación, pero hay %q", got.Continuation)
	}
	if kind == VerdictWrongBranch && (got.Continuation == "" || got.Detail == "") {
		t.Errorf("todo wrong_branch lleva detalle y continuación: %+v", got)
	}
}

// REQ-25.18, escenario «Rama del cambio en un monorepo es correcta».
func TestDecideTheChangeBranchIsCorrectInAMonorepo(t *testing.T) {
	verdict := Decide(monorepoInput("feat/x"))
	wantVerdict(t, verdict, VerdictCorrect)
	if verdict.Repo != "proyecto" || verdict.Current != "feat/x" || verdict.Detached {
		t.Errorf("el veredicto no describe el repositorio y su rama: %+v", verdict)
	}
}

// REQ-25.18, escenario «La rama de integración es incorrecta para un cambio en monorepo»:
// nombra la rama actual y la esperada y da la orden exacta de `ensure`.
func TestDecideTheIntegrationBranchIsWrongInAMonorepo(t *testing.T) {
	verdict := Decide(monorepoInput("main"))
	wantVerdict(t, verdict, VerdictWrongBranch)
	if verdict.Current != "main" || !reflect.DeepEqual(verdict.Expected, []string{"feat/x"}) {
		t.Errorf("rama actual %q y esperada %v, se esperaba main y [feat/x]", verdict.Current, verdict.Expected)
	}
	if want := `axiom branch ensure --cwd "/proyecto" --change x`; verdict.Continuation != want {
		t.Errorf("continuación = %q, se esperaba %q", verdict.Continuation, want)
	}
	for _, fragment := range []string{"proyecto", "«main»", "«feat/x»"} {
		if !strings.Contains(verdict.Detail, fragment) {
			t.Errorf("el detalle %q debe nombrar %s", verdict.Detail, fragment)
		}
	}
}

// REQ-25.18, escenario «Una rama de rol es correcta cuando hay ramas de rol»: también lo es
// la de seguimiento, la del cambio.
func TestDecideARoleBranchIsCorrectWhenThereAreRoleBranches(t *testing.T) {
	for _, current := range []string{"feat/x-qa", "feat/x-core", "feat/x"} {
		wantVerdict(t, Decide(withRoles(monorepoInput(current))), VerdictCorrect)
	}
	wrong := Decide(withRoles(monorepoInput("feat/x-otro")))
	wantVerdict(t, wrong, VerdictWrongBranch)
	if want := []string{"feat/x", "feat/x-core", "feat/x-qa"}; !reflect.DeepEqual(wrong.Expected, want) {
		t.Errorf("ramas esperadas = %v, se esperaban %v (la del cambio primero)", wrong.Expected, want)
	}
}

// REQ-25.18, escenario «La rama de archivado es incorrecta en el contexto ordinario».
func TestDecideTheArchiveBranchIsWrongInTheOrdinaryContext(t *testing.T) {
	verdict := Decide(monorepoInput("feat/x-archive"))
	wantVerdict(t, verdict, VerdictWrongBranch)
	if !reflect.DeepEqual(verdict.Expected, []string{"feat/x"}) {
		t.Errorf("rama esperada = %v, se esperaba [feat/x]", verdict.Expected)
	}
	if strings.Contains(verdict.Continuation, "--archive") {
		t.Errorf("el contexto ordinario no pide --archive: %q", verdict.Continuation)
	}
}

// REQ-25.18, escenarios «En el contexto de archivado solo la rama de archivado es correcta»
// y «...la rama del cambio, la de integración y las de rol son incorrectas».
func TestDecideInTheArchiveContextOnlyTheArchiveBranchIsCorrect(t *testing.T) {
	archive := func(current string) VerdictInput {
		in := withRoles(monorepoInput(current))
		in.Context = ContextArchive
		return in
	}
	wantVerdict(t, Decide(archive("feat/x-archive")), VerdictCorrect)

	for _, current := range []string{"feat/x", "main", "feat/x-core", "feat/ajena"} {
		t.Run(current, func(t *testing.T) {
			verdict := Decide(archive(current))
			wantVerdict(t, verdict, VerdictWrongBranch)
			if !reflect.DeepEqual(verdict.Expected, []string{"feat/x-archive"}) {
				t.Errorf("rama esperada = %v, se esperaba [feat/x-archive]", verdict.Expected)
			}
			if want := `axiom branch ensure --cwd "/proyecto" --change x --archive`; verdict.Continuation != want {
				t.Errorf("continuación = %q, se esperaba %q", verdict.Continuation, want)
			}
		})
	}
}

// REQ-25.18, escenario «Un HEAD desacoplado es incorrecto»: en los dos contextos, aunque la
// rama actual que se pase coincida con una esperada.
func TestDecideADetachedHeadIsWrong(t *testing.T) {
	for _, context := range []Context{ContextOrdinary, ContextArchive} {
		in := monorepoInput("")
		in.Context, in.Detached = context, true
		verdict := Decide(in)
		wantVerdict(t, verdict, VerdictWrongBranch)
		if !verdict.Detached || !strings.Contains(verdict.Detail, "desacoplado") {
			t.Errorf("contexto %s: el detalle debe decir que el HEAD está desacoplado: %+v", context, verdict)
		}
	}
	in := monorepoInput("feat/x")
	in.Detached = true
	wantVerdict(t, Decide(in), VerdictWrongBranch)
}

// REQ-25.18, escenario «El repositorio de specs de un multirepo exige su rama de
// integración»: también en el contexto de archivado, donde no hay rama de archivado, y una
// rama de rol o la del cambio son incorrectas.
func TestDecideTheSpecsRepositoryOfAMultirepoRequiresItsIntegrationBranch(t *testing.T) {
	specs := func(current string, context Context) VerdictInput {
		in := withRoles(monorepoInput(current))
		in.Role, in.Repo, in.Context = RepoSpecsInPlace, "repo-specs", context
		return in
	}
	for _, context := range []Context{ContextOrdinary, ContextArchive} {
		wantVerdict(t, Decide(specs("main", context)), VerdictCorrect)
		for _, current := range []string{"feat/x-core", "feat/x", "feat/x-archive"} {
			verdict := Decide(specs(current, context))
			wantVerdict(t, verdict, VerdictWrongBranch)
			if !reflect.DeepEqual(verdict.Expected, []string{"main"}) {
				t.Errorf("contexto %s, rama %s: esperada %v, se esperaba [main]", context, current, verdict.Expected)
			}
			if strings.Contains(verdict.Continuation, "--archive") {
				t.Errorf("el repositorio de specs no tiene rama de archivado: %q", verdict.Continuation)
			}
		}
	}
	detached := specs("", ContextOrdinary)
	detached.Detached = true
	wantVerdict(t, Decide(detached), VerdictWrongBranch)
}

// REQ-25.18, escenario «El repositorio de código de un multirepo solo se evalúa en apply»:
// con cualquier otro consultante es `not_applicable`.
func TestDecideACodeRepositoryOnlyAnswersToTheApplyConsultant(t *testing.T) {
	code := func(current string, consultant Consultant) VerdictInput {
		in := monorepoInput(current)
		in.Role, in.Repo, in.Consultant = RepoCode, "repo-backend", consultant
		return in
	}
	for _, consultant := range []Consultant{ConsultantWrite, ConsultantStatus, ConsultantGuard} {
		verdict := Decide(code("main", consultant))
		wantVerdict(t, verdict, VerdictNotApplicable)
		if verdict.Detail == "" {
			t.Errorf("consultante %s: un not_applicable explica por qué", consultant)
		}
	}
	wantVerdict(t, Decide(code("main", ConsultantApply)), VerdictWrongBranch)
	wantVerdict(t, Decide(code("feat/x", ConsultantApply)), VerdictCorrect)
	wantVerdict(t, Decide(withRoles(code("feat/x-qa", ConsultantApply))), VerdictCorrect)

	// En el contexto de archivado un repositorio de código sigue exigiendo la rama del cambio
	// o la de rol: no tiene rama de archivado.
	archive := code("feat/x", ConsultantApply)
	archive.Context = ContextArchive
	wantVerdict(t, Decide(archive), VerdictCorrect)
}

// REQ-25.18: los cuatro consultantes dan la misma respuesta para un repositorio con rama de
// cambio; solo el de código distingue a `apply`.
func TestDecideAnswersTheSameToEveryConsultantInARepositoryWithAChangeBranch(t *testing.T) {
	for _, consultant := range []Consultant{ConsultantWrite, ConsultantStatus, ConsultantApply, ConsultantGuard} {
		for current, want := range map[string]VerdictKind{"feat/x": VerdictCorrect, "main": VerdictWrongBranch} {
			in := monorepoInput(current)
			in.Consultant = consultant
			if got := Decide(in).Kind; got != want {
				t.Errorf("consultante %s en %s: %s, se esperaba %s", consultant, current, got, want)
			}
		}
	}
}

// REQ-25.18 y REQ-25.37: un cambio sin instantánea, los sumideros acumulativos y una ruta
// fuera de git quedan exentos: `not_applicable`, con el motivo, sin continuación y aunque la
// rama sea la equivocada.
func TestDecideAnExemptChangeIsNotApplicableWhateverTheBranch(t *testing.T) {
	for _, current := range []string{"main", "feat/x", "feat/x-archive"} {
		in := monorepoInput(current)
		in.Exempt = "el cambio no tiene instantánea de política git"
		verdict := Decide(in)
		wantVerdict(t, verdict, VerdictNotApplicable)
		if verdict.Detail != in.Exempt {
			t.Errorf("el detalle = %q, se esperaba el motivo de la exención %q", verdict.Detail, in.Exempt)
		}
	}
}

// El carril ODD no tiene archive: el contexto de archivado se evalúa como el ordinario y la
// continuación repite el carril.
func TestDecideTheODDLaneHasNoArchiveContextAndRepeatsTheLane(t *testing.T) {
	odd := func(current string, context Context) VerdictInput {
		in := monorepoInput(current)
		in.Lane, in.Context = LaneODD, context
		in.Expected.Archive = ""
		return in
	}
	wantVerdict(t, Decide(odd("feat/x", ContextArchive)), VerdictCorrect)
	verdict := Decide(odd("main", ContextArchive))
	wantVerdict(t, verdict, VerdictWrongBranch)
	if want := `axiom branch ensure --cwd "/proyecto" --change x --lane odd`; verdict.Continuation != want {
		t.Errorf("continuación = %q, se esperaba %q", verdict.Continuation, want)
	}
}

// Sin rama de archivado conocida, el contexto de archivado no tiene ninguna rama correcta.
func TestDecideTheArchiveContextWithoutAnArchiveBranchHasNoCorrectBranch(t *testing.T) {
	in := monorepoInput("feat/x")
	in.Context = ContextArchive
	in.Expected.Archive = ""
	verdict := Decide(in)
	wantVerdict(t, verdict, VerdictWrongBranch)
	if len(verdict.Expected) != 0 {
		t.Errorf("ramas esperadas = %v, no hay ninguna", verdict.Expected)
	}
}

// La continuación cita la carpeta maestra como una ruta (pathquote.Quote): con espacios sigue
// siendo una sola palabra; sin carpeta maestra queda la plantilla.
func TestDecideQuotesTheMasterFolderInTheContinuation(t *testing.T) {
	in := monorepoInput("main")
	in.Master = `C:\mis proyectos\axiom`
	if want := `axiom branch ensure --cwd "C:\mis proyectos\axiom" --change x`; Decide(in).Continuation != want {
		t.Errorf("continuación = %q, se esperaba %q", Decide(in).Continuation, want)
	}
	in.Master = ""
	if want := `axiom branch ensure --cwd <ruta> --change x`; Decide(in).Continuation != want {
		t.Errorf("sin carpeta maestra = %q, se esperaba %q", Decide(in).Continuation, want)
	}
}

// Decide es puro: no altera su entrada y el veredicto no comparte memoria con ella.
func TestDecideDoesNotAliasOrMutateItsInput(t *testing.T) {
	in := withRoles(monorepoInput("main"))
	before := withRoles(monorepoInput("main"))
	verdict := Decide(in)
	if !reflect.DeepEqual(in, before) {
		t.Errorf("Decide alteró su entrada: %+v", in)
	}
	verdict.Expected[0] = "otra"
	if in.Expected.Change != "feat/x" || in.Expected.Roles[0] != "feat/x-core" {
		t.Errorf("el veredicto comparte memoria con la entrada: %+v", in.Expected)
	}
}

// REQ-25.19: el rechazo `wrong_branch` lleva el código tipado, la rama actual, la esperada,
// el repositorio y la continuación exacta, y se detecta con errors.As.
func TestWrongBranchRefusalNamesTheTypedCodeAndTheContinuation(t *testing.T) {
	verdict := Decide(monorepoInput("main"))
	refusal := WrongBranch(verdict, "x")
	if refusal.Code != CodeWrongBranch || !strings.HasPrefix(refusal.Message, "wrong_branch: ") {
		t.Errorf("código %q y mensaje %q: se esperaba wrong_branch nombrado en el mensaje", refusal.Code, refusal.Message)
	}
	if !strings.Contains(refusal.Continuation, verdict.Continuation) {
		t.Errorf("la continuación %q debe contener la orden %q", refusal.Continuation, verdict.Continuation)
	}
	want := map[string]string{"change": "x", "repository": "proyecto", "current": "main", "expected": "feat/x", "detached": "false"}
	if !reflect.DeepEqual(refusal.Details, want) {
		t.Errorf("detalles = %v, se esperaban %v", refusal.Details, want)
	}

	var err error = &RefusalError{Refusal: *refusal}
	var typed *RefusalError
	if !errors.As(err, &typed) || typed.Code != CodeWrongBranch {
		t.Fatalf("errors.As no recupera el rechazo tipado de %v", err)
	}
	if got, want := err.Error(), refusal.Message+". "+refusal.Continuation; got != want {
		t.Errorf("Error() = %q, se esperaba mensaje + continuación %q", got, want)
	}
}
