package assets

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runV2Plugin(t *testing.T, name, harness string) {
	t.Helper()
	source, err := Read("opencode/plugins-v2/" + name + ".ts")
	if err != nil {
		t.Fatal(err)
	}
	source = strings.Replace(source, `import { spawn } from "node:child_process"`, `const spawn = (...args: any[]) => (globalThis as any).__spawn(...args)`, 1)
	source = strings.Replace(source, `import { Plugin } from "@opencode/plugin"`, `const Plugin = { define: (value: any) => value }`, 1)
	source = strings.Replace(source, `import { execFile } from "node:child_process"`, `const execFile = (...args: any[]) => (globalThis as any).__exec(...args)`, 1)
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
	cmd.Env = append(os.Environ(), "CI=0", "GITHUB_ACTIONS=0", "HOME="+dir, "XDG_CONFIG_HOME="+dir, "XDG_DATA_HOME="+dir)
	if out, err := cmd.CombinedOutput(); err != nil || strings.Contains(string(out), "SECRET") {
		t.Fatalf("V2 harness: %v\n%s", err, out)
	}
}

func TestOpenCodeV2CatalogAndRegistry(t *testing.T) {
	legacy, err := Read("opencode/plugins/skill-registry.ts")
	if err != nil {
		t.Fatal(err)
	}
	native, err := Read("opencode/plugins-v2/skill-registry.ts")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(legacy, "\n") {
		if strings.HasPrefix(line, "const PROJECT_MARKERS =") && !strings.Contains(native, line) {
			t.Fatal("V2 project marker guard diverged")
		}
	}

	runV2Plugin(t, "model-variants", `
const fs=await import('node:fs/promises');const path=await import('node:path');const os=await import('node:os');const crypto=await import('node:crypto');
const root=await fs.mkdtemp(path.join(os.tmpdir(),'catalog-v2-'));process.env.HOME=root;process.env.USERPROFILE=root;
let signal,wake;const queue=[];let revision=0;let subscribed=false;
const location={directory:'/project',workspaceID:'one'};
const ctx={location,model:{async list(){if(!subscribed)throw Error("snapshot before subscription");return {location:{directory:location.directory},data:[{providerID:'openai',id:'model',variants:[{id:revision?'high':'low'}]}]}}},event:{subscribe(opts){signal=opts.signal;signal.addEventListener('abort',()=>wake?.());return {[Symbol.asyncIterator]:async function*(){subscribed=true;while(!signal.aborted){if(!queue.length)await new Promise(r=>wake=r);while(queue.length)yield queue.shift()}}}}}};
const cleanup=await plugin.setup(ctx);const tick=()=>new Promise(r=>setTimeout(r,30));await tick();
const dir=path.join(root,'.gentle-ai','cache','opencode-v2');const read=async()=>JSON.parse(await fs.readFile(path.join(dir,(await fs.readdir(dir))[0]),'utf8'));
if((await read()).openai.model[0]!=='low')throw Error('catalog');revision=1;queue.push({type:'model.updated',location});wake?.();await tick();if((await read()).openai.model[0]!=='high')throw Error('refresh');
if(await fs.stat(path.join(root,'.gentle-ai','cache','model-variants.json')).then(()=>true,()=>false))throw Error('legacy cache overwritten');await cleanup();if(!signal.aborted)throw Error('catalog disposal');
const second=await plugin.setup({location:{...location,workspaceID:'two'},model:ctx.model,event:{subscribe(opts){return {[Symbol.asyncIterator]:async function*(){await new Promise(r=>opts.signal.addEventListener('abort',r))}}}}});await tick();
if((await fs.readdir(dir)).length!==2)throw Error('workspace caches collide');await second();await fs.rm(root,{recursive:true});
`)
	runV2Plugin(t, "skill-registry", `
const fs=await import('node:fs/promises');const os=await import('node:os');const path=await import('node:path');const root=await fs.mkdtemp(path.join(os.tmpdir(),'registry-v2-'));await fs.mkdir(path.join(root,'.git'));
const calls=[];globalThis.__exec=(cmd,args,opts,cb)=>{const child={kill(){this.killed=true}};calls.push({cmd,args,opts,child});return child};
const cleanup=await plugin.setup({location:{directory:root,project:{directory:'/wrong'}}});if(calls.length!==1||calls[0].args.at(-1)!==root||calls[0].opts.cwd!==root)throw Error('registry location');await cleanup();if(!calls[0].child.killed)throw Error('registry child cleanup');await plugin.setup({location:{directory:os.homedir()}});if(calls.length!==1)throw Error('home guard');await fs.rm(root,{recursive:true});
`)
}
