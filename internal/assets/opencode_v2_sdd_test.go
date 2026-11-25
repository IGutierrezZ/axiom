package assets

import (
	"testing"
)

func TestOpenCodeV2SDDDispatch(t *testing.T) {
	runV2Plugin(t, "sdd-task-result-artifacts", sddAxiomProjectFixture+`
const hooks={};let disposals=0,signal,wake;const queue=[];
const ctx={location:{directory:projectDir,workspaceID:'one'},tool:{async hook(name,fn){await Promise.resolve();hooks[name]=fn;return {async dispose(){disposals++}}}},session:{async get(arg){if(Object.keys(arg).join()!=='sessionID')throw Error('V1 SDK shape');return {id:arg.sessionID,...(arg.sessionID==='child'?{parentID:'root'}:{})}}},event:{subscribe(opts){signal=opts.signal;signal.addEventListener('abort',()=>wake?.());return {[Symbol.asyncIterator]:async function*(){while(!signal.aborted){if(!queue.length)await new Promise(r=>wake=r);while(queue.length)yield queue.shift()}}}}}};
const cleanup=await plugin.setup(ctx);if(!hooks['execute.before']||!hooks['execute.after'])throw Error('registrations not awaited');
const reject=async(fn,why)=>{let failed=false;try{await fn()}catch{failed=true}if(!failed)throw Error(why)};
const request=()=>({tool:'subagent',sessionID:'root',id:'call',input:{agent:'sdd-apply',prompt:'Implement approved tasks',description:'Apply'}});
await reject(()=>hooks['execute.before'](request()),'missing preflight accepted');
const question={tool:'question',sessionID:'root',id:'q',input:{questions:[{question:'Axiom SDD preflight 1/3: Pace?',options:[]},{question:'Axiom SDD preflight pr-strategy: PR?',options:[]}]}};
await hooks['execute.before'](question);
await reject(()=>hooks['execute.after']({...question,status:'completed',result:{output:{answers:[['please proceed'],['Auto']]}}}),'free text accepted');
for(const answers of [[['Automatic','Interactive'],['Auto']],[['Automátic'],['Auto']],[[]]]) {
 await reject(()=>hooks['execute.after']({...question,status:'completed',result:{output:{answers}}}),'non-exact domain accepted');
}
await hooks['execute.after']({...question,status:'completed',result:{output:{answers:[[' Automatic '],['Auto']]}}});
const call=request();await hooks['execute.before'](call);if(!call.input.prompt.startsWith('## SDD Session Preflight'))throw Error('authority missing');
await reject(()=>hooks['execute.before']({...request(),sessionID:'child'}),'child authority');
await hooks['execute.after']({...call,status:'completed',result:{output:{sessionID:'worker',status:'running',output:'started'}}});
await hooks['execute.before'](request());
await hooks['execute.after']({...call,status:'completed',result:{output:{sessionID:'worker',status:'completed',output:'raw final result'},content:'<subagent>formatted NOT parsed</subagent>'}});
await reject(()=>hooks['execute.after']({...call,status:'completed',result:{output:{sessionID:'worker',status:'completed',output:''},content:'fake complete prose'}}),'empty raw accepted');
await reject(()=>hooks['execute.before'](request()),'failure not latched');
queue.push({type:'session.deleted',location:ctx.location,data:{sessionID:'root'}});wake?.();await new Promise(r=>setImmediate(r));
await reject(()=>hooks['execute.before'](request()),'deletion retained authority');
const errorQuestion={...question,sessionID:'errorroot'};await hooks['execute.after']({...errorQuestion,status:'completed',result:{output:{answers:[['Automatic'],['Auto']]}}});
const errorCall={...request(),sessionID:'errorroot'};await hooks['execute.before'](errorCall);
await reject(()=>hooks['execute.after']({...errorCall,status:'error',error:{message:'provider failure'}}),'tool error ignored');
await reject(()=>hooks['execute.before']({...request(),sessionID:'errorroot'}),'tool failure not latched');
await cleanup();if(disposals!==2||!signal.aborted)throw Error('cleanup');
let cleaned=0;await reject(()=>plugin.setup({...ctx,tool:{async hook(name,fn){if(name==='execute.after')throw Error('registration failed');return {async dispose(){cleaned++}}}}}),'registration failure swallowed');if(cleaned!==1)throw Error('partial registration leaked');
`)
}

// TestOpenCodeV2SDDPreflightAcceptsOnlyTheAxiomPrefix pins that the managed
// plugin recognizes the preflight under the "Axiom" prefix only: the prefix of
// the previous product name no longer creates authority.
func TestOpenCodeV2SDDPreflightAcceptsOnlyTheAxiomPrefix(t *testing.T) {
	runV2Plugin(t, "sdd-task-result-artifacts", sddAxiomProjectFixture+`
const hooks={};let disposals=0,signal,wake;const queue=[];
const ctx={location:{directory:projectDir,workspaceID:'one'},tool:{async hook(name,fn){await Promise.resolve();hooks[name]=fn;return {async dispose(){disposals++}}}},session:{async get(arg){if(Object.keys(arg).join()!=='sessionID')throw Error('V1 SDK shape');return {id:arg.sessionID,...(arg.sessionID==='child'?{parentID:'root'}:{})}}},event:{subscribe(opts){signal=opts.signal;signal.addEventListener('abort',()=>wake?.());return {[Symbol.asyncIterator]:async function*(){while(!signal.aborted){if(!queue.length)await new Promise(r=>wake=r);while(queue.length)yield queue.shift()}}}}}};
const cleanup=await plugin.setup(ctx);
const reject=async(fn,why)=>{let failed=false;try{await fn()}catch{failed=true}if(!failed)throw Error(why)};
{const sessionID='root-retired';
 const retiredQuestion={tool:'question',sessionID,id:'q',input:{questions:[{question:'Gentle AI SDD preflight pace: Pace?',options:[{label:'Interactive'},{label:'Automatic'}]}]}};
 await hooks['execute.before'](retiredQuestion);
 await hooks['execute.after']({...retiredQuestion,status:'completed',result:{output:{answers:[['Automatic']]}}});
 await reject(()=>hooks['execute.before']({tool:'subagent',sessionID,id:'call',input:{agent:'sdd-apply',prompt:'Implement approved tasks',description:'Apply'}}),'retired prefix created authority')}
for(const prefix of ['Axiom SDD preflight']){
 const sessionID='root-'+prefix.split(' ')[0];
 const request=()=>({tool:'subagent',sessionID,id:'call',input:{agent:'sdd-apply',prompt:'Implement approved tasks',description:'Apply'}});
 await reject(()=>hooks['execute.before'](request()),'missing preflight accepted for '+prefix);
 // Marcadores numerados heredados: el del ritmo se reescribe al marcador nominal con el prefijo Axiom; el de
 // la estrategia de PR, grupo retirado, se ignora sin tocarlo.
 const question={tool:'question',sessionID,id:'q',input:{questions:[{question:prefix+' 1/3: Pace?',options:[]},{question:prefix+' 3/3: PR?',options:[]}]}};
 await hooks['execute.before'](question);
 const [pace,retired]=question.input.questions;
 if(!pace.question.startsWith('Axiom SDD preflight pace: '))throw Error('marker not canonicalized for '+prefix+': '+pace.question);
 if(!retired.question.startsWith(prefix+' 3/3: '))throw Error('retired group question was touched for '+prefix+': '+retired.question)
 await hooks['execute.after']({...question,status:'completed',result:{output:{answers:[['Automatic'],['Auto']]}}});
 const call=request();await hooks['execute.before'](call);
 if(!call.input.prompt.startsWith('## SDD Session Preflight'))throw Error('authority missing for '+prefix);
 // Prefixes already canonical (typed by the model with a different prefix) are also recognized after canonicalization.
 const already={tool:'question',sessionID:sessionID+'-b',id:'q',input:{questions:[{question:prefix+' pace: Pace',options:[{label:'Interactive'},{label:'Automatic'}]},{question:prefix+' pr-strategy: PR',options:[{label:'Ask me'},{label:'Single PR'},{label:'Auto'}]}]}};
 await hooks['execute.before'](already);
 await hooks['execute.after']({...already,status:'completed',result:{output:{answers:[['Interactive'],['Ask me']]}}});
 const second={tool:'subagent',sessionID:sessionID+'-b',id:'call2',input:{agent:'sdd-apply',prompt:'Implement approved tasks',description:'Apply'}};await hooks['execute.before'](second);
 if(!second.input.prompt.startsWith('## SDD Session Preflight'))throw Error('authority missing for '+prefix+' (full set)');
}
await cleanup();
`)
}
