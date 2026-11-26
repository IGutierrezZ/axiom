package adotest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Relaciones de jerarquía que simula el servidor.
const (
	relHierarchyForward = "System.LinkTypes.Hierarchy-Forward"
	relHierarchyReverse = "System.LinkTypes.Hierarchy-Reverse"
)

var (
	relationItemID = regexp.MustCompile(`(?i)/workitems/(\d+)$`)
	wiqlType       = regexp.MustCompile(`\[System\.WorkItemType\]\s*=\s*'((?:[^']|'')*)'`)
	wiqlTag        = regexp.MustCompile(`\[System\.Tags\]\s+CONTAINS\s+'((?:[^']|'')*)'`)
	membersPath    = regexp.MustCompile(`/_apis/projects/[^/]+/teams/[^/]+/members$`)
	typeFieldsPath = regexp.MustCompile(`/_apis/wit/workitemtypes/([^/]+)/fields$`)
)

// Member es un miembro del equipo del servidor simulado. UniqueName es el correo: el adaptador
// nunca lo decodifica, y las pruebas lo siembran para comprobarlo.
type Member struct {
	ID, DisplayName, UniqueName string
}

// LoseNextCreateResponse hace que el siguiente POST de un work item nuevo se aplique pero la conexión
// se corte antes de responder: el work item existe y el cliente no conoce su identificador.
func (s *Server) LoseNextCreateResponse() {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.loseCreate = true
}

// RejectCreateRelations hace que un POST de un work item nuevo que trae relaciones responda 400 sin
// crear nada, como un proceso que no admite relacionar al crear. Sin relaciones se crea con normalidad.
func (s *Server) RejectCreateRelations(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.rejectRelations = on
}

// FailWIQL hace que la consulta WIQL responda siempre con status (0 la restablece).
func (s *Server) FailWIQL(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.wiqlStatus = status
}

// Created devuelve cuántos POST de work items nuevos crearon uno.
func (s *Server) Created() int {
	s.mu.Lock()
	st := s.items
	s.mu.Unlock()
	if st == nil {
		return 0
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.created
}

// SeedTypeFields da de alta los campos (por su nombre de referencia) que el proceso define para el
// tipo typ. Activa el simulacro de GET .../_apis/wit/workitemtypes/{tipo}/fields.
func (s *Server) SeedTypeFields(typ string, references ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fields == nil {
		s.fields = map[string][]string{}
	}
	s.fields[strings.ToLower(typ)] = append([]string(nil), references...)
}

// SeedMembers da de alta los miembros del equipo. Activa el simulacro de GET
// .../_apis/projects/{proyecto}/teams/{equipo}/members, paginado con $top y $skip.
func (s *Server) SeedMembers(members ...Member) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members = append([]Member(nil), members...)
}

// extraGet devuelve el manejador de los campos de tipo y los miembros sembrados. Se llama con s.mu
// tomado.
func (s *Server) extraGet(req *Request) Handler {
	if m := typeFieldsPath.FindStringSubmatch(req.Path); m != nil && s.fields != nil {
		refs, ok := s.fields[strings.ToLower(m[1])]
		if !ok {
			return nil
		}
		return func(*Request) Response {
			value := make([]map[string]any, 0, len(refs))
			for _, ref := range refs {
				value = append(value, map[string]any{"referenceName": ref, "name": ref, "alwaysRequired": false})
			}
			return Response{JSON: map[string]any{"count": len(value), "value": value}}
		}
	}
	if membersPath.MatchString(req.Path) && s.members != nil {
		members := append([]Member(nil), s.members...)
		return func(r *Request) Response {
			top, _ := strconv.Atoi(r.Query.Get("$top"))
			skip, _ := strconv.Atoi(r.Query.Get("$skip"))
			if top <= 0 {
				top = len(members)
			}
			start := min(max(skip, 0), len(members))
			end := min(start+top, len(members))
			value := make([]map[string]any, 0, end-start)
			for _, m := range members[start:end] {
				value = append(value, map[string]any{"identity": map[string]any{
					"id": m.ID, "displayName": m.DisplayName, "uniqueName": m.UniqueName,
				}})
			}
			return Response{JSON: map[string]any{"count": len(value), "value": value}}
		}
	}
	return nil
}

// orgOf devuelve la organización: el primer segmento de la ruta decodificada.
func orgOf(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	return segments[0]
}

// linkParent refleja en el padre la relación Hierarchy-Reverse que el hijo childID acaba de añadir: le
// añade la relación Hierarchy-Forward hacia el hijo, con una URL del propio servidor. Se llama con
// st.mu tomado y solo cuando la modificación se aplicó entera.
func (st *workItemStore) linkParent(path string, childID int, rel map[string]any) {
	if rel["rel"] != relHierarchyReverse {
		return
	}
	raw, _ := rel["url"].(string)
	m := relationItemID.FindStringSubmatch(raw)
	if m == nil {
		return
	}
	parentID, _ := strconv.Atoi(m[1])
	parent, ok := st.items[parentID]
	if !ok {
		return
	}
	url := fmt.Sprintf("%s/%s/_apis/wit/workItems/%d", strings.TrimRight(st.base, "/"), orgOf(path), childID)
	parent.Relations = append(append([]map[string]any(nil), parent.Relations...), map[string]any{"rel": relHierarchyForward, "url": url})
}

// create simula POST .../_apis/wit/workitems/${tipo}: asigna el identificador y la revisión 1, aplica
// los campos y las relaciones del cuerpo y responde con el work item creado.
func (st *workItemStore) create(typ string, req *Request) Response {
	if !strings.HasPrefix(req.Header.Get("Content-Type"), "application/json-patch+json") {
		return Response{Status: http.StatusUnsupportedMediaType, JSON: map[string]any{"message": "adotest: se espera application/json-patch+json"}}
	}
	var ops []patchOp
	if err := json.Unmarshal(req.Body, &ops); err != nil || len(ops) == 0 {
		return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "adotest: documento JSON Patch inválido"}}
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	fields := map[string]any{"System.WorkItemType": typ, "System.State": "To Do"}
	var relations []map[string]any
	for _, op := range ops {
		switch {
		case op.Op == "add" && strings.HasPrefix(op.Path, "/fields/"):
			var value any
			_ = json.Unmarshal(op.Value, &value)
			fields[strings.TrimPrefix(op.Path, "/fields/")] = value
		case op.Op == "add" && op.Path == "/relations/-":
			var rel map[string]any
			if err := json.Unmarshal(op.Value, &rel); err != nil || rel["rel"] == nil || rel["url"] == nil {
				return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "adotest: relación inválida en add /relations/-"}}
			}
			relations = append(relations, rel)
		default:
			return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "adotest: operación no soportada " + op.Op + " " + op.Path}}
		}
	}
	if len(relations) > 0 && st.rejectRelations {
		return Response{Status: http.StatusBadRequest, JSON: map[string]any{
			"message": "VS403303: the process does not allow relating a work item on creation.",
			"typeKey": "RuleValidationException",
		}}
	}

	id := 1
	for existing := range st.items {
		id = max(id, existing+1)
	}
	it := &WorkItem{ID: id, Rev: 1, Fields: fields, Relations: relations}
	st.items[id] = it
	st.created++
	for _, rel := range relations {
		st.linkParent(req.Path, id, rel)
	}
	if st.loseCreate {
		st.loseCreate = false
		return Response{Drop: true}
	}
	return Response{JSON: render(it, true)}
}

// wiql simula POST .../_apis/wit/wiql para las consultas por tipo y por etiqueta que hace el
// adaptador: devuelve los work items del tipo cuyas etiquetas contienen la pedida, sin distinguir
// mayúsculas.
func (st *workItemStore) wiql(req *Request) Response {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.wiqlStatus != 0 {
		return Response{Status: st.wiqlStatus, JSON: map[string]any{"message": "adotest: fallo programado de la consulta WIQL"}}
	}
	var body struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(req.Body, &body); err != nil || body.Query == "" {
		return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "adotest: consulta WIQL inválida"}}
	}
	unquote := func(s string) string { return strings.ReplaceAll(s, "''", "'") }
	var typ, tag string
	if m := wiqlType.FindStringSubmatch(body.Query); m != nil {
		typ = unquote(m[1])
	}
	if m := wiqlTag.FindStringSubmatch(body.Query); m != nil {
		tag = unquote(m[1])
	}
	var ids []int
	for id, it := range st.items {
		if typ != "" && !strings.EqualFold(fmt.Sprint(it.Fields["System.WorkItemType"]), typ) {
			continue
		}
		if tag != "" && !hasTag(it.Fields["System.Tags"], tag) {
			continue
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	value := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		value = append(value, map[string]any{"id": id, "url": fmt.Sprintf("https://adotest.invalid/workItems/%d", id)})
	}
	return Response{JSON: map[string]any{"queryType": "flat", "workItems": value}}
}

// hasTag indica si System.Tags (separado por «;») contiene tag, sin distinguir mayúsculas.
func hasTag(raw any, tag string) bool {
	s, _ := raw.(string)
	for _, t := range strings.Split(s, ";") {
		if strings.EqualFold(strings.TrimSpace(t), tag) {
			return true
		}
	}
	return false
}
