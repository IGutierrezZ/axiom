package azuredevops

import (
	"context"
	"slices"
)

// Repository es un repositorio git del proyecto: nombre y GUID.
type Repository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// repositoriesResponse es la respuesta de _apis/git/repositories.
type repositoriesResponse struct {
	Value []Repository `json:"value"`
}

// ListRepositories lista los repositorios git del proyecto con GET .../_apis/git/repositories y los
// cachea por proyecto durante la vida del proceso (el estado es de la organización): una segunda
// consulta no envía petición. Un fallo no se cachea. Devuelve una copia: modificarla no altera la
// caché.
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	project := c.target.Project
	c.state.mu.Lock()
	cached, ok := c.state.repos[project]
	c.state.mu.Unlock()
	if ok {
		return slices.Clone(cached), nil
	}

	var resp repositoriesResponse
	if err := c.do(ctx, request{
		method:    "GET",
		segments:  []string{project, "_apis", "git", "repositories"},
		scope:     scopeCode,
		operation: "listar los repositorios del proyecto",
		retryable: true,
	}, &resp); err != nil {
		return nil, err
	}
	repos := resp.Value
	if repos == nil {
		repos = []Repository{}
	}

	c.state.mu.Lock()
	if c.state.repos == nil {
		c.state.repos = map[string][]Repository{}
	}
	c.state.repos[project] = slices.Clone(repos)
	c.state.mu.Unlock()
	return slices.Clone(repos), nil
}
