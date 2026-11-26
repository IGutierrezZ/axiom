package registry

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// RF-BRA-15 y RF-BRA-28: el modo de promoción y su origen hacen ida y vuelta por el registro.
func TestSaveAndLoadKeepThePromotionAndItsSource(t *testing.T) {
	root, commonDir := newRepository(t)
	record := validRecord(root, commonDir)
	record.Repositories[0].Promotion = branch.PromotionBranch
	record.Repositories[0].PromotionSource = branch.PromotionSourceAnswer

	if err := New().Save(commonDir, record); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(recordFile(commonDir, "sdd", "x"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"promotion": "branch"`, `"promotionSource": "answer"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("el registro debe contener %s:\n%s", want, data)
		}
	}
	got, err := load(t, commonDir, branch.LaneSDD, "x")
	if err != nil || got == nil || !reflect.DeepEqual(*got, record) {
		t.Fatalf("Load = %+v, %v; se esperaba %+v", got, err, record)
	}
}

// Sin los campos, los bytes del registro son idénticos a los de antes: omitempty.
func TestRecordBytesAreIdenticalWithoutThePromotionFields(t *testing.T) {
	root, commonDir := newRepository(t)
	record := validRecord(root, commonDir)

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("promotion")) {
		t.Errorf("sin modo de promoción el registro no lleva ninguna clave nueva:\n%s", data)
	}
	if err := New().Save(commonDir, record); err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(recordFile(commonDir, "sdd", "x"))
	if err != nil || !bytes.Equal(onDisk, append(data, '\n')) {
		t.Errorf("bytes en disco distintos de los esperados (%v):\n%s", err, onDisk)
	}
}

// Un registro con `release` y cada origen conocido es válido; uno con un vocabulario desconocido
// es `config_invalid` y la causa dice que el campo es derivado.
func TestLoadValidatesThePromotionVocabulary(t *testing.T) {
	root, commonDir := newRepository(t)
	valid := validRecord(root, commonDir)

	mutate := func(change func(*branch.RecordRepo)) string {
		record := valid
		record.Repositories = append([]branch.RecordRepo(nil), valid.Repositories...)
		change(&record.Repositories[0])
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	for _, source := range []string{"answer", "flag", "repository", "git", "default", "snapshot", "legacy"} {
		t.Run("origen válido "+source, func(t *testing.T) {
			writeFile(t, recordFile(commonDir, "sdd", "x"), mutate(func(repo *branch.RecordRepo) {
				repo.Promotion, repo.PromotionSource = branch.PromotionRelease, source
			}))
			if record, err := load(t, commonDir, branch.LaneSDD, "x"); err != nil || record == nil {
				t.Errorf("Load = %+v, %v; el origen %s es válido", record, err, source)
			}
		})
	}

	for name, tc := range map[string]struct {
		content string
		wantIn  string
	}{
		"modo ask no se registra":    {mutate(func(repo *branch.RecordRepo) { repo.Promotion = branch.PromotionAsk }), "promotion"},
		"modo desconocido":           {mutate(func(repo *branch.RecordRepo) { repo.Promotion = "trunk" }), "promotion"},
		"mayúscula":                  {mutate(func(repo *branch.RecordRepo) { repo.Promotion = "Branch" }), "promotion"},
		"origen desconocido":         {mutate(func(repo *branch.RecordRepo) { repo.PromotionSource = "azar" }), "promotionSource"},
		"origen del modo de trabajo": {mutate(func(repo *branch.RecordRepo) { repo.PromotionSource = branch.ModeSourceHub }), "promotionSource"},
	} {
		t.Run(name, func(t *testing.T) {
			writeFile(t, recordFile(commonDir, "sdd", "x"), tc.content)
			record, err := load(t, commonDir, branch.LaneSDD, "x")
			if record != nil || err == nil || !strings.Contains(err.Error(), tc.wantIn) || !strings.Contains(err.Error(), "bórralo") {
				t.Errorf("Load = %+v, %v; se esperaba un error que nombre %q y diga que se borre", record, err, tc.wantIn)
			}
		})
	}
}
