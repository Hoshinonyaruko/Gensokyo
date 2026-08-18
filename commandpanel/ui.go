package commandpanel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getUI 提供 Command Panel 本地管理页面.
//
// Phase 7B:
//   - 查看 QQ 远端状态
//   - 从普通配置列表选择配置
//   - 结构化编辑 Menu
//   - 结构化编辑 target_type=all 的 Panels
//   - specific Panels 只读保留
//   - 保存到本地 ConfigFile
//   - 使用普通 Apply 应用配置
//
// Full Restore 不在本页面暴露。
func (h *HTTPHandler) getUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header(
		"Content-Security-Policy",
		"default-src 'none'; "+
			"connect-src 'self'; "+
			"style-src 'unsafe-inline'; "+
			"script-src 'unsafe-inline'; "+
			"img-src 'self' data:; "+
			"base-uri 'none'; "+
			"frame-ancestors 'none'; "+
			"form-action 'none'",
	)

	c.Data(
		http.StatusOK,
		"text/html; charset=utf-8",
		[]byte(commandPanelUIHTML),
	)
}

const commandPanelUIHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Gensokyo Command Panel</title>
<style>
:root{color-scheme:dark;--bg:#0b0d10;--surface:#12151a;--surface2:#171b21;--surface3:#0e1115;--border:#282e37;--text:#e8edf3;--muted:#89929e;--green:#6fd6a0;--yellow:#e5c77a;--red:#ef8686;--blue:#79b7ff;--radius:10px}
*{box-sizing:border-box}
html,body{margin:0;min-height:100%;background:var(--bg);color:var(--text);font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI","Microsoft YaHei",sans-serif}
body{padding:0 18px 60px}
main{width:min(1420px,100%);margin:0 auto}
header{min-height:76px;display:flex;align-items:center;justify-content:space-between;gap:20px;border-bottom:1px solid var(--border)}
.brand{display:flex;align-items:baseline;gap:12px}.brand h1{margin:0;font-size:20px;font-weight:680}.brand span{color:var(--muted);font-size:13px}
.toolbar,.row,.actions{display:flex;align-items:center;gap:9px;flex-wrap:wrap}
button,select,input,textarea{font:inherit}
button{appearance:none;border:1px solid var(--border);border-radius:7px;background:var(--surface2);color:var(--text);padding:7px 12px;font-size:13px;cursor:pointer}
button:hover:not(:disabled){border-color:#4a5565}button:disabled{opacity:.45;cursor:default}
button.primary{border-color:#315f89;background:#15304a;color:#d8ecff}button.success{border-color:#32634b;background:#173226;color:#cff3de}button.danger{border-color:#653535;background:#331b1d;color:#f3b3b3}button.small{padding:4px 8px;font-size:11px}
.status{display:inline-flex;align-items:center;gap:7px;color:var(--muted);font-size:13px}.status-dot{width:8px;height:8px;border-radius:50%;background:var(--muted)}
.status.ok .status-dot{background:var(--green)}.status.error .status-dot{background:var(--red)}.status.busy .status-dot{background:var(--yellow)}
.summary{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-top:18px}
.metric,.card{border:1px solid var(--border);border-radius:var(--radius);background:var(--surface)}.metric{padding:16px}
.metric-label{color:var(--muted);font-size:12px}.metric-value{margin-top:7px;font-size:22px;line-height:1.2;font-weight:680;overflow-wrap:anywhere}
.section{margin-top:26px}.section-header{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:10px}.section-header h2{margin:0;font-size:15px;font-weight:680}.section-note{color:var(--muted);font-size:12px}
.message{display:none;margin-top:15px;padding:11px 13px;border:1px solid var(--border);border-radius:8px;font-size:13px;white-space:pre-wrap}.message.visible{display:block}
.message.error{border-color:#603535;background:#281719;color:#f0abab}.message.success{border-color:#315c46;background:#15271e;color:#b8e7cb}.message.info{border-color:#34526d;background:#142332;color:#b8d8f3}
.config-bar{padding:14px;display:grid;grid-template-columns:minmax(240px,1fr) auto;gap:12px;align-items:end}
.control label,.field label{display:block;margin-bottom:5px;color:var(--muted);font-size:11px}
.control select,.control input,.field input,.field textarea{width:100%;border:1px solid var(--border);border-radius:6px;background:#0d1014;color:var(--text);padding:8px 9px;outline:none}
.control select:focus,.control input:focus,.field input:focus,.field textarea:focus{border-color:#49657c}
.field textarea{min-height:64px;resize:vertical;line-height:1.45}.field input[readonly],.field textarea[readonly]{color:#aeb7c2;background:#0b0e12}
.editor-shell{display:none}.editor-shell.visible{display:block}.editor-head{padding:14px}.editor-title{display:flex;align-items:center;gap:9px;flex-wrap:wrap}.editor-title strong{font-size:15px}
.dirty{display:none;border:1px solid #715f32;border-radius:999px;color:#e4c97c;padding:2px 8px;font-size:11px}.dirty.visible{display:inline-flex}
.meta-grid,.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px 12px}.meta-grid{margin-top:14px}.field.full{grid-column:1/-1}
.block{margin-top:12px;border:1px solid var(--border);border-radius:8px;background:var(--surface2)}.block-head{min-height:44px;padding:9px 11px;display:flex;align-items:center;justify-content:space-between;gap:10px;border-bottom:1px solid var(--border)}.block-head strong{font-size:13px}.block-body{padding:11px}
.badges{display:flex;align-items:center;gap:6px;flex-wrap:wrap}.badge{display:inline-flex;align-items:center;border:1px solid var(--border);border-radius:999px;padding:2px 8px;color:var(--muted);font-size:11px}.badge.warn{color:#e6cf89;border-color:#665932}.badge.readonly{color:#b8c0c9;border-color:#46505d}
.items{display:grid;gap:10px}.item-card{border:1px solid var(--border);border-radius:7px;background:var(--surface3)}.item-head{padding:8px 10px;display:flex;align-items:center;justify-content:space-between;gap:8px;border-bottom:1px solid var(--border)}.item-name{font-size:12px;font-weight:650;overflow-wrap:anywhere}.item-body{padding:10px}
.submenu{margin-top:10px;padding-left:12px;border-left:2px solid var(--border)}.submenu-label{margin-bottom:8px;color:var(--muted);font-size:11px}
.panel-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.panel-card{padding:0}.panel-card.full-width{grid-column:1/-1}
.remote-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px;padding:12px}.remote-card{border:1px solid var(--border);border-radius:7px;background:var(--surface3);padding:10px}.remote-card h3{margin:0 0 8px;font-size:13px}.remote-line{color:var(--muted);font-size:12px;line-height:1.6;overflow-wrap:anywhere}
.empty{padding:24px;text-align:center;color:var(--muted);font-size:13px}
details{margin-top:10px}summary{width:fit-content;color:var(--blue);font-size:12px;cursor:pointer;user-select:none}
pre{margin:9px 0 0;max-height:520px;overflow:auto;border:1px solid var(--border);border-radius:7px;background:#090b0e;padding:11px;color:#cbd3dd;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:12px;line-height:1.5;white-space:pre-wrap;overflow-wrap:anywhere}
.file-table{width:100%;border-collapse:collapse}.file-table th,.file-table td{padding:10px 12px;text-align:left;border-bottom:1px solid var(--border);font-size:12px}.file-table th{color:var(--muted);font-size:11px;font-weight:550}.file-table tr:last-child td{border-bottom:0}.mono{font-family:ui-monospace,SFMono-Regular,Consolas,monospace}
.warning-box{margin-top:12px;padding:10px 12px;border:1px solid #5e5030;border-radius:7px;background:#211d13;color:#dcc989;font-size:12px;line-height:1.55}
.apply-result{margin-top:12px;display:none}.apply-result.visible{display:block}
@media(max-width:900px){.summary,.panel-grid,.remote-grid{grid-template-columns:repeat(2,minmax(0,1fr))}header{align-items:flex-start;flex-direction:column;padding:17px 0}}
@media(max-width:700px){body{padding-left:10px;padding-right:10px}.summary,.panel-grid,.remote-grid,.meta-grid,.form-grid,.config-bar{grid-template-columns:1fr}.panel-card.full-width{grid-column:auto}.config-bar{align-items:stretch}}
</style>
</head>
<body>
<main>
<header>
<div class="brand"><h1>Command Panel</h1><span>Gensokyo · Phase 7B</span></div>
<div class="toolbar"><div id="status" class="status"><span class="status-dot"></span><span id="statusText">等待连接</span></div><button id="refreshButton" type="button">刷新远端</button></div>
</header>

<div id="messageBox" class="message"></div>

<section class="summary">
<div class="metric"><div class="metric-label">AppID</div><div id="appID" class="metric-value">—</div></div>
<div class="metric"><div class="metric-label">远端 Panel</div><div id="panelCount" class="metric-value">—</div></div>
<div class="metric"><div class="metric-label">普通配置</div><div id="configCount" class="metric-value">—</div></div>
<div class="metric"><div class="metric-label">自动备份</div><div id="backupCount" class="metric-value">—</div></div>
</section>

<section class="section">
<div class="section-header"><h2>配置驾驶舱</h2><span class="section-note">编辑本地 ConfigFile，Apply 前必须先保存</span></div>
<div class="card config-bar">
<div class="control"><label for="configSelect">普通配置</label><select id="configSelect"><option value="">请选择配置</option></select></div>
<div class="actions"><button id="loadConfigButton" type="button">加载配置</button><button id="saveConfigButton" class="success" type="button" disabled>保存本地</button><button id="applyConfigButton" class="primary" type="button" disabled>Apply 到 QQ</button></div>
</div>
<div class="card config-bar" style="margin-top:10px">
<div class="control"><label for="snapshotName">从当前 QQ 远端创建普通配置</label><input id="snapshotName" type="text" placeholder="例如：当前线上 / 七夕活动前"></div>
<div class="actions"><button id="snapshotButton" type="button">保存远端快照为配置</button></div>
</div>
</section>

<section id="editorShell" class="section editor-shell">
<div class="section-header"><h2>配置编辑器</h2><span class="section-note">specific Panels 仅查看、不参与普通编辑</span></div>
<div class="card editor-head">
<div class="editor-title"><strong id="editorFileName">—</strong><span id="dirtyBadge" class="dirty">未保存</span></div>
<div class="meta-grid">
<div class="field"><label>Format</label><input id="metaFormat" readonly></div>
<div class="field"><label>Schema Version</label><input id="metaSchema" readonly></div>
<div class="field"><label>Source AppID</label><input id="metaSourceAppID" readonly></div>
<div class="field"><label>Saved At</label><input id="metaSavedAt" readonly></div>
</div>
<div class="warning-box">普通 Apply 只管理全局 Menu 与 target_type=all 的 Panels。specific Panels 和 targets 会完整保留在配置文件中，但本编辑器不会修改它们。Apply 前会由后端再次校验并自动备份当前 QQ 远端状态。</div>
</div>

<section class="section"><div class="section-header"><h2>Menu</h2><span id="menuCountNote" class="section-note">—</span></div><div id="menuEditor" class="card"><div class="empty">尚未加载</div></div></section>
<section class="section"><div class="section-header"><h2>Global Panels</h2><span id="globalPanelCountNote" class="section-note">—</span></div><div id="globalPanelEditor" class="panel-grid"></div></section>
<section class="section"><div class="section-header"><h2>Specific Panels</h2><span id="specificPanelCountNote" class="section-note">只读保留</span></div><div id="specificPanelViewer" class="panel-grid"></div></section>
<section class="section"><div class="section-header"><h2>完整配置 JSON</h2><span class="section-note">随表单修改实时更新</span></div><div class="card" style="padding:12px"><pre id="configJSON"></pre></div></section>
<div id="applyResult" class="message info apply-result"></div>
</section>

<section class="section">
<div class="section-header"><h2>当前 QQ 远端概览</h2><span class="section-note">只读</span></div>
<div class="card"><div id="remoteOverview" class="remote-grid"><div class="empty">尚未读取</div></div><details style="margin:0 12px 12px"><summary>查看远端完整 JSON</summary><pre id="remoteJSON">尚未读取</pre></details></div>
</section>

<section class="section">
<div class="section-header"><h2>自动备份</h2><span class="section-note">只读 · command_panels/autobackup</span></div>
<div class="card"><table class="file-table"><thead><tr><th>文件</th><th>大小</th><th>修改时间</th></tr></thead><tbody id="backupRows"></tbody></table><div id="backupEmpty" class="empty">尚未读取</div></div>
</section>
</main>

<script>
(function(){
'use strict';

var state={remote:null,configs:[],backups:[],currentName:'',currentConfig:null,dirty:false,busy:false};
function $(id){return document.getElementById(id)}
var status=$('status'),statusText=$('statusText'),messageBox=$('messageBox');
var refreshButton=$('refreshButton'),loadConfigButton=$('loadConfigButton'),saveConfigButton=$('saveConfigButton'),applyConfigButton=$('applyConfigButton'),snapshotButton=$('snapshotButton');
var configSelect=$('configSelect'),snapshotName=$('snapshotName');
var appID=$('appID'),panelCount=$('panelCount'),configCount=$('configCount'),backupCount=$('backupCount');
var editorShell=$('editorShell'),editorFileName=$('editorFileName'),dirtyBadge=$('dirtyBadge');
var metaFormat=$('metaFormat'),metaSchema=$('metaSchema'),metaSourceAppID=$('metaSourceAppID'),metaSavedAt=$('metaSavedAt');
var menuEditor=$('menuEditor'),menuCountNote=$('menuCountNote'),globalPanelEditor=$('globalPanelEditor'),globalPanelCountNote=$('globalPanelCountNote'),specificPanelViewer=$('specificPanelViewer'),specificPanelCountNote=$('specificPanelCountNote'),configJSON=$('configJSON');
var remoteOverview=$('remoteOverview'),remoteJSON=$('remoteJSON'),backupRows=$('backupRows'),backupEmpty=$('backupEmpty'),applyResult=$('applyResult');

function obj(v){return v&&typeof v==='object'?v:{}}
function arr(v){return Array.isArray(v)?v:[]}
function clone(v){return JSON.parse(JSON.stringify(v))}
function pretty(v){return JSON.stringify(v,null,2)}
function own(o,k){return Object.prototype.hasOwnProperty.call(obj(o),k)}
function keyOf(o,keys,fallback){o=obj(o);for(var i=0;i<keys.length;i++){if(own(o,keys[i]))return keys[i]}return fallback||keys[0]}
function read(o,keys,fallback){o=obj(o);for(var i=0;i<keys.length;i++){var k=keys[i];if(own(o,k)&&o[k]!==undefined&&o[k]!==null)return o[k]}return fallback}
function write(o,keys,v,fallback){o=obj(o);o[keyOf(o,keys,fallback)]=v;markDirty()}
function setStatus(kind,text){status.className='status'+(kind?' '+kind:'');statusText.textContent=text}
function show(kind,text){messageBox.className='message visible '+kind;messageBox.textContent=text}
function clear(){messageBox.className='message';messageBox.textContent=''}
function setBusy(v,text){state.busy=!!v;refreshButton.disabled=state.busy;loadConfigButton.disabled=state.busy;snapshotButton.disabled=state.busy;saveConfigButton.disabled=state.busy||!state.currentConfig;applyConfigButton.disabled=state.busy||!state.currentConfig;configSelect.disabled=state.busy;if(state.busy)setStatus('busy',text||'处理中')}
async function api(method,path,payload){
 var options={method:method,cache:'no-store',headers:{'Accept':'application/json'}};
 if(payload!==undefined){options.headers['Content-Type']='application/json';options.body=JSON.stringify(payload)}
 var response=await fetch('/cmd/api'+path,options),body;
 try{body=await response.json()}catch(_){throw new Error('服务器返回无法解析的响应，HTTP '+response.status)}
 if(!response.ok||!body||body.ok!==true){var m=body&&body.error?body.error:'HTTP '+response.status;if(body&&body.data)m+='\n\n部分执行结果：\n'+pretty(body.data);throw new Error(m)}
 return body.data
}
function bytes(v){var n=Number(v||0);if(!Number.isFinite(n)||n<=0)return'0 B';if(n<1024)return n+' B';if(n<1048576)return(n/1024).toFixed(1)+' KiB';return(n/1048576).toFixed(2)+' MiB'}
function time(v){if(!v)return'—';var d=new Date(v);return Number.isNaN(d.getTime())?String(v):d.toLocaleString()}
function text(parent,tag,cls,value){var e=document.createElement(tag);if(cls)e.className=cls;e.textContent=value==null?'':String(value);parent.appendChild(e);return e}
function badge(value,cls){var e=document.createElement('span');e.className='badge'+(cls?' '+cls:'');e.textContent=value;return e}
function empty(value){var e=document.createElement('div');e.className='empty';e.textContent=value;return e}
function details(title,value){var d=document.createElement('details');text(d,'summary','',title);var p=document.createElement('pre');p.textContent=pretty(value);d.appendChild(p);return d}
function field(label,value,onInput,options){
 options=options||{};var w=document.createElement('div');w.className='field'+(options.full?' full':'');text(w,'label','',label);
 var c=options.multiline?document.createElement('textarea'):document.createElement('input');if(!options.multiline)c.type='text';else c.rows=options.rows||2;
 c.value=value==null?'':String(value);if(options.placeholder)c.placeholder=options.placeholder;if(options.readonly)c.readOnly=true;
 if(!options.readonly&&onInput)c.addEventListener('input',function(){onInput(c.value)});
 w.appendChild(c);return w
}
function updateRaw(){configJSON.textContent=state.currentConfig?pretty(state.currentConfig):''}
function markDirty(){if(!state.currentConfig)return;state.dirty=true;dirtyBadge.classList.add('visible');updateRaw()}
function markClean(){state.dirty=false;dirtyBadge.classList.remove('visible');updateRaw()}
function confirmLose(){return !state.dirty||window.confirm('当前配置有未保存修改，继续会丢失这些修改。确定继续吗？')}

function itemActions(items,index,rerender,minimum,maximum){
 var a=document.createElement('div');a.className='actions';
 function b(label,fn,danger){var e=document.createElement('button');e.type='button';e.className='small'+(danger?' danger':'');e.textContent=label;e.addEventListener('click',fn);a.appendChild(e);return e}
 var up=b('上移',function(){if(index<=0)return;var t=items[index-1];items[index-1]=items[index];items[index]=t;markDirty();rerender()});up.disabled=index<=0;
 var down=b('下移',function(){if(index>=items.length-1)return;var t=items[index+1];items[index+1]=items[index];items[index]=t;markDirty();rerender()});down.disabled=index>=items.length-1;
 var copyButton=b('复制',function(){if(maximum&&items.length>=maximum){show('error','最多允许 '+maximum+' 个项目。');return}items.splice(index+1,0,clone(items[index]));markDirty();rerender()});if(maximum&&items.length>=maximum)copyButton.disabled=true;
 var remove=b('删除',function(){if(minimum&&items.length<=minimum){show('error','至少需要保留 '+minimum+' 个项目。');return}if(!window.confirm('确定删除这一项吗？'))return;items.splice(index,1);markDirty();rerender()},true);
 if(minimum&&items.length<=minimum)remove.disabled=true;
 return a
}

function renderMenuSub(item,index,items){
 item=obj(item);var card=document.createElement('div');card.className='item-card';var head=document.createElement('div');head.className='item-head';
 text(head,'div','item-name','子项 '+(index+1)+' · '+(read(item,['name'],'未命名')||'未命名'));head.appendChild(itemActions(items,index,renderMenu,0,5));card.appendChild(head);
 var body=document.createElement('div');body.className='item-body';var grid=document.createElement('div');grid.className='form-grid';
 grid.appendChild(field('名称',read(item,['name'],''),function(v){write(item,['name'],v)}));
 grid.appendChild(field('类型（只读）',read(item,['type'],''),null,{readonly:true}));
 var sk=keyOf(item,['send_message','sendMessage'],'send_message');if(own(item,sk))grid.appendChild(field('发送消息',item[sk],function(v){item[sk]=v;markDirty()},{full:true,multiline:true}));
 var link=read(item,['link'],undefined);if(link!==undefined)grid.appendChild(field('链接',link,function(v){write(item,['link'],v)},{full:true}));
 body.appendChild(grid);body.appendChild(details('查看此子项 JSON',item));card.appendChild(body);return card
}

function renderMenuItem(item,index,items){
 item=obj(item);var card=document.createElement('div');card.className='item-card';var head=document.createElement('div');head.className='item-head';
 text(head,'div','item-name','#'+(index+1)+' · '+(read(item,['name'],'未命名')||'未命名'));head.appendChild(itemActions(items,index,renderMenu,0,10));card.appendChild(head);
 var body=document.createElement('div');body.className='item-body';var grid=document.createElement('div');grid.className='form-grid';
 grid.appendChild(field('名称',read(item,['name'],''),function(v){write(item,['name'],v)}));
 grid.appendChild(field('类型（只读）',read(item,['type'],''),null,{readonly:true}));
 var sk=keyOf(item,['send_message','sendMessage'],'send_message');if(own(item,sk))grid.appendChild(field('发送消息',item[sk],function(v){item[sk]=v;markDirty()},{full:true,multiline:true}));
 var link=read(item,['link'],undefined);if(link!==undefined)grid.appendChild(field('链接',link,function(v){write(item,['link'],v)},{full:true}));
 var sw=read(item,['switch'],undefined);if(sw&&typeof sw==='object')grid.appendChild(field('Switch ID',read(sw,['switch_id','switchId'],''),function(v){write(sw,['switch_id','switchId'],v,'switch_id')},{full:true}));
 body.appendChild(grid);
 var subs=arr(read(item,['sub_menu_items','subMenuItems'],[]));if(subs.length){var sub=document.createElement('div');sub.className='submenu';text(sub,'div','submenu-label','子菜单 · '+subs.length+' 项');var list=document.createElement('div');list.className='items';subs.forEach(function(s,i){list.appendChild(renderMenuSub(s,i,subs))});sub.appendChild(list);body.appendChild(sub)}
 body.appendChild(details('查看此 Menu Item JSON',item));card.appendChild(body);return card
}

function renderMenu(){
 menuEditor.textContent='';if(!state.currentConfig){menuEditor.appendChild(empty('尚未加载'));return}
 var menu=state.currentConfig.menu;if(!menu){menuCountNote.textContent='当前为 null';menuEditor.appendChild(empty('当前配置没有全局 Menu'));return}
 var items=arr(read(menu,['items'],[]));menuCountNote.textContent=items.length+' / 10 项';var list=document.createElement('div');list.className='items';list.style.padding='11px';
 if(!items.length)list.appendChild(empty('Menu 存在，但当前没有项目'));else items.forEach(function(item,index){list.appendChild(renderMenuItem(item,index,items))});
 menuEditor.appendChild(list);var raw=document.createElement('div');raw.style.padding='0 11px 11px';raw.appendChild(details('查看 Menu JSON',menu));menuEditor.appendChild(raw);updateRaw()
}

function renderPanelItem(item,index,items,rerender,editable){
 item=obj(item);var card=document.createElement('div');card.className='item-card';var head=document.createElement('div');head.className='item-head';
 text(head,'div','item-name','#'+(index+1)+' · '+(read(item,['name'],'未命名')||'未命名'));
 if(editable)head.appendChild(itemActions(items,index,rerender,1,20));else head.appendChild(badge('只读','readonly'));card.appendChild(head);
 var body=document.createElement('div');body.className='item-body';var grid=document.createElement('div');grid.className='form-grid';
 grid.appendChild(field('名称',read(item,['name'],''),editable?function(v){write(item,['name'],v)}:null,{readonly:!editable}));
 grid.appendChild(field('类型（只读）',read(item,['type'],''),null,{readonly:true}));
 grid.appendChild(field('描述',read(item,['desc'],''),editable?function(v){write(item,['desc'],v)}:null,{full:true,multiline:true,readonly:!editable}));
 var link=read(item,['link'],undefined);if(link!==undefined)grid.appendChild(field('链接',link,editable?function(v){write(item,['link'],v)}:null,{full:true,readonly:!editable}));
 body.appendChild(grid);body.appendChild(details('查看此 Panel Item JSON',item));card.appendChild(body);return card
}

function isGlobal(record){return String(read(record,['target_type','targetType'],''))==='all'}

function renderGlobal(record,index){
 record=obj(record);var panel=obj(read(record,['panel'],{})),items=arr(read(panel,['items'],[]));var card=document.createElement('div');card.className='card panel-card full-width';
 var head=document.createElement('div');head.className='block-head';text(head,'strong','',read(record,['panel_id','panelId'],'Global Panel '+(index+1)));
 var badges=document.createElement('div');badges.className='badges';badges.appendChild(badge('scope: '+read(record,['scope'],'unknown')));badges.appendChild(badge('target: '+read(record,['target_type','targetType'],'unknown')));badges.appendChild(badge('items: '+items.length));head.appendChild(badges);card.appendChild(head);
 var body=document.createElement('div');body.className='block-body';var grid=document.createElement('div');grid.className='form-grid';grid.appendChild(field('Remark',read(panel,['remark'],''),function(v){write(panel,['remark'],v)},{full:true,multiline:true}));body.appendChild(grid);
 var list=document.createElement('div');list.className='items';list.style.marginTop='10px';items.forEach(function(item,i){list.appendChild(renderPanelItem(item,i,items,renderPanels,true))});if(!items.length)list.appendChild(empty('当前 Panel 没有 items；Apply 时后端会进行合法性校验。'));body.appendChild(list);body.appendChild(details('查看完整 Global Panel JSON',record));card.appendChild(body);return card
}

function renderSpecific(record,index){
 record=obj(record);var panel=obj(read(record,['panel'],{})),items=arr(read(panel,['items'],[])),users=arr(read(record,['user_openids','userOpenIDs'],[])),groups=arr(read(record,['group_openids','groupOpenIDs'],[]));
 var card=document.createElement('div');card.className='card panel-card';var head=document.createElement('div');head.className='block-head';text(head,'strong','',read(record,['panel_id','panelId'],'Specific Panel '+(index+1)));
 var badges=document.createElement('div');badges.className='badges';badges.appendChild(badge('scope: '+read(record,['scope'],'unknown')));badges.appendChild(badge('targets: '+(users.length+groups.length)));badges.appendChild(badge('只读','readonly'));head.appendChild(badges);card.appendChild(head);
 var body=document.createElement('div');body.className='block-body';var grid=document.createElement('div');grid.className='form-grid';
 grid.appendChild(field('Remark',read(panel,['remark'],''),null,{full:true,multiline:true,readonly:true}));grid.appendChild(field('Items',items.length,null,{readonly:true}));grid.appendChild(field('Targets',users.length+groups.length,null,{readonly:true}));body.appendChild(grid);body.appendChild(details('查看 Specific Panel / Targets JSON',record));card.appendChild(body);return card
}

function renderPanels(){
 globalPanelEditor.textContent='';specificPanelViewer.textContent='';if(!state.currentConfig){globalPanelEditor.appendChild(empty('尚未加载'));specificPanelViewer.appendChild(empty('尚未加载'));return}
 var panels=arr(state.currentConfig.panels),globals=panels.filter(isGlobal),specifics=panels.filter(function(r){return!isGlobal(r)});
 globalPanelCountNote.textContent=globals.length+' 个';specificPanelCountNote.textContent=specifics.length+' 个 · 只读保留';
 if(!globals.length)globalPanelEditor.appendChild(empty('当前配置没有 target_type=all 的 Panel'));else globals.forEach(function(r,i){globalPanelEditor.appendChild(renderGlobal(r,i))});
 if(!specifics.length)specificPanelViewer.appendChild(empty('当前配置没有 specific Panel'));else specifics.forEach(function(r,i){specificPanelViewer.appendChild(renderSpecific(r,i))});updateRaw()
}

function renderEditor(){
 if(!state.currentConfig){editorShell.classList.remove('visible');saveConfigButton.disabled=true;applyConfigButton.disabled=true;return}
 editorShell.classList.add('visible');editorFileName.textContent=state.currentName||'—';metaFormat.value=state.currentConfig.format||'';metaSchema.value=state.currentConfig.schema_version===undefined?'':String(state.currentConfig.schema_version);metaSourceAppID.value=state.currentConfig.source_appid||'';metaSavedAt.value=state.currentConfig.saved_at||'';
 saveConfigButton.disabled=state.busy;applyConfigButton.disabled=state.busy;renderMenu();renderPanels();markClean()
}

function renderRemote(remote){
 remoteOverview.textContent='';remote=obj(remote);var menu=obj(remote.menu),menuItems=arr(read(menu,['items'],[]));var mc=document.createElement('div');mc.className='remote-card';text(mc,'h3','','Menu · '+menuItems.length+' 项');
 if(!menuItems.length)text(mc,'div','remote-line','当前没有全局 Menu');else menuItems.forEach(function(item,i){text(mc,'div','remote-line','#'+(i+1)+' '+read(item,['name'],'未命名')+' · '+read(item,['type'],'unknown'))});remoteOverview.appendChild(mc);
 arr(remote.panels).forEach(function(record,index){record=obj(record);var panel=obj(read(record,['panel'],{})),items=arr(read(panel,['items'],[])),users=arr(read(record,['user_openids','userOpenIDs'],[])),groups=arr(read(record,['group_openids','groupOpenIDs'],[]));var c=document.createElement('div');c.className='remote-card';text(c,'h3','',read(record,['panel_id','panelId'],'Panel #'+(index+1)));text(c,'div','remote-line','scope='+read(record,['scope'],'unknown')+' · target='+read(record,['target_type','targetType'],'unknown'));text(c,'div','remote-line','items='+items.length+' · targets='+(users.length+groups.length));var remark=read(panel,['remark'],'');if(remark)text(c,'div','remote-line','remark: '+remark);items.slice(0,8).forEach(function(item,i){text(c,'div','remote-line','  '+(i+1)+'. '+read(item,['name'],'未命名'))});if(items.length>8)text(c,'div','remote-line','  … 还有 '+(items.length-8)+' 项');remoteOverview.appendChild(c)});
 remoteJSON.textContent=pretty(remote)
}

function renderSelect(){
 var selected=configSelect.value||state.currentName;configSelect.textContent='';var p=document.createElement('option');p.value='';p.textContent=state.configs.length?'请选择配置':'没有普通配置';configSelect.appendChild(p);
 state.configs.forEach(function(c){var o=document.createElement('option');o.value=c.name||'';o.textContent=(c.name||'—')+' · '+bytes(c.size);configSelect.appendChild(o)});
 if(selected&&state.configs.some(function(c){return c.name===selected}))configSelect.value=selected
}
function renderBackups(){
 backupRows.textContent='';if(!state.backups.length){backupEmpty.style.display='block';backupEmpty.textContent='没有自动备份';return}backupEmpty.style.display='none';
 state.backups.forEach(function(b){var r=document.createElement('tr');text(r,'td','mono',b.name||'—');text(r,'td','',bytes(b.size));text(r,'td','',time(b.modified_at));backupRows.appendChild(r)})
}
function renderDashboard(){
 var remote=obj(state.remote),panels=arr(remote.panels);appID.textContent=remote.app_id||'—';panelCount.textContent=String(panels.length);configCount.textContent=String(state.configs.length);backupCount.textContent=String(state.backups.length);renderSelect();renderBackups();renderRemote(remote)
}

async function refreshDashboard(options){
 options=options||{};if(!options.ignoreDirty&&!confirmLose())return;clear();setBusy(true,'正在读取');
 try{var result=await Promise.all([api('GET','/remote'),api('GET','/configs')]);state.remote=result[0]||{};var files=result[1]||{};state.configs=arr(files.configs);state.backups=arr(files.backups);renderDashboard();setStatus('ok','已连接')}
 catch(err){setStatus('error','读取失败');show('error',err.message||String(err))}finally{setBusy(false)}
}
async function refreshFiles(){
 var files=await api('GET','/configs');files=files||{};state.configs=arr(files.configs);state.backups=arr(files.backups);renderSelect();renderBackups();configCount.textContent=String(state.configs.length);backupCount.textContent=String(state.backups.length)
}
async function loadConfig(name,force){
 name=(name||configSelect.value||'').trim();if(!name){show('error','请先选择一个普通配置。');return}if(!force&&!confirmLose()){if(state.currentName)configSelect.value=state.currentName;return}
 clear();applyResult.classList.remove('visible');setBusy(true,'正在加载配置');
 try{var cfg=await api('GET','/configs/'+encodeURIComponent(name));state.currentName=name;state.currentConfig=clone(cfg);configSelect.value=name;renderEditor();show('success','已加载本地配置：'+name);setStatus('ok','配置已加载')}
 catch(err){setStatus('error','加载失败');show('error',err.message||String(err))}finally{setBusy(false)}
}
async function saveConfig(){
 if(!state.currentConfig||!state.currentName){show('error','当前没有已加载的普通配置。');return false}clear();setBusy(true,'正在保存本地配置');
 try{var saved=await api('PUT','/configs/'+encodeURIComponent(state.currentName),state.currentConfig);state.currentConfig=clone(saved);await refreshFiles();renderEditor();show('success','本地配置已保存：'+state.currentName);setStatus('ok','已保存');return true}
 catch(err){setStatus('error','保存失败');show('error',err.message||String(err));return false}finally{setBusy(false)}
}
async function applyConfig(){
 if(!state.currentConfig||!state.currentName){show('error','当前没有已加载的普通配置。');return}
 if(state.dirty){show('error','当前配置还有未保存修改。请先点击“保存本地”，再执行 Apply。');return}
 var app=state.remote&&state.remote.app_id?state.remote.app_id:'当前 Bot';
 if(!window.confirm('即将把配置“'+state.currentName+'”普通 Apply 到 AppID '+app+'。\n\n后端会先自动备份远端状态；普通 Apply 只修改 Menu 与 target_type=all 的 Panels，specific Panels 不会被修改或删除。\n\n确定继续吗？'))return;
 clear();applyResult.classList.remove('visible');setBusy(true,'正在 Apply');
 try{var result=await api('POST','/apply',{name:state.currentName});applyResult.textContent='Apply 完成\n\n'+pretty(result);applyResult.className='message info apply-result visible';show('success','Apply 已完成。自动备份与执行结果见下方。');setStatus('ok','Apply 完成');await refreshDashboard({ignoreDirty:true});configSelect.value=state.currentName}
 catch(err){setStatus('error','Apply 失败');show('error',err.message||String(err))}finally{setBusy(false)}
}
async function snapshot(){
 var name=snapshotName.value.trim();if(!name){show('error','请输入快照配置名称。');return}if(!confirmLose())return;clear();setBusy(true,'正在保存远端快照');
 try{await api('POST','/configs/snapshot',{name:name});await refreshFiles();var candidates=[name,name.toLowerCase().endsWith('.json')?name:name+'.json'],actual='';
 for(var i=0;i<state.configs.length;i++){if(candidates.indexOf(state.configs[i].name)>=0){actual=state.configs[i].name;break}}
 snapshotName.value='';show('success','当前 QQ 远端状态已保存为普通配置。');if(actual){configSelect.value=actual;await loadConfig(actual,true)}else setStatus('ok','快照已保存')}
 catch(err){setStatus('error','快照保存失败');show('error',err.message||String(err))}finally{setBusy(false)}
}

refreshButton.addEventListener('click',function(){refreshDashboard()});
loadConfigButton.addEventListener('click',function(){loadConfig()});
saveConfigButton.addEventListener('click',saveConfig);
applyConfigButton.addEventListener('click',applyConfig);
snapshotButton.addEventListener('click',snapshot);
configSelect.addEventListener('change',function(){if(!state.currentName||configSelect.value===state.currentName)return;if(state.dirty){var wanted=configSelect.value;if(!confirmLose()){configSelect.value=state.currentName;return}configSelect.value=wanted}});
window.addEventListener('beforeunload',function(event){if(!state.dirty)return;event.preventDefault();event.returnValue=''});
refreshDashboard({ignoreDirty:true});
})();
</script>
</body>
</html>`
