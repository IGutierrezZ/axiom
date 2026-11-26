package assets

import (
	"strings"
	"testing"
)

// Marcas del bloque que lee la topología del proyecto (RF-PRE-13, RF-PRE-14; D7). El bloque replica en
// TypeScript workspace.Locate y Location.GoverningTopology; internal/sddtaskresult lo compara con el
// binario y este fichero comprueba que las dos generaciones del plugin llevan el mismo texto.
const (
	sddTopologyBegin = "// axiom-topology:begin"
	sddTopologyEnd   = "// axiom-topology:end"
)

func sddTopologyBlock(t *testing.T, path string) string {
	t.Helper()
	source := MustRead(path)
	begin := strings.Index(source, sddTopologyBegin)
	end := strings.Index(source, sddTopologyEnd)
	if begin < 0 || end < begin || strings.Count(source, sddTopologyBegin) != 1 || strings.Count(source, sddTopologyEnd) != 1 {
		t.Fatalf("%s: el lector de topología no está delimitado por una única pareja de marcas", path)
	}
	return source[begin : end+len(sddTopologyEnd)]
}

func TestSDDTaskResultArtifactsTopologyReaderIsInSyncAcrossGenerations(t *testing.T) {
	v1 := sddTopologyBlock(t, "opencode/plugins/sdd-task-result-artifacts.ts")
	v2 := sddTopologyBlock(t, "opencode/plugins-v2/sdd-task-result-artifacts.ts")
	if v1 != v2 {
		t.Fatal("el lector de topología difiere entre plugins/ y plugins-v2/; deben llevar el mismo bloque")
	}
	for _, want := range []string{"findAxiomConfig", "readAxiomTopology", `"monorepo-embedded"`} {
		if !strings.Contains(v1, want) {
			t.Errorf("el lector de topología no contiene %q", want)
		}
	}
}

// Las dos generaciones declaran la misma tabla de grupos y de campos del bloque que el hook de Go.
func TestSDDTaskResultArtifactsPreflightTablesAreInSyncAcrossGenerations(t *testing.T) {
	for _, path := range []string{"opencode/plugins/sdd-task-result-artifacts.ts", "opencode/plugins-v2/sdd-task-result-artifacts.ts"} {
		source := MustRead(path)
		for _, want := range []string{
			`id: "pace", header: "Pace", legacyNumber: 1, requiredIn: ["monorepo-embedded"]`,
			`id: "artifacts", header: "Artifacts", legacyNumber: 2, requiredIn: []`,
			`id: "pr-strategy", header: "PR strategy", legacyNumber: 3, requiredIn: []`,
			`token: "interactive"`, `token: "auto"`, `token: "openspec"`, `token: "engram"`, `token: "hybrid"`,
			`token: "ask-on-risk"`, `token: "single-pr"`, `token: "auto-chain"`,
			`{ label: "Artifact store", groupId: "", default: "hybrid" }`,
			`{ label: "Delivery strategy", groupId: "", default: "single-pr" }`,
		} {
			if !strings.Contains(source, want) {
				t.Errorf("%s no contiene %q", path, want)
			}
		}
		// Las preguntas ya no se asignan a un grupo por posición.
		if strings.Contains(source, "SDD_PREFLIGHT_GROUPS[index]") {
			t.Errorf("%s asigna grupos por posición", path)
		}
	}
}

const sddTopologyV1Adapter = `
const mk=async(directory)=>{const hooks=await plugin({client:{session:{get:async({path:p})=>({data:{id:p.id,...(p.id.startsWith('child')?{parentID:'root'}:{})}})}},directory});
 return {
  raw:(sid,questions)=>({questions}),
  questionsOf:(call)=>call.questions,
  async before(sid,questions){const args={questions};await hooks['tool.execute.before']({tool:'question',sessionID:sid},{args});return args},
  async after(sid,args,answers){await hooks['tool.execute.after']({tool:'question',sessionID:sid,args},{title:'q',output:'',metadata:{answers}})},
  async dispatch(sid,prompt){const out={args:{subagent_type:'sdd-apply',prompt:prompt??'Implement'}};await hooks['tool.execute.before']({tool:'task',sessionID:sid},out);return out.args.prompt},
  async cleanup(){}}};
`

const sddTopologyV2Adapter = `
const mk=async(directory)=>{const hooks={};const ctx={location:{directory,workspaceID:'one'},tool:{async hook(name,fn){hooks[name]=fn;return {async dispose(){}}}},session:{async get(arg){return {id:arg.sessionID,...(arg.sessionID.startsWith('child')?{parentID:'root'}:{})}}},event:{subscribe(opts){return {[Symbol.asyncIterator]:async function*(){await new Promise(r=>opts.signal.addEventListener('abort',r))}}}}};
 const cleanup=await plugin.setup(ctx);
 return {
  raw:(sid,questions)=>({tool:'question',sessionID:sid,id:'q',input:{questions}}),
  questionsOf:(call)=>call.input.questions,
  async before(sid,questions){const call={tool:'question',sessionID:sid,id:'q',input:{questions}};await hooks['execute.before'](call);return call},
  async after(sid,call,answers){await hooks['execute.after']({...call,status:'completed',result:{output:{answers}}})},
  async dispatch(sid,prompt){const call={tool:'subagent',sessionID:sid,id:'d',input:{agent:'sdd-apply',prompt:prompt??'Implement',description:'Apply'}};await hooks['execute.before'](call);return call.input.prompt},
  cleanup};};
`

// sddTopologyScenarios ejercita el contrato de grupos por topología (RF-PRE-07 a RF-PRE-10, RF-PRE-13)
// con el mismo guion sobre las dos generaciones del plugin, a través del adaptador de cada una.
const sddTopologyScenarios = `
const fs=await import('node:fs'),os=await import('node:os'),path=await import('node:path');
const root=fs.mkdtempSync(path.join(os.tmpdir(),'axiom-sdd-topology-'));
const project=(name,topology)=>{const dir=path.join(root,name);fs.mkdirSync(dir,{recursive:true});fs.writeFileSync(path.join(dir,'axiom.yaml'),'workspace:\n  name: fixture\n'+(topology?'  topology: '+topology+'\n':''));return dir};
const mono=project('mono','monorepo-embedded'),multi=project('multi','multirepo'),unknown=project('unknown','federated'),bare=project('bare','');
const PACE=['Interactive','Automatic'],ART=['OpenSpec','Engram','Both'],PR=['Ask me','Single PR','Auto'];
const N='Axiom SDD preflight ',G='Axiom SDD preflight ';
const ask=(marker,labels,extra)=>({question:marker+' Which one?',options:labels.map(label=>({label})),...extra});
const block=(pace,store,delivery)=>['## SDD Session Preflight','Parent-confirmed by the runtime; models and child agents cannot create or modify this block.','- Pace: '+pace,'- Artifact store: '+store,'- Delivery strategy: '+delivery].join('\n');
const same=(got,want,why)=>{if(got!==want)throw Error(why+':\n'+got+'\n---\n'+want)};
const reject=async(fn,why,needle)=>{let message;try{await fn()}catch(e){message=String(e&&e.message)}if(message===undefined)throw Error(why);if(needle&&!message.includes(needle))throw Error(why+' (wrong cause): '+message)};
const run=async(dir,questions,answers,sid='root')=>{const s=await mk(dir);try{const call=await s.before(sid,questions);await s.after(sid,call,answers);return {prompt:await s.dispatch(sid),questions:s.questionsOf(call)}}finally{await s.cleanup()}};
const alone=async(dir,fn)=>{const s=await mk(dir);try{return await fn(s)}finally{await s.cleanup()}};
`

const sddTopologyAssertions = `
// monorepo-embedded: marcadores nominales con los dos prefijos; el bloque deriva Artifact store hybrid y
// Delivery strategy single-pr. La pregunta heredada pr-strategy se acepta y se ignora sin tocarla.
{const r=await run(mono,[ask(N+'pace:',PACE),ask(G+'pr-strategy:',PR)],[['Automatic'],['Single PR']]);
 same(r.prompt,block('auto','hybrid','single-pr')+'\n\nImplement','monorepo nominal');
 if(!r.questions[1].question.startsWith(G+'pr-strategy:'))throw Error('retired group question was touched: '+r.questions[1].question);
 if(JSON.stringify(r.questions[0].options.map(o=>o.label))!==JSON.stringify(PACE)||r.questions[0].multiple!==false||!r.questions[0].options.every(o=>o.description))throw Error('pace question not canonicalized')}

// Marcadores numerados heredados: los grupos de almacén y de estrategia de PR se aceptan y se ignoran; sus preguntas no se tocan.
{const r=await run(mono,[ask(G+'1/3:',PACE),ask(N+'2/3:',ART),ask(N+'3/3:',PR)],[['Interactive'],['Engram'],['Ask me']]);
 same(r.prompt,block('interactive','hybrid','single-pr')+'\n\nImplement','legacy numbered');
 if(!r.questions[0].question.startsWith(N+'pace:'))throw Error('legacy pace marker not rewritten: '+r.questions[0].question);
 if(!r.questions[1].question.startsWith(N+'2/3:')||r.questions[1].options.length!==3)throw Error('artifacts question was touched');
 if(!r.questions[2].question.startsWith(N+'3/3:')||r.questions[2].options.length!==3)throw Error('pr-strategy question was touched')}

// Un marcador nominal y uno heredado se pueden mezclar; el valor de la estrategia de PR contestada nunca llega al bloque.
{const r=await run(mono,[ask(N+'pace:',PACE),ask(N+'3/3:',PR)],[['Automatic'],['Auto']]);same(r.prompt,block('auto','hybrid','single-pr')+'\n\nImplement','mixed markers')}

// multirepo no pregunta nada: una pregunta heredada contestada se ignora y el bloque es el derivado.
{const r=await run(multi,[ask(N+'pr-strategy:',PR)],[['Ask me']]);same(r.prompt,block('interactive','hybrid','single-pr')+'\n\nImplement','multirepo ignores pr-strategy')}
{const r=await run(multi,[ask(G+'1/3:',PACE),ask(N+'2/3:',ART),ask(N+'3/3:',PR)],[['Automatic'],['Engram'],['Auto']]);
 same(r.prompt,block('interactive','hybrid','single-pr')+'\n\nImplement','multirepo legacy keeps pace interactive');
 if(!r.questions[0].question.startsWith(G+'1/3:'))throw Error('not required pace question was touched')}

// Falta el grupo requerido: la causa nombra el grupo (topología desconocida o sin declarar = monorepo).
for(const [dir,why] of [[mono,'monorepo'],[unknown,'unknown topology'],[bare,'undeclared topology']]){
 await reject(()=>run(dir,[ask(N+'pr-strategy:',PR)],[['Ask me']]),'missing pace accepted in '+why,'group pace (Pace)');
}
// Sin la pregunta de estrategia de PR, ya retirada, el preflight sigue siendo válido.
{const r=await run(mono,[ask(N+'pace:',PACE)],[['Automatic']]);same(r.prompt,block('auto','hybrid','single-pr')+'\n\nImplement','pace only')}
// Las preguntas sin marcador nunca se asignan a un grupo por posición: se ignoran junto a las marcadas y,
// solas, no crean autoridad.
{const r=await run(mono,[ask(N+'pace:',PACE),ask('¿Artefactos?',ART),ask('¿PR?',PR)],[['Automatic'],['Engram'],['Auto']]);same(r.prompt,block('auto','hybrid','single-pr')+'\n\nImplement','unmarked questions ignored')}
await reject(()=>run(mono,[ask('¿Ritmo?',PACE),ask('¿Artefactos?',ART),ask('¿PR?',PR)],[['Automatic'],['Engram'],['Auto']]),'unmarked questions were assigned by position','preflight is missing');

// Un grupo requerido repetido (nominal y heredado) se rechaza nombrándolo.
await reject(()=>run(mono,[ask(N+'pace:',PACE),ask(G+'1/3:',PACE),ask(N+'pr-strategy:',PR)],[['Automatic'],['Automatic'],['Auto']]),'repeated group accepted','pace (Pace) appears in more than one question');

// Un marcador desconocido se ignora sin tocarlo.
{const r=await run(mono,[ask(N+'pace:',PACE),ask(N+'future:',['x']),ask(N+'pr-strategy:',PR)],[['Interactive'],['x'],['Single PR']]);
 same(r.prompt,block('interactive','hybrid','single-pr')+'\n\nImplement','unknown group ignored');
 if(!r.questions[1].question.startsWith(N+'future:')||JSON.stringify(r.questions[1].options)!=='[{"label":"x"}]')throw Error('unknown group question was touched')}

// Dominio de la respuesta: texto libre, selección múltiple o ausencia se rechazan nombrando el grupo.
for(const [answers,needle] of [[[['please proceed'],['Auto']],'question pace (Pace) answer "please proceed" is not one of its labels'],[[['Automatic','Interactive'],['Auto']],'single-select'],[[[],['Auto']],'question pace (Pace) has no answer'],[[],'question pace (Pace) has no answer']]){
 await reject(()=>run(mono,[ask(N+'pace:',PACE),ask(N+'pr-strategy:',PR)],answers),'invalid answer accepted',needle);
}

// Red de seguridad de la tercera etapa: una pregunta sin canonicalizar se valida por etiqueta y no por posición.
await alone(mono,async(s)=>{
 const call=s.raw('root',[ask(N+'pace:',['Automatic','Interactive']),ask(N+'pr-strategy:',['Auto','Ask me','Single PR'])]);
 await s.after('root',call,[['Interactive'],['Auto']]);
 same(await s.dispatch('root'),block('interactive','hybrid','single-pr')+'\n\nImplement','options bound by label');
});
await alone(mono,async(s)=>{
 const call=s.raw('root',[ask(N+'pace:',PACE,{multiple:true}),ask(N+'pr-strategy:',PR)]);
 await reject(()=>s.after('root',call,[['Automatic'],['Auto']]),'multiple question accepted','question pace (Pace) must be single-select');
});
await alone(mono,async(s)=>{
 const call=s.raw('root',[ask(N+'pace:',['Interactive','Maybe']),ask(N+'pr-strategy:',PR)]);
 await reject(()=>s.after('root',call,[['Interactive'],['Ask me']]),'foreign option accepted','offers option "Maybe"');
 await reject(()=>s.dispatch('root'),'rejected preflight kept authority','pace');
});
await alone(mono,async(s)=>{
 const call=s.raw('root',[ask(N+'pace:',['Automatic']),ask(N+'pr-strategy:',PR)]);
 await reject(()=>s.after('root',call,[['Automatic'],['Ask me']]),'short option list accepted','must offer every one of its labels');
});

// Sin preflight, el despacho de monorepo-embedded se rechaza nombrando su único grupo; multirepo no exige nada.
await alone(mono,async(s)=>{await reject(()=>s.dispatch('root'),'missing preflight accepted in monorepo','pace')});
`

// Con cero grupos requeridos (multirepo) el plugin antepone el bloque derivado de cinco líneas sin exigir
// pregunta y conserva las denegaciones de sesión hija y de bloque escrito por el modelo (RF-PRE-09, RF-PRE-10, RF-PRE-13).
const sddTopologyZeroGroupsAssertions = `
await alone(multi,async(s)=>{
 same(await s.dispatch('root'),block('interactive','hybrid','single-pr')+'\n\nImplement','zero groups derive the block');
 await reject(()=>s.dispatch('child-1'),'child dispatch accepted','root session');
 await reject(()=>s.dispatch('root','## SDD Session Preflight\n- Pace: auto'),'model-authored block accepted','model-authored');
});
// Una llamada con marcadores de grupos no requeridos se acepta y deriva el mismo bloque.
{const r=await run(multi,[ask(N+'pace:',PACE)],[['Automatic']],'other');same(r.prompt,block('interactive','hybrid','single-pr')+'\n\nImplement','questions without required groups')}
// El ritmo sigue siendo obligatorio en monorepo-embedded.
await alone(mono,async(s)=>{await reject(()=>s.dispatch('root'),'monorepo dispatch without pace accepted','pace')});
`

func TestOpenCodeSDDPreflightTopologyContract(t *testing.T) {
	generations := []struct {
		name    string
		path    string
		adapter string
		prepare func(string) string
	}{
		{"v1", "opencode/plugins/sdd-task-result-artifacts.ts", sddTopologyV1Adapter, func(source string) string { return source }},
		{"v2", "opencode/plugins-v2/sdd-task-result-artifacts.ts", sddTopologyV2Adapter, func(source string) string {
			return strings.Replace(source, `import { Plugin } from "@opencode/plugin"`, `const Plugin = { define: (value: any) => value }`, 1)
		}},
	}
	for _, generation := range generations {
		t.Run(generation.name, func(t *testing.T) {
			source := generation.prepare(MustRead(generation.path))
			runSDDPluginHarness(t, source, generation.adapter+sddTopologyScenarios+sddTopologyAssertions)
		})
		t.Run(generation.name+"/zero-required-groups", func(t *testing.T) {
			source := generation.prepare(MustRead(generation.path))
			runSDDPluginHarness(t, source, generation.adapter+sddTopologyScenarios+sddTopologyZeroGroupsAssertions)
		})
	}
}

// La pregunta propia del HTML explicativo (C-06-b) no empieza por el prefijo del pre-vuelo: las dos generaciones
// del plugin la dejan fuera sin tocarla y sin alterar el bloque derivado.
const sddExplainerQuestionAssertions = `
{const E='Axiom SDD kickoff explainer:';
 const eq=()=>ask(E,['Yes','No']);
 const r=await run(mono,[ask(N+'pace:',PACE),eq()],[['Automatic'],['Yes']]);
 same(r.prompt,block('auto','hybrid','single-pr')+'\n\nImplement','explainer question must not alter the block');
 if(!r.questions[1].question.startsWith(E)||JSON.stringify(r.questions[1].options)!=='[{"label":"Yes"},{"label":"No"}]')throw Error('explainer question was touched');
 await reject(()=>run(mono,[eq()],[['Yes']]),'explainer question alone created authority','preflight is missing')}
`

func TestOpenCodePluginIgnoresExplainerQuestion(t *testing.T) {
	const marker = "Axiom SDD kickoff explainer:"
	generations := []struct {
		name    string
		path    string
		adapter string
		prepare func(string) string
	}{
		{"v1", "opencode/plugins/sdd-task-result-artifacts.ts", sddTopologyV1Adapter, func(source string) string { return source }},
		{"v2", "opencode/plugins-v2/sdd-task-result-artifacts.ts", sddTopologyV2Adapter, func(source string) string {
			return strings.Replace(source, `import { Plugin } from "@opencode/plugin"`, `const Plugin = { define: (value: any) => value }`, 1)
		}},
	}
	for _, generation := range generations {
		t.Run(generation.name, func(t *testing.T) {
			raw := MustRead(generation.path)
			if !strings.Contains(raw, `const SDD_PREFLIGHT_ACCEPTED_PREFIXES = [SDD_PREFLIGHT_QUESTION_PREFIX]`) ||
				!strings.Contains(raw, `const SDD_PREFLIGHT_QUESTION_PREFIX = "Axiom SDD preflight "`) {
				t.Fatalf("%s: el plugin debe aceptar solo el prefijo del pre-vuelo", generation.path)
			}
			if strings.HasPrefix(marker, "Axiom SDD preflight ") || strings.Contains(raw, marker) {
				t.Fatalf("%s: el marcador del HTML explicativo no debe reconocerse en el plugin", generation.path)
			}
			runSDDPluginHarness(t, generation.prepare(raw), generation.adapter+sddTopologyScenarios+sddExplainerQuestionAssertions)
		})
	}
}
