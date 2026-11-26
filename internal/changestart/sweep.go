package changestart

// sweepTemps borra los temporales del cambio bajo <v1>/.tmp/<cambio>/. Solo se llama con el cerrojo
// del cambio tomado: quien lo sostiene sabe que ningún otro proceso escribe ahí, así que se barre
// entero sin heurísticas de antigüedad (D-24). Un fallo solo avisa.
func (e Env) sweepTemps(change string) []string {
	if err := e.FS.RemoveAll(e.tmpDir(change)); err != nil {
		return []string{"no se pudieron barrer los temporales de un arranque anterior del cambio; se barrerán en la siguiente operación"}
	}
	return nil
}
