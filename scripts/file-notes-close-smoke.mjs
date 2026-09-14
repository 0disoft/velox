import fs from 'node:fs/promises';
import {resolve,join} from 'node:path';
import {spawn,spawnSync} from 'node:child_process';
import {createServer} from 'node:net';
import {createHash} from 'node:crypto';
const root=resolve('.'), work=join(root,'.cache','close16',String(Date.now()));
await fs.mkdir(work,{recursive:true});
await fs.cp(join(root,'examples/file-notes/web'),join(work,'web'),{recursive:true});
const index=join(work,'web/index.html');
await fs.writeFile(index,(await fs.readFile(index,'utf8')).replace('<head>','<head><script>Object.defineProperty(window,"indexedDB",{value:{open(){throw Error("Controlled unavailable recovery")}}});</script>'));
await fs.writeFile(join(work,'runtime.json'),JSON.stringify({runtimeVersion:1,app:{id:'dev.velox.closeprobe',name:'Velox close probe',version:'0.1.0'},assets:{root:'web',entry:'index.html'},window:{width:800,height:600},security:{permissions:[]}}));
const server=createServer(); await new Promise(r=>server.listen(0,'127.0.0.1',r));const port=server.address().port;await new Promise(r=>server.close(r));
const child=spawn(join(root,'dist/velox-host.exe'),['--config',join(work,'runtime.json'),'--debug'],{cwd:root,env:{...process.env,VELOX_DATA_DIR:join(work,'profile'),WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS:'--remote-debugging-address=127.0.0.1 --remote-debugging-port='+port},stdio:['ignore','pipe','pipe'],windowsHide:true});
let output='',exited=false;child.on('exit',()=>{exited=true});for(const s of[child.stdout,child.stderr])s.on('data',b=>output+=b);
const wait=ms=>new Promise(r=>setTimeout(r,ms));let ws,id=0;const pending=new Map(),dialogs=[];
const result={work,baseline:process.argv.includes('--baseline'),hostSha256:createHash('sha256').update(await fs.readFile(join(root,'dist/velox-host.exe'))).digest('hex')};
try{
 let target;for(let n=0;n<100&&!exited;n++){try{target=(await(await fetch('http://127.0.0.1:'+port+'/json/list')).json()).find(p=>p.type==='page'&&p.url.startsWith('https://'));if(target)break}catch{}await wait(200)}
 if(!target)throw Error('No target: '+output);
 ws=new WebSocket(target.webSocketDebuggerUrl);await new Promise((r,j)=>{ws.onopen=r;ws.onerror=j});
 ws.onmessage=e=>{const m=JSON.parse(e.data);if(m.method==='Page.javascriptDialogOpening')dialogs.push(m.params);if(pending.has(m.id)){pending.get(m.id)(m);pending.delete(m.id)}};
 const call=(method,params={})=>new Promise((r,j)=>{const n=++id,t=setTimeout(()=>{pending.delete(n);j(Error(method+' timeout'))},10000);pending.set(n,m=>{clearTimeout(t);m.error?j(Error(JSON.stringify(m.error))):r(m.result)});ws.send(JSON.stringify({id:n,method,params}))});
 const evalJS=async expression=>(await call('Runtime.evaluate',{expression,returnByValue:true})).result.value;
 await call('Page.enable');
 result.browser=await call('Browser.getVersion');
 for(let n=0;n<50;n++){if(await evalJS('!!document.querySelector("textarea")'))break;await wait(100)}
 const rect=await evalJS('(()=>{const r=document.querySelector("textarea").getBoundingClientRect();return {x:r.x+30,y:r.y+30}})()');
 await call('Input.dispatchMouseEvent',{type:'mousePressed',...rect,button:'left',clickCount:1});await call('Input.dispatchMouseEvent',{type:'mouseReleased',...rect,button:'left',clickCount:1});await call('Input.insertText',{text:'UNSAVED-CLOSE-PROBE'});
 await evalJS('(()=>{const e=document.querySelector("textarea");e.value="UNSAVED-CLOSE-PROBE";e.dispatchEvent(new Event("input",{bubbles:true}))})()');
 result.activation=await evalJS('navigator.userActivation.hasBeenActive');
 result.recoveryUnavailable=await evalJS('document.body.textContent.includes("Draft recovery unavailable")');
 if(!result.activation||!result.recoveryUnavailable)throw Error('Fixture preconditions failed');
 result.beforeNavigation=await evalJS('({text:document.querySelector("textarea").value,dirty:document.body.textContent.includes("Unsaved changes")})');
 if(result.beforeNavigation.text!=='UNSAVED-CLOSE-PROBE'||!result.beforeNavigation.dirty)throw Error('Document was not dirty');
 const reload=call('Runtime.evaluate',{expression:'location.reload()'}).catch(()=>{});
 for(let n=0;n<50&&!dialogs.length;n++)await wait(100);
 if(dialogs[0]?.type!=='beforeunload')throw Error('Navigation control did not produce beforeunload');
 await call('Page.handleJavaScriptDialog',{accept:false});await reload;
 result.navigationCancelPreservesText=await evalJS('document.querySelector("textarea").value === "UNSAVED-CLOSE-PROBE"');dialogs.length=0;
 const nativeClose=()=>{const p=spawnSync('pwsh',['-NoProfile','-NonInteractive','-Command',`Add-Type -TypeDefinition 'using System; using System.Text; using System.Runtime.InteropServices; public static class CloseProbe { public delegate bool Callback(IntPtr h, IntPtr p); [DllImport("user32.dll")] public static extern bool EnumWindows(Callback c, IntPtr p); [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid); [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr h, StringBuilder s, int n); [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l); public static bool Close(uint target) { IntPtr found=IntPtr.Zero; EnumWindows((h,p)=>{uint id;GetWindowThreadProcessId(h,out id);var s=new StringBuilder(256);GetWindowText(h,s,256);if(id==target && s.ToString()=="Velox close probe")found=h;return true;},IntPtr.Zero);return found!=IntPtr.Zero && PostMessage(found,0x10,IntPtr.Zero,IntPtr.Zero); } }'; if(-not [CloseProbe]::Close(${child.pid})){throw 'No owned native window or PostMessage failed'}`],{windowsHide:true,encoding:'utf8',timeout:15000});if(p.status!==0)throw Error(p.stderr)};
 nativeClose();for(let n=0;n<50&&!exited&&!dialogs.length;n++)await wait(100);
 result.nativeCloseOffersConsent=dialogs[0]?.type==='beforeunload';result.closedWithoutConsent=exited;
 if(result.nativeCloseOffersConsent){await call('Page.handleJavaScriptDialog',{accept:false});result.nativeCancelPreservesText=await evalJS('document.querySelector("textarea").value === "UNSAVED-CLOSE-PROBE"');dialogs.length=0;nativeClose();for(let n=0;n<50&&!dialogs.length&&!exited;n++)await wait(100);if(dialogs[0]?.type!=='beforeunload')throw Error('Repeated close lacks consent');await call('Page.handleJavaScriptDialog',{accept:true});for(let n=0;n<100&&!exited;n++)await wait(100);result.acceptedCloseExits=exited;}
 if(exited){for(let n=0;n<100;n++){try{await fs.rename(join(work,'profile'),join(work,'profile-released'));result.profileRenameSucceeded=true;break}catch{await wait(100)}}}
 result.passed=result.navigationCancelPreservesText&&result.nativeCloseOffersConsent&&result.nativeCancelPreservesText&&result.acceptedCloseExits&&result.profileRenameSucceeded;
 if(!result.baseline&&!result.passed)throw Error('Native close contract failed');
}catch(e){result.error=String(e);throw e}finally{ws?.close();if(!exited)spawnSync('taskkill.exe',['/PID',String(child.pid),'/T','/F'],{windowsHide:true,encoding:'utf8'});await fs.writeFile(join(work,'result.json'),JSON.stringify(result,null,2));await fs.writeFile(join(work,'host.log'),output);console.log(JSON.stringify(result));}
