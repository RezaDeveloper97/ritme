import re,sys,html,glob,os
from html.parser import HTMLParser
class P(HTMLParser):
    def __init__(s): super().__init__(); s.out=[]; s.skip=0
    def handle_starttag(s,t,a):
        if t in('style','script','svg','helmet'): s.skip+=1
        a=dict(a)
        if t=='a' and a.get('href','#') not in ('#',None): s.out.append(f' [→{a["href"].replace(".dc.html","")}] ')
        if t in('input','textarea') : s.out.append(f' <{t}:{a.get("placeholder") or a.get("value") or a.get("aria-label","")}> ')
        if t in('div','section','header','li','h1','h2','h3','p','button','nav','footer','aside'): s.out.append('\n')
    def handle_endtag(s,t):
        if t in('style','script','svg','helmet'): s.skip-=1
    def handle_data(s,d):
        if not s.skip and d.strip(): s.out.append(d.strip()+' ')
for f in sys.argv[1:]:
    p=P(); p.feed(open(f).read())
    txt=re.sub(r'\n\s*\n+','\n',''.join(p.out))
    txt=re.sub(r'^۹:۴۱ *\n','',txt.strip())
    print(f'===== {os.path.basename(f)}\n{txt}\n')
