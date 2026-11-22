package azuredevops_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const reposRoute = "/contoso/Fabrikam/_apis/git/repositories"

func twoRepos() adotest.Response {
	return adotest.Response{JSON: map[string]any{"count": 2, "value": []map[string]any{
		{"id": "11111111-1111-1111-1111-111111111111", "name": "api"},
		{"id": "22222222-2222-2222-2222-222222222222", "name": "portal"},
	}}}
}

func TestListRepositoriesReturnsNamesAndGUIDs(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", reposRoute, twoRepos())
	client, _ := newTestClient(t, srv, nil)

	repos, err := client.ListRepositories(context.Background())

	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	want := []azuredevops.Repository{
		{ID: "11111111-1111-1111-1111-111111111111", Name: "api"},
		{ID: "22222222-2222-2222-2222-222222222222", Name: "portal"},
	}
	if !reflect.DeepEqual(repos, want) {
		t.Fatalf("repositorios = %+v", repos)
	}
}

func TestListRepositoriesSecondCallComesFromTheCache(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", reposRoute, twoRepos())
	client, _ := newTestClient(t, srv, nil)

	first, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if got := srv.Count("GET", reposRoute); got != 1 {
		t.Fatalf("peticiones = %d, esperaba 1: la segunda consulta no envía nada", got)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("la caché devuelve el mismo listado")
	}
}

func TestListRepositoriesFailureIsNotCached(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Sequence("GET", reposRoute, adotest.Response{Status: 500, JSON: map[string]any{"message": "fallo"}}, twoRepos())
	client, _ := newTestClient(t, srv, nil)

	_, err := client.ListRepositories(context.Background())
	if !errors.Is(err, azuredevops.ErrServer) {
		t.Fatalf("la primera consulta debe fallar con ErrServer: %v", err)
	}
	repos, err := client.ListRepositories(context.Background())

	if err != nil || len(repos) != 2 {
		t.Fatalf("la segunda consulta debe devolver el listado: %v, %v", repos, err)
	}
	if got := srv.Count("GET", reposRoute); got != 2 {
		t.Fatalf("peticiones = %d, esperaba 2 (el fallo no se cachea)", got)
	}
}

func TestListRepositoriesForbiddenNamesVsoCode(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", reposRoute, adotest.Response{Status: 403})
	client, _ := newTestClient(t, srv, nil)

	_, err := client.ListRepositories(context.Background())

	var auth *azuredevops.AuthError
	if !errors.As(err, &auth) || auth.Scope != "vso.code" {
		t.Fatalf("esperaba AuthError con vso.code: %v", err)
	}
}

func TestListRepositoriesCacheIsPerOrganizationAndProcess(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", reposRoute, twoRepos())
	srv.Reply("GET", "/fabrikam/Fabrikam/_apis/git/repositories", twoRepos())
	srv.Reply("GET", "/contoso/Otro/_apis/git/repositories", twoRepos())
	plugin := azuredevops.New(srv.Options())
	src := &patSource{pat: testPAT}
	newClient := func(org, project string) *azuredevops.Client {
		c, err := plugin.NewClient(map[string]any{"organization": org, "project": project, "team": "t"}, src)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}

	if _, err := newClient("contoso", "Fabrikam").ListRepositories(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Otro cliente de la misma organización (aunque cambie la capitalización) y proyecto: caché.
	if _, err := newClient("Contoso", "Fabrikam").ListRepositories(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := len(srv.Requests()); got != 1 {
		t.Fatalf("peticiones = %d, esperaba 1: la caché es de la organización", got)
	}
	// Otro proyecto de la misma organización y otra organización: peticiones nuevas.
	if _, err := newClient("contoso", "Otro").ListRepositories(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := newClient("fabrikam", "Fabrikam").ListRepositories(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := len(srv.Requests()); got != 3 {
		t.Fatalf("peticiones = %d, esperaba 3 (un proyecto y una organización nuevos)", got)
	}
}

func TestListRepositoriesReturnsACopy(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", reposRoute, twoRepos())
	client, _ := newTestClient(t, srv, nil)

	first, _ := client.ListRepositories(context.Background())
	first[0].Name = "mutado"
	second, _ := client.ListRepositories(context.Background())

	if second[0].Name != "api" {
		t.Fatalf("modificar el resultado no debe alterar la caché: %+v", second)
	}
}
