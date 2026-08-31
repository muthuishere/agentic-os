"use strict";
// agentic-os, drawn.
//
// One surface, two callers, three platforms. Same visual language as the
// rag_api diagram it is a sibling of: warm off-white plate, hairline cards, one
// orange accent, one travelling blue dot, no gradients and no glow. Everything
// is laid out in a virtual 1600x1000 space and scaled to whatever viewport it
// lands in.

const VW = 1600, VH = 1000;
const INK   = "#1c1b19";
const MUTED = "#8a857c";
const HAIR  = "#d6d2c8";
const CARD  = "#ffffff";
const ACC   = "#e08a1e";          // orange accent
const DOT   = "#4a5cd0";          // travelling dot blue

const SANS = '-apple-system,BlinkMacSystemFont,"Segoe UI",Inter,Helvetica,Arial,sans-serif';
const MONO = 'ui-monospace,SFMono-Regular,Menlo,Consolas,monospace';

// Counts come from the generated registry snapshot, injected by the page. The
// fallback is only there so the file is still readable on its own.
const STATS = Object.assign({commands:96, groups:34}, window.AOS_STATS || {});
const SURFACE = `${STATS.commands} commands · ${STATS.groups} groups`;

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

function roundRect(x,y,w,h,r){
  ctx.beginPath();
  ctx.moveTo(x+r,y); ctx.lineTo(x+w-r,y); ctx.quadraticCurveTo(x+w,y,x+w,y+r);
  ctx.lineTo(x+w,y+h-r); ctx.quadraticCurveTo(x+w,y+h,x+w-r,y+h);
  ctx.lineTo(x+r,y+h); ctx.quadraticCurveTo(x,y+h,x,y+h-r);
  ctx.lineTo(x,y+r); ctx.quadraticCurveTo(x,y,x+r,y); ctx.closePath();
}
function text(str,x,y,{size=14,weight=400,color=INK,font=SANS,align="left",spacing=0}={}){
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
// The two callers. Both arrive at the same place; that is the whole point.
const callers = [
  {id:"human", n:"01", x:60, y:330, w:250, h:118, icon:"›_", title:"A person",
   detail:"at a terminal"},
  {id:"agent", n:"02", x:60, y:562, w:250, h:118, icon:"⌁", title:"An agent",
   detail:"over MCP"},
];

// The command surface — the single slab everything routes through.
const surface = {x:430, y:300, w:280, h:470};

// What the surface reaches on the machine.
const caps = [
  "windows", "input", "screen", "files", "processes", "packages", "network",
];
const machine = {x:840, y:292, w:300, h:486};

// One backend per platform, behind the same verbs.
const platforms = [
  {name:"macOS",   sub:"CoreGraphics · osascript",  y:330},
  {name:"Windows", sub:"Win32 · PowerShell",        y:466},
  {name:"Linux",   sub:"wmctrl · xdotool · Xvfb",   y:602},
];
const PX = 1250, PW = 290, PH = 116;

// The two lines that are the same call, said in each caller's language.
const sameCall = [
  {kind:"cli", txt:"$ agentic-os window move Chrome --zone=1B"},
  {kind:"mcp", txt:'window_move {"app":"Chrome","zone":"1B"}'},
];

// ---------------------------------------------------------------- anchors
function boxAnchor(b, side){
  const m={left:V(b.x,b.y+b.h/2), right:V(b.x+b.w,b.y+b.h/2),
           top:V(b.x+b.w/2,b.y),  bottom:V(b.x+b.w/2,b.y+b.h)};
  return m[side];
}
function ctrl(p,q,bend){
  const dx=q.x-p.x, dy=q.y-p.y, len=Math.hypot(dx,dy)||1;
  const nx=-dy/len, ny=dx/len, off=len*bend;
  return [V(p.x+dx*0.32+nx*off, p.y+dy*0.32+ny*off),
          V(p.x+dx*0.68+nx*off, p.y+dy*0.68+ny*off)];
}

// active caller alternates slowly: the same surface, either way in.
const activeCaller = t => Math.floor(t/3.2) % 2;

// ---------------------------------------------------------------- edges
function edgeList(t){
  const a = activeCaller(t);
  const sl = boxAnchor(surface,"left");
  const es = [];

  callers.forEach((c,i)=>{
    es.push({
      p0: V(c.x+c.w, c.y+c.h/2),
      p3: V(sl.x, surface.y + surface.h*(i===0?0.30:0.70)),
      label: i===0 ? "TYPES" : "CALLS",
      bend: 0, dots: 2, hot: a===i,
    });
  });

  // surface -> the machine, one line per capability row
  const rows = capRows();
  rows.forEach((r,i)=>{
    es.push({
      p0: V(surface.x+surface.w, surface.y + surface.h*(0.18 + 0.64*(i/(rows.length-1)))),
      p3: V(machine.x, r.y + r.h/2),
      label: null, bend: 0, dots: 1, thin: true, phase: i/rows.length,
    });
  });

  // the machine -> one backend per platform
  platforms.forEach((p,i)=>{
    es.push({
      p0: V(machine.x+machine.w, machine.y + machine.h*(0.22 + 0.28*i)),
      p3: V(PX, p.y + PH/2),
      label: null,
      bend: i===0 ? -0.10 : (i===2 ? 0.10 : 0), dots: 1, lo: -22,
    });
  });

  // and the result, coming back the long way underneath
  es.push({
    p0: V(machine.x+40, machine.y+machine.h),
    p3: V(callers[1].x+40, callers[1].y+callers[1].h),
    label: "RESULT", bend: -0.20, dots: 3, rev: true, lo: -26, lt: 0.5,
  });

  return es;
}

function capRows(){
  const h = 44, gap = 12, top = machine.y + 62;
  return caps.map((c,i)=>({label:c, x:machine.x+24, y:top+i*(h+gap), w:machine.w-48, h}));
}

// ---------------------------------------------------------------- drawing
function drawCard(c){
  const p=T(V(c.x,c.y));
  ctx.save();
  ctx.shadowColor="rgba(30,28,24,.10)"; ctx.shadowBlur=S(18); ctx.shadowOffsetY=S(5);
  ctx.fillStyle=CARD; roundRect(p.x,p.y,S(c.w),S(c.h),S(12)); ctx.fill();
  ctx.restore();
  ctx.strokeStyle=HAIR; ctx.lineWidth=Math.max(1,S(1));
  roundRect(p.x,p.y,S(c.w),S(c.h),S(12)); ctx.stroke();

  text(c.n, c.x+20, c.y+40, {size:21, weight:300, color:"#b8b3a8", font:MONO});

  const ip=T(V(c.x+20,c.y+56));
  ctx.fillStyle="#f3f1ec"; roundRect(ip.x,ip.y,S(28),S(28),S(7)); ctx.fill();
  ctx.strokeStyle="#e6e2d9"; ctx.lineWidth=Math.max(1,S(1));
  roundRect(ip.x,ip.y,S(28),S(28),S(7)); ctx.stroke();
  text(c.icon, c.x+34, c.y+75, {size:14, color:"#6f6a61", align:"center", font:MONO});

  text(c.title, c.x+58, c.y+76, {size:17, weight:700});
  text(c.detail, c.x+20, c.y+c.h-20, {size:12.5, color:MUTED});
}

function drawSurface(t){
  const p=T(V(surface.x,surface.y));
  ctx.save();
  ctx.shadowColor="rgba(74,92,208,.16)"; ctx.shadowBlur=S(30); ctx.shadowOffsetY=S(6);
  ctx.fillStyle=CARD; roundRect(p.x,p.y,S(surface.w),S(surface.h),S(14)); ctx.fill();
  ctx.restore();
  ctx.strokeStyle="#c3c9ee"; ctx.lineWidth=S(3);
  roundRect(p.x,p.y,S(surface.w),S(surface.h),S(14)); ctx.stroke();

  const cx = surface.x + surface.w/2;
  text("03", surface.x+22, surface.y+42, {size:21, weight:300, color:"#b8b3a8", font:MONO});
  text("The command surface", cx, surface.y+92, {size:20, weight:700, align:"center"});
  text(SURFACE, cx, surface.y+116, {size:12.5, color:MUTED, align:"center"});

  // the same call, said two ways — whichever caller is active right now
  const a = activeCaller(t);
  const call = sameCall[a];
  const bw = surface.w-40, bx = surface.x+20, by = surface.y+142;
  const q=T(V(bx,by));
  ctx.fillStyle="#151412"; roundRect(q.x,q.y,S(bw),S(74),S(9)); ctx.fill();
  text(a===0 ? "CLI" : "MCP TOOL", bx+14, by+22,
    {size:9.5, color:"#7f7a70", font:MONO, weight:700, spacing:1.2});
  // wrap the mono line by hand — it is short and known
  const words = call.txt.split(" ");
  let line = "", ln = 0;
  const emit = s => { text(s, bx+14, by+44+ln*17, {size:11.5, color:"#e6e2d9", font:MONO}); ln++; };
  for(const w of words){
    const next = line ? line+" "+w : w;
    if(measure(next,11.5,400,MONO) > bw-28){ emit(line); line = w; } else line = next;
  }
  if(line) emit(line);

  // the one fact the whole diagram exists to make: one Runner underneath
  const ry = surface.y+244;
  const r=T(V(surface.x+20,ry));
  ctx.fillStyle="#fbf4e8"; roundRect(r.x,r.y,S(bw),S(52),S(9)); ctx.fill();
  ctx.strokeStyle="#e3b877"; ctx.lineWidth=Math.max(1,S(1.4));
  roundRect(r.x,r.y,S(bw),S(52),S(9)); ctx.stroke();
  text("one Runner, one code path", cx, ry+22, {size:13, weight:700, color:"#8a5a12", align:"center"});
  text("no second implementation", cx, ry+40, {size:11.5, color:"#a07a3a", align:"center"});

  // the structural refusals live here too
  const gates = ["needs a display? refused, with a reason",
                 "delete a root or $HOME? refused",
                 "serve binds loopback"];
  gates.forEach((g,i)=>{
    const y = surface.y+330+i*26;
    const d=T(V(surface.x+24,y-4));
    ctx.fillStyle="#d9d4c9"; ctx.beginPath(); ctx.arc(d.x,d.y,S(2.4),0,7); ctx.fill();
    text(g, surface.x+38, y, {size:11.5, color:"#6f6a61"});
  });

  text("SAFETY IS STRUCTURE", cx, surface.y+surface.h-22,
    {size:9.5, color:MUTED, align:"center", weight:700, spacing:1.4});
}

function drawMachine(t){
  const p=T(V(machine.x,machine.y));
  ctx.fillStyle="#fbfaf7"; roundRect(p.x,p.y,S(machine.w),S(machine.h),S(14)); ctx.fill();
  ctx.strokeStyle="#e2ddd2"; ctx.lineWidth=Math.max(1,S(1));
  roundRect(p.x,p.y,S(machine.w),S(machine.h),S(14)); ctx.stroke();

  text("04", machine.x+22, machine.y+40, {size:21, weight:300, color:"#b8b3a8", font:MONO});
  text("the machine", machine.x+machine.w/2, machine.y+40, {size:17, weight:700, align:"center"});

  const rows = capRows();
  rows.forEach((r,i)=>{
    const on = (Math.floor(t*0.9) % rows.length) === i;
    const q=T(V(r.x,r.y));
    ctx.fillStyle = on ? "#fdf3e3" : CARD;
    roundRect(q.x,q.y,S(r.w),S(r.h),S(8)); ctx.fill();
    ctx.strokeStyle = on ? ACC : "#e6e2d9"; ctx.lineWidth=Math.max(1,S(on?1.6:1));
    roundRect(q.x,q.y,S(r.w),S(r.h),S(8)); ctx.stroke();
    text(r.label, r.x+16, r.y+28, {size:13, weight:on?700:500, color:on?"#8a5a12":"#5e594f"});
  });
}

function drawPlatforms(t){
  platforms.forEach((pf,i)=>{
    const p=T(V(PX,pf.y));
    ctx.save();
    ctx.shadowColor="rgba(30,28,24,.08)"; ctx.shadowBlur=S(14); ctx.shadowOffsetY=S(4);
    ctx.fillStyle=CARD; roundRect(p.x,p.y,S(PW),S(PH),S(12)); ctx.fill();
    ctx.restore();
    ctx.strokeStyle=HAIR; ctx.lineWidth=Math.max(1,S(1));
    roundRect(p.x,p.y,S(PW),S(PH),S(12)); ctx.stroke();

    text(String(i+5).padStart(2,"0"), PX+20, pf.y+34,
      {size:18, weight:300, color:"#b8b3a8", font:MONO});
    text(pf.name, PX+20, pf.y+68, {size:20, weight:700});
    text(pf.sub, PX+20, pf.y+94, {size:12, color:MUTED, font:MONO});

    // a quiet verified tick — all three are actually tested
    const tp=T(V(PX+PW-34, pf.y+30));
    ctx.strokeStyle="#9aa2dd"; ctx.lineWidth=S(1.6); ctx.lineCap="round";
    ctx.beginPath();
    ctx.moveTo(tp.x-S(6), tp.y); ctx.lineTo(tp.x-S(2), tp.y+S(5)); ctx.lineTo(tp.x+S(6), tp.y-S(6));
    ctx.stroke(); ctx.lineCap="butt";
  });
  text("ONE BACKEND EACH · VERIFIED ON ALL THREE", PX+PW, platforms[2].y+PH+30,
    {size:9.5, color:MUTED, align:"right", weight:700, spacing:1.4});
}

function drawEdges(t){
  for(const e of edgeList(t)){
    const p0=e.p0, p3=e.p3;
    const [p1,p2]=ctrl(p0,p3,e.bend||0);
    const A=T(p0),B=T(p1),C=T(p2),D=T(p3);

    ctx.strokeStyle = e.hot ? "#b9c0ea" : (e.thin ? "#ddd8cd" : "#cfcabf");
    ctx.lineWidth = Math.max(1, S(e.hot ? 1.8 : (e.thin ? 1 : 1.3)));
    ctx.beginPath(); ctx.moveTo(A.x,A.y); ctx.bezierCurveTo(B.x,B.y,C.x,C.y,D.x,D.y); ctx.stroke();

    const tg=bezT(p0,p1,p2,p3,1), tip=T(p3), ang=Math.atan2(tg.y,tg.x);
    ctx.save(); ctx.translate(tip.x,tip.y); ctx.rotate(ang);
    ctx.strokeStyle="#b6b0a4"; ctx.lineWidth=Math.max(1,S(1.4));
    ctx.beginPath(); ctx.moveTo(-S(8),-S(4.5)); ctx.lineTo(0,0); ctx.lineTo(-S(8),S(4.5)); ctx.stroke();
    ctx.restore();

    // travelling dots: the call going in, and the result coming back
    const n = e.dots || 1;
    const dim = (e.hot === false) ? 0.28 : 1;
    for(let i=0;i<n;i++){
      let u=((t*0.19 + i/n + (e.phase||0)) % 1);
      if(e.rev) u=1-u;
      const q=T(bez(p0,p1,p2,p3,u));
      const fade=Math.min(1, Math.sin(Math.PI*Math.min(u,1-u)*2.6)+0.35) * dim;
      ctx.fillStyle=`rgba(74,92,208,${0.85*fade})`;
      ctx.beginPath(); ctx.arc(q.x,q.y,S(e.thin?3:4),0,7); ctx.fill();
      ctx.fillStyle=`rgba(74,92,208,${0.13*fade})`;
      ctx.beginPath(); ctx.arc(q.x,q.y,S(e.thin?6.5:9),0,7); ctx.fill();
    }

    if(e.label){
      const lt=e.lt||0.5, lo=(e.lo===undefined?20:e.lo);
      const m=bez(p0,p1,p2,p3,lt), mt=bezT(p0,p1,p2,p3,lt);
      text(e.label, m.x - mt.y*lo, m.y + mt.x*lo + 4,
        {size:10.5, color:MUTED, align:"center", weight:600, spacing:1.2});
    }
  }
}

function drawTitle(){
  text("agentic-os", VW-60, 132, {size:68, weight:800, align:"right"});
  text("A computer-use MCP server that is also a CLI you can type.", VW-60, 172,
    {size:16.5, color:"#57534b", align:"right"});
  text("One surface, two callers, three platforms.", VW-60, 201,
    {size:16.5, weight:700, align:"right"});
  text(SURFACE + " · macOS · Windows · Linux", VW-60, 230,
    {size:12.5, color:ACC, align:"right", weight:700, spacing:.6});
}

function drawStatusBar(t){
  const w=640,h=48,x=60,y=74;
  const p=T(V(x,y));
  ctx.save();
  ctx.shadowColor="rgba(0,0,0,.22)"; ctx.shadowBlur=S(16); ctx.shadowOffsetY=S(4);
  ctx.fillStyle="#151412"; roundRect(p.x,p.y,S(w),S(h),S(10)); ctx.fill();
  ctx.restore();

  const segs=[["agentic-os","#e6e2d9"],["|","#5a564e"],[STATS.commands+" commands","#7fd6a0"],
              ["|","#5a564e"],[STATS.groups+" groups","#e6e2d9"],["|","#5a564e"],
              ["3 platforms","#e0a955"]];
  let cx=x+22;
  for(const [s,col] of segs){
    text(s, cx, y+31, {size:14, color:col, font:MONO, weight:600});
    cx += measure(s,14,600,MONO)+10;
  }
  if(Math.floor(t*1.8)%2){
    const q=T(V(cx+2,y+16)); ctx.fillStyle="#7fd6a0"; ctx.fillRect(q.x,q.y,S(8),S(17));
  }
}

function drawChips(){
  const chips=["One static binary","No account, no phone-home","OS APIs, not pixels",
               "Loopback by default","Adapters + plugins","Runs headless"];
  let x=60; const y=VH-72;
  for(const s of chips){
    const w=measure(s,12.5,500)+30;
    const p=T(V(x,y));
    ctx.fillStyle="#fbfaf7"; roundRect(p.x,p.y,S(w),S(32),S(16)); ctx.fill();
    ctx.strokeStyle="#e2ddd2"; ctx.lineWidth=Math.max(1,S(1));
    roundRect(p.x,p.y,S(w),S(32),S(16)); ctx.stroke();
    text(s, x+w/2, y+21, {size:12.5, color:"#5e594f", align:"center"});
    x += w+12;
  }
  const rx=VW-60;
  text("1·2·3", rx-238, y+24, {size:22, weight:800, color:ACC, align:"right"});
  text("one surface, two callers, three platforms", rx-228, y+23,
    {size:12.5, color:"#57534b"});
}

function drawFooterRule(){
  const a=T(V(0,VH-120)), b=T(V(VW,VH-120));
  ctx.strokeStyle="#e6e2d9"; ctx.lineWidth=Math.max(1,S(1));
  ctx.beginPath(); ctx.moveTo(a.x,a.y); ctx.lineTo(b.x,b.y); ctx.stroke();
}

// ---------------------------------------------------------------- loop
const still = matchMedia("(prefers-reduced-motion: reduce)").matches;
let t0=performance.now(), frames=0, facc=0;
const fpsEl = document.getElementById("fps");

function draw(t){
  ctx.fillStyle="#f4f3ef"; ctx.fillRect(0,0,innerWidth,innerHeight);
  const p=T(V(0,0));
  ctx.fillStyle="#fbfaf8"; ctx.fillRect(p.x,p.y,S(VW),S(VH));

  drawFooterRule();
  drawEdges(t);
  drawMachine(t);
  drawPlatforms(t);
  drawSurface(t);
  for(const c of callers) drawCard(c);
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

if(still){ draw(2.0); addEventListener("resize", ()=>{ resize(); draw(2.0); }); }
else requestAnimationFrame(frame);
