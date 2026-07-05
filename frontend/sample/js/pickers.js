// ===== PICKER COMPONENTS: wheel, ruler, calendar =====

// ---- vertical wheel ----
function wheelHtml(id, values, sel, width){
  return `<div class="wheel" id="${id}" data-sel="${sel}" style="width:${width}px">
    <div class="wpad"></div>
    ${values.map(v=>`<div class="wi">${v}</div>`).join('')}
    <div class="wpad"></div>
  </div>`;
}
function initWheel(id, onChange){
  const el = document.getElementById(id); if(!el) return;
  const items = [...el.querySelectorAll('.wi')];
  const H = 44, sel = +el.dataset.sel || 0;
  el.scrollTop = sel*H;
  const upd = ()=>{
    const idx = Math.max(0, Math.min(items.length-1, Math.round(el.scrollTop/H)));
    items.forEach((it,k)=> it.classList.toggle('on', k===idx));
    onChange(idx);
  };
  let raf;
  el.addEventListener('scroll', ()=>{ cancelAnimationFrame(raf); raf=requestAnimationFrame(upd); });
  upd();
}

// ---- horizontal ruler ----
function rulerHtml(id, min, max, val, unit, label){
  let ticks='';
  for(let i=min;i<=max;i++){
    const major=(i%5===0);
    ticks += `<div class="tk ${major?'major':''}"><div class="line"></div>${major?`<div class="num">${faNum(i)}</div>`:'<div class="num"></div>'}</div>`;
  }
  return `<div style="text-align:center;margin-bottom:16px">
      <span id="${id}_v" style="font-size:48px;font-weight:800;color:var(--ink);font-variant-numeric:tabular-nums">${faNum(val.toFixed(1))}</span>
      <span style="font-size:15px;color:var(--muted);margin-right:6px">${unit}</span>
    </div>
    <div class="ruler-wrap">
      <div class="rpoint"></div>
      <div class="ruler" id="${id}" dir="ltr"><div class="rpad"></div>${ticks}<div class="rpad"></div></div>
      <div class="ruler-fade"></div>
    </div>`;
}
function initRuler(id, min, max, val, onChange){
  const el = document.getElementById(id); if(!el) return;
  const disp = document.getElementById(id+'_v');
  const W = 12;
  el.scrollLeft = (val-min)*W;
  const upd = ()=>{
    const idx = Math.round(el.scrollLeft/W);
    const v = Math.max(min, Math.min(max, min+idx));
    if(disp) disp.textContent = faNum(v.toFixed(1));
    onChange(v);
  };
  let raf;
  el.addEventListener('scroll', ()=>{ cancelAnimationFrame(raf); raf=requestAnimationFrame(upd); });
  upd();
}

// ---- calendar (last period) ----
function calendarHtml(){
  const month = 'اردیبهشت ۱۴۰۴';
  const wd = ['ش','ی','د','س','چ','پ','ج'];
  const offset = 3;              // first day column offset
  const days = 31;
  let cells = '';
  for(let i=0;i<offset;i++) cells += `<span></span>`;
  for(let d=1; d<=days; d++){
    cells += `<button class="cday" data-d="${d}" onclick="pickDay(${d})">${faNum(d)}</button>`;
  }
  return `<div class="card" style="padding:16px 14px;margin-top:2px">
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:14px">
      <span class="iconbtn">${ic('chevronRight',20)}</span>
      <span style="font-weight:800;font-size:15px;color:var(--ink)">${month}</span>
      <span class="iconbtn">${ic('chevronLeft',20)}</span>
    </div>
    <div class="cal-grid" style="margin-bottom:8px">
      ${wd.map(w=>`<span style="font-size:11px;color:var(--muted);font-weight:700;text-align:center">${w}</span>`).join('')}
    </div>
    <div class="cal-grid">${cells}</div>
  </div>
  <p class="sub" style="text-align:center;margin-top:16px">روزی که آخرین پریودتان آغاز شد را انتخاب کنید</p>`;
}
function pickDay(d){
  store.lastPeriod = d;
  document.querySelectorAll('.cday').forEach(b=> b.classList.toggle('on', +b.dataset.d===d));
}
