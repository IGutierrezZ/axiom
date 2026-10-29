#!/usr/bin/env bash
# scripts/axiom-worktree.sh — Envoltorio local de `axiom branch` para desarrollar Axiom.
#
# Una sola implementación de ramas y worktrees: la del producto (`axiom branch`). Este script
# aporta la costumbre local y el verbo `pr`, que el producto deja fuera (la publicación es una
# decisión de la persona):
#
#   new  <slug> [prefijo]  axiom branch ensure --cwd <checkout principal> --change <slug> [--prefix p] --json
#   pr   <slug>            push de la rama del cambio y PR hacia el prTarget de axiom branch status --json
#   done <slug> [--force]  axiom branch done --cwd <checkout principal> --change <slug> y, después,
#                          el avance rápido de main
#
# Los worktrees viven en <carpeta del proyecto>-wt/<slug>/<repositorio> (p. ej. axiom-wt/<slug>/axiom).
# Necesita un `axiom` con el verbo `axiom branch` en el PATH (`go install ./cmd/axiom`).
set -euo pipefail

VERB="${1:-}"
SLUG="${2:-}"
PREFIX=""
FORCE=0

usage() {
    echo "Uso: $0 {new|pr|done} <slug> [feat|inc|fix|refactor] [--force]"
    exit 1
}

if [ -z "$VERB" ] || [ -z "$SLUG" ]; then
    usage
fi

# El tercer argumento posicional, si no es una bandera, es el prefijo de la rama.
if [ -n "${3:-}" ] && [ "${3#-}" = "$3" ]; then
    PREFIX="$3"
fi

for arg in "$@"; do
    if [ "$arg" = "--force" ] || [ "$arg" = "-f" ]; then
        FORCE=1
    fi
done

case "$VERB" in
    new|pr|done) ;;
    *) echo "ERROR: Verbo desconocido '$VERB'. Usa 'new', 'pr' o 'done'." >&2; exit 1 ;;
esac

if ! [[ "$SLUG" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
    echo "ERROR: Slug inválido '$SLUG'. Usa kebab-case en minúsculas (ej. worktree-and-pr-governance)." >&2
    exit 1
fi

if [ -n "$PREFIX" ] && ! [[ "$PREFIX" =~ ^(feat|inc|fix|refactor)$ ]]; then
    echo "ERROR: Prefijo inválido '$PREFIX'. Usa feat, inc, fix o refactor." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

if ! git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "ERROR: No es un repositorio git: $REPO_ROOT" >&2
    exit 1
fi

# El checkout principal es el padre del git common dir: desde un worktree, REPO_ROOT ya no lo es.
COMMON_DIR="$(git -C "$REPO_ROOT" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
if [ -z "$COMMON_DIR" ]; then
    CD_REL="$(git -C "$REPO_ROOT" rev-parse --git-common-dir 2>/dev/null || true)"
    if [ -n "$CD_REL" ]; then
        COMMON_DIR="$(cd "$REPO_ROOT/$CD_REL" && pwd)"
    else
        COMMON_DIR="$REPO_ROOT/.git"
    fi
fi
MAIN_ROOT="$(cd "$COMMON_DIR/.." && pwd)"

# Sin un `axiom` que traiga `axiom branch` no hay nada que envolver.
if ! command -v axiom >/dev/null 2>&1 || ! axiom branch --help >/dev/null 2>&1; then
    printf 'ERROR: instala un `axiom` con `axiom branch` (`go install ./cmd/axiom`).\n' >&2
    exit 1
fi

# Valor de una clave de texto del JSON de axiom (salida con sangría fija), sin los escapes `\\`.
json_paths() {
    sed -n "s/^ *\"$1\": \"\(.*\)\",\{0,1\}\$/\1/p" | sed 's#\\\\#\\#g'
}

case "$VERB" in
    new)
        PREFIX_ARGS=()
        if [ -n "$PREFIX" ]; then
            PREFIX_ARGS=(--prefix "$PREFIX")
        fi

        # `--mode worktree`: la política local exige un worktree propio y un checkout principal
        # siempre en main, sea cual sea la preferencia personal del hub.
        if ! RESULT="$(axiom branch ensure --cwd "$MAIN_ROOT" --change "$SLUG" --mode worktree ${PREFIX_ARGS[@]+"${PREFIX_ARGS[@]}"} --json)"; then
            echo "$RESULT" >&2
            echo "ERROR: axiom branch ensure rechazó el cambio '$SLUG'." >&2
            exit 1
        fi

        WORKTREES="$(printf '%s\n' "$RESULT" | json_paths worktree)"
        if [ -z "$WORKTREES" ]; then
            echo "ERROR: axiom branch ensure no devolvió ningún worktree para '$SLUG'." >&2
            exit 1
        fi
        while IFS= read -r path; do
            echo "[OK] Worktree: $path"
        done <<< "$WORKTREES"
        echo "[INFO] Trabaja con rutas absolutas dentro de esos directorios (--cwd y git -C)."
        echo "[INFO] Al verificar, ejecuta: scripts/axiom-worktree.sh pr $SLUG"
        ;;

    pr)
        STATUS="$(axiom branch status --cwd "$MAIN_ROOT" --change "$SLUG" --json)"
        # Rama del cambio, su worktree y la base del PR (`prTarget`) salen del plan del producto.
        CHANGE_BRANCH="$(printf '%s\n' "$STATUS" | awk '
            /^          "name": "/    { name=$0; sub(/^ *"name": "/, "", name); sub(/",?$/, "", name); kind=""; wt="" }
            /^          "kind": "/    { kind=$0; sub(/^ *"kind": "/, "", kind); sub(/",?$/, "", kind) }
            /^          "worktree": "/ { wt=$0; sub(/^ *"worktree": "/, "", wt); sub(/",?$/, "", wt); gsub(/\\\\/, "\\", wt) }
            /^          "prTarget": "/ {
                target=$0; sub(/^ *"prTarget": "/, "", target); sub(/",?$/, "", target)
                if (kind == "change") { print name "\001" wt "\001" target; exit }
            }')"
        if [ -z "$CHANGE_BRANCH" ]; then
            echo "ERROR: axiom branch status no encontró la rama del cambio '$SLUG'. Créala con: scripts/axiom-worktree.sh new $SLUG" >&2
            exit 1
        fi
        # El separador \001 no es un espacio en blanco de IFS: un worktree vacío no desplaza los campos.
        IFS=$'\001' read -r BRANCH WT_PATH PR_TARGET <<< "$CHANGE_BRANCH"
        if [ -z "$WT_PATH" ] || [ ! -d "$WT_PATH" ]; then
            echo "ERROR: La rama $BRANCH no tiene worktree en este equipo (worktree: '${WT_PATH:-ninguno}')." >&2
            exit 1
        fi
        if [ -z "$PR_TARGET" ]; then
            echo "ERROR: axiom branch status no propuso una rama destino (prTarget) para $BRANCH." >&2
            exit 1
        fi

        DIRTY="$(git -C "$WT_PATH" status --porcelain)"
        if [ -n "$DIRTY" ]; then
            echo "ERROR: El worktree tiene cambios sin commitear. Haz commit de todo antes de crear el PR:" >&2
            echo "$DIRTY" >&2
            exit 1
        fi

        echo "[INFO] Subiendo $BRANCH a origin..."
        git -C "$WT_PATH" push -u origin "$BRANCH"

        REMOTE_URL="$(git -C "$WT_PATH" remote get-url origin | sed 's/\.git$//')"
        COMPARE_URL="$REMOTE_URL/compare/$PR_TARGET...$BRANCH?expand=1"
        # HOST/OWNER/REPO de origin (https o ssh) para no depender de 'gh repo set-default'.
        GH_REPO="$(printf '%s' "$REMOTE_URL" | sed -E 's#^([a-zA-Z][a-zA-Z0-9+.-]*://)?([^@/]+@)?([^/:]+)[:/]#\3/#')"

        if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
            echo "[INFO] Creando Pull Request mediante gh CLI (base: $PR_TARGET)..."
            (cd "$WT_PATH" && gh pr create --repo "$GH_REPO" --base "$PR_TARGET" --head "$BRANCH" --fill)
            echo "[OK] Pull Request creado exitosamente."
        else
            echo "[AVISO] gh CLI no disponible o no autenticado. Abre el PR manualmente en GitHub:"
            echo "$COMPARE_URL"
        fi
        ;;

    done)
        FORCE_ARGS=()
        if [ "$FORCE" -eq 1 ]; then
            FORCE_ARGS=(--force)
        fi

        # `axiom branch done` confirma la fusión y desmonta el worktree y la rama local del plan.
        if ! axiom branch done --cwd "$MAIN_ROOT" --change "$SLUG" ${FORCE_ARGS[@]+"${FORCE_ARGS[@]}"}; then
            echo "ERROR: axiom branch done no desmanteló por completo el cambio '$SLUG' (revisa el mensaje anterior)." >&2
            exit 1
        fi

        CURRENT_BRANCH="$(git -C "$MAIN_ROOT" rev-parse --abbrev-ref HEAD)"
        if [ "$CURRENT_BRANCH" = "main" ]; then
            echo "[INFO] Actualizando main local con origin/main..."
            git -C "$MAIN_ROOT" fetch origin main 2>/dev/null || true
            git -C "$MAIN_ROOT" merge --ff-only origin/main 2>/dev/null || echo "[AVISO] No se pudo hacer ff-only en main."
        fi

        git -C "$MAIN_ROOT" worktree prune
        echo "[OK] Ciclo de trabajo '$SLUG' desmantelado y entorno limpio."
        ;;
esac
