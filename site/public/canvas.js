"use strict";
// aos, drawn — and clickable.
//
// Warm off-white plate, hairline cards, one orange accent, one travelling blue
// dot, no gradients and no glow. Everything is laid out in a virtual 1600x1000
// space and scaled to whatever viewport it lands in.
//
// Two things this diagram is built to say, in order:
//   1. an agent drives this the way a person does, because it installs a skill
//      and runs the CLI. MCP is the second way in, not the first.
//   2. the surface is not an abstraction — every group is here, and clicking
//      one lists the real commands, straight from the generated registry.

const VW = 1600, VH = 1000;

// One palette per scheme, taken from the docs theme tokens so the landing page
// and the documentation read as the same product. Dark is the default there, so
// it is the default here; the page follows the reader's system setting rather
// than insisting on one look.
const PALETTES = {
  dark: {
    page:"#0f0e0c", plate:"#161512", card:"#1f1e1b", panel:"#1b1a17",
    ink:"#f4f3ef", muted:"#8a857c", dim:"#a8a299", faint:"#6f6a61",
    hair:"#2b2926", hair2:"#302d29", panelEdge:"#2b2926", rule:"#2b2926",
    num:"#514c44", bullet:"#4f4a42", subtle:"#262420",
    codeBg:"#0b0a09", codeInk:"#e6e2d9", codeInk2:"#c4bfb4",
    acc:"#e08a1e", accEdge:"#6b4a15", accInk:"#e8a44f", accInk2:"#c08f4a",
    accNote:"#a8895c", accFill:"#241a0b", accFill2:"#241a0b", accFill3:"#2e2210",
    surfaceEdge:"#5c4318", surfaceGlow:"rgba(224,138,30,.18)",
    chipInk:"#c4bfb4", routeInk:"#e6e2d9",
    hover:"#262420", hoverEdge:"#3d3a35",
    edge:"#3a352c", thin:"#2f2b25", arrow:"#5a544a", hot:"#f2c489",
    dotRGB:"232,164,79", tick:"#7fd6a0",
    statusBg:"#242220", sep:"#575249", green:"#7fd6a0", amber:"#e0a955",
    shadow:"rgba(0,0,0,.55)", shadowSoft:"rgba(0,0,0,.45)", shadowBar:"rgba(0,0,0,.5)",
  },
  light: {
    page:"#f4f3ef", plate:"#fbfaf8", card:"#ffffff", panel:"#fbfaf7",
    ink:"#1c1b19", muted:"#8a857c", dim:"#6f6a61", faint:"#b0aa9e",
    hair:"#d6d2c8", hair2:"#e6e2d9", panelEdge:"#e2ddd2", rule:"#eeebe4",
    num:"#b8b3a8", bullet:"#d9d4c9", subtle:"#f3f1ec",
    codeBg:"#151412", codeInk:"#e6e2d9", codeInk2:"#4a463f",
    acc:"#e08a1e", accEdge:"#e3b877", accInk:"#8a5a12", accInk2:"#a07a3a",
    accNote:"#8a7a5a", accFill:"#fbf4e8", accFill2:"#fdf6ea", accFill3:"#fdf3e3",
    surfaceEdge:"#c3c9ee", surfaceGlow:"rgba(74,92,208,.16)",
    chipInk:"#5e594f", routeInk:"#3d3a34",
    hover:"#f7f5f0", hoverEdge:"#c9c4b8",
    edge:"#cfcabf", thin:"#ddd8cd", arrow:"#b6b0a4", hot:"#b9c0ea",
    dotRGB:"74,92,208", tick:"#9aa2dd",
    statusBg:"#151412", sep:"#5a564e", green:"#7fd6a0", amber:"#e0a955",
    shadow:"rgba(30,28,24,.10)", shadowSoft:"rgba(30,28,24,.08)", shadowBar:"rgba(0,0,0,.22)",
  },
};

const darkQuery = matchMedia("(prefers-color-scheme: dark)");
let P = PALETTES[darkQuery.matches ? "dark" : "light"];

const SANS = '-apple-system,BlinkMacSystemFont,"Segoe UI",Inter,Helvetica,Arial,sans-serif';
const MONO = 'ui-monospace,SFMono-Regular,Menlo,Consolas,monospace';

// The registry snapshot, injected by the page. Generated from the binary, so
// the diagram cannot claim a command that does not exist. The fallback keeps
// the file readable on its own.
const DATA = Object.assign({commands:0, groups:0, binary:"aos", groupList:[]}, window.AOS_STATS || {});
const BIN = DATA.binary || "aos";
const GROUPS = DATA.groupList || [];
const SURFACE_LABEL = `${DATA.commands} commands · ${DATA.groups} groups`;

const cv = document.getElementById("c"), ctx = cv.getContext("2d", {alpha:false});
let SC = 1, OX = 0, OY = 0;

function resize(){
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  const w = innerWidth, h = innerHeight;
  cv.width = Math.floor(w*dpr); cv.height = Math.floor(h*dpr);
  SC = Math.min(w/VW, h/VH);
  OX = (w - VW*SC)/2; OY = (h - VH*SC)/2;
  ctx.setTransform(dpr,0,0,dpr,0,0);
}
addEventListener("resize", resize); resize();

// ---------------------------------------------------------------- helpers
const V = (x,y)=>({x,y});
function T(p){ return V(OX + p.x*SC, OY + p.y*SC); }
const S = v => v*SC;
// screen -> virtual, so a click can be tested against the layout
function inv(px,py){ return V((px-OX)/SC, (py-OY)/SC); }
function hit(b,p){ return p.x>=b.x && p.x<=b.x+b.w && p.y>=b.y && p.y<=b.y+b.h; }

function roundRect(x,y,w,h,r){
  ctx.beginPath();
  ctx.moveTo(x+r,y); ctx.lineTo(x+w-r,y); ctx.quadraticCurveTo(x+w,y,x+w,y+r);
  ctx.lineTo(x+w,y+h-r); ctx.quadraticCurveTo(x+w,y+h,x+w-r,y+h);
  ctx.lineTo(x+r,y+h); ctx.quadraticCurveTo(x,y+h,x,y+h-r);
  ctx.lineTo(x,y+r); ctx.quadraticCurveTo(x,y,x+r,y); ctx.closePath();
}
function text(str,x,y,{size=14,weight=400,color=P.ink,font=SANS,align="left",spacing=0}={}){
  const p=T(V(x,y));
  ctx.font = `${weight} ${S(size)}px ${font}`;
  ctx.fillStyle=color; ctx.textAlign=align; ctx.textBaseline="alphabetic";
  if(spacing){
    let cx=p.x;
    const total=[...str].reduce((a,ch)=>a+ctx.measureText(ch).width+S(spacing),0)-S(spacing);
    if(align==="center") cx=p.x-total/2; if(align==="right") cx=p.x-total;
    ctx.textAlign="left";
    for(const ch of str){ ctx.fillText(ch,cx,p.y); cx+=ctx.measureText(ch).width+S(spacing); }
  } else ctx.fillText(str,p.x,p.y);
}
function measure(str,size,weight=400,font=SANS){
  ctx.font=`${weight} ${S(size)}px ${font}`; return ctx.measureText(str).width/SC;
}
function ellipsize(str,max,size,weight=400,font=SANS){
  if(measure(str,size,weight,font) <= max) return str;
  let s = str;
  while(s.length > 1 && measure(s+"…",size,weight,font) > max) s = s.slice(0,-1);
  return s+"…";
}
function bez(p0,p1,p2,p3,t){
  const u=1-t, a=u*u*u, b=3*u*u*t, c=3*u*t*t, d=t*t*t;
  return V(a*p0.x+b*p1.x+c*p2.x+d*p3.x, a*p0.y+b*p1.y+c*p2.y+d*p3.y);
}
function bezT(p0,p1,p2,p3,t){
  const u=1-t;
  const x=3*u*u*(p1.x-p0.x)+6*u*t*(p2.x-p1.x)+3*t*t*(p3.x-p2.x);
  const y=3*u*u*(p1.y-p0.y)+6*u*t*(p2.y-p1.y)+3*t*t*(p3.y-p2.y);
  const m=Math.hypot(x,y)||1; return V(x/m,y/m);
}

// ---------------------------------------------------------------- the model
// Two callers. The agent is not a different kind of client: it installs the
// skill and runs the same commands, which is why both arrive at one surface.
const callers = [
  {id:"human", n:"01", x:60, y:292, w:196, h:104, icon:"›_", title:"A person",
   detail:"at a terminal", edge:"TYPES"},
  {id:"agent", n:"02", x:60, y:424, w:196, h:104, icon:"✦", title:"An agent",
   detail:"skill installed", edge:"RUNS"},
];

const surface   = {x:348, y:272, w:244, h:272};
const ways      = {x:622, y:272, w:554, h:272};
const platforms = [
  {name:"macOS",   sub:"CoreGraphics · osascript",  y:272},
  {name:"Windows", sub:"Win32 · PowerShell",        y:368},
  {name:"Linux",   sub:"wmctrl · xdotool · Xvfb",   y:464},
];
const PX = 1216, PW = 324, PH = 84;

// Band B: every group, and the commands inside the one you pick.
const grid   = {x:60, y:658, cols:6, cw:150, ch:34, gx:12, gy:10};
const detail = {x:1050, y:634, w:490, h:296};

function chipBox(i){
  const c = i % grid.cols, r = (i - c) / grid.cols;
  return {x: grid.x + c*(grid.cw+grid.gx), y: grid.y + r*(grid.ch+grid.gy),
          w: grid.cw, h: grid.ch};
}

// ---------------------------------------------------------------- selection
// It cycles on its own so the page is alive on arrival, and stops the moment
// someone clicks: an autoplay that fights the user is worse than no autoplay.
let selected = Math.max(0, GROUPS.findIndex(g => g.name === "window"));
let pinned = false;
let hovered = -1;

function autoSelect(t){
  if(pinned || !GROUPS.length) return;
  selected = Math.floor(t/2.6) % GROUPS.length;
}

function pointFromEvent(e){
  const r = cv.getBoundingClientRect();
  return inv(e.clientX - r.left, e.clientY - r.top);
}
cv.addEventListener("mousemove", e => {
  const p = pointFromEvent(e);
  hovered = -1;
  for(let i=0;i<GROUPS.length;i++){ if(hit(chipBox(i),p)){ hovered = i; break; } }
  cv.style.cursor = hovered >= 0 ? "pointer" : "default";
});
cv.addEventListener("mouseleave", () => { hovered = -1; });
cv.addEventListener("click", e => {
  const p = pointFromEvent(e);
  for(let i=0;i<GROUPS.length;i++){
    if(hit(chipBox(i),p)){
      // clicking the pinned group again releases it and resumes the tour
      if(pinned && selected === i){ pinned = false; }
      else { selected = i; pinned = true; }
      if(still) draw(2.0);
      return;
    }
  }
});
// Keyboard is not decoration here: the grid is the navigation, so it should
// work without a pointer.
addEventListener("keydown", e => {
  if(!GROUPS.length) return;
  const step = {ArrowRight:1, ArrowLeft:-1, ArrowDown:grid.cols, ArrowUp:-grid.cols}[e.key];
  if(step === undefined){
    if(e.key === "Escape"){ pinned = false; if(still) draw(2.0); }
    return;
  }
  e.preventDefault();
  pinned = true;
  selected = (selected + step + GROUPS.length) % GROUPS.length;
  if(still) draw(2.0);
});

// ---------------------------------------------------------------- anchors
function ctrl(p,q,bend){
  const dx=q.x-p.x, dy=q.y-p.y, len=Math.hypot(dx,dy)||1;
  const nx=-dy/len, ny=dx/len, off=len*bend;
  return [V(p.x+dx*0.32+nx*off, p.y+dy*0.32+ny*off),
          V(p.x+dx*0.68+nx*off, p.y+dy*0.68+ny*off)];
}
const activeCaller = t => Math.floor(t/3.2) % 2;

// ---------------------------------------------------------------- edges
function edgeList(t){
  const a = activeCaller(t);
  const es = [];

  callers.forEach((c,i)=>{
    es.push({
      p0: V(c.x+c.w, c.y+c.h/2),
      p3: V(surface.x, surface.y + surface.h*(i===0?0.30:0.70)),
      label: c.edge, bend: 0, dots: 2, hot: a===i,
    });
  });

  // the surface, said two ways
  es.push({p0: V(surface.x+surface.w, surface.y+surface.h*0.34),
           p3: V(ways.x, ways.y+92), label:null, bend:0, dots:1});
  es.push({p0: V(surface.x+surface.w, surface.y+surface.h*0.66),
           p3: V(ways.x, ways.y+186), label:null, bend:0, dots:1, thin:true});

  // one backend per platform
  platforms.forEach((p,i)=>{
    es.push({
      p0: V(ways.x+ways.w, ways.y + ways.h*(0.22 + 0.28*i)),
      p3: V(PX, p.y + PH/2),
      label: null, bend: i===0 ? -0.08 : (i===2 ? 0.08 : 0), dots: 1, lo:-22,
    });
  });

  // the surface down into the full command grid — the point being that the
  // band below is the same surface, enumerated
  es.push({p0: V(surface.x+surface.w/2, surface.y+surface.h),
           p3: V(grid.x+180, grid.y-26), label:null, bend:0.08, dots:2, thin:true});

  return es;
}

// ---------------------------------------------------------------- drawing
function drawCard(c, hot){
  const p=T(V(c.x,c.y));
  ctx.save();
  ctx.shadowColor=P.shadow; ctx.shadowBlur=S(18); ctx.shadowOffsetY=S(5);
  ctx.fillStyle=P.card; roundRect(p.x,p.y,S(c.w),S(c.h),S(12)); ctx.fill();
  ctx.restore();
  ctx.strokeStyle=hot?P.hot:P.hair; ctx.lineWidth=Math.max(1,S(hot?1.8:1));
  roundRect(p.x,p.y,S(c.w),S(c.h),S(12)); ctx.stroke();

  text(c.n, c.x+18, c.y+34, {size:19, weight:300, color:P.num, font:MONO});

  const ip=T(V(c.x+18,c.y+48));
  ctx.fillStyle=P.subtle; roundRect(ip.x,ip.y,S(26),S(26),S(7)); ctx.fill();
  ctx.strokeStyle=P.hair2; ctx.lineWidth=Math.max(1,S(1));
  roundRect(ip.x,ip.y,S(26),S(26),S(7)); ctx.stroke();
  text(c.icon, c.x+31, c.y+66, {size:13, color:P.dim, align:"center", font:MONO});

  text(c.title, c.x+54, c.y+67, {size:16.5, weight:700});
  text(c.detail, c.x+18, c.y+c.h-16, {size:12, color:P.muted});
}

function drawSurface(){
  const p=T(V(surface.x,surface.y));
  ctx.save();
  ctx.shadowColor=P.surfaceGlow; ctx.shadowBlur=S(30); ctx.shadowOffsetY=S(6);
  ctx.fillStyle=P.card; roundRect(p.x,p.y,S(surface.w),S(surface.h),S(14)); ctx.fill();
  ctx.restore();
  ctx.strokeStyle=P.surfaceEdge; ctx.lineWidth=S(3);
  roundRect(p.x,p.y,S(surface.w),S(surface.h),S(14)); ctx.stroke();

  const cx = surface.x + surface.w/2;
  text("03", surface.x+20, surface.y+38, {size:19, weight:300, color:P.num, font:MONO});
  text("One command surface", cx, surface.y+84, {size:18, weight:700, align:"center"});
  text(SURFACE_LABEL, cx, surface.y+106, {size:12, color:P.muted, align:"center"});

  const bw = surface.w-40, ry = surface.y+126;
  const r=T(V(surface.x+20,ry));
  ctx.fillStyle=P.accFill; roundRect(r.x,r.y,S(bw),S(50),S(9)); ctx.fill();
  ctx.strokeStyle=P.accEdge; ctx.lineWidth=Math.max(1,S(1.4));
  roundRect(r.x,r.y,S(bw),S(50),S(9)); ctx.stroke();
  text("one Runner, one code path", cx, ry+21, {size:12.5, weight:700, color:P.accInk, align:"center"});
  text("no second implementation", cx, ry+38, {size:11, color:P.accInk2, align:"center"});

  const gates = ["no display? refused, with a reason",
                 "delete a root or $HOME? refused",
                 "serve is token-gated, on loopback"];
  gates.forEach((g,i)=>{
    const y = surface.y+198+i*20;
    const d=T(V(surface.x+22,y-4));
    ctx.fillStyle=P.bullet; ctx.beginPath(); ctx.arc(d.x,d.y,S(2.4),0,7); ctx.fill();
    text(ellipsize(g, bw-22, 11), surface.x+34, y, {size:11, color:P.dim});
  });

  text("SAFETY IS STRUCTURE", cx, surface.y+surface.h-14,
    {size:9, color:P.muted, align:"center", weight:700, spacing:1.4});
}

// The heart of the reordering: the skill is the headline path, MCP is the
// footnote. Both show the same call so the claim above stays legible.
function drawWays(t){
  const p=T(V(ways.x,ways.y));
  ctx.fillStyle=P.panel; roundRect(p.x,p.y,S(ways.w),S(ways.h),S(14)); ctx.fill();
  ctx.strokeStyle=P.panelEdge; ctx.lineWidth=Math.max(1,S(1));
  roundRect(p.x,p.y,S(ways.w),S(ways.h),S(14)); ctx.stroke();

  text("two ways in", ways.x+22, ways.y+36, {size:16.5, weight:700});
  text("the skill first · MCP second", ways.x+ways.w-22, ways.y+36,
    {size:11.5, color:P.muted, align:"right"});

  const bw = ways.w-44;

  // --- primary: the agent skill
  const ay = ways.y+56;
  const a=T(V(ways.x+22,ay));
  ctx.fillStyle=P.accFill2; roundRect(a.x,a.y,S(bw),S(104),S(10)); ctx.fill();
  ctx.strokeStyle=P.accEdge; ctx.lineWidth=Math.max(1,S(1.6));
  roundRect(a.x,a.y,S(bw),S(104),S(10)); ctx.stroke();

  const pill=T(V(ways.x+36,ay+14));
  const pw = measure("PRIMARY",9,700,MONO)+18;
  ctx.fillStyle=P.acc; roundRect(pill.x,pill.y,S(pw),S(17),S(8.5)); ctx.fill();
  text("PRIMARY", ways.x+36+pw/2, ay+26, {size:9, color:"#fff", font:MONO, weight:700, align:"center", spacing:1});
  text("agent skill", ways.x+42+pw+10, ay+27, {size:13.5, weight:700, color:P.accInk});
  text(`${BIN} skill install`, ways.x+bw-2, ay+27,
    {size:11.5, color:P.accInk2, font:MONO, align:"right"});

  const cmd=T(V(ways.x+36,ay+40));
  ctx.fillStyle=P.codeBg; roundRect(cmd.x,cmd.y,S(bw-28),S(30),S(7)); ctx.fill();
  text(`$ ${BIN} window move Chrome --zone=1B`, ways.x+48, ay+60,
    {size:12, color:P.codeInk, font:MONO});
  text("nothing running · nothing to connect to · works over ssh", ways.x+36, ay+90,
    {size:11.5, color:P.accNote});

  // --- secondary: MCP
  const my = ways.y+176;
  const m=T(V(ways.x+22,my));
  ctx.fillStyle=P.card; roundRect(m.x,m.y,S(bw),S(74),S(10)); ctx.fill();
  ctx.strokeStyle=P.hair2; ctx.lineWidth=Math.max(1,S(1));
  roundRect(m.x,m.y,S(bw),S(74),S(10)); ctx.stroke();

  text("MCP", ways.x+36, my+24, {size:11, color:P.dim, font:MONO, weight:700, spacing:1.2});
  text("for agents that want typed tools", ways.x+36+measure("MCP",11,700,MONO)+14, my+24,
    {size:11.5, color:P.muted});
  text(`${BIN} serve mcp`, ways.x+bw-2, my+24, {size:11.5, color:P.muted, font:MONO, align:"right"});

  const q=T(V(ways.x+36,my+34));
  ctx.fillStyle=P.subtle; roundRect(q.x,q.y,S(bw-28),S(28),S(7)); ctx.fill();
  text('window_move {"app":"Chrome","zone":"1B"}', ways.x+48, my+53,
    {size:11.5, color:P.codeInk2, font:MONO});
}

function drawPlatforms(){
  platforms.forEach((pf,i)=>{
    const p=T(V(PX,pf.y));
    ctx.save();
    ctx.shadowColor=P.shadowSoft; ctx.shadowBlur=S(14); ctx.shadowOffsetY=S(4);
    ctx.fillStyle=P.card; roundRect(p.x,p.y,S(PW),S(PH),S(12)); ctx.fill();
    ctx.restore();
    ctx.strokeStyle=P.hair; ctx.lineWidth=Math.max(1,S(1));
    roundRect(p.x,p.y,S(PW),S(PH),S(12)); ctx.stroke();

    text(String(i+4).padStart(2,"0"), PX+18, pf.y+28,
      {size:16, weight:300, color:P.num, font:MONO});
    text(pf.name, PX+18, pf.y+58, {size:18, weight:700});
    text(pf.sub, PX+PW-18, pf.y+58, {size:11, color:P.muted, font:MONO, align:"right"});

    const tp=T(V(PX+PW-24, pf.y+24));
    ctx.strokeStyle=P.tick; ctx.lineWidth=S(1.6); ctx.lineCap="round";
    ctx.beginPath();
    ctx.moveTo(tp.x-S(6), tp.y); ctx.lineTo(tp.x-S(2), tp.y+S(5)); ctx.lineTo(tp.x+S(6), tp.y-S(6));
    ctx.stroke(); ctx.lineCap="butt";
  });
  text("ONE BACKEND EACH · VERIFIED ON ALL THREE", PX+PW, platforms[2].y+PH+26,
    {size:9, color:P.muted, align:"right", weight:700, spacing:1.4});
}

// ------------------------------------------------------- band B: every group
function drawGrid(t){
  text("07", grid.x, grid.y-58, {size:19, weight:300, color:P.num, font:MONO});
  text("every command", grid.x+38, grid.y-58, {size:17, weight:700});
  text("click a group — these are the real routes, generated from the binary",
    grid.x+38+measure("every command",17,700)+16, grid.y-58, {size:12, color:P.muted});

  GROUPS.forEach((g,i)=>{
    const b = chipBox(i);
    const on = i === selected, hov = i === hovered;
    const p=T(V(b.x,b.y));
    ctx.fillStyle = on ? P.accFill3 : (hov ? P.hover : P.card);
    roundRect(p.x,p.y,S(b.w),S(b.h),S(8)); ctx.fill();
    ctx.strokeStyle = on ? P.acc : (hov ? P.hoverEdge : P.hair2);
    ctx.lineWidth = Math.max(1,S(on?1.8:1));
    roundRect(p.x,p.y,S(b.w),S(b.h),S(8)); ctx.stroke();

    text(g.name, b.x+12, b.y+22,
      {size:12.5, weight:on?700:500, color:on?P.accInk:P.chipInk, font:MONO});
    text(String(g.count), b.x+b.w-12, b.y+22,
      {size:11, color:on?P.accInk2:P.faint, align:"right", font:MONO});
    // a quiet dot marks a group that needs a screen
    if(g.gui){
      const d=T(V(b.x+b.w-26,b.y+17));
      ctx.fillStyle = on ? P.accEdge : P.thin;
      ctx.beginPath(); ctx.arc(d.x,d.y,S(2.6),0,7); ctx.fill();
    }
  });

  const last = chipBox(GROUPS.length-1);
  text("● needs a display", grid.x, last.y+last.h+26, {size:11, color:P.muted});
  text(pinned ? "pinned — click again or press Esc to resume the tour"
              : "touring · click to pin · arrow keys to move",
    grid.x+140, last.y+last.h+26, {size:11, color:P.faint});
}

function drawDetail(){
  const g = GROUPS[selected];
  const p=T(V(detail.x,detail.y));
  ctx.save();
  ctx.shadowColor=P.shadow; ctx.shadowBlur=S(20); ctx.shadowOffsetY=S(6);
  ctx.fillStyle=P.card; roundRect(p.x,p.y,S(detail.w),S(detail.h),S(14)); ctx.fill();
  ctx.restore();
  ctx.strokeStyle=P.accEdge; ctx.lineWidth=Math.max(1,S(1.6));
  roundRect(p.x,p.y,S(detail.w),S(detail.h),S(14)); ctx.stroke();
  if(!g) return;

  text(`${BIN} ${g.name}`, detail.x+22, detail.y+40, {size:19, weight:700, font:MONO});
  text(`${g.count} command${g.count===1?"":"s"}`, detail.x+detail.w-22, detail.y+40,
    {size:12, color:P.muted, align:"right", font:MONO});

  const rule=T(V(detail.x+22,detail.y+54));
  ctx.strokeStyle=P.rule; ctx.lineWidth=Math.max(1,S(1));
  ctx.beginPath(); ctx.moveTo(rule.x,rule.y); ctx.lineTo(rule.x+S(detail.w-44),rule.y); ctx.stroke();

  const rowH = 27, top = detail.y+78;
  const max = Math.floor((detail.h - 78 - 16) / rowH);
  g.commands.slice(0,max).forEach((c,i)=>{
    const y = top + i*rowH;
    // a group default is invoked as the bare group name
    const route = c.name ? `${g.name} ${c.name}` : g.name;
    const rw = measure(route,12.5,600,MONO);
    text(route, detail.x+22, y, {size:12.5, weight:600, color:P.routeInk, font:MONO});
    if(c.gui){
      const d=T(V(detail.x+30+rw, y-4));
      ctx.fillStyle=P.accEdge; ctx.beginPath(); ctx.arc(d.x,d.y,S(2.4),0,7); ctx.fill();
    }
    text(ellipsize(c.summary, detail.w-44, 11.5), detail.x+22, y+14,
      {size:11.5, color:P.muted});
  });
  if(g.commands.length > max){
    text(`+${g.commands.length-max} more`, detail.x+22, top+max*rowH,
      {size:11.5, color:P.faint});
  }
}

function drawEdges(t){
  for(const e of edgeList(t)){
    const p0=e.p0, p3=e.p3;
    const [p1,p2]=ctrl(p0,p3,e.bend||0);
    const A=T(p0),B=T(p1),C=T(p2),D=T(p3);

    ctx.strokeStyle = e.hot ? P.hot : (e.thin ? P.thin : P.edge);
    ctx.lineWidth = Math.max(1, S(e.hot ? 1.8 : (e.thin ? 1 : 1.3)));
    ctx.beginPath(); ctx.moveTo(A.x,A.y); ctx.bezierCurveTo(B.x,B.y,C.x,C.y,D.x,D.y); ctx.stroke();

    const tg=bezT(p0,p1,p2,p3,1), tip=T(p3), ang=Math.atan2(tg.y,tg.x);
    ctx.save(); ctx.translate(tip.x,tip.y); ctx.rotate(ang);
    ctx.strokeStyle=P.arrow; ctx.lineWidth=Math.max(1,S(1.4));
    ctx.beginPath(); ctx.moveTo(-S(8),-S(4.5)); ctx.lineTo(0,0); ctx.lineTo(-S(8),S(4.5)); ctx.stroke();
    ctx.restore();

    const n = e.dots || 1;
    const dim = (e.hot === false) ? 0.28 : 1;
    for(let i=0;i<n;i++){
      let u=((t*0.19 + i/n + (e.phase||0)) % 1);
      if(e.rev) u=1-u;
      const q=T(bez(p0,p1,p2,p3,u));
      const fade=Math.min(1, Math.sin(Math.PI*Math.min(u,1-u)*2.6)+0.35) * dim;
      ctx.fillStyle=`rgba(${P.dotRGB},${0.85*fade})`;
      ctx.beginPath(); ctx.arc(q.x,q.y,S(e.thin?3:4),0,7); ctx.fill();
      ctx.fillStyle=`rgba(${P.dotRGB},${0.13*fade})`;
      ctx.beginPath(); ctx.arc(q.x,q.y,S(e.thin?6.5:9),0,7); ctx.fill();
    }

    if(e.label){
      const lt=e.lt||0.5, lo=(e.lo===undefined?20:e.lo);
      const m=bez(p0,p1,p2,p3,lt), mt=bezT(p0,p1,p2,p3,lt);
      text(e.label, m.x - mt.y*lo, m.y + mt.x*lo + 4,
        {size:10.5, color:P.muted, align:"center", weight:600, spacing:1.2});
    }
  }
}

function drawTitle(){
  // The name is the thing you type. That is the whole idea.
  text(BIN, VW-60, 130, {size:82, weight:800, align:"right", spacing:-1});
  text("One CLI over the machine that an agent drives the way you do.", VW-60, 182,
    {size:16, color:P.dim, align:"right"});
  text("Install the skill. Or serve it over MCP.", VW-60, 207,
    {size:16, weight:700, align:"right"});
  text(SURFACE_LABEL + " · macOS · Windows · Linux", VW-60, 231,
    {size:12, color:P.acc, align:"right", weight:700, spacing:.6});
}

function drawStatusBar(t){
  const w=640,h=48,x=60,y=74;
  const p=T(V(x,y));
  ctx.save();
  ctx.shadowColor=P.shadowBar; ctx.shadowBlur=S(16); ctx.shadowOffsetY=S(4);
  ctx.fillStyle=P.statusBg; roundRect(p.x,p.y,S(w),S(h),S(10)); ctx.fill();
  ctx.restore();

  const segs=[[BIN,P.codeInk],["|",P.sep],[DATA.commands+" commands",P.green],
              ["|",P.sep],[DATA.groups+" groups",P.codeInk],["|",P.sep],
              ["3 platforms",P.amber]];
  let cx=x+22;
  for(const [s,col] of segs){
    text(s, cx, y+31, {size:14, color:col, font:MONO, weight:600});
    cx += measure(s,14,600,MONO)+10;
  }
  if(Math.floor(t*1.8)%2){
    const q=T(V(cx+2,y+16)); ctx.fillStyle=P.green; ctx.fillRect(q.x,q.y,S(8),S(17));
  }
}

function drawChips(){
  const chips=["One static binary","No account, no phone-home","OS APIs, not pixels","Runs headless"];
  let x=60; const y=176;
  for(const s of chips){
    const w=measure(s,12,500)+26;
    const p=T(V(x,y));
    ctx.fillStyle=P.panel; roundRect(p.x,p.y,S(w),S(30),S(15)); ctx.fill();
    ctx.strokeStyle=P.panelEdge; ctx.lineWidth=Math.max(1,S(1));
    roundRect(p.x,p.y,S(w),S(30),S(15)); ctx.stroke();
    text(s, x+w/2, y+20, {size:12, color:P.chipInk, align:"center"});
    x += w+10;
  }
}

// ---------------------------------------------------------------- loop
const still = matchMedia("(prefers-reduced-motion: reduce)").matches;
let t0=performance.now(), frames=0, facc=0;
const fpsEl = document.getElementById("fps");

function draw(t){
  autoSelect(t);
  ctx.fillStyle=P.page; ctx.fillRect(0,0,innerWidth,innerHeight);
  const p=T(V(0,0));
  ctx.fillStyle=P.plate; ctx.fillRect(p.x,p.y,S(VW),S(VH));

  drawEdges(t);
  drawWays(t);
  drawPlatforms();
  drawSurface();
  callers.forEach((c,i)=>drawCard(c, activeCaller(t)===i));
  drawGrid(t);
  drawDetail();
  drawTitle();
  drawStatusBar(t);
  drawChips();
}

function frame(now){
  const dt=(now-t0)/1000; t0=now; const t=now/1000;
  facc+=dt; frames++;
  if(facc>0.5 && fpsEl){ fpsEl.textContent=Math.round(frames/facc)+" fps"; frames=0; facc=0; }
  draw(t);
  requestAnimationFrame(frame);
}

// Follow the reader's system setting live, the way the docs theme does.
darkQuery.addEventListener("change", e => {
  P = PALETTES[e.matches ? "dark" : "light"];
  if(still) draw(2.0);
});

if(still){ draw(2.0); addEventListener("resize", ()=>{ resize(); draw(2.0); }); }
else requestAnimationFrame(frame);
