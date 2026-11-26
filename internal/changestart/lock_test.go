package changestart

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"
)

type startOutcome struct {
	res StartResult
	err error
}

func runConcurrent(env Env, trackers ...*stubTracker) []startOutcome {
	out := make([]startOutcome, len(trackers))
	var wg sync.WaitGroup
	for i, tr := range trackers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i].res, out[i].err = Start(context.Background(), env, tr, "2525")
		}()
	}
	wg.Wait()
	return out
}

func TestStartConcurrentSameItem(t *testing.T) {
	env := newRepo(t)
	env.LockWait = 10 * time.Second
	out := runConcurrent(env, newStub(), newStub())
	staged := 0
	for _, o := range out {
		switch {
		case o.err == nil && o.res.Status == StatusStaged:
			staged++
		case o.err == nil && o.res.Status == StatusAlreadyStaged:
		default:
			if r := refusalOf(t, o.err); r.Code != CodeChangeStartBusy {
				t.Fatalf("resultado inesperado: %+v %v", o.res, o.err)
			}
		}
	}
	if staged != 1 || len(stagings(t, env)) != 1 {
		t.Fatalf("debe existir un único staging: staged=%d stagings=%v", staged, stagings(t, env))
	}
	if _, ok := env.stagingFor(stagings(t, env)[0]); !ok {
		t.Fatal("el staging debe ser íntegro")
	}
}

func TestStartConcurrentSameItemDifferentTitles(t *testing.T) {
	env := newRepo(t)
	env.LockWait = 10 * time.Second
	a, b := newStub(), newStub()
	b.item.Title = "Alta de usuarios con SSO"
	out := runConcurrent(env, a, b)
	var statuses []string
	for _, o := range out {
		if o.err != nil {
			t.Fatalf("Start: %v", o.err)
		}
		statuses = append(statuses, o.res.Status)
	}
	sort.Strings(statuses)
	if statuses[0] != StatusAlreadyStaged || statuses[1] != StatusStaged {
		t.Fatalf("estados = %v", statuses)
	}
	if got := stagings(t, env); len(got) != 1 {
		t.Fatalf("stagings = %v", got)
	}
}

func TestStartOrphanLockRecovered(t *testing.T) {
	env := newRepo(t)
	// Un cerrojo de un proceso que ya no existe: queda el fichero con su carga, sin bloqueo del SO.
	writeFile(t, env.lockPath("item", "azure-devops-2525"), `{"pid":999999,"host":"fantasma"}`)
	writeFile(t, env.lockPath("change", linkedName), `{"pid":999999}`)
	if res := mustStart(t, env, newStub()); res.Status != StatusStaged {
		t.Fatalf("estado = %q", res.Status)
	}
}

func TestStartBusyAfterBoundedWait(t *testing.T) {
	for name, hold := range map[string]func(Env) (func(), *Refusal){
		"cambio": func(e Env) (func(), *Refusal) { return e.lockChange(linkedName) },
		"item":   func(e Env) (func(), *Refusal) { return e.lockItem("azure-devops", "2525") },
	} {
		t.Run(name, func(t *testing.T) {
			env := newRepo(t)
			env.LockWait = 300 * time.Millisecond
			release, rf := hold(env.withDefaults())
			if rf != nil {
				t.Fatal(rf)
			}
			defer release()
			begin := time.Now()
			r := startRefusal(t, env, newStub())
			if r.Code != CodeChangeStartBusy {
				t.Fatalf("código = %s", r.Code)
			}
			if waited := time.Since(begin); waited < 250*time.Millisecond || waited > 5*time.Second {
				t.Fatalf("espera acotada fuera de rango: %v", waited)
			}
			if got := stagings(t, env); len(got) != 0 {
				t.Fatalf("no debe quedar staging: %v", got)
			}
		})
	}
}

func TestSweepOnlyUnderChangeLock(t *testing.T) {
	env := newRepo(t)
	env.LockWait = 200 * time.Millisecond
	mine := filepath.Join(env.tmpDir(linkedName), "restos", "a.txt")
	other := filepath.Join(env.tmpDir("otro-cambio"), "restos", "b.txt")
	writeFile(t, mine, "x")
	writeFile(t, other, "y")

	release, rf := env.withDefaults().lockChange(linkedName)
	if rf != nil {
		t.Fatal(rf)
	}
	if r := startRefusal(t, env, newStub()); r.Code != CodeChangeStartBusy {
		t.Fatalf("código = %s", r.Code)
	}
	if _, err := os.Stat(mine); err != nil {
		t.Fatalf("sin el cerrojo del cambio no se barre nada: %v", err)
	}
	release()

	mustStart(t, env, newStub())
	if _, err := os.Stat(mine); !os.IsNotExist(err) {
		t.Fatalf("los temporales del cambio deben barrerse bajo su cerrojo: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("los temporales de otro cambio no se tocan: %v", err)
	}
}
