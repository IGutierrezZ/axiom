package assets

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Marcas del bloque de la guarda de proyecto Axiom (D3) de sdd-task-result-artifacts.ts. El bloque
// replica workspace.IsAxiomProject en TypeScript; internal/sddtaskresult lo compara con el criterio
// de Go y este fichero comprueba que las dos generaciones del plugin llevan el mismo texto.
const (
	sddProjectGateBegin = "// axiom-project-gate:begin"
	sddProjectGateEnd   = "// axiom-project-gate:end"
)

// sddAxiomProjectFixture crea en el arnés de Node un proyecto Axiom mínimo (axiom.yaml en la raíz y
// un subdirectorio) y deja su ruta en projectDir. Los arneses de v2 que ejercitan la validación SDD
// lo usan como directorio de sesión: fuera de un proyecto Axiom el plugin es inerte.
const sddAxiomProjectFixture = `
const projectDir=await (async()=>{const fs=await import('node:fs'),os=await import('node:os'),path=await import('node:path');const dir=fs.mkdtempSync(path.join(os.tmpdir(),'axiom-sdd-project-'));fs.writeFileSync(path.join(dir,'axiom.yaml'),'name: fixture\n');return dir})();
`

// sddProjectGateBlock devuelve el texto entre las marcas del plugin indicado.
func sddProjectGateBlock(t *testing.T, path string) string {
	t.Helper()
	source, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	begin := strings.Index(source, sddProjectGateBegin)
	end := strings.Index(source, sddProjectGateEnd)
	if begin < 0 || end < begin || strings.Count(source, sddProjectGateBegin) != 1 || strings.Count(source, sddProjectGateEnd) != 1 {
		t.Fatalf("%s: la guarda de proyecto Axiom no está delimitada por una única pareja de marcas", path)
	}
	return source[begin : end+len(sddProjectGateEnd)]
}

func TestSDDTaskResultArtifactsProjectGateIsInSyncAcrossGenerations(t *testing.T) {
	v1 := sddProjectGateBlock(t, "opencode/plugins/sdd-task-result-artifacts.ts")
	v2 := sddProjectGateBlock(t, "opencode/plugins-v2/sdd-task-result-artifacts.ts")
	if v1 != v2 {
		t.Fatal("la guarda de proyecto Axiom difiere entre plugins/ y plugins-v2/; deben llevar el mismo bloque")
	}
	for _, want := range []string{`"axiom.yaml"`, `".axiom-workspace"`, `catch {`, `return false`} {
		if !strings.Contains(v1, want) {
			t.Errorf("la guarda de proyecto Axiom no contiene %q", want)
		}
	}
	// openspec/ no es marcador de Axiom (D3): un repositorio ajeno que lo use no activa el plugin.
	for _, line := range strings.Split(v1, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "//") && strings.Contains(line, "openspec") {
			t.Errorf("la guarda de proyecto Axiom no puede aceptar openspec/ como marcador: %q", line)
		}
	}
	for _, path := range []string{"opencode/plugins/sdd-task-result-artifacts.ts", "opencode/plugins-v2/sdd-task-result-artifacts.ts"} {
		source := MustRead(path)
		// Los tres ganchos que actúan (preflight, despacho y resultado) pasan por la guarda.
		if got := strings.Count(source, "await isAxiomProject("); got != 4 {
			t.Errorf("%s consulta la guarda %d veces, se esperaban 4 (question before/after, subagente before/after)", path, got)
		}
	}
}

// runSDDPluginHarness ejecuta el plugin indicado en Node (sin tipos) con el arnés dado. La generación
// v1 solo importa tipos de @opencode-ai/plugin; la v2 se prepara como en runV2Plugin.
func runSDDPluginHarness(t *testing.T, source, harness string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.mts"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "harness.mjs"), []byte("import plugin from './plugin.mts'\n"+harness), 0600); err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	cmd := exec.Command(node, "--experimental-strip-types", filepath.Join(dir, "harness.mjs"))
	cmd.Env = append(os.Environ(), "CI=0", "GITHUB_ACTIONS=0", "HOME="+dir, "XDG_CONFIG_HOME="+dir, "XDG_DATA_HOME="+dir, "TMPDIR="+dir, "TEMP="+dir, "TMP="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("SDD plugin harness: %v\n%s", err, out)
	}
}

// sddGateFixtures crea tres directorios de sesión: uno ajeno que usa openspec/, un proyecto Axiom con
// un subdirectorio (el criterio asciende) y un multirepo cuyo puntero .axiom-workspace resuelve a un
// axiom.yaml existente.
const sddGateFixtures = `
const fs=await import('node:fs'),os=await import('node:os'),path=await import('node:path');
const root=fs.mkdtempSync(path.join(os.tmpdir(),'axiom-sdd-gate-'));
const foreign=path.join(root,'foreign');fs.mkdirSync(path.join(foreign,'openspec','changes','x'),{recursive:true});fs.mkdirSync(path.join(foreign,'.git'),{recursive:true});
const proj=path.join(root,'proj');fs.mkdirSync(path.join(proj,'sub','deeper'),{recursive:true});fs.writeFileSync(path.join(proj,'axiom.yaml'),'name: fixture\n');
const multi=path.join(root,'multi');fs.mkdirSync(path.join(multi,'repo-specs'),{recursive:true});fs.mkdirSync(path.join(multi,'backend'),{recursive:true});fs.writeFileSync(path.join(multi,'repo-specs','axiom.yaml'),'name: specs\n');fs.writeFileSync(path.join(multi,'.axiom-workspace'),'config: repo-specs/axiom.yaml # puntero\n');
const broken=path.join(root,'broken');fs.mkdirSync(broken,{recursive:true});fs.writeFileSync(path.join(broken,'.axiom-workspace'),'config: no-existe/axiom.yaml\n');
const reject=async(fn,why)=>{let failed=false;try{await fn()}catch{failed=true}if(!failed)throw Error(why)};
`

func TestOpenCodeV1SDDPluginIsInertOutsideAxiomProjects(t *testing.T) {
	runSDDPluginHarness(t, MustRead("opencode/plugins/sdd-task-result-artifacts.ts"), sddGateFixtures+`
const make=(directory,worktree)=>plugin({client:{session:{get:async({path})=>({data:{id:path.id}})}},directory,worktree});
const dispatch=()=>({args:{subagent_type:'sdd-apply',prompt:'Implement approved tasks'}});
const twoQuestions=()=>({args:{questions:[{question:'Axiom SDD preflight 1/3: Pace',options:[{label:'x'}]},{question:'Axiom SDD preflight 2/3: Artifacts',options:[]}]}});

// Proyecto ajeno: nada se rechaza, se canonicaliza ni se valida, ni queda estado retenido.
for(const directory of [foreign,broken]){
 const hooks=await make(directory);
 const out=dispatch();await hooks['tool.execute.before']({tool:'task',sessionID:'s'},out);
 if(out.args.prompt!=='Implement approved tasks')throw Error('foreign project prompt rewritten: '+directory);
 const q=twoQuestions();const before=JSON.stringify(q);await hooks['tool.execute.before']({tool:'question',sessionID:'s'},q);
 if(JSON.stringify(q)!==before)throw Error('foreign project question canonicalized: '+directory);
 await hooks['tool.execute.after']({tool:'question',sessionID:'s',args:q.args},{title:'q',output:'',metadata:{answers:[]}});
 await hooks['tool.execute.after']({tool:'task',sessionID:'s',args:out.args},{title:'t',output:'',metadata:{}});
 await hooks['tool.execute.before']({tool:'task',sessionID:'s'},dispatch());
}
// El directorio del worktree solo vale si no hay directorio de sesión.
{const hooks=await make('',foreign);await hooks['tool.execute.before']({tool:'task',sessionID:'s'},dispatch())}

// Proyecto Axiom (raíz, subdirectorio y puntero multirepo): el comportamiento actual sigue intacto.
for(const directory of [proj,path.join(proj,'sub','deeper'),multi,path.join(multi,'backend')]){
 const hooks=await make(directory);
 await reject(()=>hooks['tool.execute.before']({tool:'task',sessionID:'s'},dispatch()),'missing preflight accepted in '+directory);
 await reject(()=>hooks['tool.execute.before']({tool:'question',sessionID:'s'},twoQuestions()),'two-question preflight accepted in '+directory);
}
{const hooks=await make(proj);const out=dispatch();await reject(()=>hooks['tool.execute.after']({tool:'task',sessionID:'s',args:out.args},{title:'t',output:'',metadata:{}}),'empty result accepted');await reject(()=>hooks['tool.execute.before']({tool:'task',sessionID:'s'},dispatch()),'failure not latched')}
`)
}

func TestOpenCodeV2SDDPluginIsInertOutsideAxiomProjects(t *testing.T) {
	source := MustRead("opencode/plugins-v2/sdd-task-result-artifacts.ts")
	runSDDPluginHarness(t, strings.Replace(source, `import { Plugin } from "@opencode/plugin"`, `const Plugin = { define: (value: any) => value }`, 1), sddGateFixtures+`
const make=async(directory)=>{const hooks={};const ctx={location:{directory,workspaceID:'one'},tool:{async hook(name,fn){hooks[name]=fn;return {async dispose(){}}}},session:{async get(arg){return {id:arg.sessionID}}},event:{subscribe(opts){return {[Symbol.asyncIterator]:async function*(){await new Promise(r=>opts.signal.addEventListener('abort',r))}}}}};const cleanup=await plugin.setup(ctx);return {hooks,cleanup}};
const dispatch=()=>({tool:'subagent',sessionID:'s',id:'call',input:{agent:'sdd-apply',prompt:'Implement approved tasks',description:'Apply'}});
const twoQuestions=()=>({tool:'question',sessionID:'s',id:'q',input:{questions:[{question:'Axiom SDD preflight 1/3: Pace',options:[{label:'x'}]},{question:'Axiom SDD preflight 2/3: Artifacts',options:[]}]}});

// Proyecto ajeno: nada se rechaza, se canonicaliza ni se valida, ni queda estado retenido.
for(const directory of [foreign,broken]){
 const {hooks,cleanup}=await make(directory);
 const call=dispatch();await hooks['execute.before'](call);
 if(call.input.prompt!=='Implement approved tasks')throw Error('foreign project prompt rewritten: '+directory);
 const q=twoQuestions();const before=JSON.stringify(q);await hooks['execute.before'](q);
 if(JSON.stringify(q)!==before)throw Error('foreign project question canonicalized: '+directory);
 await hooks['execute.after']({...q,status:'completed',result:{output:{answers:[]}}});
 await hooks['execute.after']({...call,status:'completed',result:{output:{sessionID:'w',status:'completed',output:''}}});
 await hooks['execute.after']({...call,status:'error',error:{message:'provider failure'}});
 await hooks['execute.before'](dispatch());
 await cleanup();
}

// Proyecto Axiom (raíz, subdirectorio y puntero multirepo): el comportamiento actual sigue intacto.
for(const directory of [proj,path.join(proj,'sub','deeper'),multi,path.join(multi,'backend')]){
 const {hooks,cleanup}=await make(directory);
 await reject(()=>hooks['execute.before'](dispatch()),'missing preflight accepted in '+directory);
 await reject(()=>hooks['execute.before'](twoQuestions()),'two-question preflight accepted in '+directory);
 await cleanup();
}
{const {hooks,cleanup}=await make(proj);const call=dispatch();await reject(()=>hooks['execute.after']({...call,status:'completed',result:{output:{sessionID:'w',status:'completed',output:''}}}),'empty result accepted');await reject(()=>hooks['execute.before'](dispatch()),'failure not latched');await cleanup()}
`)
}
