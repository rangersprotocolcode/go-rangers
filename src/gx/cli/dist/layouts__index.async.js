"use strict";(self.webpackChunk=self.webpackChunk||[]).push([[717],{23055:function(w,A,a){a.d(A,{Z:function(){return y}});var B=a(9669),u=a.n(B),x=function(){var o=window.location.port;return Number(o)+2e3},m=function(){return window.location.protocol+"//"+window.location.hostname+":"},d=m()+x()+"/api";console.log(d);var f=function(o){return Promise.resolve(o)};function y(n){var o=n.data,r=n.method,e=r===void 0?"get":r,F=n.responseType,H=F===void 0?null:F;return n.url=d,n.method=e,n.method.toUpperCase()==="GET"?(n.params=o||{},delete n.data):n.data=o||{},u()(n).then(function(i){var l=i.data,s=l.result,g={status:s&&s.status,success:s&&s.status===0,data:s&&s.data,message:s&&s.message};return s&&s.status===0||n.skipError||!(s&&s.status!==0)&&n.customError?f(g):Promise.reject(g)}).catch(function(i){var l=i.response,s={};if(l&&l instanceof Object){var g=l.data,P=l.statusText;s=g,s.status=l.status,s.message=g.message||P}else s.status=600,s.message=i.message||"Network Error";return console.log(i),Promise.reject(s)})}},52345:function(w,A,a){a.d(A,{Z:function(){return n}});var B=a(99165),u=a(70647),x=a(62435),m={"paragraph-copy":"paragraph-copy___t0NqJ",iconCopy:"iconCopy___d7iW5"},d=a(86074),f=u.Z.Paragraph,y=function(r){var e=r.text,F=r.tooltips,H=r.children,i=r.iconColor;return(0,d.jsx)("div",{className:"paragraph-copy",children:(0,d.jsx)(f,{copyable:{text:e,tooltips:[F||"Copy"],icon:(0,d.jsx)(B.Z,{className:m.iconCopy,style:{color:i}})},children:H})})},n=y},3025:function(w,A,a){a.r(A),a.d(A,{default:function(){return R}});var B=a(17061),u=a.n(B),x=a(17156),m=a.n(x),d=a(27424),f=a.n(d),y=a(52345),n=a(28070),o=a(62435),r={headers:"headers___FDPD7",header:"header___vLq0P",logoBox:"logoBox___L63Wv",logo:"logo___GXqbR",name:"name___AhPba",nodeId:"nodeId___XEK5r",address:"address___Kr7Rj"},e=a(86074),F=function(j){var t=(0,o.useContext)(n.R0),C=(t==null?void 0:t.miner)&&JSON.parse(t==null?void 0:t.miner).id;return(0,e.jsx)("div",{className:r.headers,children:(0,e.jsxs)("div",{className:r.header,children:[(0,e.jsxs)("div",{className:r.logoBox,children:[(0,e.jsx)("img",{src:a(66949),className:r.logo}),(0,e.jsx)("h1",{className:r.name,children:"Rangers Node"})]}),(0,e.jsxs)("div",{className:r.nodeId,children:["Node ID: \xA0",(0,e.jsx)(y.Z,{text:C,tooltips:C,iconColor:"#fff",children:(0,e.jsxs)("span",{className:r.address,children:[C,"\xA0"]})})]})]})})},H=F,i=a(99870),l={tabsBox:"tabsBox___Krq71",tabs:"tabs___uDhIn",list:"list___gr_jp",item:"item___JtK5u",active:"active___lOoKj"},s=function(j){var t=(0,i.UO)(),C=(0,o.useState)(i.m8.location.pathname),N=f()(C,2),T=N[0],b=N[1];return(0,o.useEffect)(function(){if(Object.keys(t).length>0){var h="/",v=i.m8.location.pathname;for(var k in t)v=v.replace(h+t[k],"");b(v)}},[i.m8.location.pathname]),(0,e.jsx)("div",{className:l.tabsBox,children:(0,e.jsx)("div",{className:l.tabs,children:(0,e.jsx)("ul",{className:l.list,children:n.dr.map(function(h,v){return(0,e.jsx)("li",{className:"".concat(l.item," ").concat(h.value.includes(T)?l.active:""),children:(0,e.jsx)("a",{href:h.value[0],children:h.name})},h.key)})})})})},g=s,P=a(23055);function U(){return E.apply(this,arguments)}function E(){return E=m()(u()().mark(function p(){return u()().wrap(function(t){for(;;)switch(t.prev=t.next){case 0:return t.abrupt("return",(0,P.Z)({method:"post",data:{method:"Rocket_dashboard",jsonrpc:"2.0",id:1,params:[]}}));case 1:case"end":return t.stop()}},p)})),E.apply(this,arguments)}var I={minersBox:"minersBox___PL3gC",miners:"miners___wUFLP",minersRight:"minersRight___YHYvO"};function R(){var p=(0,o.useState)(),j=f()(p,2),t=j[0],C=j[1],N=(0,o.useRef)();(0,o.useEffect)(function(){return N.current=setInterval(function(){T()},2e3),function(){clearInterval(N.current)}},[]);var T=function(){var b=m()(u()().mark(function h(){var v;return u()().wrap(function(c){for(;;)switch(c.prev=c.next){case 0:return c.prev=0,c.next=3,U();case 3:v=c.sent,C(v==null?void 0:v.data),c.next=10;break;case 7:c.prev=7,c.t0=c.catch(0),console.log(c.t0);case 10:case"end":return c.stop()}},h,null,[[0,7]])}));return function(){return b.apply(this,arguments)}}();return(0,e.jsx)("div",{className:I.minersBox,children:(0,e.jsxs)(n.R0.Provider,{value:t,children:[(0,e.jsx)(H,{}),(0,e.jsxs)("div",{className:I.miners,children:[(0,e.jsx)(g,{}),(0,e.jsx)("div",{className:I.minersRight,children:(0,e.jsx)(i.j3,{})})]})]})})}},28070:function(w,A,a){a.d(A,{R0:function(){return d},UZ:function(){return m},b8:function(){return x},dr:function(){return u}});var B=a(62435),u=[{key:"Dashboard",name:"Dashboard",value:["/index","/","/blockDetail"]},{key:"MinerInfo",name:"Miner Info",value:["/minerInfo"]}],x=[{key:"BlockChain",name:"Block Chain",value:"blockChain"},{key:"GroupChain",name:"Group Chain",value:"groupChain"}],m=[{key:"BlockRewards",name:"Block Rewards",value:"blockRewards"},{key:"TransactionsDetails",name:"Transactions Details",value:"transactionsDetails"}],d=B.createContext({})},66949:function(w){w.exports="data:image/png;base64,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"}}]);
