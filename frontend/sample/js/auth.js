// ===== AUTH FLOW =====

// 1) Splash
SCREENS.splash = () => `
  <div class="view" style="position:relative;height:100%;background:linear-gradient(160deg,#E91E63 0%,#C2399E 55%,#A56BD6 100%);color:#fff;cursor:pointer"
       onclick="navigate('signup')">
    ${statusBar('light')}
    <div style="position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:14px">
      <div style="width:84px;height:84px;border-radius:26px;background:rgba(255,255,255,.16);backdrop-filter:blur(6px);display:flex;align-items:center;justify-content:center;box-shadow:0 18px 40px -12px rgba(0,0,0,.4)">
        ${ic('sparkle',46,{fill:'#fff',sw:0})}
      </div>
      <div style="font-weight:900;font-size:34px;letter-spacing:1px;margin-top:6px">ریـتمی</div>
      <div style="font-size:13px;opacity:.92;font-weight:500">همراه سلامتی شما</div>
    </div>
    <div style="position:absolute;bottom:30px;left:0;right:0;display:flex;flex-direction:column;align-items:center;gap:10px">
      <span style="animation:spin 1s linear infinite;display:inline-flex;opacity:.95">${ic('loader',24)}</span>
      <span style="font-size:10px;opacity:.8">تمامی حقوق محفوظ است ۱۴۰۵</span>
    </div>
  </div>
  <style>@keyframes spin{to{transform:rotate(360deg)}}</style>`;
MOUNT.splash = () => { clearTimeout(window._sp); window._sp = setTimeout(()=>{ if(current==='splash') navigate('signup'); }, 2200); };

// 2) Sign up — phone number
SCREENS.signup = () => `
  <div class="view" style="background:#fff">
    ${statusBar()}
    <div class="hdr">
      <span></span>
      <span class="iconbtn" onclick="navigate('splash')" style="cursor:pointer">${ic('chevronRight',24)}</span>
    </div>
    <div class="scroll" style="padding:18px 22px">
      <div style="display:flex;align-items:center;justify-content:flex-end;gap:8px;margin-top:8px">
        <span class="titr" style="font-size:18px">همین امروز عضو ریتمی شو</span>
        <span style="color:var(--pink)">${ic('sparkle',22,{fill:'currentColor',sw:0})}</span>
      </div>
      <p class="sub" style="text-align:right;margin:10px 0 30px">برای ورود یا ثبت‌نام شماره‌ی موبایل خود را وارد کنید</p>

      <label class="lbl">شماره موبایل</label>
      <div class="field">
        <input id="ph" inputmode="numeric" maxlength="11" placeholder="۰۹۱۲ ۳۴۵ ۶۷۸۹" oninput="onPhone(this)" />
        <span style="color:#A9B2BC">${ic('user',18)}</span>
      </div>

      <label style="display:flex;align-items:flex-start;gap:10px;margin-top:26px;cursor:pointer;justify-content:flex-end" onclick="toggleTerms(this)">
        <span class="sub" style="text-align:right;flex:1">ثبت‌نام به منزله‌ی قبول <b style="color:var(--brand);font-weight:700">قوانین و مقررات</b> و حریم خصوصی است.</span>
        <span class="cbx" data-on="0">${ic('check',14,{stroke:'#fff'})}</span>
      </label>
    </div>
    <div style="padding:14px 16px 8px">
      <button id="getCode" class="btn btn-primary" disabled onclick="goOtp()">دریافت کد تأیید</button>
    </div>
    ${homeIndicator()}
  </div>`;

function onPhone(el){
  el.value = el.value.replace(/[^0-9۰-۹]/g,'');
  store.phone = el.value;
  refreshSignup();
}
function toggleTerms(row){
  const cb = row.querySelector('.cbx');
  const on = cb.getAttribute('data-on')==='1';
  cb.setAttribute('data-on', on?'0':'1');
  cb.classList.toggle('on', !on);
  store.terms = !on;
  refreshSignup();
}
function refreshSignup(){
  const ok = (store.phone||'').replace(/\D/g,'').length>=10 && store.terms;
  const b = document.getElementById('getCode'); if(b) b.disabled = !ok;
}
function goOtp(){ navigate('otp'); }

// 3) OTP
SCREENS.otp = () => `
  <div class="view" style="background:#fff">
    ${statusBar()}
    <div class="hdr">
      <span></span>
      <span class="iconbtn" onclick="back()" style="cursor:pointer">${ic('chevronRight',24)}</span>
    </div>
    <div class="scroll" style="padding:18px 22px;display:flex;flex-direction:column;align-items:center">
      <div style="width:100%;text-align:right">
        <div class="titr">کد یکبار مصرف 🔐</div>
        <p class="sub" style="margin:12px 0 0">کد یکبار مصرف به شماره‌ی ${faNum(store.phone||'۰۹۱۲۳۴۵۶۷۸۹')} ارسال شد</p>
      </div>
      <button class="btn-soft" style="width:auto;padding:0 14px;margin:16px 0 30px;display:flex;align-items:center;gap:6px" onclick="back()">
        ${ic('pencil',15)} ویرایش شماره موبایل
      </button>

      <div dir="ltr" style="display:flex;gap:12px;justify-content:center">
        ${[0,1,2,3].map(i=>`<input id="o${i}" class="otp-box" inputmode="numeric" maxlength="1" oninput="otpType(${i})" onkeydown="otpKey(event,${i})" />`).join('')}
      </div>

      <div class="sub" style="margin-top:26px;display:flex;align-items:center;gap:6px">
        <span id="rs">ارسال مجدد کد تا <b id="t" style="color:var(--ink)">۰۰:۵۹</b></span>
      </div>
    </div>
    <div style="padding:14px 16px 8px">
      <button id="verify" class="btn btn-primary" disabled onclick="navigate('name')">تأیید کد</button>
    </div>
    ${homeIndicator()}
  </div>
  <style>
    .otp-box{ width:62px;height:64px;border:1.5px solid #E6EAF0;border-radius:16px;text-align:center;
      font-size:26px;font-weight:800;font-family:'Vazirmatn';color:var(--ink);outline:none;background:#fff; }
    .otp-box:focus{ border-color:var(--pink);box-shadow:0 0 0 4px rgba(251,100,182,.14); }
    .otp-box.filled{ border-color:var(--brand);background:#FFF1F7; }
  </style>`;
MOUNT.otp = () => { setTimeout(()=>{ const f=document.getElementById('o0'); if(f) f.focus(); },120); otpTimer(); };

function otpType(i){
  const el = document.getElementById('o'+i);
  el.value = el.value.replace(/[^0-9۰-۹]/g,'').slice(0,1);
  el.classList.toggle('filled', !!el.value);
  if(el.value && i<3){ document.getElementById('o'+(i+1)).focus(); }
  let code=''; for(let k=0;k<4;k++) code += (document.getElementById('o'+k).value||'');
  store.otp = code;
  document.getElementById('verify').disabled = code.length<4;
}
function otpKey(e,i){
  if(e.key==='Backspace' && !e.target.value && i>0) document.getElementById('o'+(i-1)).focus();
}
function otpTimer(){
  clearInterval(window._ot); let s=59;
  window._ot = setInterval(()=>{
    s--; const t=document.getElementById('t');
    if(!t){ clearInterval(window._ot); return; }
    if(s<=0){ clearInterval(window._ot); document.getElementById('rs').innerHTML='<b style="color:var(--brand);cursor:pointer">ارسال مجدد کد</b>'; return; }
    t.textContent = '۰۰:'+faNum(String(s).padStart(2,'0'));
  },1000);
}
