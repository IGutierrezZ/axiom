package azuredevops

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

const (
	// memberPageSize es el tamaño de página ($top) de la lista de miembros del equipo.
	memberPageSize = 100
	// defaultMaxMemberPages es el tope de páginas de miembros: un equipo de 100.000 personas no es un
	// caso real y un servidor que nunca termina no puede dejar el verbo colgado.
	defaultMaxMemberPages = 1000
)

// maxMemberPages es el tope efectivo de páginas de miembros. Es una variable solo para que las
// pruebas lo reduzcan (SetMaxMemberPagesForTest).
var maxMemberPages = defaultMaxMemberPages

var _ integrations.Discoverer = (*Plugin)(nil)

// Discover implementa el puerto de descubrimiento (integrations.Discoverer): tipos de work item con
// sus estados y campos, iteraciones del equipo y miembros. Las peticiones son secuenciales y un fallo
// en cualquiera hace fallar el descubrimiento entero: nunca se devuelve un resultado parcial como si
// fuera completo (D-12).
func (p *Plugin) Discover(ctx context.Context, req integrations.PortRequest) (integrations.Discovery, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.Discovery{}, err
	}
	return client.discover(ctx)
}

// processType es un tipo de work item de la lista de tipos del proyecto.
type processType struct {
	Name          string `json:"name"`
	ReferenceName string `json:"referenceName"`
}

type processTypesResponse struct {
	Value []processType `json:"value"`
}

type processStatesResponse struct {
	Value []struct {
		Name     string `json:"name"`
		Color    string `json:"color"`
		Category string `json:"category"`
	} `json:"value"`
}

type processFieldsResponse struct {
	Value []struct {
		ReferenceName  string `json:"referenceName"`
		Name           string `json:"name"`
		AlwaysRequired bool   `json:"alwaysRequired"`
	} `json:"value"`
}

type iterationItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Attributes struct {
		StartDate  string `json:"startDate"`
		FinishDate string `json:"finishDate"`
		TimeFrame  string `json:"timeFrame"`
	} `json:"attributes"`
}

// iterationsResponse acepta los dos nombres de la lista: la documentación y algunas respuestas
// difieren (values y value).
type iterationsResponse struct {
	Values []iterationItem `json:"values"`
	Value  []iterationItem `json:"value"`
}

// membersResponse solo decodifica el id y el nombre visible: uniqueName (el correo) no se decodifica
// ni se guarda nunca.
type membersResponse struct {
	Value []struct {
		Identity struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"identity"`
	} `json:"value"`
}

func (c *Client) discover(ctx context.Context) (integrations.Discovery, error) {
	project := c.target.Project

	var types processTypesResponse
	if err := c.do(ctx, request{
		method:    "GET",
		segments:  []string{project, "_apis", "wit", "workitemtypes"},
		scope:     scopeWorkWrite,
		operation: "listar los tipos de work item",
		retryable: true,
	}, &types); err != nil {
		return integrations.Discovery{}, err
	}

	itemTypes := make([]integrations.ItemType, 0, len(types.Value))
	for _, t := range types.Value {
		item := integrations.ItemType{Name: t.Name, ReferenceName: t.ReferenceName,
			States: []integrations.ItemState{}, Fields: []integrations.ItemField{}}

		var states processStatesResponse
		if err := c.do(ctx, request{
			method:    "GET",
			segments:  []string{project, "_apis", "wit", "workitemtypes", t.Name, "states"},
			scope:     scopeWorkWrite,
			operation: fmt.Sprintf("leer los estados del tipo %s", t.Name),
			retryable: true,
		}, &states); err != nil {
			return integrations.Discovery{}, err
		}
		for _, s := range states.Value {
			// La categoría se copia tal cual: Proposed, InProgress, Resolved, Completed, Removed.
			item.States = append(item.States, integrations.ItemState{Name: s.Name, Category: s.Category, Color: s.Color})
		}

		var fields processFieldsResponse
		if err := c.do(ctx, request{
			method:    "GET",
			segments:  []string{project, "_apis", "wit", "workitemtypes", t.Name, "fields"},
			scope:     scopeWorkWrite,
			operation: fmt.Sprintf("leer los campos del tipo %s", t.Name),
			retryable: true,
		}, &fields); err != nil {
			return integrations.Discovery{}, err
		}
		for _, f := range fields.Value {
			item.Fields = append(item.Fields, integrations.ItemField{ReferenceName: f.ReferenceName, Name: f.Name, AlwaysRequired: f.AlwaysRequired})
		}
		itemTypes = append(itemTypes, item)
	}

	var iterations iterationsResponse
	if err := c.do(ctx, request{
		method:    "GET",
		segments:  []string{project, c.target.Team, "_apis", "work", "teamsettings", "iterations"},
		scope:     scopeWork,
		operation: "listar las iteraciones del equipo",
		retryable: true,
	}, &iterations); err != nil {
		return integrations.Discovery{}, err
	}
	items := iterations.Values
	if len(items) == 0 {
		items = iterations.Value
	}
	sprints := make([]integrations.Iteration, 0, len(items))
	for _, it := range items {
		sprints = append(sprints, integrations.Iteration{
			ID: it.ID, Name: it.Name, Path: it.Path,
			StartDate: it.Attributes.StartDate, FinishDate: it.Attributes.FinishDate, TimeFrame: it.Attributes.TimeFrame,
		})
	}

	members, err := c.listMembers(ctx)
	if err != nil {
		return integrations.Discovery{}, err
	}

	return integrations.Discovery{
		Schema: integrations.DiscoverySchemaV1,
		Plugin: manifest.AzureDevOpsID,
		Scope: map[string]string{
			"organization": c.target.Organization,
			"project":      project,
			"team":         c.target.Team,
		},
		ItemTypes:  itemTypes,
		Iterations: sprints,
		Members:    members,
	}, nil
}

// listMembers lista los miembros del equipo con GET {org}/_apis/projects/{proyecto}/teams/{equipo}/members
// paginado con $top y $skip, sin repetidos por id. La paginación termina con una página incompleta.
// Una página llena que no aporta ningún id nuevo (el servidor ignora $skip) o el tope de páginas son
// un *PaginationError: nunca se devuelve una lista truncada como si fuera completa. Solo lee el id y
// el nombre visible. La reutiliza la resolución de identidad.
func (c *Client) listMembers(ctx context.Context) ([]integrations.Member, error) {
	const operation = "listar los miembros del equipo"
	members := []integrations.Member{}
	seen := map[string]bool{}

	for page := 1; ; page++ {
		var resp membersResponse
		if err := c.do(ctx, request{
			method:    "GET",
			segments:  []string{"_apis", "projects", c.target.Project, "teams", c.target.Team, "members"},
			query:     url.Values{"$top": {strconv.Itoa(memberPageSize)}, "$skip": {strconv.Itoa((page - 1) * memberPageSize)}},
			scope:     scopeProject,
			operation: operation,
			retryable: true,
		}, &resp); err != nil {
			return nil, err
		}

		added := 0
		for _, entry := range resp.Value {
			id := entry.Identity.ID
			if id == "" {
				return nil, invalidResponse(operation, "un miembro no trae identificador")
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			added++
			members = append(members, integrations.Member{ID: id, DisplayName: entry.Identity.DisplayName})
		}

		if len(resp.Value) < memberPageSize {
			return members, nil
		}
		if added == 0 {
			return nil, &PaginationError{Operation: operation, Pages: page,
				Reason: "una página completa no aportó ningún miembro nuevo (el servidor ignora $skip)"}
		}
		if page >= maxMemberPages {
			return nil, &PaginationError{Operation: operation, Pages: page,
				Reason: fmt.Sprintf("se alcanzó el tope de %d páginas", maxMemberPages)}
		}
	}
}
