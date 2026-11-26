package changestart

import (
	"errors"
	"os"
)

// Discard retira el staging abandonado de un cambio (D-12, D-25): bajo el cerrojo del cambio, lo
// renombra atómicamente a un temporal y lo borra, y barre los temporales del cambio. No usa la red, no
// toca el destino, el repositorio ni el work item, y nunca pide el cerrojo del work item. Sin staging
// es un éxito sin efecto con aviso.
func Discard(env Env, change string) (DiscardResult, error) {
	env = env.withDefaults()
	if err := ValidChangeName(change); err != nil {
		return DiscardResult{}, refuse(CodeUsageInvalid, err)
	}
	unlock, rf := env.lockChange(change)
	if rf != nil {
		return DiscardResult{}, rf
	}
	defer unlock()

	res := DiscardResult{Change: change, Staging: env.StagingDir(change)}
	if _, err := os.Lstat(res.Staging); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return DiscardResult{}, refuse(CodeWriteFailed, err, "step", "discard")
		}
		res.Status = StatusNothingToDiscard
		res.Warnings = append(res.Warnings, "no había ningún staging del cambio "+change+" que retirar")
		res.Warnings = append(res.Warnings, env.sweepTemps(change)...)
		return res, nil
	}
	removeErr, err := env.retireReporting(res.Staging, change)
	if err != nil {
		return DiscardResult{}, refuse(CodeWriteFailed, err, "step", "discard")
	}
	res.Status = StatusDiscarded
	if removeErr != nil {
		res.Warnings = append(res.Warnings, "el staging se retiró, pero no se pudo borrar del todo; se barrerá en la siguiente operación del cambio")
	}
	res.Warnings = append(res.Warnings, env.sweepTemps(change)...)
	return res, nil
}
