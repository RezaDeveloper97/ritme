INSERT IGNORE INTO users(name,mobile,mobile_verified_at,created_at,updated_at) VALUES('QA partner','09977000009',NOW(),NOW(),NOW());
INSERT INTO user_profiles(user_id,user_goal,created_at,updated_at) SELECT id,'non_ttc',NOW(),NOW() FROM users WHERE mobile='09977000009' AND NOT EXISTS(SELECT 1 FROM user_profiles WHERE user_id=users.id);
INSERT INTO user_life_profiles(user_id,gender,onboarding_started_at,onboarding_completed_at,created_at,updated_at) SELECT id,'male',NOW(),NOW(),NOW(),NOW() FROM users WHERE mobile='09977000009' AND NOT EXISTS(SELECT 1 FROM user_life_profiles WHERE user_id=users.id);
INSERT INTO user_consents(user_id,consent,granted,version,granted_at,created_at,updated_at) SELECT u.id,c.c,1,1,NOW(),NOW(),NOW() FROM users u JOIN (SELECT 'terms' c UNION SELECT 'health_data_processing') c WHERE u.mobile='09977000009' ON DUPLICATE KEY UPDATE granted=1;
INSERT INTO user_revisions(user_id,engine_rev,analysis_rev,updated_at) SELECT id,1,1,NOW() FROM users WHERE mobile='09977000009' ON DUPLICATE KEY UPDATE engine_rev=engine_rev+1;
INSERT INTO companions(owner_id,companion_user_id,type,status,display_name,invited_at,accepted_at,created_at,updated_at,role,template,urgent_sms,owner_confirmed_at)
 SELECT o.id,p.id,'partner','active',o.name,NOW(),NOW(),NOW(),NOW(),'partner','legacy',0,NOW() FROM users o, users p WHERE p.mobile='09977000009' AND o.mobile BETWEEN '09977000001' AND '09977000008'
 AND NOT EXISTS(SELECT 1 FROM companions c WHERE c.owner_id=o.id AND c.companion_user_id=p.id);
INSERT IGNORE INTO companion_grants(companion_id,section,level,created_at,updated_at) SELECT c.id,'cycle','view',NOW(),NOW() FROM companions c JOIN users p ON p.id=c.companion_user_id WHERE p.mobile='09977000009';
