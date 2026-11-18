package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Identidad git como actor de los artefactos versionados (capacidad change-actor-identity):
// `gate record` registra la identidad como `actor` y `kickoff seal` como `sealed_by`. Sin
// identidad completa se conserva `cli` y se avisa por stderr; nunca se bloquea.

// resolveActorIdentity es la costura de prueba sobre gitidentity.Resolve.
var resolveActorIdentity = gitidentity.Resolve

// actorIdentityWarnings recibe el aviso de identidad git incompleta. Es propio: no comparte
// writer con los avisos de rama de seal ni de los escritores, cuyas pruebas afirman cero avisos.
var actorIdentityWarnings io.Writer = os.Stderr

// actorTarget nombra el campo que se rellena y si el verbo admite --actor.
type actorTarget struct {
	field     string // "actor" o "sealed_by"
	actorFlag bool   // true en gate record: el aviso sugiere --actor
}

var (
	gateActorTarget = actorTarget{field: "actor", actorFlag: true}
	sealActorTarget = actorTarget{field: "sealed_by"}
)

// defaultActor resuelve la identidad git desde workspaceRoot. Devuelve el actor
// ("Nombre <email>" o kickoff.FallbackActor) y el aviso que hay que emitir si el registro llega
// a escribirse ("" con identidad completa).
func defaultActor(workspaceRoot string, target actorTarget) (actor, warning string) {
	identity := resolveActorIdentity(context.Background(), workspaceRoot)
	if actor, ok := identity.Actor(); ok {
		return actor, ""
	}
	return kickoff.FallbackActor, actorIdentityWarning(workspaceRoot, identity, target)
}

// actorIdentityWarning compone la línea de aviso (D5): nombra la raíz de la que se leyó, las
// claves que faltan y las órdenes que las configuran.
func actorIdentityWarning(workspaceRoot string, identity gitidentity.Identity, target actorTarget) string {
	missing := identity.Missing()
	var missingText string
	if len(missing) == 1 {
		missingText = "falta " + missing[0]
	} else {
		missingText = "faltan " + strings.Join(missing, " y ")
	}
	commands := make([]string, 0, len(missing))
	for _, key := range missing {
		placeholder := "<nombre>"
		if key == gitidentity.KeyEmail {
			placeholder = "<email>"
		}
		commands = append(commands, fmt.Sprintf("git config --global %s %q", key, placeholder))
	}
	hint := "Configúrala con " + strings.Join(commands, " y ")
	if target.actorFlag {
		hint += ", o indica --actor"
	}
	return fmt.Sprintf("Aviso: la identidad git no está completa en %s (%s): se registra %s %q. %s.",
		pathquote.Quote(workspaceRoot), missingText, target.field, kickoff.FallbackActor, hint)
}

// emitActorIdentityWarning escribe warning en actorIdentityWarnings si no está vacío (una
// línea). El aviso es informativo: un fallo al escribirlo nunca falla el verbo.
func emitActorIdentityWarning(warning string) {
	if warning == "" {
		return
	}
	_, _ = fmt.Fprintln(actorIdentityWarnings, warning)
}
