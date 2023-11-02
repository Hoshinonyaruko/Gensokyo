"use strict";(globalThis["webpackChunkfrontend"]=globalThis["webpackChunkfrontend"]||[]).push([[853],{73853:(e,t,n)=>{n.r(t),n.d(t,{CompletionAdapter:()=>ye,DefinitionAdapter:()=>Oe,DiagnosticsAdapter:()=>we,DocumentColorAdapter:()=>$e,DocumentFormattingEditProvider:()=>qe,DocumentHighlightAdapter:()=>Le,DocumentLinkAdapter:()=>ze,DocumentRangeFormattingEditProvider:()=>Xe,DocumentSymbolAdapter:()=>He,FoldingRangeAdapter:()=>Qe,HoverAdapter:()=>De,ReferenceAdapter:()=>We,RenameAdapter:()=>Ue,SelectionRangeAdapter:()=>Ye,WorkerManager:()=>N,fromPosition:()=>Ee,fromRange:()=>Ae,setupMode:()=>wt,toRange:()=>xe,toTextEdit:()=>Te});var r=n(73512),i=Object.defineProperty,o=Object.getOwnPropertyDescriptor,a=Object.getOwnPropertyNames,s=Object.prototype.hasOwnProperty,c=(e,t,n,r)=>{if(t&&"object"===typeof t||"function"===typeof t)for(let c of a(t))s.call(e,c)||c===n||i(e,c,{get:()=>t[c],enumerable:!(r=o(t,c))||r.enumerable});return e},u=(e,t,n)=>(c(e,t,"default"),n&&c(n,t,"default")),d={};
/*!-----------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Version: 0.44.0(3e047efd345ff102c8c61b5398fb30845aaac166)
 * Released under the MIT license
 * https://github.com/microsoft/monaco-editor/blob/main/LICENSE.txt
 *-----------------------------------------------------------------------------*/u(d,r);var g,l,h,f,p,m,v,b,k,w,_,C,y,E,A,x,I,S,T,R,D,P,M,j,L,F,O=12e4,N=class{_defaults;_idleCheckInterval;_lastUsedTime;_configChangeListener;_worker;_client;constructor(e){this._defaults=e,this._worker=null,this._client=null,this._idleCheckInterval=window.setInterval((()=>this._checkIfIdle()),3e4),this._lastUsedTime=0,this._configChangeListener=this._defaults.onDidChange((()=>this._stopWorker()))}_stopWorker(){this._worker&&(this._worker.dispose(),this._worker=null),this._client=null}dispose(){clearInterval(this._idleCheckInterval),this._configChangeListener.dispose(),this._stopWorker()}_checkIfIdle(){if(!this._worker)return;let e=Date.now()-this._lastUsedTime;e>O&&this._stopWorker()}_getClient(){return this._lastUsedTime=Date.now(),this._client||(this._worker=d.editor.createWebWorker({moduleId:"vs/language/json/jsonWorker",label:this._defaults.languageId,createData:{languageSettings:this._defaults.diagnosticsOptions,languageId:this._defaults.languageId,enableSchemaRequest:this._defaults.diagnosticsOptions.enableSchemaRequest}}),this._client=this._worker.getProxy()),this._client}getLanguageServiceWorker(...e){let t;return this._getClient().then((e=>{t=e})).then((t=>{if(this._worker)return this._worker.withSyncedResources(e)})).then((e=>t))}};(function(e){e.MIN_VALUE=-2147483648,e.MAX_VALUE=2147483647})(g||(g={})),function(e){e.MIN_VALUE=0,e.MAX_VALUE=2147483647}(l||(l={})),function(e){function t(e,t){return e===Number.MAX_VALUE&&(e=l.MAX_VALUE),t===Number.MAX_VALUE&&(t=l.MAX_VALUE),{line:e,character:t}}function n(e){var t=e;return be.objectLiteral(t)&&be.uinteger(t.line)&&be.uinteger(t.character)}e.create=t,e.is=n}(h||(h={})),function(e){function t(e,t,n,r){if(be.uinteger(e)&&be.uinteger(t)&&be.uinteger(n)&&be.uinteger(r))return{start:h.create(e,t),end:h.create(n,r)};if(h.is(e)&&h.is(t))return{start:e,end:t};throw new Error("Range#create called with invalid arguments["+e+", "+t+", "+n+", "+r+"]")}function n(e){var t=e;return be.objectLiteral(t)&&h.is(t.start)&&h.is(t.end)}e.create=t,e.is=n}(f||(f={})),function(e){function t(e,t){return{uri:e,range:t}}function n(e){var t=e;return be.defined(t)&&f.is(t.range)&&(be.string(t.uri)||be.undefined(t.uri))}e.create=t,e.is=n}(p||(p={})),function(e){function t(e,t,n,r){return{targetUri:e,targetRange:t,targetSelectionRange:n,originSelectionRange:r}}function n(e){var t=e;return be.defined(t)&&f.is(t.targetRange)&&be.string(t.targetUri)&&(f.is(t.targetSelectionRange)||be.undefined(t.targetSelectionRange))&&(f.is(t.originSelectionRange)||be.undefined(t.originSelectionRange))}e.create=t,e.is=n}(m||(m={})),function(e){function t(e,t,n,r){return{red:e,green:t,blue:n,alpha:r}}function n(e){var t=e;return be.numberRange(t.red,0,1)&&be.numberRange(t.green,0,1)&&be.numberRange(t.blue,0,1)&&be.numberRange(t.alpha,0,1)}e.create=t,e.is=n}(v||(v={})),function(e){function t(e,t){return{range:e,color:t}}function n(e){var t=e;return f.is(t.range)&&v.is(t.color)}e.create=t,e.is=n}(b||(b={})),function(e){function t(e,t,n){return{label:e,textEdit:t,additionalTextEdits:n}}function n(e){var t=e;return be.string(t.label)&&(be.undefined(t.textEdit)||S.is(t))&&(be.undefined(t.additionalTextEdits)||be.typedArray(t.additionalTextEdits,S.is))}e.create=t,e.is=n}(k||(k={})),function(e){e["Comment"]="comment",e["Imports"]="imports",e["Region"]="region"}(w||(w={})),function(e){function t(e,t,n,r,i){var o={startLine:e,endLine:t};return be.defined(n)&&(o.startCharacter=n),be.defined(r)&&(o.endCharacter=r),be.defined(i)&&(o.kind=i),o}function n(e){var t=e;return be.uinteger(t.startLine)&&be.uinteger(t.startLine)&&(be.undefined(t.startCharacter)||be.uinteger(t.startCharacter))&&(be.undefined(t.endCharacter)||be.uinteger(t.endCharacter))&&(be.undefined(t.kind)||be.string(t.kind))}e.create=t,e.is=n}(_||(_={})),function(e){function t(e,t){return{location:e,message:t}}function n(e){var t=e;return be.defined(t)&&p.is(t.location)&&be.string(t.message)}e.create=t,e.is=n}(C||(C={})),function(e){e.Error=1,e.Warning=2,e.Information=3,e.Hint=4}(y||(y={})),function(e){e.Unnecessary=1,e.Deprecated=2}(E||(E={})),function(e){function t(e){var t=e;return void 0!==t&&null!==t&&be.string(t.href)}e.is=t}(A||(A={})),function(e){function t(e,t,n,r,i,o){var a={range:e,message:t};return be.defined(n)&&(a.severity=n),be.defined(r)&&(a.code=r),be.defined(i)&&(a.source=i),be.defined(o)&&(a.relatedInformation=o),a}function n(e){var t,n=e;return be.defined(n)&&f.is(n.range)&&be.string(n.message)&&(be.number(n.severity)||be.undefined(n.severity))&&(be.integer(n.code)||be.string(n.code)||be.undefined(n.code))&&(be.undefined(n.codeDescription)||be.string(null===(t=n.codeDescription)||void 0===t?void 0:t.href))&&(be.string(n.source)||be.undefined(n.source))&&(be.undefined(n.relatedInformation)||be.typedArray(n.relatedInformation,C.is))}e.create=t,e.is=n}(x||(x={})),function(e){function t(e,t){for(var n=[],r=2;r<arguments.length;r++)n[r-2]=arguments[r];var i={title:e,command:t};return be.defined(n)&&n.length>0&&(i.arguments=n),i}function n(e){var t=e;return be.defined(t)&&be.string(t.title)&&be.string(t.command)}e.create=t,e.is=n}(I||(I={})),function(e){function t(e,t){return{range:e,newText:t}}function n(e,t){return{range:{start:e,end:e},newText:t}}function r(e){return{range:e,newText:""}}function i(e){var t=e;return be.objectLiteral(t)&&be.string(t.newText)&&f.is(t.range)}e.replace=t,e.insert=n,e.del=r,e.is=i}(S||(S={})),function(e){function t(e,t,n){var r={label:e};return void 0!==t&&(r.needsConfirmation=t),void 0!==n&&(r.description=n),r}function n(e){var t=e;return void 0!==t&&be.objectLiteral(t)&&be.string(t.label)&&(be.boolean(t.needsConfirmation)||void 0===t.needsConfirmation)&&(be.string(t.description)||void 0===t.description)}e.create=t,e.is=n}(T||(T={})),function(e){function t(e){var t=e;return"string"===typeof t}e.is=t}(R||(R={})),function(e){function t(e,t,n){return{range:e,newText:t,annotationId:n}}function n(e,t,n){return{range:{start:e,end:e},newText:t,annotationId:n}}function r(e,t){return{range:e,newText:"",annotationId:t}}function i(e){var t=e;return S.is(t)&&(T.is(t.annotationId)||R.is(t.annotationId))}e.replace=t,e.insert=n,e.del=r,e.is=i}(D||(D={})),function(e){function t(e,t){return{textDocument:e,edits:t}}function n(e){var t=e;return be.defined(t)&&V.is(t.textDocument)&&Array.isArray(t.edits)}e.create=t,e.is=n}(P||(P={})),function(e){function t(e,t,n){var r={kind:"create",uri:e};return void 0===t||void 0===t.overwrite&&void 0===t.ignoreIfExists||(r.options=t),void 0!==n&&(r.annotationId=n),r}function n(e){var t=e;return t&&"create"===t.kind&&be.string(t.uri)&&(void 0===t.options||(void 0===t.options.overwrite||be.boolean(t.options.overwrite))&&(void 0===t.options.ignoreIfExists||be.boolean(t.options.ignoreIfExists)))&&(void 0===t.annotationId||R.is(t.annotationId))}e.create=t,e.is=n}(M||(M={})),function(e){function t(e,t,n,r){var i={kind:"rename",oldUri:e,newUri:t};return void 0===n||void 0===n.overwrite&&void 0===n.ignoreIfExists||(i.options=n),void 0!==r&&(i.annotationId=r),i}function n(e){var t=e;return t&&"rename"===t.kind&&be.string(t.oldUri)&&be.string(t.newUri)&&(void 0===t.options||(void 0===t.options.overwrite||be.boolean(t.options.overwrite))&&(void 0===t.options.ignoreIfExists||be.boolean(t.options.ignoreIfExists)))&&(void 0===t.annotationId||R.is(t.annotationId))}e.create=t,e.is=n}(j||(j={})),function(e){function t(e,t,n){var r={kind:"delete",uri:e};return void 0===t||void 0===t.recursive&&void 0===t.ignoreIfNotExists||(r.options=t),void 0!==n&&(r.annotationId=n),r}function n(e){var t=e;return t&&"delete"===t.kind&&be.string(t.uri)&&(void 0===t.options||(void 0===t.options.recursive||be.boolean(t.options.recursive))&&(void 0===t.options.ignoreIfNotExists||be.boolean(t.options.ignoreIfNotExists)))&&(void 0===t.annotationId||R.is(t.annotationId))}e.create=t,e.is=n}(L||(L={})),function(e){function t(e){var t=e;return t&&(void 0!==t.changes||void 0!==t.documentChanges)&&(void 0===t.documentChanges||t.documentChanges.every((function(e){return be.string(e.kind)?M.is(e)||j.is(e)||L.is(e):P.is(e)})))}e.is=t}(F||(F={}));var W,U,V,H,K,z,q,X,B,$,Q,G,J,Y,Z,ee,te,ne,re,ie,oe,ae,se,ce,ue,de,ge,le,he,fe,pe,me=function(){function e(e,t){this.edits=e,this.changeAnnotations=t}return e.prototype.insert=function(e,t,n){var r,i;if(void 0===n?r=S.insert(e,t):R.is(n)?(i=n,r=D.insert(e,t,n)):(this.assertChangeAnnotations(this.changeAnnotations),i=this.changeAnnotations.manage(n),r=D.insert(e,t,i)),this.edits.push(r),void 0!==i)return i},e.prototype.replace=function(e,t,n){var r,i;if(void 0===n?r=S.replace(e,t):R.is(n)?(i=n,r=D.replace(e,t,n)):(this.assertChangeAnnotations(this.changeAnnotations),i=this.changeAnnotations.manage(n),r=D.replace(e,t,i)),this.edits.push(r),void 0!==i)return i},e.prototype.delete=function(e,t){var n,r;if(void 0===t?n=S.del(e):R.is(t)?(r=t,n=D.del(e,t)):(this.assertChangeAnnotations(this.changeAnnotations),r=this.changeAnnotations.manage(t),n=D.del(e,r)),this.edits.push(n),void 0!==r)return r},e.prototype.add=function(e){this.edits.push(e)},e.prototype.all=function(){return this.edits},e.prototype.clear=function(){this.edits.splice(0,this.edits.length)},e.prototype.assertChangeAnnotations=function(e){if(void 0===e)throw new Error("Text edit change is not configured to manage change annotations.")},e}(),ve=function(){function e(e){this._annotations=void 0===e?Object.create(null):e,this._counter=0,this._size=0}return e.prototype.all=function(){return this._annotations},Object.defineProperty(e.prototype,"size",{get:function(){return this._size},enumerable:!1,configurable:!0}),e.prototype.manage=function(e,t){var n;if(R.is(e)?n=e:(n=this.nextId(),t=e),void 0!==this._annotations[n])throw new Error("Id "+n+" is already in use.");if(void 0===t)throw new Error("No annotation provided for id "+n);return this._annotations[n]=t,this._size++,n},e.prototype.nextId=function(){return this._counter++,this._counter.toString()},e}();(function(){function e(e){var t=this;this._textEditChanges=Object.create(null),void 0!==e?(this._workspaceEdit=e,e.documentChanges?(this._changeAnnotations=new ve(e.changeAnnotations),e.changeAnnotations=this._changeAnnotations.all(),e.documentChanges.forEach((function(e){if(P.is(e)){var n=new me(e.edits,t._changeAnnotations);t._textEditChanges[e.textDocument.uri]=n}}))):e.changes&&Object.keys(e.changes).forEach((function(n){var r=new me(e.changes[n]);t._textEditChanges[n]=r}))):this._workspaceEdit={}}Object.defineProperty(e.prototype,"edit",{get:function(){return this.initDocumentChanges(),void 0!==this._changeAnnotations&&(0===this._changeAnnotations.size?this._workspaceEdit.changeAnnotations=void 0:this._workspaceEdit.changeAnnotations=this._changeAnnotations.all()),this._workspaceEdit},enumerable:!1,configurable:!0}),e.prototype.getTextEditChange=function(e){if(V.is(e)){if(this.initDocumentChanges(),void 0===this._workspaceEdit.documentChanges)throw new Error("Workspace edit is not configured for document changes.");var t={uri:e.uri,version:e.version},n=this._textEditChanges[t.uri];if(!n){var r=[],i={textDocument:t,edits:r};this._workspaceEdit.documentChanges.push(i),n=new me(r,this._changeAnnotations),this._textEditChanges[t.uri]=n}return n}if(this.initChanges(),void 0===this._workspaceEdit.changes)throw new Error("Workspace edit is not configured for normal text edit changes.");n=this._textEditChanges[e];if(!n){r=[];this._workspaceEdit.changes[e]=r,n=new me(r),this._textEditChanges[e]=n}return n},e.prototype.initDocumentChanges=function(){void 0===this._workspaceEdit.documentChanges&&void 0===this._workspaceEdit.changes&&(this._changeAnnotations=new ve,this._workspaceEdit.documentChanges=[],this._workspaceEdit.changeAnnotations=this._changeAnnotations.all())},e.prototype.initChanges=function(){void 0===this._workspaceEdit.documentChanges&&void 0===this._workspaceEdit.changes&&(this._workspaceEdit.changes=Object.create(null))},e.prototype.createFile=function(e,t,n){if(this.initDocumentChanges(),void 0===this._workspaceEdit.documentChanges)throw new Error("Workspace edit is not configured for document changes.");var r,i,o;if(T.is(t)||R.is(t)?r=t:n=t,void 0===r?i=M.create(e,n):(o=R.is(r)?r:this._changeAnnotations.manage(r),i=M.create(e,n,o)),this._workspaceEdit.documentChanges.push(i),void 0!==o)return o},e.prototype.renameFile=function(e,t,n,r){if(this.initDocumentChanges(),void 0===this._workspaceEdit.documentChanges)throw new Error("Workspace edit is not configured for document changes.");var i,o,a;if(T.is(n)||R.is(n)?i=n:r=n,void 0===i?o=j.create(e,t,r):(a=R.is(i)?i:this._changeAnnotations.manage(i),o=j.create(e,t,r,a)),this._workspaceEdit.documentChanges.push(o),void 0!==a)return a},e.prototype.deleteFile=function(e,t,n){if(this.initDocumentChanges(),void 0===this._workspaceEdit.documentChanges)throw new Error("Workspace edit is not configured for document changes.");var r,i,o;if(T.is(t)||R.is(t)?r=t:n=t,void 0===r?i=L.create(e,n):(o=R.is(r)?r:this._changeAnnotations.manage(r),i=L.create(e,n,o)),this._workspaceEdit.documentChanges.push(i),void 0!==o)return o}})();(function(e){function t(e){return{uri:e}}function n(e){var t=e;return be.defined(t)&&be.string(t.uri)}e.create=t,e.is=n})(W||(W={})),function(e){function t(e,t){return{uri:e,version:t}}function n(e){var t=e;return be.defined(t)&&be.string(t.uri)&&be.integer(t.version)}e.create=t,e.is=n}(U||(U={})),function(e){function t(e,t){return{uri:e,version:t}}function n(e){var t=e;return be.defined(t)&&be.string(t.uri)&&(null===t.version||be.integer(t.version))}e.create=t,e.is=n}(V||(V={})),function(e){function t(e,t,n,r){return{uri:e,languageId:t,version:n,text:r}}function n(e){var t=e;return be.defined(t)&&be.string(t.uri)&&be.string(t.languageId)&&be.integer(t.version)&&be.string(t.text)}e.create=t,e.is=n}(H||(H={})),function(e){e.PlainText="plaintext",e.Markdown="markdown"}(K||(K={})),function(e){function t(t){var n=t;return n===e.PlainText||n===e.Markdown}e.is=t}(K||(K={})),function(e){function t(e){var t=e;return be.objectLiteral(e)&&K.is(t.kind)&&be.string(t.value)}e.is=t}(z||(z={})),function(e){e.Text=1,e.Method=2,e.Function=3,e.Constructor=4,e.Field=5,e.Variable=6,e.Class=7,e.Interface=8,e.Module=9,e.Property=10,e.Unit=11,e.Value=12,e.Enum=13,e.Keyword=14,e.Snippet=15,e.Color=16,e.File=17,e.Reference=18,e.Folder=19,e.EnumMember=20,e.Constant=21,e.Struct=22,e.Event=23,e.Operator=24,e.TypeParameter=25}(q||(q={})),function(e){e.PlainText=1,e.Snippet=2}(X||(X={})),function(e){e.Deprecated=1}(B||(B={})),function(e){function t(e,t,n){return{newText:e,insert:t,replace:n}}function n(e){var t=e;return t&&be.string(t.newText)&&f.is(t.insert)&&f.is(t.replace)}e.create=t,e.is=n}($||($={})),function(e){e.asIs=1,e.adjustIndentation=2}(Q||(Q={})),function(e){function t(e){return{label:e}}e.create=t}(G||(G={})),function(e){function t(e,t){return{items:e||[],isIncomplete:!!t}}e.create=t}(J||(J={})),function(e){function t(e){return e.replace(/[\\`*_{}[\]()#+\-.!]/g,"\\$&")}function n(e){var t=e;return be.string(t)||be.objectLiteral(t)&&be.string(t.language)&&be.string(t.value)}e.fromPlainText=t,e.is=n}(Y||(Y={})),function(e){function t(e){var t=e;return!!t&&be.objectLiteral(t)&&(z.is(t.contents)||Y.is(t.contents)||be.typedArray(t.contents,Y.is))&&(void 0===e.range||f.is(e.range))}e.is=t}(Z||(Z={})),function(e){function t(e,t){return t?{label:e,documentation:t}:{label:e}}e.create=t}(ee||(ee={})),function(e){function t(e,t){for(var n=[],r=2;r<arguments.length;r++)n[r-2]=arguments[r];var i={label:e};return be.defined(t)&&(i.documentation=t),be.defined(n)?i.parameters=n:i.parameters=[],i}e.create=t}(te||(te={})),function(e){e.Text=1,e.Read=2,e.Write=3}(ne||(ne={})),function(e){function t(e,t){var n={range:e};return be.number(t)&&(n.kind=t),n}e.create=t}(re||(re={})),function(e){e.File=1,e.Module=2,e.Namespace=3,e.Package=4,e.Class=5,e.Method=6,e.Property=7,e.Field=8,e.Constructor=9,e.Enum=10,e.Interface=11,e.Function=12,e.Variable=13,e.Constant=14,e.String=15,e.Number=16,e.Boolean=17,e.Array=18,e.Object=19,e.Key=20,e.Null=21,e.EnumMember=22,e.Struct=23,e.Event=24,e.Operator=25,e.TypeParameter=26}(ie||(ie={})),function(e){e.Deprecated=1}(oe||(oe={})),function(e){function t(e,t,n,r,i){var o={name:e,kind:t,location:{uri:r,range:n}};return i&&(o.containerName=i),o}e.create=t}(ae||(ae={})),function(e){function t(e,t,n,r,i,o){var a={name:e,detail:t,kind:n,range:r,selectionRange:i};return void 0!==o&&(a.children=o),a}function n(e){var t=e;return t&&be.string(t.name)&&be.number(t.kind)&&f.is(t.range)&&f.is(t.selectionRange)&&(void 0===t.detail||be.string(t.detail))&&(void 0===t.deprecated||be.boolean(t.deprecated))&&(void 0===t.children||Array.isArray(t.children))&&(void 0===t.tags||Array.isArray(t.tags))}e.create=t,e.is=n}(se||(se={})),function(e){e.Empty="",e.QuickFix="quickfix",e.Refactor="refactor",e.RefactorExtract="refactor.extract",e.RefactorInline="refactor.inline",e.RefactorRewrite="refactor.rewrite",e.Source="source",e.SourceOrganizeImports="source.organizeImports",e.SourceFixAll="source.fixAll"}(ce||(ce={})),function(e){function t(e,t){var n={diagnostics:e};return void 0!==t&&null!==t&&(n.only=t),n}function n(e){var t=e;return be.defined(t)&&be.typedArray(t.diagnostics,x.is)&&(void 0===t.only||be.typedArray(t.only,be.string))}e.create=t,e.is=n}(ue||(ue={})),function(e){function t(e,t,n){var r={title:e},i=!0;return"string"===typeof t?(i=!1,r.kind=t):I.is(t)?r.command=t:r.edit=t,i&&void 0!==n&&(r.kind=n),r}function n(e){var t=e;return t&&be.string(t.title)&&(void 0===t.diagnostics||be.typedArray(t.diagnostics,x.is))&&(void 0===t.kind||be.string(t.kind))&&(void 0!==t.edit||void 0!==t.command)&&(void 0===t.command||I.is(t.command))&&(void 0===t.isPreferred||be.boolean(t.isPreferred))&&(void 0===t.edit||F.is(t.edit))}e.create=t,e.is=n}(de||(de={})),function(e){function t(e,t){var n={range:e};return be.defined(t)&&(n.data=t),n}function n(e){var t=e;return be.defined(t)&&f.is(t.range)&&(be.undefined(t.command)||I.is(t.command))}e.create=t,e.is=n}(ge||(ge={})),function(e){function t(e,t){return{tabSize:e,insertSpaces:t}}function n(e){var t=e;return be.defined(t)&&be.uinteger(t.tabSize)&&be.boolean(t.insertSpaces)}e.create=t,e.is=n}(le||(le={})),function(e){function t(e,t,n){return{range:e,target:t,data:n}}function n(e){var t=e;return be.defined(t)&&f.is(t.range)&&(be.undefined(t.target)||be.string(t.target))}e.create=t,e.is=n}(he||(he={})),function(e){function t(e,t){return{range:e,parent:t}}function n(t){var n=t;return void 0!==n&&f.is(n.range)&&(void 0===n.parent||e.is(n.parent))}e.create=t,e.is=n}(fe||(fe={})),function(e){function t(e,t,n,r){return new ke(e,t,n,r)}function n(e){var t=e;return!!(be.defined(t)&&be.string(t.uri)&&(be.undefined(t.languageId)||be.string(t.languageId))&&be.uinteger(t.lineCount)&&be.func(t.getText)&&be.func(t.positionAt)&&be.func(t.offsetAt))}function r(e,t){for(var n=e.getText(),r=i(t,(function(e,t){var n=e.range.start.line-t.range.start.line;return 0===n?e.range.start.character-t.range.start.character:n})),o=n.length,a=r.length-1;a>=0;a--){var s=r[a],c=e.offsetAt(s.range.start),u=e.offsetAt(s.range.end);if(!(u<=o))throw new Error("Overlapping edit");n=n.substring(0,c)+s.newText+n.substring(u,n.length),o=c}return n}function i(e,t){if(e.length<=1)return e;var n=e.length/2|0,r=e.slice(0,n),o=e.slice(n);i(r,t),i(o,t);var a=0,s=0,c=0;while(a<r.length&&s<o.length){var u=t(r[a],o[s]);e[c++]=u<=0?r[a++]:o[s++]}while(a<r.length)e[c++]=r[a++];while(s<o.length)e[c++]=o[s++];return e}e.create=t,e.is=n,e.applyEdits=r}(pe||(pe={}));var be,ke=function(){function e(e,t,n,r){this._uri=e,this._languageId=t,this._version=n,this._content=r,this._lineOffsets=void 0}return Object.defineProperty(e.prototype,"uri",{get:function(){return this._uri},enumerable:!1,configurable:!0}),Object.defineProperty(e.prototype,"languageId",{get:function(){return this._languageId},enumerable:!1,configurable:!0}),Object.defineProperty(e.prototype,"version",{get:function(){return this._version},enumerable:!1,configurable:!0}),e.prototype.getText=function(e){if(e){var t=this.offsetAt(e.start),n=this.offsetAt(e.end);return this._content.substring(t,n)}return this._content},e.prototype.update=function(e,t){this._content=e.text,this._version=t,this._lineOffsets=void 0},e.prototype.getLineOffsets=function(){if(void 0===this._lineOffsets){for(var e=[],t=this._content,n=!0,r=0;r<t.length;r++){n&&(e.push(r),n=!1);var i=t.charAt(r);n="\r"===i||"\n"===i,"\r"===i&&r+1<t.length&&"\n"===t.charAt(r+1)&&r++}n&&t.length>0&&e.push(t.length),this._lineOffsets=e}return this._lineOffsets},e.prototype.positionAt=function(e){e=Math.max(Math.min(e,this._content.length),0);var t=this.getLineOffsets(),n=0,r=t.length;if(0===r)return h.create(0,e);while(n<r){var i=Math.floor((n+r)/2);t[i]>e?r=i:n=i+1}var o=n-1;return h.create(o,e-t[o])},e.prototype.offsetAt=function(e){var t=this.getLineOffsets();if(e.line>=t.length)return this._content.length;if(e.line<0)return 0;var n=t[e.line],r=e.line+1<t.length?t[e.line+1]:this._content.length;return Math.max(Math.min(n+e.character,r),n)},Object.defineProperty(e.prototype,"lineCount",{get:function(){return this.getLineOffsets().length},enumerable:!1,configurable:!0}),e}();(function(e){var t=Object.prototype.toString;function n(e){return"undefined"!==typeof e}function r(e){return"undefined"===typeof e}function i(e){return!0===e||!1===e}function o(e){return"[object String]"===t.call(e)}function a(e){return"[object Number]"===t.call(e)}function s(e,n,r){return"[object Number]"===t.call(e)&&n<=e&&e<=r}function c(e){return"[object Number]"===t.call(e)&&-2147483648<=e&&e<=2147483647}function u(e){return"[object Number]"===t.call(e)&&0<=e&&e<=2147483647}function d(e){return"[object Function]"===t.call(e)}function g(e){return null!==e&&"object"===typeof e}function l(e,t){return Array.isArray(e)&&e.every(t)}e.defined=n,e.undefined=r,e.boolean=i,e.string=o,e.number=a,e.numberRange=s,e.integer=c,e.uinteger=u,e.func=d,e.objectLiteral=g,e.typedArray=l})(be||(be={}));var we=class{constructor(e,t,n){this._languageId=e,this._worker=t;const r=e=>{let t,n=e.getLanguageId();n===this._languageId&&(this._listener[e.uri.toString()]=e.onDidChangeContent((()=>{window.clearTimeout(t),t=window.setTimeout((()=>this._doValidate(e.uri,n)),500)})),this._doValidate(e.uri,n))},i=e=>{d.editor.setModelMarkers(e,this._languageId,[]);let t=e.uri.toString(),n=this._listener[t];n&&(n.dispose(),delete this._listener[t])};this._disposables.push(d.editor.onDidCreateModel(r)),this._disposables.push(d.editor.onWillDisposeModel(i)),this._disposables.push(d.editor.onDidChangeModelLanguage((e=>{i(e.model),r(e.model)}))),this._disposables.push(n((e=>{d.editor.getModels().forEach((e=>{e.getLanguageId()===this._languageId&&(i(e),r(e))}))}))),this._disposables.push({dispose:()=>{d.editor.getModels().forEach(i);for(let e in this._listener)this._listener[e].dispose()}}),d.editor.getModels().forEach(r)}_disposables=[];_listener=Object.create(null);dispose(){this._disposables.forEach((e=>e&&e.dispose())),this._disposables.length=0}_doValidate(e,t){this._worker(e).then((t=>t.doValidation(e.toString()))).then((n=>{const r=n.map((t=>Ce(e,t)));let i=d.editor.getModel(e);i&&i.getLanguageId()===t&&d.editor.setModelMarkers(i,t,r)})).then(void 0,(e=>{console.error(e)}))}};function _e(e){switch(e){case y.Error:return d.MarkerSeverity.Error;case y.Warning:return d.MarkerSeverity.Warning;case y.Information:return d.MarkerSeverity.Info;case y.Hint:return d.MarkerSeverity.Hint;default:return d.MarkerSeverity.Info}}function Ce(e,t){let n="number"===typeof t.code?String(t.code):t.code;return{severity:_e(t.severity),startLineNumber:t.range.start.line+1,startColumn:t.range.start.character+1,endLineNumber:t.range.end.line+1,endColumn:t.range.end.character+1,message:t.message,code:n,source:t.source}}var ye=class{constructor(e,t){this._worker=e,this._triggerCharacters=t}get triggerCharacters(){return this._triggerCharacters}provideCompletionItems(e,t,n,r){const i=e.uri;return this._worker(i).then((e=>e.doComplete(i.toString(),Ee(t)))).then((n=>{if(!n)return;const r=e.getWordUntilPosition(t),i=new d.Range(t.lineNumber,r.startColumn,t.lineNumber,r.endColumn),o=n.items.map((e=>{const t={label:e.label,insertText:e.insertText||e.label,sortText:e.sortText,filterText:e.filterText,documentation:e.documentation,detail:e.detail,command:Re(e.command),range:i,kind:Se(e.kind)};return e.textEdit&&(Ie(e.textEdit)?t.range={insert:xe(e.textEdit.insert),replace:xe(e.textEdit.replace)}:t.range=xe(e.textEdit.range),t.insertText=e.textEdit.newText),e.additionalTextEdits&&(t.additionalTextEdits=e.additionalTextEdits.map(Te)),e.insertTextFormat===X.Snippet&&(t.insertTextRules=d.languages.CompletionItemInsertTextRule.InsertAsSnippet),t}));return{isIncomplete:n.isIncomplete,suggestions:o}}))}};function Ee(e){if(e)return{character:e.column-1,line:e.lineNumber-1}}function Ae(e){if(e)return{start:{line:e.startLineNumber-1,character:e.startColumn-1},end:{line:e.endLineNumber-1,character:e.endColumn-1}}}function xe(e){if(e)return new d.Range(e.start.line+1,e.start.character+1,e.end.line+1,e.end.character+1)}function Ie(e){return"undefined"!==typeof e.insert&&"undefined"!==typeof e.replace}function Se(e){const t=d.languages.CompletionItemKind;switch(e){case q.Text:return t.Text;case q.Method:return t.Method;case q.Function:return t.Function;case q.Constructor:return t.Constructor;case q.Field:return t.Field;case q.Variable:return t.Variable;case q.Class:return t.Class;case q.Interface:return t.Interface;case q.Module:return t.Module;case q.Property:return t.Property;case q.Unit:return t.Unit;case q.Value:return t.Value;case q.Enum:return t.Enum;case q.Keyword:return t.Keyword;case q.Snippet:return t.Snippet;case q.Color:return t.Color;case q.File:return t.File;case q.Reference:return t.Reference}return t.Property}function Te(e){if(e)return{range:xe(e.range),text:e.newText}}function Re(e){return e&&"editor.action.triggerSuggest"===e.command?{id:e.command,title:e.title,arguments:e.arguments}:void 0}var De=class{constructor(e){this._worker=e}provideHover(e,t,n){let r=e.uri;return this._worker(r).then((e=>e.doHover(r.toString(),Ee(t)))).then((e=>{if(e)return{range:xe(e.range),contents:je(e.contents)}}))}};function Pe(e){return e&&"object"===typeof e&&"string"===typeof e.kind}function Me(e){return"string"===typeof e?{value:e}:Pe(e)?"plaintext"===e.kind?{value:e.value.replace(/[\\`*_{}[\]()#+\-.!]/g,"\\$&")}:{value:e.value}:{value:"```"+e.language+"\n"+e.value+"\n```\n"}}function je(e){if(e)return Array.isArray(e)?e.map(Me):[Me(e)]}var Le=class{constructor(e){this._worker=e}provideDocumentHighlights(e,t,n){const r=e.uri;return this._worker(r).then((e=>e.findDocumentHighlights(r.toString(),Ee(t)))).then((e=>{if(e)return e.map((e=>({range:xe(e.range),kind:Fe(e.kind)})))}))}};function Fe(e){switch(e){case ne.Read:return d.languages.DocumentHighlightKind.Read;case ne.Write:return d.languages.DocumentHighlightKind.Write;case ne.Text:return d.languages.DocumentHighlightKind.Text}return d.languages.DocumentHighlightKind.Text}var Oe=class{constructor(e){this._worker=e}provideDefinition(e,t,n){const r=e.uri;return this._worker(r).then((e=>e.findDefinition(r.toString(),Ee(t)))).then((e=>{if(e)return[Ne(e)]}))}};function Ne(e){return{uri:d.Uri.parse(e.uri),range:xe(e.range)}}var We=class{constructor(e){this._worker=e}provideReferences(e,t,n,r){const i=e.uri;return this._worker(i).then((e=>e.findReferences(i.toString(),Ee(t)))).then((e=>{if(e)return e.map(Ne)}))}},Ue=class{constructor(e){this._worker=e}provideRenameEdits(e,t,n,r){const i=e.uri;return this._worker(i).then((e=>e.doRename(i.toString(),Ee(t),n))).then((e=>Ve(e)))}};function Ve(e){if(!e||!e.changes)return;let t=[];for(let n in e.changes){const r=d.Uri.parse(n);for(let i of e.changes[n])t.push({resource:r,versionId:void 0,textEdit:{range:xe(i.range),text:i.newText}})}return{edits:t}}var He=class{constructor(e){this._worker=e}provideDocumentSymbols(e,t){const n=e.uri;return this._worker(n).then((e=>e.findDocumentSymbols(n.toString()))).then((e=>{if(e)return e.map((e=>({name:e.name,detail:"",containerName:e.containerName,kind:Ke(e.kind),range:xe(e.location.range),selectionRange:xe(e.location.range),tags:[]})))}))}};function Ke(e){let t=d.languages.SymbolKind;switch(e){case ie.File:return t.Array;case ie.Module:return t.Module;case ie.Namespace:return t.Namespace;case ie.Package:return t.Package;case ie.Class:return t.Class;case ie.Method:return t.Method;case ie.Property:return t.Property;case ie.Field:return t.Field;case ie.Constructor:return t.Constructor;case ie.Enum:return t.Enum;case ie.Interface:return t.Interface;case ie.Function:return t.Function;case ie.Variable:return t.Variable;case ie.Constant:return t.Constant;case ie.String:return t.String;case ie.Number:return t.Number;case ie.Boolean:return t.Boolean;case ie.Array:return t.Array}return t.Function}var ze=class{constructor(e){this._worker=e}provideLinks(e,t){const n=e.uri;return this._worker(n).then((e=>e.findDocumentLinks(n.toString()))).then((e=>{if(e)return{links:e.map((e=>({range:xe(e.range),url:e.target})))}}))}},qe=class{constructor(e){this._worker=e}provideDocumentFormattingEdits(e,t,n){const r=e.uri;return this._worker(r).then((e=>e.format(r.toString(),null,Be(t)).then((e=>{if(e&&0!==e.length)return e.map(Te)}))))}},Xe=class{constructor(e){this._worker=e}canFormatMultipleRanges=!1;provideDocumentRangeFormattingEdits(e,t,n,r){const i=e.uri;return this._worker(i).then((e=>e.format(i.toString(),Ae(t),Be(n)).then((e=>{if(e&&0!==e.length)return e.map(Te)}))))}};function Be(e){return{tabSize:e.tabSize,insertSpaces:e.insertSpaces}}var $e=class{constructor(e){this._worker=e}provideDocumentColors(e,t){const n=e.uri;return this._worker(n).then((e=>e.findDocumentColors(n.toString()))).then((e=>{if(e)return e.map((e=>({color:e.color,range:xe(e.range)})))}))}provideColorPresentations(e,t,n){const r=e.uri;return this._worker(r).then((e=>e.getColorPresentations(r.toString(),t.color,Ae(t.range)))).then((e=>{if(e)return e.map((e=>{let t={label:e.label};return e.textEdit&&(t.textEdit=Te(e.textEdit)),e.additionalTextEdits&&(t.additionalTextEdits=e.additionalTextEdits.map(Te)),t}))}))}},Qe=class{constructor(e){this._worker=e}provideFoldingRanges(e,t,n){const r=e.uri;return this._worker(r).then((e=>e.getFoldingRanges(r.toString(),t))).then((e=>{if(e)return e.map((e=>{const t={start:e.startLine+1,end:e.endLine+1};return"undefined"!==typeof e.kind&&(t.kind=Ge(e.kind)),t}))}))}};function Ge(e){switch(e){case w.Comment:return d.languages.FoldingRangeKind.Comment;case w.Imports:return d.languages.FoldingRangeKind.Imports;case w.Region:return d.languages.FoldingRangeKind.Region}}var Je,Ye=class{constructor(e){this._worker=e}provideSelectionRanges(e,t,n){const r=e.uri;return this._worker(r).then((e=>e.getSelectionRanges(r.toString(),t.map(Ee)))).then((e=>{if(e)return e.map((e=>{const t=[];while(e)t.push({range:xe(e.range)}),e=e.parent;return t}))}))}};function Ze(e,t){void 0===t&&(t=!1);var n=e.length,r=0,i="",o=0,a=16,s=0,c=0,u=0,d=0,g=0;function l(t,n){var i=0,o=0;while(i<t||!n){var a=e.charCodeAt(r);if(a>=48&&a<=57)o=16*o+a-48;else if(a>=65&&a<=70)o=16*o+a-65+10;else{if(!(a>=97&&a<=102))break;o=16*o+a-97+10}r++,i++}return i<t&&(o=-1),o}function h(e){r=e,i="",o=0,a=16,g=0}function f(){var t=r;if(48===e.charCodeAt(r))r++;else{r++;while(r<e.length&&nt(e.charCodeAt(r)))r++}if(r<e.length&&46===e.charCodeAt(r)){if(r++,!(r<e.length&&nt(e.charCodeAt(r))))return g=3,e.substring(t,r);r++;while(r<e.length&&nt(e.charCodeAt(r)))r++}var n=r;if(r<e.length&&(69===e.charCodeAt(r)||101===e.charCodeAt(r)))if(r++,(r<e.length&&43===e.charCodeAt(r)||45===e.charCodeAt(r))&&r++,r<e.length&&nt(e.charCodeAt(r))){r++;while(r<e.length&&nt(e.charCodeAt(r)))r++;n=r}else g=3;return e.substring(t,n)}function p(){var t="",i=r;while(1){if(r>=n){t+=e.substring(i,r),g=2;break}var o=e.charCodeAt(r);if(34===o){t+=e.substring(i,r),r++;break}if(92!==o){if(o>=0&&o<=31){if(tt(o)){t+=e.substring(i,r),g=2;break}g=6}r++}else{if(t+=e.substring(i,r),r++,r>=n){g=2;break}var a=e.charCodeAt(r++);switch(a){case 34:t+='"';break;case 92:t+="\\";break;case 47:t+="/";break;case 98:t+="\b";break;case 102:t+="\f";break;case 110:t+="\n";break;case 114:t+="\r";break;case 116:t+="\t";break;case 117:var s=l(4,!0);s>=0?t+=String.fromCharCode(s):g=4;break;default:g=5}i=r}}return t}function m(){if(i="",g=0,o=r,c=s,d=u,r>=n)return o=n,a=17;var t=e.charCodeAt(r);if(et(t)){do{r++,i+=String.fromCharCode(t),t=e.charCodeAt(r)}while(et(t));return a=15}if(tt(t))return r++,i+=String.fromCharCode(t),13===t&&10===e.charCodeAt(r)&&(r++,i+="\n"),s++,u=r,a=14;switch(t){case 123:return r++,a=1;case 125:return r++,a=2;case 91:return r++,a=3;case 93:return r++,a=4;case 58:return r++,a=6;case 44:return r++,a=5;case 34:return r++,i=p(),a=10;case 47:var l=r-1;if(47===e.charCodeAt(r+1)){r+=2;while(r<n){if(tt(e.charCodeAt(r)))break;r++}return i=e.substring(l,r),a=12}if(42===e.charCodeAt(r+1)){r+=2;var h=n-1,m=!1;while(r<h){var b=e.charCodeAt(r);if(42===b&&47===e.charCodeAt(r+1)){r+=2,m=!0;break}r++,tt(b)&&(13===b&&10===e.charCodeAt(r)&&r++,s++,u=r)}return m||(r++,g=1),i=e.substring(l,r),a=13}return i+=String.fromCharCode(t),r++,a=16;case 45:if(i+=String.fromCharCode(t),r++,r===n||!nt(e.charCodeAt(r)))return a=16;case 48:case 49:case 50:case 51:case 52:case 53:case 54:case 55:case 56:case 57:return i+=f(),a=11;default:while(r<n&&v(t))r++,t=e.charCodeAt(r);if(o!==r){switch(i=e.substring(o,r),i){case"true":return a=8;case"false":return a=9;case"null":return a=7}return a=16}return i+=String.fromCharCode(t),r++,a=16}}function v(e){if(et(e)||tt(e))return!1;switch(e){case 125:case 93:case 123:case 91:case 34:case 58:case 44:case 47:return!1}return!0}function b(){var e;do{e=m()}while(e>=12&&e<=15);return e}return{setPosition:h,getPosition:function(){return r},scan:t?b:m,getToken:function(){return a},getTokenValue:function(){return i},getTokenOffset:function(){return o},getTokenLength:function(){return r-o},getTokenStartLine:function(){return c},getTokenStartCharacter:function(){return o-d},getTokenError:function(){return g}}}function et(e){return 32===e||9===e||11===e||12===e||160===e||5760===e||e>=8192&&e<=8203||8239===e||8287===e||12288===e||65279===e}function tt(e){return 10===e||13===e||8232===e||8233===e}function nt(e){return e>=48&&e<=57}(function(e){e.DEFAULT={allowTrailingComma:!1}})(Je||(Je={}));var rt=Ze;function it(e){return{getInitialState:()=>new vt(null,null,!1,null),tokenize:(t,n)=>bt(e,t,n)}}var ot="delimiter.bracket.json",at="delimiter.array.json",st="delimiter.colon.json",ct="delimiter.comma.json",ut="keyword.json",dt="keyword.json",gt="string.value.json",lt="number.json",ht="string.key.json",ft="comment.block.json",pt="comment.line.json",mt=class{constructor(e,t){this.parent=e,this.type=t}static pop(e){return e?e.parent:null}static push(e,t){return new mt(e,t)}static equals(e,t){if(!e&&!t)return!0;if(!e||!t)return!1;while(e&&t){if(e===t)return!0;if(e.type!==t.type)return!1;e=e.parent,t=t.parent}return!0}},vt=class{_state;scanError;lastWasColon;parents;constructor(e,t,n,r){this._state=e,this.scanError=t,this.lastWasColon=n,this.parents=r}clone(){return new vt(this._state,this.scanError,this.lastWasColon,this.parents)}equals(e){return e===this||!!(e&&e instanceof vt)&&(this.scanError===e.scanError&&this.lastWasColon===e.lastWasColon&&mt.equals(this.parents,e.parents))}getStateData(){return this._state}setStateData(e){this._state=e}};function bt(e,t,n,r=0){let i=0,o=!1;switch(n.scanError){case 2:t='"'+t,i=1;break;case 1:t="/*"+t,i=2;break}const a=rt(t);let s=n.lastWasColon,c=n.parents;const u={tokens:[],endState:n.clone()};while(1){let d=r+a.getPosition(),g="";const l=a.scan();if(17===l)break;if(d===r+a.getPosition())throw new Error("Scanner did not advance, next 3 characters are: "+t.substr(a.getPosition(),3));switch(o&&(d-=i),o=i>0,l){case 1:c=mt.push(c,0),g=ot,s=!1;break;case 2:c=mt.pop(c),g=ot,s=!1;break;case 3:c=mt.push(c,1),g=at,s=!1;break;case 4:c=mt.pop(c),g=at,s=!1;break;case 6:g=st,s=!0;break;case 5:g=ct,s=!1;break;case 8:case 9:g=ut,s=!1;break;case 7:g=dt,s=!1;break;case 10:const e=c?c.type:0,t=1===e;g=s||t?gt:ht,s=!1;break;case 11:g=lt,s=!1;break}if(e)switch(l){case 12:g=pt;break;case 13:g=ft;break}u.endState=new vt(n.getStateData(),a.getTokenError(),s,c),u.tokens.push({startIndex:d,scopes:g})}return u}var kt=class extends we{constructor(e,t,n){super(e,t,n.onDidChange),this._disposables.push(d.editor.onWillDisposeModel((e=>{this._resetSchema(e.uri)}))),this._disposables.push(d.editor.onDidChangeModelLanguage((e=>{this._resetSchema(e.model.uri)})))}_resetSchema(e){this._worker().then((t=>{t.resetSchema(e.toString())}))}};function wt(e){const t=[],n=[],r=new N(e);t.push(r);const i=(...e)=>r.getLanguageServiceWorker(...e);function o(){const{languageId:t,modeConfiguration:r}=e;Ct(n),r.documentFormattingEdits&&n.push(d.languages.registerDocumentFormattingEditProvider(t,new qe(i))),r.documentRangeFormattingEdits&&n.push(d.languages.registerDocumentRangeFormattingEditProvider(t,new Xe(i))),r.completionItems&&n.push(d.languages.registerCompletionItemProvider(t,new ye(i,[" ",":",'"']))),r.hovers&&n.push(d.languages.registerHoverProvider(t,new De(i))),r.documentSymbols&&n.push(d.languages.registerDocumentSymbolProvider(t,new He(i))),r.tokens&&n.push(d.languages.setTokensProvider(t,it(!0))),r.colors&&n.push(d.languages.registerColorProvider(t,new $e(i))),r.foldingRanges&&n.push(d.languages.registerFoldingRangeProvider(t,new Qe(i))),r.diagnostics&&n.push(new kt(t,i,e)),r.selectionRanges&&n.push(d.languages.registerSelectionRangeProvider(t,new Ye(i)))}o(),t.push(d.languages.setLanguageConfiguration(e.languageId,yt));let a=e.modeConfiguration;return e.onDidChange((e=>{e.modeConfiguration!==a&&(a=e.modeConfiguration,o())})),t.push(_t(n)),_t(t)}function _t(e){return{dispose:()=>Ct(e)}}function Ct(e){while(e.length)e.pop().dispose()}var yt={wordPattern:/(-?\d*\.\d\w*)|([^\[\{\]\}\:\"\,\s]+)/g,comments:{lineComment:"//",blockComment:["/*","*/"]},brackets:[["{","}"],["[","]"]],autoClosingPairs:[{open:"{",close:"}",notIn:["string"]},{open:"[",close:"]",notIn:["string"]},{open:'"',close:'"',notIn:["string"]}]}}}]);