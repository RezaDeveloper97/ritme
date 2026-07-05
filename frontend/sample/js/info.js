// ===== INFORMATION FLOW =====

// shared shell for an onboarding step
function infoShell(step, title, sub, content, btnLabel, nextId, btnId){
  return `
  <div class="view" style="background:#fff">
    ${statusBar()}
    ${infoHeader(step)}
    <div class="scroll" style="padding:8px 22px 0;display:flex;flex-direction:column">
      <div style="text-align:right;margin:6px 0 6px">
        <div class="titr">${title}</div>
        ${sub?`<p class="sub" style="margin:10px 0 0">${sub}</p>`:''}
      </div>
      <div style="flex:1;display:flex;flex-direction:column;justify-content:center;padding:8px 0">
        ${content}
      </div>
    </div>
    <div style="padding:14px 16px 8px">
      <button class="btn btn-primary" ${btnId?`id="${btnId}"`:''} onclick="navigate('${nextId}')">${btnLabel}</button>
    </div>
    ${homeIndicator()}
  </div>`;
}

// --- Step 1: Name
SCREENS.name = () => infoShell(1,'اسم شما چیست؟','با چه نامی شما را صدا کنیم؟',`
  <div style="margin-top:8px">
    <label class="lbl">نام و نام خانوادگی</label>
    <div class="field">
      <input id="nm" placeholder="مثلاً ایسابلا" value="${store.name||''}" oninput="store.name=this.value;document.getElementById('nmBtn').disabled=!this.value.trim()" />
      ${ic('pencil',18,{stroke:'#A9B2BC'})}
    </div>
  </div>`,'ادامه','gender','nmBtn');
MOUNT.name = () => { const b=document.getElementById('nmBtn'); if(b) b.disabled=!(store.name||'').trim(); };

// --- Step 2: Gender
SCREENS.gender = () => infoShell(2,'جنسیت شما چیست؟','ما را برای شناخت بهترتان یاری کنید',`
  <div style="display:flex;gap:18px;justify-content:center;margin-top:6px">
    ${genderCard('female','زن','#FB64B6','#FFF0F7','<path d="M12 3a6 6 0 1 0 0 12 6 6 0 0 0 0-12zM12 15v6M9 19h6"/>')}
    ${genderCard('male','مرد','#3B9EF0','#EEF6FF','<circle cx="10" cy="14" r="6"/><path d="M14.5 9.5L20 4M20 4h-5M20 4v5"/>')}
  </div>
  <div style="display:flex;justify-content:center;margin-top:26px">
    <button class="btn-soft" style="width:auto;padding:0 20px" onclick="pickGender('none');navigate('birthday')">ترجیح می‌دهم نگویم</button>
  </div>`,'ادامه','birthday');

function genderCard(val,label,color,bg,path){
  const on = store.gender===val;
  return `<button onclick="pickGender('${val}')" data-g="${val}"
    style="width:108px;height:140px;border-radius:20px;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:12px;
    border:2px solid ${on?color:'#EBEEF2'};background:${on?bg:'#fff'};transition:.18s">
    <span style="width:60px;height:60px;border-radius:50%;background:${bg};display:flex;align-items:center;justify-content:center;color:${color}">
      <svg viewBox="0 0 24 24" width="30" height="30" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${path}</svg></span>
    <span style="font-weight:700;font-size:15px;color:${on?color:'var(--ink)'}">${label}</span>
  </button>`;
}
function pickGender(v){ store.gender=v; render('gender'); }

// --- Step 3: Birthday (3 wheels)
const FA_MONTHS = ['فروردین','اردیبهشت','خرداد','تیر','مرداد','شهریور','مهر','آبان','آذر','دی','بهمن','اسفند'];
SCREENS.birthday = () => infoShell(3,'تاریخ تولد شما؟','برای شناخت بهتر شما، ما را همراهی کنید',`
  <div style="display:flex;gap:8px;justify-content:center;margin-top:4px;position:relative">
    <div class="wheel-band"></div>
    ${wheelHtml('wD', Array.from({length:31},(_,i)=>faNum(i+1)), store.birth.d-1, 56)}
    ${wheelHtml('wM', FA_MONTHS, store.birth.m, 112)}
    ${wheelHtml('wY', Array.from({length:60},(_,i)=>faNum(1340+i)), store.birth.y-1340, 80)}
  </div>`,'ادامه','weight');
MOUNT.birthday = () => {
  initWheel('wD', i=>store.birth.d=i+1);
  initWheel('wM', i=>store.birth.m=i);
  initWheel('wY', i=>store.birth.y=1340+i);
};

// --- Step 4: Weight (segmented + ruler)
SCREENS.weight = () => infoShell(4,'وزن شما چقدر است؟','برای شناخت بهتر شما، ما را همراهی کنید',`
  <div style="display:flex;justify-content:center;margin-bottom:30px">
    <div class="seg" style="width:170px">
      <button class="${store.weightUnit==='lb'?'on':''}" onclick="setUnit('weight','lb')">lb</button>
      <button class="${store.weightUnit==='kg'?'on':''}" onclick="setUnit('weight','kg')">kg</button>
    </div>
  </div>
  ${rulerHtml('rW', 30, 150, store.weight, store.weightUnit, 'وزن')}
`,'ادامه','height');
MOUNT.weight = () => initRuler('rW', 30, 150, store.weight, v=>store.weight=v);

// --- Step 5: Height
SCREENS.height = () => infoShell(5,'قد شما چقدر است؟','برای شناخت بهتر شما، ما را همراهی کنید',`
  <div style="display:flex;justify-content:center;margin-bottom:30px">
    <div class="seg" style="width:170px">
      <button class="${store.heightUnit==='ft'?'on':''}" onclick="setUnit('height','ft')">ft</button>
      <button class="${store.heightUnit==='cm'?'on':''}" onclick="setUnit('height','cm')">cm</button>
    </div>
  </div>
  ${rulerHtml('rH', 120, 210, store.height, store.heightUnit, 'قد')}
`,'ادامه','periodLen');
MOUNT.height = () => initRuler('rH', 120, 210, store.height, v=>store.height=v);

function setUnit(which,u){ store[which+'Unit']=u; render(which); }

// --- Step 6: Period duration (vertical wheel of days)
SCREENS.periodLen = () => infoShell(6,'پریود شما معمولاً چند روز طول می‌کشد؟','برای شناخت بهتر شما، ما را همراهی کنید',`
  <div style="display:flex;justify-content:center;position:relative">
    <div class="wheel-band" style="width:150px;left:50%;transform:translateX(-50%)"></div>
    ${wheelHtml('wP', Array.from({length:10},(_,i)=>faNum(i+1)+' روز'), store.periodLen-1, 150)}
  </div>`,'ادامه','cycleLen');
MOUNT.periodLen = () => initWheel('wP', i=>store.periodLen=i+1);

// --- Step 7: Cycle length / last period calendar
SCREENS.cycleLen = () => infoShell(7,'چه زمانی آخرین پریود شما آغاز شد؟','برای شناخت بهتر شما، ما را همراهی کنید',
  calendarHtml(),'پایان','settingUp');
MOUNT.cycleLen = () => {};

// ---------- Setting up loader ----------
SCREENS.settingUp = () => `
  <div class="view" style="background:#fff">
    ${statusBar()}
    <div class="scroll" style="padding:80px 22px 0;display:flex;flex-direction:column;align-items:center;text-align:center">
      <div class="titr" style="font-size:19px;line-height:1.7">ما در حال تنظیم تقویم<br>شخصی‌سازی‌شده‌ی شما هستیم…</div>
      <p class="sub" style="margin:12px 0 50px">برای شناخت بهتر شما، ما را همراهی کنید</p>
      <div style="position:relative;width:200px;height:200px">
        <svg width="200" height="200" viewBox="0 0 200 200" style="transform:rotate(-90deg)">
          <circle cx="100" cy="100" r="88" fill="none" stroke="#E6EAF0" stroke-width="13"/>
          <circle id="ring" cx="100" cy="100" r="88" fill="none" stroke="var(--pink)" stroke-width="13" stroke-linecap="round"
            stroke-dasharray="553" stroke-dashoffset="553"/>
        </svg>
        <div style="position:absolute;inset:0;display:flex;align-items:center;justify-content:center;font-size:40px;font-weight:800;color:var(--slate)"><span id="pct">۰٪</span></div>
      </div>
    </div>
    <div style="padding:20px;text-align:center"><span class="sub">چند ثانیه صبر کنید …</span></div>
  </div>`;
MOUNT.settingUp = () => {
  let p=0; const ring=document.getElementById('ring'), pct=document.getElementById('pct'), C=553;
  clearInterval(window._su);
  window._su = setInterval(()=>{
    p+=2; if(p>100){ clearInterval(window._su); replace('home'); return; }
    if(ring){ ring.style.strokeDashoffset = C*(1-p/100); pct.textContent=faNum(p)+'٪'; }
  }, 45);
};
