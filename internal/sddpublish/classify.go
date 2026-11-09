package sddpublish

import (
	"fmt"
	"strings"
	"time"
)

// Clasificación de las negativas de `git commit` y `git push` (DD-09, paso 10). Son funciones
// puras sobre la salida de git (`--porcelain` y `stderr`, con `LC_ALL=C`): lo que decide el
// código de la negativa se prueba con entradas fijas.

// unavailableSignals son las señales de `stderr` de un remoto inalcanzable o sin credenciales.
var unavailableSignals = []string{
	"Could not read from remote repository",
	"does not appear to be a git repository",
	"unable to access",
	"Authentication failed",
	"could not read Username",
}

// identitySignals son las señales de `stderr` de un `git commit` sin identidad.
var identitySignals = []string{
	"Author identity unknown",
	"unable to auto-detect email address",
	"empty ident name",
	"Please tell me who you are",
}

// stderrTail devuelve las últimas líneas de lo que dijo git, para que la negativa conserve el
// motivo sin volcar toda la salida.
func stderrTail(stderr string) string {
	const lines = 8
	all := splitLines(stderr)
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.Join(all, " | ")
}

// classifyCommit clasifica el fallo de `git commit` (paso 7): un hook de commit presente que
// rechaza es `commit_hook_rejected`; la falta de identidad, `identity_missing`; cualquier otro
// fallo, `commit_failed` con la cola de `stderr`. Las rutas pudieron quedar preparadas: es el
// estado que deja git y la negativa lo dice.
func classifyCommit(failure *CommandError, hookPresent bool) *Refusal {
	const staged = "las rutas de documentación pueden haber quedado preparadas en el índice"
	for _, signal := range identitySignals {
		if strings.Contains(failure.Stderr, signal) {
			return refuse(CodeIdentityMissing,
				"git no tiene identidad con la que firmar el commit de los documentos; no se ha confirmado nada",
				"configura `git config user.name` y `git config user.email` y repite la orden", nil)
		}
	}
	tail := stderrTail(failure.Stderr)
	if hookPresent {
		return refuse(CodeCommitHookRejected,
			fmt.Sprintf("un hook de git rechazó el commit de los documentos (código %d): %s; no se ha confirmado nada y %s", failure.ExitCode, tail, staged),
			"corrige lo que pide el hook, o confirma tú los documentos con `git commit`, y repite la orden",
			map[string]any{"exitCode": failure.ExitCode, "stderr": tail})
	}
	return refuse(CodeCommitFailed,
		fmt.Sprintf("el commit de los documentos falló (código %d): %s; no se ha confirmado nada y %s", failure.ExitCode, tail, staged),
		"revisa el estado con `git status` y repite la orden",
		map[string]any{"exitCode": failure.ExitCode, "stderr": tail})
}

// classifyPush clasifica el resultado de `git push` con código distinto de cero o con tiempo
// agotado (paso 10). prePushHook dice si hay un hook `pre-push` local. El orden es el de la
// salida de git: primero las líneas `!` de `--porcelain` (el remoto contestó), después un remoto
// inalcanzable (código 128 con sus señales o el tiempo máximo) y, sin línea de estado de la
// referencia, un hook `pre-push` local que rechazó. Ninguna negativa reintenta, hace `fetch` ni
// fuerza.
func classifyPush(outcome PushOutcome, prePushHook bool, timeout time.Duration, branch string) *Refusal {
	if outcome.TimedOut {
		return refuse(CodeRemoteUnavailable,
			fmt.Sprintf("el push de %s a origin superó el tiempo máximo de %s", branch, timeout),
			"comprueba la conexión y las credenciales del remoto y repite la orden; el commit de los documentos se conserva en local",
			map[string]any{"timeout": timeout.String()})
	}
	tail := stderrTail(outcome.Stderr)
	for _, line := range splitLines(outcome.Stdout) {
		if !strings.HasPrefix(line, "!") {
			continue
		}
		switch {
		case strings.Contains(line, "[remote rejected]"):
			return refuse(CodePushHookRejected,
				fmt.Sprintf("el remoto rechazó el push de %s (un hook del servidor o una rama protegida): %s", branch, tail),
				"revisa la protección de la rama o el hook del servidor; Axiom no fuerza el push ni se salta hooks",
				map[string]any{"stderr": tail})
		case strings.Contains(line, "[rejected]") && (strings.Contains(line, "(non-fast-forward)") || strings.Contains(line, "(fetch first)")):
			return refuse(CodePushNonFastForward,
				fmt.Sprintf("el remoto tiene commits de %s que no están en la rama local: el push no es un avance rápido", branch),
				fmt.Sprintf("trae el remoto con `git fetch origin` e intégralo (`git rebase origin/%s` o `git merge origin/%s`), y repite la orden; Axiom no hace fetch ni fuerza el push", branch, branch),
				nil)
		}
		return refuse(CodePushFailed,
			fmt.Sprintf("el push de %s falló: %s", branch, strings.TrimSpace(line)),
			"revisa el estado del remoto y repite la orden",
			map[string]any{"stderr": tail})
	}
	if outcome.ExitCode == 128 {
		for _, signal := range unavailableSignals {
			if strings.Contains(outcome.Stderr, signal) {
				return refuse(CodeRemoteUnavailable,
					fmt.Sprintf("origin no está disponible o no hay credenciales para subir %s: %s", branch, tail),
					"comprueba la URL de `origin` y las credenciales de git ya configuradas y repite la orden; el commit de los documentos se conserva en local",
					map[string]any{"stderr": tail})
			}
		}
	}
	if prePushHook {
		return refuse(CodePushHookRejected,
			fmt.Sprintf("el hook pre-push rechazó el push de %s (código %d): %s", branch, outcome.ExitCode, tail),
			"corrige lo que pide el hook y repite la orden; Axiom no se salta hooks",
			map[string]any{"exitCode": outcome.ExitCode, "stderr": tail})
	}
	return refuse(CodePushFailed,
		fmt.Sprintf("el push de %s falló (código %d): %s", branch, outcome.ExitCode, tail),
		"revisa el estado del remoto y repite la orden; el commit de los documentos se conserva en local",
		map[string]any{"exitCode": outcome.ExitCode, "stderr": tail})
}
