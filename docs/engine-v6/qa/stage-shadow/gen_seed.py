#!/usr/bin/env python3
r"""Stage QA personas (T-M9-48): prints the seed SQL for users 09977000001..08 from the engine goldens' personas,
shifted so the persona's `today` lands on TODAY. Stage only.
Run (from the repo root):  python3 docs/engine-v6/qa/stage-shadow/gen_seed.py > /tmp/seed.sql
  ssh root@89.251.8.115 "docker exec -i ritme-stage-mysql-1 sh -c 'mariadb -uroot -p\"\$MYSQL_ROOT_PASSWORD\" ritme_stage'" < /tmp/seed.sql
Persona 9 (QA partner linked to 1..8 with a cycle grant): seed_companion.sql the same way.
"""
import json,datetime as dt,sys,os
D=os.path.join(os.path.dirname(os.path.abspath(__file__)),'../../../../backend-go/internal/cycle/engine/testdata/personas/')
TODAY=dt.date(2026,11,2)
P=[('regular','regular_28','09977000001'),('late 8','late_8','09977000002'),('late 15','late_15','09977000003'),('pill','pill','09977000004'),
('postpartum','postpartum_breastfeeding','09977000005'),('onboarding-only','onboarding_fresh','09977000006'),('stale anchor','onboarding_stale','09977000007'),('skip','skip','09977000008')]
o=[]
for i,(label,f,m) in enumerate(P,1):
    d=json.load(open(D+f+'.json'));sh=TODAY-dt.date.fromisoformat(d['today']);inp=d['input']
    S_=lambda s:(dt.date.fromisoformat(s)+sh).isoformat()
    uid=f"(SELECT id FROM users WHERE mobile='{m}')"
    o.append(f"INSERT IGNORE INTO users(name,mobile,mobile_verified_at,created_at,updated_at) VALUES('QA {label}','{m}',NOW(),NOW(),NOW());")
    dec=inp.get('declared',{})
    lps=S_(dec['last_period_start']) if dec.get('last_period_start') else None
    q=lambda v:'NULL' if v is None else (f"'{v}'" if isinstance(v,str) else str(v))
    pers=inp.get('periods',[])
    lp=pers[-1]['start'] if pers else None
    prof_lp=S_(lp) if lp else lps
    o.append(f"INSERT INTO user_profiles(user_id,period_duration,cycle_duration,last_period_start,onboarding_last_period_start,user_goal,created_at,updated_at) SELECT id,{q(dec.get('period_length') or (5 if pers else None))},{q(dec.get('cycle_length') or (28 if pers else None))},{q(prof_lp)},{q(lps)},'non_ttc',NOW(),NOW() FROM users WHERE mobile='{m}' AND NOT EXISTS(SELECT 1 FROM user_profiles WHERE user_id=users.id);")
    mode="'postpartum'" if inp.get('mode')=='postpartum' else 'NULL'
    o.append(f"INSERT INTO user_life_profiles(user_id,gender,life_mode,onboarding_started_at,onboarding_completed_at,created_at,updated_at) SELECT id,'female',{mode},NOW(),NOW(),NOW(),NOW() FROM users WHERE mobile='{m}' AND NOT EXISTS(SELECT 1 FROM user_life_profiles WHERE user_id=users.id);")
    for p in pers:
        o.append(f"INSERT IGNORE INTO cycle_histories(user_id,period_start_date,period_end_date,is_confirmed,is_estimated,source,created_at,updated_at) VALUES({uid},'{S_(p['start'])}',{q(S_(p['end']) if p.get('end') else None)},1,0,'user_logged',NOW(),NOW());")
    if inp.get('hormonal'):
        o.append(f"INSERT IGNORE INTO contraception_methods(user_id,method,pack_type,pack_started_on,created_at,updated_at) VALUES({uid},'combined_pill','21','{S_(pers[-1]['start'])}',NOW(),NOW());")
    if inp.get('mode')=='postpartum':
        o.append(f"INSERT IGNORE INTO postpartum_profiles(user_id,birth_date,delivery_type,baby_count,source,breastfeeding,created_at,updated_at) VALUES({uid},'{S_(inp['delivery_date'])}','vaginal',1,'direct',1,NOW(),NOW());")
    for s in inp.get('skips',[]):
        o.append(f"INSERT IGNORE INTO cycle_skips(user_id,expected_start,created_at,updated_at) VALUES({uid},'{S_(s)}',NOW(),NOW());")
    for c in ('terms','health_data_processing'):
        o.append(f"INSERT INTO user_consents(user_id,consent,granted,version,granted_at,created_at,updated_at) VALUES({uid},'{c}',1,1,NOW(),NOW(),NOW()) ON DUPLICATE KEY UPDATE granted=1,revoked_at=NULL;")
    o.append(f"INSERT INTO user_revisions(user_id,engine_rev,analysis_rev,updated_at) VALUES({uid},1,1,NOW()) ON DUPLICATE KEY UPDATE engine_rev=engine_rev+1,analysis_rev=analysis_rev+1,updated_at=NOW();")
print("\n".join(o))
