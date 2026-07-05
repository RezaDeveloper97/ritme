// ---- shared UI bits + app state ----
window.SCREENS = window.SCREENS || {};
window.MOUNT = window.MOUNT || {};
const store = {
  phone:'', otp:'', name:'', gender:'', birth:{d:25,m:9,y:1373},
  weightUnit:'kg', weight:60.0, heightUnit:'cm', height:165.0,
  periodLen:5, cycleLen:28, lastPeriod:30,
};

// status bar — tone: 'dark' (default, dark glyphs) | 'light'
function statusBar(tone){
  const c = tone === 'light' ? '#fff' : '#11202F';
  return `<div class="statusbar" style="color:${c}">
    <span class="time">9:41</span>
    <span class="si">
      <svg width="18" height="12" viewBox="0 0 18 12" fill="${c}"><rect x="0" y="7" width="3" height="5" rx="1"/><rect x="5" y="4.5" width="3" height="7.5" rx="1"/><rect x="10" y="2" width="3" height="10" rx="1"/><rect x="15" y="0" width="3" height="12" rx="1"/></svg>
      <svg width="17" height="12" viewBox="0 0 17 12" fill="none" stroke="${c}" stroke-width="1.6" stroke-linecap="round"><path d="M1 4.2A11 11 0 0 1 16 4.2"/><path d="M3.5 6.6A7.4 7.4 0 0 1 13.5 6.6"/><path d="M6 9A3.8 3.8 0 0 1 11 9"/><circle cx="8.5" cy="11" r=".6" fill="${c}" stroke="none"/></svg>
      <svg width="26" height="13" viewBox="0 0 26 13" fill="none"><rect x="1" y="1" width="21" height="11" rx="3" stroke="${c}" stroke-opacity=".5"/><rect x="3" y="3" width="16" height="7" rx="1.5" fill="${c}"/><rect x="23" y="4" width="2" height="5" rx="1" fill="${c}" fill-opacity=".5"/></svg>
    </span>
  </div>`;
}

function homeIndicator(dark){
  return `<div style="height:22px;display:flex;align-items:center;justify-content:center">
    <div style="width:130px;height:5px;border-radius:99px;background:${dark?'#fff':'#11202F'};opacity:.85"></div></div>`;
}

// brand wordmark
function logo(size){
  size = size || 22;
  return `<div style="display:flex;align-items:center;gap:7px;justify-content:center">
    <span style="font-weight:900;font-size:${size}px;color:var(--brand-deep);letter-spacing:.5px">ریتمی</span>
    <span style="color:var(--brand)">${ic('sparkle', size, {fill:'currentColor', sw:0})}</span>
  </div>`;
}

// info-flow header with back chevron + step counter
function infoHeader(step){
  return `<div class="hdr">
    <span class="iconbtn" onclick="back()" style="cursor:pointer">${ic('chevronRight',24)}</span>
    <span class="stepcount">${faNum(step)}<span style="opacity:.5"> / ۷</span></span>
  </div>`;
}

// persian digits
const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
function faNum(n){ return String(n).replace(/[0-9]/g, d=>FA[+d]); }
