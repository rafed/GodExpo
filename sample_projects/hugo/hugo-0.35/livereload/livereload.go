// Copyright 2015 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Contains an embedded version of livereload.js
//
// Copyright (c) 2010-2015 Andrey Tarantsov
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package livereload

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gorilla/websocket"
)

// Prefix to signal to LiveReload that we need to navigate to another path.
const hugoNavigatePrefix = "__hugo_navigate"

var upgrader = &websocket.Upgrader{
	// Hugo may potentially spin up multiple HTTP servers, so we need to exclude the
	// port when checking the origin.
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header["Origin"]
		if len(origin) == 0 {
			return true
		}
		u, err := url.Parse(origin[0])
		if err != nil {
			return false
		}

		if u.Host == r.Host {
			return true
		}

		h1, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			return false
		}
		h2, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			return false
		}

		return h1 == h2
	},
	ReadBufferSize: 1024, WriteBufferSize: 1024}

// Handler is a HandlerFunc handling the livereload
// Websocket interaction.
func Handler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws}
	wsHub.register <- c
	defer func() { wsHub.unregister <- c }()
	go c.writer()
	c.reader()
}

// Initialize starts the Websocket Hub handling live reloads.
func Initialize() {
	go wsHub.run()
}

// ForceRefresh tells livereload to force a hard refresh.
func ForceRefresh() {
	RefreshPath("/x.js")
}

// NavigateToPath tells livereload to navigate to the given path.
// This translates to `window.location.href = path` in the client.
func NavigateToPath(path string) {
	RefreshPath(hugoNavigatePrefix + path)
}

// NavigateToPathForPort is similar to NavigateToPath but will also
// set window.location.port to the given port value.
func NavigateToPathForPort(path string, port int) {
	refreshPathForPort(hugoNavigatePrefix+path, port)
}

// RefreshPath tells livereload to refresh only the given path.
// If that path points to a CSS stylesheet or an image, only the changes
// will be updated in the browser, not the entire page.
func RefreshPath(s string) {
	refreshPathForPort(s, -1)
}

func refreshPathForPort(s string, port int) {
	// Tell livereload a file has changed - will force a hard refresh if not CSS or an image
	urlPath := filepath.ToSlash(s)
	portStr := ""
	if port > 0 {
		portStr = fmt.Sprintf(`, "overrideURL": %d`, port)
	}
	msg := fmt.Sprintf(`{"command":"reload","path":%q,"originalPath":"","liveCSS":true,"liveImg":true%s}`, urlPath, portStr)
	wsHub.broadcast <- []byte(msg)
}

// ServeJS serves the liverreload.js who's reference is injected into the page.
func ServeJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(liveReloadJS())
}

func liveReloadJS() []byte {
	return []byte(livereloadJS + hugoLiveReloadPlugin)
}

var (
	// This is temporary patched with this PR (enables sensible error messages):
	// https://github.com/livereload/livereload-js/pull/64
	// TODO(bep) replace with distribution once merged.
	livereloadJS         = `(function e(t,n,o){function i(s,l){if(!n[s]){if(!t[s]){var c=typeof require=="function"&&require;if(!l&&c)return c(s,!0);if(r)return r(s,!0);var a=new Error("Cannot find module '"+s+"'");throw a.code="MODULE_NOT_FOUND",a}var h=n[s]={exports:{}};t[s][0].call(h.exports,function(e){var n=t[s][1][e];return i(n?n:e)},h,h.exports,e,t,n,o)}return n[s].exports}var r=typeof require=="function"&&require;for(var s=0;s<o.length;s++)i(o[s]);return i})({1:[function(e,t,n){(function(){var t,o,i,r,s,l;l=e("./protocol"),r=l.Parser,o=l.PROTOCOL_6,i=l.PROTOCOL_7;s="2.2.2";n.Connector=t=function(){function e(e,t,n,o){this.options=e;this.WebSocket=t;this.Timer=n;this.handlers=o;this._uri="ws"+(this.options.https?"s":"")+"://"+this.options.host+":"+this.options.port+"/livereload";this._nextDelay=this.options.mindelay;this._connectionDesired=false;this.protocol=0;this.protocolParser=new r({connected:function(e){return function(t){e.protocol=t;e._handshakeTimeout.stop();e._nextDelay=e.options.mindelay;e._disconnectionReason="broken";return e.handlers.connected(t)}}(this),error:function(e){return function(t){e.handlers.error(t);return e._closeOnError()}}(this),message:function(e){return function(t){return e.handlers.message(t)}}(this)});this._handshakeTimeout=new n(function(e){return function(){if(!e._isSocketConnected()){return}e._disconnectionReason="handshake-timeout";return e.socket.close()}}(this));this._reconnectTimer=new n(function(e){return function(){if(!e._connectionDesired){return}return e.connect()}}(this));this.connect()}e.prototype._isSocketConnected=function(){return this.socket&&this.socket.readyState===this.WebSocket.OPEN};e.prototype.connect=function(){this._connectionDesired=true;if(this._isSocketConnected()){return}this._reconnectTimer.stop();this._disconnectionReason="cannot-connect";this.protocolParser.reset();this.handlers.connecting();this.socket=new this.WebSocket(this._uri);this.socket.onopen=function(e){return function(t){return e._onopen(t)}}(this);this.socket.onclose=function(e){return function(t){return e._onclose(t)}}(this);this.socket.onmessage=function(e){return function(t){return e._onmessage(t)}}(this);return this.socket.onerror=function(e){return function(t){return e._onerror(t)}}(this)};e.prototype.disconnect=function(){this._connectionDesired=false;this._reconnectTimer.stop();if(!this._isSocketConnected()){return}this._disconnectionReason="manual";return this.socket.close()};e.prototype._scheduleReconnection=function(){if(!this._connectionDesired){return}if(!this._reconnectTimer.running){this._reconnectTimer.start(this._nextDelay);return this._nextDelay=Math.min(this.options.maxdelay,this._nextDelay*2)}};e.prototype.sendCommand=function(e){if(this.protocol==null){return}return this._sendCommand(e)};e.prototype._sendCommand=function(e){return this.socket.send(JSON.stringify(e))};e.prototype._closeOnError=function(){this._handshakeTimeout.stop();this._disconnectionReason="error";return this.socket.close()};e.prototype._onopen=function(e){var t;this.handlers.socketConnected();this._disconnectionReason="handshake-failed";t={command:"hello",protocols:[o,i]};t.ver=s;if(this.options.ext){t.ext=this.options.ext}if(this.options.extver){t.extver=this.options.extver}if(this.options.snipver){t.snipver=this.options.snipver}this._sendCommand(t);return this._handshakeTimeout.start(this.options.handshake_timeout)};e.prototype._onclose=function(e){this.protocol=0;this.handlers.disconnected(this._disconnectionReason,this._nextDelay);return this._scheduleReconnection()};e.prototype._onerror=function(e){};e.prototype._onmessage=function(e){return this.protocolParser.process(e.data)};return e}()}).call(this)},{"./protocol":6}],2:[function(e,t,n){(function(){var e;e={bind:function(e,t,n){if(e.addEventListener){return e.addEventListener(t,n,false)}else if(e.attachEvent){e[t]=1;return e.attachEvent("onpropertychange",function(e){if(e.propertyName===t){return n()}})}else{throw new Error("Attempt to attach custom event "+t+" to something which isn't a DOMElement")}},fire:function(e,t){var n;if(e.addEventListener){n=document.createEvent("HTMLEvents");n.initEvent(t,true,true);return document.dispatchEvent(n)}else if(e.attachEvent){if(e[t]){return e[t]++}}else{throw new Error("Attempt to fire custom event "+t+" on something which isn't a DOMElement")}}};n.bind=e.bind;n.fire=e.fire}).call(this)},{}],3:[function(e,t,n){(function(){var e;t.exports=e=function(){e.identifier="less";e.version="1.0";function e(e,t){this.window=e;this.host=t}e.prototype.reload=function(e,t){if(this.window.less&&this.window.less.refresh){if(e.match(/\.less$/i)){return this.reloadLess(e)}if(t.originalPath.match(/\.less$/i)){return this.reloadLess(t.originalPath)}}return false};e.prototype.reloadLess=function(e){var t,n,o,i;n=function(){var e,n,o,i;o=document.getElementsByTagName("link");i=[];for(e=0,n=o.length;e<n;e++){t=o[e];if(t.href&&t.rel.match(/^stylesheet\/less$/i)||t.rel.match(/stylesheet/i)&&t.type.match(/^text\/(x-)?less$/i)){i.push(t)}}return i}();if(n.length===0){return false}for(o=0,i=n.length;o<i;o++){t=n[o];t.href=this.host.generateCacheBustUrl(t.href)}this.host.console.log("LiveReload is asking LESS to recompile all stylesheets");this.window.less.refresh(true);return true};e.prototype.analyze=function(){return{disable:!!(this.window.less&&this.window.less.refresh)}};return e}()}).call(this)},{}],4:[function(e,t,n){(function(){var t,o,i,r,s,l,c={}.hasOwnProperty;t=e("./connector").Connector;l=e("./timer").Timer;i=e("./options").Options;s=e("./reloader").Reloader;r=e("./protocol").ProtocolError;n.LiveReload=o=function(){function e(e){var n,o,a;this.window=e;this.listeners={};this.plugins=[];this.pluginIdentifiers={};this.console=this.window.console&&this.window.console.log&&this.window.console.error?this.window.location.href.match(/LR-verbose/)?this.window.console:{log:function(){},error:this.window.console.error.bind(this.window.console)}:{log:function(){},error:function(){}};if(!(this.WebSocket=this.window.WebSocket||this.window.MozWebSocket)){this.console.error("LiveReload disabled because the browser does not seem to support web sockets");return}if("LiveReloadOptions"in e){this.options=new i;a=e["LiveReloadOptions"];for(n in a){if(!c.call(a,n))continue;o=a[n];this.options.set(n,o)}}else{this.options=i.extract(this.window.document);if(!this.options){this.console.error("LiveReload disabled because it could not find its own <SCRIPT> tag");return}}this.reloader=new s(this.window,this.console,l);this.connector=new t(this.options,this.WebSocket,l,{connecting:function(e){return function(){}}(this),socketConnected:function(e){return function(){}}(this),connected:function(e){return function(t){var n;if(typeof(n=e.listeners).connect==="function"){n.connect()}e.log("LiveReload is connected to "+e.options.host+":"+e.options.port+" (protocol v"+t+").");return e.analyze()}}(this),error:function(e){return function(e){if(e instanceof r){if(typeof console!=="undefined"&&console!==null){return console.log(""+e.message+".")}}else{if(typeof console!=="undefined"&&console!==null){return console.log("LiveReload internal error: "+e.message)}}}}(this),disconnected:function(e){return function(t,n){var o;if(typeof(o=e.listeners).disconnect==="function"){o.disconnect()}switch(t){case"cannot-connect":return e.log("LiveReload cannot connect to "+e.options.host+":"+e.options.port+", will retry in "+n+" sec.");case"broken":return e.log("LiveReload disconnected from "+e.options.host+":"+e.options.port+", reconnecting in "+n+" sec.");case"handshake-timeout":return e.log("LiveReload cannot connect to "+e.options.host+":"+e.options.port+" (handshake timeout), will retry in "+n+" sec.");case"handshake-failed":return e.log("LiveReload cannot connect to "+e.options.host+":"+e.options.port+" (handshake failed), will retry in "+n+" sec.");case"manual":break;case"error":break;default:return e.log("LiveReload disconnected from "+e.options.host+":"+e.options.port+" ("+t+"), reconnecting in "+n+" sec.")}}}(this),message:function(e){return function(t){switch(t.command){case"reload":return e.performReload(t);case"alert":return e.performAlert(t)}}}(this)});this.initialized=true}e.prototype.on=function(e,t){return this.listeners[e]=t};e.prototype.log=function(e){return this.console.log(""+e)};e.prototype.performReload=function(e){var t,n;this.log("LiveReload received reload request: "+JSON.stringify(e,null,2));return this.reloader.reload(e.path,{liveCSS:(t=e.liveCSS)!=null?t:true,liveImg:(n=e.liveImg)!=null?n:true,originalPath:e.originalPath||"",overrideURL:e.overrideURL||"",serverURL:"http://"+this.options.host+":"+this.options.port})};e.prototype.performAlert=function(e){return alert(e.message)};e.prototype.shutDown=function(){var e;if(!this.initialized){return}this.connector.disconnect();this.log("LiveReload disconnected.");return typeof(e=this.listeners).shutdown==="function"?e.shutdown():void 0};e.prototype.hasPlugin=function(e){return!!this.pluginIdentifiers[e]};e.prototype.addPlugin=function(e){var t;if(!this.initialized){return}if(this.hasPlugin(e.identifier)){return}this.pluginIdentifiers[e.identifier]=true;t=new e(this.window,{_livereload:this,_reloader:this.reloader,_connector:this.connector,console:this.console,Timer:l,generateCacheBustUrl:function(e){return function(t){return e.reloader.generateCacheBustUrl(t)}}(this)});this.plugins.push(t);this.reloader.addPlugin(t)};e.prototype.analyze=function(){var e,t,n,o,i,r;if(!this.initialized){return}if(!(this.connector.protocol>=7)){return}n={};r=this.plugins;for(o=0,i=r.length;o<i;o++){e=r[o];n[e.constructor.identifier]=t=(typeof e.analyze==="function"?e.analyze():void 0)||{};t.version=e.constructor.version}this.connector.sendCommand({command:"info",plugins:n,url:this.window.location.href})};return e}()}).call(this)},{"./connector":1,"./options":5,"./protocol":6,"./reloader":7,"./timer":9}],5:[function(e,t,n){(function(){var e;n.Options=e=function(){function e(){this.https=false;this.host=null;this.port=35729;this.snipver=null;this.ext=null;this.extver=null;this.mindelay=1e3;this.maxdelay=6e4;this.handshake_timeout=5e3}e.prototype.set=function(e,t){if(typeof t==="undefined"){return}if(!isNaN(+t)){t=+t}return this[e]=t};return e}();e.extract=function(t){var n,o,i,r,s,l,c,a,h,u,d,f,p;f=t.getElementsByTagName("script");for(a=0,u=f.length;a<u;a++){n=f[a];if((c=n.src)&&(i=c.match(/^[^:]+:\/\/(.*)\/z?livereload\.js(?:\?(.*))?$/))){s=new e;s.https=c.indexOf("https")===0;if(r=i[1].match(/^([^\/:]+)(?::(\d+))?$/)){s.host=r[1];if(r[2]){s.port=parseInt(r[2],10)}}if(i[2]){p=i[2].split("&");for(h=0,d=p.length;h<d;h++){l=p[h];if((o=l.split("=")).length>1){s.set(o[0].replace(/-/g,"_"),o.slice(1).join("="))}}}return s}}return null}}).call(this)},{}],6:[function(e,t,n){(function(){var e,t,o,i,r=[].indexOf||function(e){for(var t=0,n=this.length;t<n;t++){if(t in this&&this[t]===e)return t}return-1};n.PROTOCOL_6=e="http://livereload.com/protocols/official-6";n.PROTOCOL_7=t="http://livereload.com/protocols/official-7";n.ProtocolError=i=function(){function e(e,t){this.message="LiveReload protocol error ("+e+') after receiving data: "'+t+'".'}return e}();n.Parser=o=function(){function n(e){this.handlers=e;this.reset()}n.prototype.reset=function(){return this.protocol=null};n.prototype.process=function(n){var o,s,l,c,a;try{if(this.protocol==null){if(n.match(/^!!ver:([\d.]+)$/)){this.protocol=6}else if(l=this._parseMessage(n,["hello"])){if(!l.protocols.length){throw new i("no protocols specified in handshake message")}else if(r.call(l.protocols,t)>=0){this.protocol=7}else if(r.call(l.protocols,e)>=0){this.protocol=6}else{throw new i("no supported protocols found")}}return this.handlers.connected(this.protocol)}else if(this.protocol===6){l=JSON.parse(n);if(!l.length){throw new i("protocol 6 messages must be arrays")}o=l[0],c=l[1];if(o!=="refresh"){throw new i("unknown protocol 6 command")}return this.handlers.message({command:"reload",path:c.path,liveCSS:(a=c.apply_css_live)!=null?a:true})}else{l=this._parseMessage(n,["reload","alert"]);return this.handlers.message(l)}}catch(e){s=e;if(s instanceof i){return this.handlers.error(s)}else{throw s}}};n.prototype._parseMessage=function(e,t){var n,o,s;try{o=JSON.parse(e)}catch(t){n=t;throw new i("unparsable JSON",e)}if(!o.command){throw new i('missing "command" key',e)}if(s=o.command,r.call(t,s)<0){throw new i("invalid command '"+o.command+"', only valid commands are: "+t.join(", ")+")",e)}return o};return n}()}).call(this)},{}],7:[function(e,t,n){(function(){var e,t,o,i,r,s,l;l=function(e){var t,n,o;if((n=e.indexOf("#"))>=0){t=e.slice(n);e=e.slice(0,n)}else{t=""}if((n=e.indexOf("?"))>=0){o=e.slice(n);e=e.slice(0,n)}else{o=""}return{url:e,params:o,hash:t}};i=function(e){var t;e=l(e).url;if(e.indexOf("file://")===0){t=e.replace(/^file:\/\/(localhost)?/,"")}else{t=e.replace(/^([^:]+:)?\/\/([^:\/]+)(:\d*)?\//,"/")}return decodeURIComponent(t)};s=function(e,t,n){var i,r,s,l,c;i={score:0};for(l=0,c=t.length;l<c;l++){r=t[l];s=o(e,n(r));if(s>i.score){i={object:r,score:s}}}if(i.score>0){return i}else{return null}};o=function(e,t){var n,o,i,r;e=e.replace(/^\/+/,"").toLowerCase();t=t.replace(/^\/+/,"").toLowerCase();if(e===t){return 1e4}n=e.split("/").reverse();o=t.split("/").reverse();r=Math.min(n.length,o.length);i=0;while(i<r&&n[i]===o[i]){++i}return i};r=function(e,t){return o(e,t)>0};e=[{selector:"background",styleNames:["backgroundImage"]},{selector:"border",styleNames:["borderImage","webkitBorderImage","MozBorderImage"]}];n.Reloader=t=function(){function t(e,t,n){this.window=e;this.console=t;this.Timer=n;this.document=this.window.document;this.importCacheWaitPeriod=200;this.plugins=[]}t.prototype.addPlugin=function(e){return this.plugins.push(e)};t.prototype.analyze=function(e){return results};t.prototype.reload=function(e,t){var n,o,i,r,s;this.options=t;if((o=this.options).stylesheetReloadTimeout==null){o.stylesheetReloadTimeout=15e3}s=this.plugins;for(i=0,r=s.length;i<r;i++){n=s[i];if(n.reload&&n.reload(e,t)){return}}if(t.liveCSS){if(e.match(/\.css$/i)){if(this.reloadStylesheet(e)){return}}}if(t.liveImg){if(e.match(/\.(jpe?g|png|gif)$/i)){this.reloadImages(e);return}}return this.reloadPage()};t.prototype.reloadPage=function(){return this.window.document.location.reload()};t.prototype.reloadImages=function(t){var n,o,s,l,c,a,h,u,d,f,p,m,g,v,y,w,R,_;n=this.generateUniqueString();v=this.document.images;for(a=0,f=v.length;a<f;a++){o=v[a];if(r(t,i(o.src))){o.src=this.generateCacheBustUrl(o.src,n)}}if(this.document.querySelectorAll){for(h=0,p=e.length;h<p;h++){y=e[h],s=y.selector,l=y.styleNames;w=this.document.querySelectorAll("[style*="+s+"]");for(u=0,m=w.length;u<m;u++){o=w[u];this.reloadStyleImages(o.style,l,t,n)}}}if(this.document.styleSheets){R=this.document.styleSheets;_=[];for(d=0,g=R.length;d<g;d++){c=R[d];_.push(this.reloadStylesheetImages(c,t,n))}return _}};t.prototype.reloadStylesheetImages=function(t,n,o){var i,r,s,l,c,a,h,u;try{s=t!=null?t.cssRules:void 0}catch(e){i=e}if(!s){return}for(c=0,h=s.length;c<h;c++){r=s[c];switch(r.type){case CSSRule.IMPORT_RULE:this.reloadStylesheetImages(r.styleSheet,n,o);break;case CSSRule.STYLE_RULE:for(a=0,u=e.length;a<u;a++){l=e[a].styleNames;this.reloadStyleImages(r.style,l,n,o)}break;case CSSRule.MEDIA_RULE:this.reloadStylesheetImages(r,n,o)}}};t.prototype.reloadStyleImages=function(e,t,n,o){var s,l,c,a,h;for(a=0,h=t.length;a<h;a++){l=t[a];c=e[l];if(typeof c==="string"){s=c.replace(/\burl\s*\(([^)]*)\)/,function(e){return function(t,s){if(r(n,i(s))){return"url("+e.generateCacheBustUrl(s,o)+")"}else{return t}}}(this));if(s!==c){e[l]=s}}}};t.prototype.reloadStylesheet=function(e){var t,n,o,r,l,c,a,h,u,d,f,p,m,g,v;o=function(){var e,t,o,i;o=this.document.getElementsByTagName("link");i=[];for(e=0,t=o.length;e<t;e++){n=o[e];if(n.rel.match(/^stylesheet$/i)&&!n.__LiveReload_pendingRemoval){i.push(n)}}return i}.call(this);t=[];g=this.document.getElementsByTagName("style");for(c=0,d=g.length;c<d;c++){l=g[c];if(l.sheet){this.collectImportedStylesheets(l,l.sheet,t)}}for(a=0,f=o.length;a<f;a++){n=o[a];this.collectImportedStylesheets(n,n.sheet,t)}if(this.window.StyleFix&&this.document.querySelectorAll){v=this.document.querySelectorAll("style[data-href]");for(h=0,p=v.length;h<p;h++){l=v[h];o.push(l)}}this.console.log("LiveReload found "+o.length+" LINKed stylesheets, "+t.length+" @imported stylesheets");r=s(e,o.concat(t),function(e){return function(t){return i(e.linkHref(t))}}(this));if(r){if(r.object.rule){this.console.log("LiveReload is reloading imported stylesheet: "+r.object.href);this.reattachImportedRule(r.object)}else{this.console.log("LiveReload is reloading stylesheet: "+this.linkHref(r.object));this.reattachStylesheetLink(r.object)}}else{this.console.log("LiveReload will reload all stylesheets because path '"+e+"' did not match any specific one");for(u=0,m=o.length;u<m;u++){n=o[u];this.reattachStylesheetLink(n)}}return true};t.prototype.collectImportedStylesheets=function(e,t,n){var o,i,r,s,l,c;try{s=t!=null?t.cssRules:void 0}catch(e){o=e}if(s&&s.length){for(i=l=0,c=s.length;l<c;i=++l){r=s[i];switch(r.type){case CSSRule.CHARSET_RULE:continue;case CSSRule.IMPORT_RULE:n.push({link:e,rule:r,index:i,href:r.href});this.collectImportedStylesheets(e,r.styleSheet,n);break;default:break}}}};t.prototype.waitUntilCssLoads=function(e,t){var n,o,i;n=false;o=function(e){return function(){if(n){return}n=true;return t()}}(this);e.onload=function(e){return function(){e.console.log("LiveReload: the new stylesheet has finished loading");e.knownToSupportCssOnLoad=true;return o()}}(this);if(!this.knownToSupportCssOnLoad){(i=function(t){return function(){if(e.sheet){t.console.log("LiveReload is polling until the new CSS finishes loading...");return o()}else{return t.Timer.start(50,i)}}}(this))()}return this.Timer.start(this.options.stylesheetReloadTimeout,o)};t.prototype.linkHref=function(e){return e.href||e.getAttribute("data-href")};t.prototype.reattachStylesheetLink=function(e){var t,n;if(e.__LiveReload_pendingRemoval){return}e.__LiveReload_pendingRemoval=true;if(e.tagName==="STYLE"){t=this.document.createElement("link");t.rel="stylesheet";t.media=e.media;t.disabled=e.disabled}else{t=e.cloneNode(false)}t.href=this.generateCacheBustUrl(this.linkHref(e));n=e.parentNode;if(n.lastChild===e){n.appendChild(t)}else{n.insertBefore(t,e.nextSibling)}return this.waitUntilCssLoads(t,function(n){return function(){var o;if(/AppleWebKit/.test(navigator.userAgent)){o=5}else{o=200}return n.Timer.start(o,function(){var o;if(!e.parentNode){return}e.parentNode.removeChild(e);t.onreadystatechange=null;return(o=n.window.StyleFix)!=null?o.link(t):void 0})}}(this))};t.prototype.reattachImportedRule=function(e){var t,n,o,i,r,s,l,c;l=e.rule,n=e.index,o=e.link;s=l.parentStyleSheet;t=this.generateCacheBustUrl(l.href);i=l.media.length?[].join.call(l.media,", "):"";r='@import url("'+t+'") '+i+";";l.__LiveReload_newHref=t;c=this.document.createElement("link");c.rel="stylesheet";c.href=t;c.__LiveReload_pendingRemoval=true;if(o.parentNode){o.parentNode.insertBefore(c,o)}return this.Timer.start(this.importCacheWaitPeriod,function(e){return function(){if(c.parentNode){c.parentNode.removeChild(c)}if(l.__LiveReload_newHref!==t){return}s.insertRule(r,n);s.deleteRule(n+1);l=s.cssRules[n];l.__LiveReload_newHref=t;return e.Timer.start(e.importCacheWaitPeriod,function(){if(l.__LiveReload_newHref!==t){return}s.insertRule(r,n);return s.deleteRule(n+1)})}}(this))};t.prototype.generateUniqueString=function(){return"livereload="+Date.now()};t.prototype.generateCacheBustUrl=function(e,t){var n,o,i,r,s;if(t==null){t=this.generateUniqueString()}s=l(e),e=s.url,n=s.hash,o=s.params;if(this.options.overrideURL){if(e.indexOf(this.options.serverURL)<0){i=e;e=this.options.serverURL+this.options.overrideURL+"?url="+encodeURIComponent(e);this.console.log("LiveReload is overriding source URL "+i+" with "+e)}}r=o.replace(/(\?|&)livereload=(\d+)/,function(e,n){return""+n+t});if(r===o){if(o.length===0){r="?"+t}else{r=""+o+"&"+t}}return e+r+n};return t}()}).call(this)},{}],8:[function(e,t,n){(function(){var t,n,o;t=e("./customevents");n=window.LiveReload=new(e("./livereload").LiveReload)(window);for(o in window){if(o.match(/^LiveReloadPlugin/)){n.addPlugin(window[o])}}n.addPlugin(e("./less"));n.on("shutdown",function(){return delete window.LiveReload});n.on("connect",function(){return t.fire(document,"LiveReloadConnect")});n.on("disconnect",function(){return t.fire(document,"LiveReloadDisconnect")});t.bind(document,"LiveReloadShutDown",function(){return n.shutDown()})}).call(this)},{"./customevents":2,"./less":3,"./livereload":4}],9:[function(e,t,n){(function(){var e;n.Timer=e=function(){function e(e){this.func=e;this.running=false;this.id=null;this._handler=function(e){return function(){e.running=false;e.id=null;return e.func()}}(this)}e.prototype.start=function(e){if(this.running){clearTimeout(this.id)}this.id=setTimeout(this._handler,e);return this.running=true};e.prototype.stop=function(){if(this.running){clearTimeout(this.id);this.running=false;return this.id=null}};return e}();e.start=function(e,t){return setTimeout(t,e)}}).call(this)},{}]},{},[8]);`
	hugoLiveReloadPlugin = fmt.Sprintf(`
/*
Hugo adds a specific prefix, "__hugo_navigate", to the path in certain situations to signal
navigation to another content page.
*/

function HugoReload() {}

HugoReload.identifier = 'hugoReloader';
HugoReload.version = '0.9';

HugoReload.prototype.reload = function(path, options) {
	var prefix = %q;

	if (path.lastIndexOf(prefix, 0) !== 0) {
		return false
	}
	
	path = path.substring(prefix.length);

	var portChanged = options.overrideURL && options.overrideURL != window.location.port
	
	if (!portChanged && window.location.pathname === path) {
		window.location.reload();
	} else {
		if (portChanged) {
			window.location = location.protocol + "//" + location.hostname + ":" + options.overrideURL + path;
		} else {
			window.location.pathname = path;
		}
	}

	return true;
};

LiveReload.addPlugin(HugoReload)
`, hugoNavigatePrefix)
)
