package kickoff

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ResolveExplainerHTML cruza la bandera --explainer-html con el valor de proyecto
// `sdd.explainer_html`:
//
//	always + (vacía|yes) -> yes        always + no  -> error
//	never  + (vacía|no)  -> no         never  + yes -> error
//	ask    + yes|no      -> la bandera
//	ask    + vacía       -> no, con defaulted verdadero (la CLI avisa)
func ResolveExplainerHTML(project workspace.ExplainerHTMLMode, flag ExplainerHTML) (value ExplainerHTML, defaulted bool, err error) {
	contradiction := func() error {
		return fmt.Errorf("--explainer-html %s contradice sdd.explainer_html: %s de axiom.yaml; omite la bandera o cambia la declaración", flag, project)
	}
	switch project {
	case workspace.ExplainerHTMLAlways:
		if flag == ExplainerHTMLNo {
			return "", false, contradiction()
		}
		return ExplainerHTMLYes, false, nil
	case workspace.ExplainerHTMLNever:
		if flag == ExplainerHTMLYes {
			return "", false, contradiction()
		}
		return ExplainerHTMLNo, false, nil
	}
	if flag == "" {
		return ExplainerHTMLNo, true, nil
	}
	return flag, false, nil
}
