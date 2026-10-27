package sddstatus

// ChangeInstanceMarkerPending dice si `PrepareChangeInstanceConsent` va a escribir el marcador de
// instancia del cambio con este status: solo cuando la resolución dejó raíces de edición sin
// autorizar y el status identifica la carpeta y el nombre del cambio. `sdd continue` lo consulta
// para aplicar la guarda A (REQ-25.19) exactamente antes de esa escritura y no en una
// continuación que no escribe nada.
func ChangeInstanceMarkerPending(status Status) bool {
	return len(status.consentPreparationRoots) > 0 && status.ChangeRoot != nil && status.ChangeName != nil
}
