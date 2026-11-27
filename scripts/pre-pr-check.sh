#!/usr/bin/env bash
# Comprobación previa a la PR: lo que la CI comprueba sobre todo el repositorio
# y la regla de pruebas localizadas no ejecuta en local. La lanza
# `scripts/axiom-worktree.{ps1,sh} pr` antes de subir la rama, y también se
# puede ejecutar a mano desde cualquier worktree.
#
#   1. gofmt de todos los .go, con la misma selección que `go run ./internal/gofmtcheck` en la CI.
#   2. Las guardas transversales de scripts/pre-pr-guards.txt, por nombre.
#   3. El deadcode ratchet (scripts/deadcode-ratchet.sh), con el deadcode instalado.
#
# Ejecuta los tres pasos aunque falle uno, para mostrar todo lo que hay que
# corregir de una vez. No sustituye a la CI: ni la batería completa ni las
# pruebas de plataforma.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"
log_dir="$(mktemp -d)"
trap 'rm -rf "${log_dir}"' EXIT
failed=0

echo "[1/3] gofmt"
if ! unformatted="$(git ls-files --cached --others --exclude-standard -z -- '*.go' | xargs -0 gofmt -l 2>&1)"; then
  printf '%s\n' "${unformatted}" >&2
  echo "gofmt no pudo analizar algún fichero." >&2
  failed=1
elif [[ -n "${unformatted}" ]]; then
  printf 'Ficheros sin el formato de gofmt (corrígelos con gofmt -w):\n%s\n' "${unformatted}" >&2
  failed=1
fi

echo "[2/3] guardas transversales (scripts/pre-pr-guards.txt)"
packages=()
names=()
while read -r pkg name _ || [[ -n "${pkg:-}" ]]; do
  pkg="${pkg%$'\r'}"
  name="${name%$'\r'}"
  [[ -z "${pkg}" || "${pkg}" == \#* ]] && continue
  packages+=("${pkg}")
  names+=("${name}")
done < scripts/pre-pr-guards.txt
if [[ ${#names[@]} -eq 0 ]]; then
  echo "scripts/pre-pr-guards.txt no declara ninguna prueba." >&2
  failed=1
else
  mapfile -t unique_packages < <(printf '%s\n' "${packages[@]}" | sort -u)
  pattern="^($(IFS='|'; printf '%s' "${names[*]}"))\$"
  guard_log="${log_dir}/guards.log"
  go test "${unique_packages[@]}" -run "${pattern}" -count=1 -v >"${guard_log}" 2>&1 || true
  missing=()
  for name in "${names[@]}"; do
    grep -q "^--- PASS: ${name} " "${guard_log}" || missing+=("${name}")
  done
  if [[ ${#missing[@]} -gt 0 ]]; then
    # Con -v, go test intercala la salida de todas las pruebas, también la de las
    # que pasan. Se agrupa por prueba (las marcas === RUN/NAME/CONT dicen a cuál
    # pertenece cada línea) y solo se muestra la de las que fallan o se omiten,
    # junto con lo que no es de ninguna (errores de compilación, FAIL del paquete).
    awk '
      /^=== (RUN|NAME|CONT|PAUSE) / { current = $3; next }
      /^[[:space:]]*--- (PASS|FAIL|SKIP): / {
        if ($2 != "PASS:") { printf "%s", buffer[$3]; print }
        delete buffer[$3]
        next
      }
      /^(PASS|FAIL)$/ || /^ok[[:space:]]/ { current = ""; next }
      /^FAIL[[:space:]]/ { current = ""; print; next }
      current != "" { buffer[current] = buffer[current] $0 "\n"; next }
      { print }
    ' "${guard_log}" >&2
    printf 'Guardas que no pasaron (fallaron, se omitieron o ya no existen): %s\n' "${missing[*]}" >&2
    failed=1
  fi
fi

echo "[3/3] deadcode ratchet"
"${BASH}" scripts/deadcode-ratchet.sh || failed=1

if [[ ${failed} -ne 0 ]]; then
  echo "Comprobación previa: FALLA. Corrige lo anterior y haz commit antes de abrir la PR." >&2
  exit 1
fi
echo "Comprobación previa: todo en verde."
