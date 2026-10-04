
/* ریتمی — اسکریپت سایت: منوی موبایل + محاسبه‌گرهای شمسی. بدون وابستگی. */
(function(){
  'use strict';
  // ---- mobile menu
  document.querySelectorAll('.rt-burger').forEach(function(b){
    var menu=document.getElementById('rt-mobile-menu');
    b.addEventListener('click',function(){
      var open=b.getAttribute('aria-expanded')==='true';
      b.setAttribute('aria-expanded',open?'false':'true'); b.setAttribute('aria-label',open?'باز کردن منو':'بستن منو');
      if(!menu) return; if(open){menu.removeAttribute('data-open');menu.hidden=true;}else{menu.hidden=false;menu.setAttribute('data-open','');}
    });
  });
  // ---- Persian digits
  var FA='۰۱۲۳۴۵۶۷۸۹', AR='٠١٢٣٤٥٦٧٨٩';
  function toEn(s){return String(s).replace(/[۰-۹]/g,function(d){return FA.indexOf(d);}).replace(/[٠-٩]/g,function(d){return AR.indexOf(d);});}
  function toFa(s){return String(s).replace(/\d/g,function(d){return FA[+d];});}
  var MONTHS=['فروردین','اردیبهشت','خرداد','تیر','مرداد','شهریور','مهر','آبان','آذر','دی','بهمن','اسفند'];
  // ---- Jalali <-> Gregorian (algorithm: jalaali-js, Behrang Noruzi Niya)
  function div(a,b){return ~~(a/b);} function mod(a,b){return a-~~(a/b)*b;}
  function jalCal(jy){var bl=[-61,9,38,199,426,686,756,818,1111,1181,1210,1635,2060,2097,2192,2262,2324,2394,2456,3178],bn=bl.length,gy=jy+621,leapJ=-14,jp=bl[0],jm,jump,leap,leapG,march,n,i;
    for(i=1;i<bn;i+=1){jm=bl[i];jump=jm-jp;if(jy<jm)break;leapJ=leapJ+div(jump,33)*8+div(mod(jump,33),4);jp=jm;}
    n=jy-jp;leapJ=leapJ+div(n,33)*8+div(mod(n,33)+3,4);if(mod(jump,33)===4&&jump-n===4)leapJ+=1;
    leapG=div(gy,4)-div((div(gy,100)+1)*3,4)-150;march=20+leapJ-leapG;
    if(jump-n<6)n=n-jump+div(jump+4,33)*33;leap=mod(mod(n+1,33)-1,4);if(leap===-1)leap=4;return{leap:leap,gy:gy,march:march};}
  function g2d(gy,gm,gd){var d=div((gy+div(gm-8,6)+100100)*1461,4)+div(153*mod(gm+9,12)+2,5)+gd-34840408;d=d-div(div(gy+100100+div(gm-8,6),100)*3,4)+752;return d;}
  function d2g(jdn){var j=4*jdn+139361631;j=j+div(div(4*jdn+183187720,146097)*3,4)*4-3908;var i=div(mod(j,1461),4)*5+308;var gd=div(mod(i,153),5)+1,gm=mod(div(i,153),12)+1,gy=div(j,1461)-100100+div(8-gm,6);return{gy:gy,gm:gm,gd:gd};}
  function j2d(jy,jm,jd){var r=jalCal(jy);return g2d(r.gy,3,r.march)+(jm-1)*31-div(jm,7)*(jm-7)+jd-1;}
  function d2j(jdn){var gy=d2g(jdn).gy,jy=gy-621,r=jalCal(jy),jdn1f=g2d(gy,3,r.march),jd,jm,k;k=jdn-jdn1f;
    if(k>=0){if(k<=185){jm=1+div(k,31);jd=mod(k,31)+1;return{jy:jy,jm:jm,jd:jd};}else{k-=186;}}else{jy-=1;k+=179;if(r.leap===1)k+=1;}
    jm=7+div(k,30);jd=mod(k,30)+1;return{jy:jy,jm:jm,jd:jd};}
  function todayJdn(){var d=new Date();return g2d(d.getFullYear(),d.getMonth()+1,d.getDate());}
  function fmt(j){return toFa(j.jd)+' '+MONTHS[j.jm-1]+' '+toFa(j.jy);}
  // parse "۱۴۰۵/۰۲/۱۲", "1405-2-12", "۱۲ اردیبهشت ۱۴۰۵"
  function parseJ(s){s=toEn(s).trim();var m=s.match(/^(\d{4})[\/\-.](\d{1,2})[\/\-.](\d{1,2})$/);if(m)return{jy:+m[1],jm:+m[2],jd:+m[3]};
    m=s.match(/^(\d{1,2})\s+(\S+)\s+(\d{4})$/);if(m){var mi=MONTHS.indexOf(m[2]);if(mi>=0)return{jy:+m[3],jm:mi+1,jd:+m[1]};}return null;}
  function valid(j){return j&&j.jy>1300&&j.jy<1500&&j.jm>=1&&j.jm<=12&&j.jd>=1&&j.jd<=31;}
  // ---- calculators
  document.querySelectorAll('form.rt-calc').forEach(function(f){
    var err=document.createElement('div');err.className='rt-err';f.insertBefore(err,f.querySelector('button[type=submit]'));
    f.addEventListener('submit',function(e){
      e.preventDefault();
      var lmp=parseJ(f.querySelector('[name=lmp]').value),cyc=parseInt(toEn(f.querySelector('[name=cycle]').value).replace(/\D/g,''),10)||28;
      var rv=f.querySelector('.rt-res-v'),rs=f.querySelector('.rt-res-s');
      if(!valid(lmp)){f.setAttribute('data-error','');err.textContent='تاریخ را مثل ۱۴۰۵/۰۲/۱۲ یا «۱۲ اردیبهشت ۱۴۰۵» بنویس.';return;}
      if(cyc<21||cyc>45){f.setAttribute('data-error','');err.textContent='طول چرخه معمولاً بین ۲۱ تا ۴۵ روز است؛ اگر خارج از این بازه است، این ابزار مناسب نیست.';return;}
      f.removeAttribute('data-error');err.textContent='';
      var base=j2d(lmp.jy,lmp.jm,lmp.jd),today=todayJdn();
      if(f.dataset.kind==='due'){
        var due=base+280+(cyc-28);var wk=Math.floor((today-base)/7),dy=(today-base)%7;
        rv.textContent=fmt(d2j(due));
        var txt=(today>=base&&today-base<300)?('الان حدود هفته '+toFa(wk)+(dy?(' و '+toFa(dy)+' روز'):'')+' بارداری هستی · '):'';
        rs.textContent=txt+'بازه معمول: '+fmt(d2j(due-14))+' تا '+fmt(d2j(due+14));
      }else{
        var ov=base+(cyc-14),ws=ov-5,we=ov,next=base+cyc;
        rv.textContent=fmt(d2j(ws))+' تا '+fmt(d2j(we));
        rs.textContent='تخمک‌گذاری احتمالی: حدود '+fmt(d2j(ov))+' · پریود بعدی احتمالاً '+fmt(d2j(next));
      }
    });
  });
})();
