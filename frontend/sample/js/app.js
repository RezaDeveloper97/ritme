// ---- router ----
window.SCREENS = window.SCREENS || {};
let navStack = [];
let current = null;

function render(id){
  const app = document.getElementById('app');
  const fn = window.SCREENS[id];
  if(!fn){ console.warn('no screen', id); return; }
  app.innerHTML = `<div class="view enter">${fn()}</div>`;
  current = id;
  // run optional onMount hook
  if(window.MOUNT && window.MOUNT[id]) window.MOUNT[id]();
  // reset scroll
  const sc = app.querySelector('.scroll');
  if(sc) sc.scrollTop = 0;
}

function navigate(id){
  if(current) navStack.push(current);
  render(id);
}
function replace(id){ render(id); }
function back(){
  const prev = navStack.pop();
  if(prev) render(prev);
}

window.MOUNT = window.MOUNT || {};

// boot
document.addEventListener('DOMContentLoaded', ()=> render('splash'));
