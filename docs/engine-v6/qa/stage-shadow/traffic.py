#!/usr/bin/env python3
"""Drive QA-persona traffic through the cycle-engine endpoints on STAGE (CYCLE_ENGINE=shadow).
Runs ON the stage host (root@89.251.8.115); talks straight to the ritme-stage-backend-go container (no nginx gate).
Only touches ritme-stage-mysql-1 (inserts a known OTP row for the QA mobiles 099770000NN).
Run:  ssh root@89.251.8.115 python3 - [--days N] [--only 1,2] < traffic.py
"""
import json, subprocess, sys, time, datetime as dt, urllib.request, urllib.error, collections
from concurrent.futures import ThreadPoolExecutor
DAYS = 365; FWD = 30; ONLY = None
a = sys.argv[1:]
if '--days' in a: DAYS = int(a[a.index('--days')+1])
if '--only' in a: ONLY = [int(x) for x in a[a.index('--only')+1].split(',')]
BASE = subprocess.check_output(['docker','inspect','ritme-stage-backend-go-1','--format','{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}']).decode().split()[0]
BASE = f'http://{BASE}:80'
PERSONAS = ['regular','late 8','late 15','pill','postpartum','onboarding-only','stale anchor','skip']
def sql(q):
    subprocess.run(['docker','exec','-i','ritme-stage-mysql-1','sh','-c','mariadb -uroot -p"$MYSQL_ROOT_PASSWORD" ritme_stage'],input=q.encode(),check=True)
def call(method, path, tok=None, body=None):
    h = {'Accept':'application/json','Content-Type':'application/json','Accept-Language':'fa'}
    if tok: h['Authorization'] = 'Bearer '+tok
    for attempt in range(4):
        req = urllib.request.Request(BASE+path, method=method, headers=h, data=json.dumps(body).encode() if body is not None else None)
        try:
            with urllib.request.urlopen(req, timeout=30) as r: return r.status, json.loads(r.read() or b'null')
        except urllib.error.HTTPError as e:
            if e.code == 429: time.sleep(5*(attempt+1)); continue
            try: return e.code, json.loads(e.read())
            except Exception: return e.code, None
        except Exception as e: return 0, str(e)
    return 429, None
def login(i):
    m = f'0997700000{i:02d}'[-11:] if False else '09977%06d' % i
    sql(f"INSERT INTO otp_verifications(mobile,code,expires_at,attempts,created_at,updated_at) VALUES('{m}','4321',NOW()+INTERVAL 1 DAY,0,NOW(),NOW());")
    s, j = call('POST','/api/v1/auth/verify-otp',body={'mobile':m,'code':'4321'})
    if s != 200: raise SystemExit(f'login {m}: {s} {j}')
    d = j.get('data', j); return d.get('token') or d.get('access_token')
seen = set()
stats = collections.defaultdict(lambda: collections.Counter())
def hit(name, tok, path):
    s, _ = call('GET', path, tok); stats[name][s] += 1
    if s != 200 and path.split('/')[-1] not in seen: seen.add(path.split('/')[-1]); print('  non-200', s, path, flush=True)
    return s
today = dt.date(2026,11,2) if False else dt.date.today()
for i, name in enumerate(PERSONAS, 1):
    if ONLY and i not in ONLY: continue
    tok = login(i)
    paths = ['/api/v1/cycle/status','/api/v1/cycle/today','/api/v1/home','/api/v1/home/cycle-overview','/api/v1/fertility/today','/api/v1/fertility/insights',
             '/api/v1/fertility/bbt','/api/v1/analysis/cycle','/api/v1/analysis/summary','/api/v1/analysis/period','/api/v1/analysis/fertility','/api/v1/cycle/history',
             '/api/v1/cycle/period/status','/api/v1/messages/feed','/api/v1/profile/cycle-settings']
    for p in ['menstrual','early_follicular','mid_follicular','high_fertility','ovulation_likely','early_luteal','mid_luteal']: paths.append(f'/api/v1/cycle/phase-content/{p}')
    for k in range(0, 14):  # months back
        y = (today.year*12 + today.month-1 - k); paths.append(f'/api/v1/cycle/month/{y//12}/{y%12+1}')
    for k in (-1, -2):      # future months
        pass
    days = [(today + dt.timedelta(days=o)).isoformat() for o in range(-DAYS, FWD+1)]
    paths += [f'/api/v1/cycle/date/{d}' for d in days] + [f'/api/v1/fertility/days/{d}' for d in days[-60:]]
    # two passes: second pass exercises the engine cache / etag path as well
    with ThreadPoolExecutor(4) as ex:
        for _ in range(2): list(ex.map(lambda p: hit(f'{i}:{name}', tok, p), paths))
    if i == 1 or not ONLY or True:
        pass
    print(f'{i} {name}: {len(paths)*2} requests', dict(stats[f'{i}:{name}']), flush=True)

# companion (partner) view of every persona: /companion/home + per-link cycle section
if not ONLY or 9 in ONLY:
    tok = login(9)
    s_, j = call('GET','/api/v1/companions/links', tok)
    ids = []
    if s_ == 200:
        d = j.get('data', j); ids = [x['id'] for x in (d if isinstance(d, list) else d.get('links', d.get('data', [])))]
    for _ in range(3):
        hit('9:partner', tok, '/api/v1/companion/home')
        for lid in ids: hit('9:partner', tok, f'/api/v1/companions/links/{lid}/sections/cycle')
    print('9 partner: links', len(ids), dict(stats['9:partner']), flush=True)
