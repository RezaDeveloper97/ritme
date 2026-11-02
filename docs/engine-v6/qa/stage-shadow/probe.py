#!/usr/bin/env python3
# Per-persona shadow probe (stage only, T-M9-48). Run ON the stage host:
#   scp probe.py root@89.251.8.115:/tmp/ && ssh root@89.251.8.115 python3 /tmp/probe.py 1,2,3,4,5,6,7,8
# Writes /tmp/shadowprobe/p<i>.jsonl and r<i>.txt (the report after persona i). Run personas one at a time
# (sequential requests => no dropped shadow jobs), so the newest samples of one hash are that persona.
# sequential /cycle/date for -365..+30 and /cycle/month for 14 back + 1 fwd,
# saving the served v1 payload subset, then the newest shadow samples of that persona's hash.
import json, subprocess, sys, time, datetime as dt, urllib.request, urllib.error
OUT='/tmp/shadowprobe'; subprocess.run(['mkdir','-p',OUT])
BASE=subprocess.check_output(['docker','inspect','ritme-stage-backend-go-1','--format','{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}']).decode().split()[0]
BASE=f'http://{BASE}:80'
def sql(q): subprocess.run(['docker','exec','-i','ritme-stage-mysql-1','sh','-c','mariadb -uroot -p"$MYSQL_ROOT_PASSWORD" ritme_stage'],input=q.encode(),check=True)
def call(path,tok=None,body=None,method='GET'):
    h={'Accept':'application/json','Content-Type':'application/json','Accept-Language':'fa'}
    if tok: h['Authorization']='Bearer '+tok
    for a in range(6):
        r=urllib.request.Request(BASE+path,method=method,headers=h,data=json.dumps(body).encode() if body is not None else None)
        try:
            with urllib.request.urlopen(r,timeout=30) as x: return x.status,json.loads(x.read() or b'null')
        except urllib.error.HTTPError as e:
            if e.code==429: time.sleep(5*(a+1)); continue
            return e.code,None
    return 429,None
def login(i):
    m='09977%06d'%i
    sql(f"INSERT INTO otp_verifications(mobile,code,expires_at,attempts,created_at,updated_at) VALUES('{m}','4321',NOW()+INTERVAL 1 DAY,0,NOW(),NOW());")
    s,j=call('/api/v1/auth/verify-otp',body={'mobile':m,'code':'4321'},method='POST'); d=j.get('data',j); return d.get('token') or d.get('access_token')
today=dt.date.today()
for i in [int(x) for x in sys.argv[1].split(',')]:
    tok=login(i); f=open(f'{OUT}/p{i}.jsonl','w')
    for o in range(-365,31):
        d=(today+dt.timedelta(days=o)).isoformat(); s,j=call(f'/api/v1/cycle/date/{d}',tok)
        dd=(j or {}).get('data',{}) if isinstance(j,dict) else {}
        f.write(json.dumps({'kind':'day','offset':o,'status':s,'calculation':dd.get('calculation'),'cycle_view':dd.get('cycle_view')})+'\n')
        time.sleep(0.03)
    for k in range(-1,14):
        y=today.year*12+today.month-1-k; s,j=call(f'/api/v1/cycle/month/{y//12}/{y%12+1}',tok)
        f.write(json.dumps({'kind':'month','y':y//12,'m':y%12+1,'status':s,'data':(j or {}).get('data') if isinstance(j,dict) else None})+'\n')
        time.sleep(0.3)
    f.close(); time.sleep(3)
    rep=subprocess.check_output(['docker','exec','ritme-stage-backend-go-1','/app/api','engine-shadow-report','-n','0']).decode()
    open(f'{OUT}/r{i}.txt','w').write(rep); print(i,'done',flush=True)
