// ===== MAIN PAGE (Homepage) =====

function sectionHead(title, action){
  return `<div style="display:flex;align-items:center;justify-content:space-between;margin:0 2px 12px">
    ${action?`<span style="font-size:12px;font-weight:700;color:var(--brand);cursor:pointer">${action}</span>`:'<span></span>'}
    <span style="font-size:16px;font-weight:800;color:var(--ink)">${title}</span>
  </div>`;
}

// ---------- header ----------
function homeHeader(){
  return `<div style="display:flex;align-items:center;justify-content:space-between;padding:6px 18px 10px">
    <button class="iconbtn" style="background:rgba(255,255,255,.6);position:relative">
      ${ic('bell',20)}<span style="position:absolute;top:7px;left:8px;width:7px;height:7px;border-radius:50%;background:var(--brand);border:1.5px solid #fff"></span>
    </button>
    <div style="text-align:center">
      <div style="font-weight:900;font-size:20px;color:var(--brand-deep)">ریتمی</div>
      <div style="font-size:10px;color:var(--muted);margin-top:1px">همراه سلامتی شما</div>
    </div>
    <button class="iconbtn" style="background:rgba(255,255,255,.6);color:var(--brand)">${ic('sparkle',20,{fill:'currentColor',sw:0})}</button>
  </div>`;
}

// ---------- week calendar ----------
function weekStrip(){
  const week = [
    {w:'شنبه',d:28},{w:'یک',d:29},{w:'دو',d:30,t:1},{w:'سه',d:31},
    {w:'چهار',d:1},{w:'پنج',d:2},{w:'جمعه',d:3}
  ];
  return `<div style="padding:2px 16px 8px">
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px">
      <span style="font-size:11px;color:var(--muted);font-weight:600">ماه کامل</span>
      <span style="font-size:13px;font-weight:800;color:var(--ink)">دی ۱۴۰۳</span>
    </div>
    <div style="display:flex;justify-content:space-between">
      ${week.map(x=>`
        <div style="display:flex;flex-direction:column;align-items:center;gap:7px;flex:1">
          <span style="font-size:11px;color:${x.t?'var(--brand)':'var(--muted)'};font-weight:600">${x.w}</span>
          <span style="width:34px;height:42px;border-radius:18px;display:flex;align-items:center;justify-content:center;font-size:14px;font-weight:700;font-variant-numeric:tabular-nums;
            ${x.t?'background:var(--brand);color:#fff;box-shadow:0 8px 16px -6px rgba(233,30,99,.6)':'color:var(--ink)'}">${faNum(x.d)}</span>
        </div>`).join('')}
    </div>
  </div>`;
}

// ---------- next-period hero card ----------
function nextPeriodCard(){
  return `<div style="margin:6px 16px 0;border-radius:20px;overflow:hidden;background:linear-gradient(135deg,#F0568F 0%,#E91E63 55%,#D81B60 100%);color:#fff;box-shadow:0 18px 34px -16px rgba(233,30,99,.7)">
    <div style="padding:18px 18px 16px">
      <div style="display:flex;align-items:flex-start;justify-content:space-between">
        <div style="background:rgba(255,255,255,.22);border-radius:14px;padding:8px 12px;text-align:center">
          <div style="font-size:18px;font-weight:800">۵۲٪</div>
          <div style="font-size:10px;opacity:.9">باروری</div>
        </div>
        <div style="text-align:right">
          <div style="font-size:13px;opacity:.92;font-weight:600">تا پریود بعدی</div>
          <div style="font-size:30px;font-weight:900;margin-top:2px">۱ روز</div>
          <div style="display:inline-flex;align-items:center;gap:5px;background:rgba(255,255,255,.18);border-radius:20px;padding:3px 10px;margin-top:8px;font-size:11px">
            ${ic('drop',13,{fill:'#fff',sw:0})} فاز فعلی: لوتیال</div>
        </div>
      </div>
      <div style="display:flex;gap:4px;margin:18px 0 12px" dir="ltr">
        ${[18,16,22,14,30].map((w,i)=>`<span style="height:7px;flex:${w};border-radius:99px;background:${i===4?'#fff':'rgba(255,255,255,.32)'}"></span>`).join('')}
      </div>
      <div style="font-size:12.5px;font-weight:700;text-align:right">پریود بعدی شروع می‌شود در ۱۷ فروردین</div>
      <div id="npx" style="font-size:11.5px;line-height:1.9;opacity:.92;text-align:right;margin-top:8px">
        بدنت در حال آماده‌سازی برای شروع چرخه‌ی تازه است. در این مرحله ممکن است نوسان خلق‌و‌خو، خستگی یا حساسیت داشته باشی — مراقب خودت باش و کمی استراحت کن.</div>
      <div style="display:flex;justify-content:center;margin-top:12px">
        <button onclick="toggleNp(this)" style="background:rgba(255,255,255,.18);border:0;color:#fff;border-radius:20px;padding:5px 16px;font-family:inherit;font-size:12px;font-weight:700;cursor:pointer;display:flex;align-items:center;gap:5px">بستن ${ic('chevronDown',15)}</button>
      </div>
    </div>
  </div>`;
}
function toggleNp(b){
  const x=document.getElementById('npx'); const open=x.style.display!=='none';
  x.style.display=open?'none':'block';
  b.innerHTML = (open?'مشاهده‌ی بیشتر':'بستن')+' '+ic('chevronDown',15,{style:open?'transform:rotate(180deg)':''});
}

// ---------- start period ----------
function startPeriodBtn(){
  return `<div style="padding:14px 16px 4px">
    <button class="btn btn-ghost" style="height:48px;border-radius:16px;gap:8px;font-size:14px">
      ${ic('drop',18,{fill:'var(--brand)',sw:0})} شروع پریود</button></div>`;
}

// ---------- phase rows ----------
function phaseRows(){
  const rows=[
    {l:'پنجره باروری', d:'۱۵ آذر', c:'#F5A623', bg:'#FFF3DF'},
    {l:'تخمک گذاری', d:'۲۰ آذر', c:'#34C77B', bg:'#E7F8EF'},
    {l:'پریود بعدی در', d:'۳۰ آذر', c:'#FB64B6', bg:'#FFEBF5'},
  ];
  return `<div style="padding:6px 16px 0"><div class="card" style="padding:6px 14px">
    ${rows.map((r,i)=>`<div style="display:flex;align-items:center;justify-content:space-between;padding:12px 2px;${i<2?'border-bottom:1px solid var(--line)':''}">
      <span style="font-size:13px;font-weight:700;color:var(--muted);font-variant-numeric:tabular-nums">${r.d}</span>
      <div style="display:flex;align-items:center;gap:10px">
        <span style="font-size:14px;font-weight:700;color:var(--ink)">${r.l}</span>
        <span class="dot" style="background:${r.bg};color:${r.c}">${dropSolid(16,r.c)}</span>
      </div>
    </div>`).join('')}
  </div></div>`;
}

// ---------- mini stat cards (horizontal) ----------
function miniCards(){
  const cards=[
    {l:'پریود بعدی در', v:'۲۰ روز', c:'#FB64B6', bg:'#FFEBF5'},
    {l:'تخمک گذاری', v:'۶ روز', c:'#34C77B', bg:'#E7F8EF'},
    {l:'پنجره باروری', v:'۳ روز', c:'#F5A623', bg:'#FFF3DF'},
  ];
  return `<div class="scroll-x" style="padding:14px 16px 0"><div style="display:flex;gap:12px;justify-content:flex-end;min-width:min-content">
    ${cards.map(c=>`<div class="card" style="width:148px;flex:0 0 148px;padding:16px 14px;display:flex;flex-direction:column;align-items:center;gap:10px">
      <span class="dot" style="width:44px;height:44px;background:${c.bg};color:${c.c}">${dropSolid(22,c.c)}</span>
      <span style="font-size:12px;color:var(--muted);font-weight:600">${c.l}</span>
      <span style="font-size:17px;font-weight:800;color:var(--ink)">${c.v}</span>
    </div>`).join('')}
  </div></div>`;
}

// ---------- daily recommendations ----------
function recommendations(){
  const chips=[['flame','#FB64B6',1],['heart','#F06292',0],['smile','#F5A623',0],['moon','#7C7CF0',0],['drop','#34C77B',0]];
  const items=[
    {t:'مصرف آهن', s:'امروز غذاهای غنی از آهن بخورید'},
    {t:'مصرف امگا ۳', s:'امروز غذاهای غنی از امگا بخورید'},
  ];
  return `<div style="padding:16px 16px 0"><div class="card" style="padding:14px 12px">
    <div style="font-size:16px;font-weight:800;color:var(--pink);text-align:right;margin-bottom:12px">توصیه‌های امروز</div>
    <div style="display:flex;gap:8px;justify-content:flex-end;margin-bottom:12px">
      ${chips.map(c=>`<span style="width:36px;height:36px;border-radius:11px;display:flex;align-items:center;justify-content:center;
        background:${c[2]?c[1]:'#F4F6F8'};color:${c[2]?'#fff':'#AEB6BF'}">${ic(c[0],18,{fill:c[0]==='drop'&&c[2]?'#fff':'none',sw:c[0]==='drop'?0:2})}</span>`).join('')}
    </div>
    ${items.map(it=>`<div style="display:flex;align-items:center;gap:10px;border:1px solid var(--line);border-radius:12px;padding:12px;margin-bottom:8px">
      <span style="color:var(--pink)">${ic('drop',20,{fill:'currentColor',sw:0})}</span>
      <div style="flex:1;text-align:right">
        <div style="font-size:14px;font-weight:700;color:var(--ink)">${it.t}</div>
        <div style="font-size:11px;color:var(--muted);margin-top:3px">${it.s}</div>
      </div>
    </div>`).join('')}
  </div></div>`;
}

// ---------- today tasks ----------
const TASKS=[
  {t:'ثبت دمای بدن', done:1},{t:'نوشیدن ۸ لیوان آب', done:1},
  {t:'مصرف ویتامین D', done:0},{t:'۳۰ دقیقه پیاده‌روی', done:0}
];
function todayTasks(){
  return `<div style="padding:14px 16px 0"><div class="card" style="padding:14px 12px">
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:14px">
      <span style="display:flex;align-items:center;gap:6px;font-size:12px;font-weight:700;color:var(--steel)"><span id="taskCount">۲/۴</span>${ic('checkCircle',16,{stroke:'var(--green)'})}</span>
      <span style="font-size:14px;font-weight:800;color:var(--ink)">وظایف امروز</span>
    </div>
    <div id="taskList">${TASKS.map((x,i)=>taskRow(x,i)).join('')}</div>
    <div style="height:1px;background:var(--line);margin:14px 0 12px"></div>
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px">
      <span id="taskPct" style="font-size:12px;font-weight:700;color:var(--steel)">۵۰درصد</span>
      <span style="font-size:12px;font-weight:700;color:var(--muted)">پیشرفت امروز</span>
    </div>
    <div class="bar"><i id="taskBar" style="width:50%"></i></div>
  </div></div>`;
}
function taskRow(x,i){
  return `<div onclick="toggleTask(${i})" style="display:flex;align-items:center;justify-content:flex-end;gap:10px;padding:9px 2px;cursor:pointer">
    <span style="font-size:14px;font-weight:600;color:${x.done?'#AEB6BF':'var(--ink)'};${x.done?'text-decoration:line-through':''}">${x.t}</span>
    <span class="cbx ${x.done?'on':''}">${ic('check',14,{stroke:'#fff'})}</span>
  </div>`;
}
function toggleTask(i){
  TASKS[i].done = TASKS[i].done?0:1;
  document.getElementById('taskList').innerHTML = TASKS.map((x,k)=>taskRow(x,k)).join('');
  const n = TASKS.filter(t=>t.done).length, pct=Math.round(n/TASKS.length*100);
  document.getElementById('taskCount').textContent = faNum(n)+'/۴';
  document.getElementById('taskPct').textContent = faNum(pct)+'درصد';
  document.getElementById('taskBar').style.width = pct+'%';
}

// ---------- challenge ----------
function challenge(){
  return `<div style="padding:14px 16px 0"><div class="card" style="padding:14px 12px">
    <div style="font-size:14px;font-weight:800;color:var(--ink);text-align:right;margin-bottom:12px">چالش امروز</div>
    <div style="display:flex;align-items:center;gap:10px;border:1.5px solid #DDEDE4;background:#F4FBF7;border-radius:12px;padding:13px 12px">
      <span style="color:var(--green)">${ic('checkCircle',20)}</span>
      <span style="flex:1;text-align:right;font-size:14px;font-weight:700;color:var(--ink)">ثبت دمای بدن</span>
    </div>
    <p class="sub" style="text-align:right;margin:10px 4px 0">این هفته روند خوبی داشتی! به همین منوال ادامه بده.</p>
  </div></div>`;
}

// ---------- reminders ----------
function reminder(title){
  return `<div style="padding:14px 16px 0"><div class="card" style="padding:14px 12px">
    <div style="font-size:14px;font-weight:800;color:var(--ink);text-align:right;margin-bottom:12px">${title}</div>
    <div style="display:flex;align-items:center;gap:10px">
      <span style="font-size:12px;font-weight:700;color:var(--brand);font-variant-numeric:tabular-nums">۱۳:۳۰</span>
      <div style="flex:1;text-align:right">
        <div style="font-size:14px;font-weight:700;color:var(--ink)">دکتر عابدزاده</div>
        <div style="font-size:11px;color:var(--muted);margin-top:2px">متخصص زنان و زایمان</div>
      </div>
      <span class="cbx">${ic('check',14,{stroke:'#fff'})}</span>
    </div>
  </div></div>`;
}

// ---------- smart tip ----------
function smartTip(){
  return `<div style="padding:14px 16px 0"><div class="card" style="padding:14px 12px">
    <div style="font-size:14px;font-weight:800;color:var(--ink);text-align:right;margin-bottom:10px">نکته هوشمند</div>
    <p class="sub" style="text-align:right;margin:0 2px 14px">در فاز لوتئال، بدنت به مراقبت بیشتری نیاز دارد. غلات کامل و سبزیجات را فراموش نکن.</p>
    <div style="display:flex;align-items:center;gap:10px;background:linear-gradient(90deg,#FFF0F7,#F3F0FF);border-radius:12px;padding:12px">
      <span style="color:var(--pink)">${ic('sparkle',20,{fill:'currentColor',sw:0})}</span>
      <span style="flex:1;text-align:right;font-size:12.5px;font-weight:700;color:var(--ink)">هر روز فرصتی تازه برای مراقبت از خودت است.</span>
    </div>
  </div></div>`;
}

// ---------- week summary ----------
function weekSummary(){
  const cards=[
    {l:'روحیه', v:'۸۱٪', ic:'smile', c:'#F06292'},
    {l:'خواب', v:'۶.۵ ساعت', ic:'moon', c:'#7C7CF0'},
    {l:'انرژی', v:'۷۳٪', ic:'zap', c:'#F5A623'},
  ];
  return `<div style="padding:18px 16px 0">
    ${sectionHead('خلاصه هفته','مشاهده کامل')}
    <div style="display:flex;gap:10px">
      ${cards.map(c=>`<div class="card" style="flex:1;padding:14px 8px;display:flex;flex-direction:column;align-items:center;gap:8px">
        <span style="color:${c.c}">${ic(c.ic,24,{stroke:'currentColor'})}</span>
        <span style="font-size:15px;font-weight:800;color:var(--ink)">${c.v}</span>
        <span style="font-size:11px;color:var(--muted);font-weight:600">${c.l}</span>
      </div>`).join('')}
    </div>
    <p class="sub" style="text-align:right;margin:10px 4px 0">این هفته روند خوبی داشتی! به همین منوال ادامه بده.</p>
  </div>`;
}

// ---------- today status (charts) ----------
function spark(color, pts){
  return `<svg width="100%" height="34" viewBox="0 0 90 34" preserveAspectRatio="none"><polyline fill="none" stroke="${color}" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" points="${pts}"/></svg>`;
}
function todayStatus(){
  const cards=[
    {l:'فشار خون', v:'۱۰۲/۷۷', u:'mmHg', c:'#F06292', p:'0,24 15,18 30,22 45,10 60,16 75,8 90,14'},
    {l:'قند خون', v:'۸', u:'mg/dl', c:'#34C77B', p:'0,20 15,24 30,14 45,18 60,10 75,16 90,12'},
    {l:'ضربان قلب', v:'۷۲', u:'bpm', c:'#7C7CF0', p:'0,16 15,10 30,20 45,12 60,22 75,14 90,18'},
  ];
  return `<div style="padding:18px 16px 0">
    ${sectionHead('وضعیت امروز','مشاهده کامل')}
    <div class="scroll-x"><div style="display:flex;gap:10px;justify-content:flex-end;min-width:min-content">
      ${cards.map(c=>`<div class="card" style="width:150px;flex:0 0 150px;padding:12px">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:6px">
          <span style="font-size:9px;font-weight:700;color:var(--green);background:#E7F8EF;border-radius:10px;padding:2px 7px">نرمال</span>
          <span style="font-size:12px;font-weight:700;color:var(--ink)">${c.l}</span>
        </div>
        <div style="text-align:right;margin-bottom:4px"><span style="font-size:18px;font-weight:800;color:var(--ink)">${c.v}</span><span style="font-size:10px;color:var(--muted);margin-right:3px">${c.u}</span></div>
        ${spark(c.c,c.p)}
      </div>`).join('')}
    </div></div>
  </div>`;
}

// ---------- cycle articles ----------
function articles(){
  const arts=[
    {t:'لخته‌های خون در دوران پریود؛ معنی آن چیست؟', m:'۹ دقیقه'},
    {t:'لخته‌های خون در دوران پریود؛ معنی آن چیست؟', m:'۱۳ دقیقه'},
  ];
  return `<div style="padding:18px 16px 0">
    ${sectionHead('بر اساس سیکل فعلی شما')}
    <div style="display:flex;gap:12px">
      ${arts.map(a=>`<div style="flex:1">
        <div style="height:150px;border-radius:14px;background:url('assets/blog1.jpg') center/cover;box-shadow:0 10px 20px -12px rgba(0,0,0,.3)"></div>
        <div style="font-size:12.5px;font-weight:700;color:var(--ink);text-align:right;line-height:1.7;margin-top:8px">${a.t}</div>
        <div style="display:flex;align-items:center;gap:5px;justify-content:flex-end;margin-top:6px;color:var(--muted)">
          ${ic('book',13)}<span style="font-size:10px">${a.m}</span></div>
      </div>`).join('')}
    </div>
    <div style="padding:14px 0 0"><button class="btn btn-primary" style="border-radius:14px">مشاهده بیشتر</button></div>
  </div>`;
}

// ---------- my cycles ----------
function myCycles(){
  return `<div style="padding:16px 16px 0"><div class="card" style="padding:16px 14px">
    ${sectionHead('سیکل‌های من')}
    <div style="display:flex;align-items:center;justify-content:space-between;padding:6px 2px">
      <span style="font-size:13px;font-weight:800;color:var(--brand)">روز ۱</span>
      <span style="font-size:13px;color:var(--muted);font-weight:600">سیکل فعلی</span>
    </div>
    <div style="display:flex;align-items:center;justify-content:space-between;padding:6px 2px">
      <span style="font-size:13px;font-weight:700;color:var(--ink)">۱۹ اکتبر</span>
      <span style="font-size:13px;color:var(--muted);font-weight:600">شروع شده در</span>
    </div>
    <button style="width:100%;margin-top:12px;background:#FFF1F7;border:0;color:var(--brand);border-radius:12px;padding:11px;font-family:inherit;font-weight:700;font-size:13px;cursor:pointer;display:flex;align-items:center;justify-content:center;gap:6px">
      ${ic('plus',16)} ثبت سیکل‌های قبلی</button>
  </div></div>`;
}

// ---------- cycle summary ----------
function cycleSummary(){
  const rows=[['طول سیکل قبلی','۲۸ روز'],['مدت پریود قبلی','۵ روز'],['نوسان طول سیکل','۳ روز']];
  return `<div style="padding:16px 16px 0"><div class="card" style="padding:16px 14px">
    ${sectionHead('خلاصه سیکل')}
    ${rows.map((r,i)=>`<div style="display:flex;align-items:center;justify-content:space-between;padding:12px 2px;${i<2?'border-bottom:1px solid var(--line)':''}">
      <span style="font-size:13px;font-weight:800;color:var(--ink);font-variant-numeric:tabular-nums">${r[1]}</span>
      <span style="font-size:13px;color:var(--muted);font-weight:600">${r[0]}</span>
    </div>`).join('')}
    <div style="padding:14px 0 0"><button class="btn btn-primary" style="border-radius:14px">مشاهده بیشتر</button></div>
  </div></div>`;
}

// ---------- bottom nav ----------
function tabbar(){
  const tabs=[
    {l:'پروفایل',ic:'user'},{l:'تحلیل',ic:'chart'},{fab:1},{l:'تقویم',ic:'calendar'},{l:'امروز',ic:'grid',on:1}
  ];
  return `<div class="tabbar">
    ${tabs.map(t=> t.fab
      ? `<div class="tab" style="flex:0 0 auto"><div class="fab">${ic('plus',26)}</div></div>`
      : `<div class="tab ${t.on?'on':''}">${ic(t.ic,22)}<span>${t.l}</span></div>`).join('')}
  </div>`;
}

// ---------- assemble ----------
SCREENS.home = () => `
  <div class="view" style="background:#EFF2F4">
    <div class="home-grad" style="position:absolute;top:0;left:0;right:0;height:430px"></div>
    <div style="position:relative;z-index:1">${statusBar()}</div>
    <div class="scroll" style="position:relative;z-index:1">
      ${homeHeader()}
      ${weekStrip()}
      ${nextPeriodCard()}
      ${startPeriodBtn()}
      ${phaseRows()}
      ${miniCards()}
      ${recommendations()}
      ${todayTasks()}
      ${challenge()}
      ${reminder('یادآور دکتر')}
      ${reminder('یادآور دارو')}
      ${smartTip()}
      ${weekSummary()}
      ${todayStatus()}
      ${articles()}
      ${myCycles()}
      ${cycleSummary()}
      <div style="height:26px"></div>
    </div>
    ${tabbar()}
  </div>`;
