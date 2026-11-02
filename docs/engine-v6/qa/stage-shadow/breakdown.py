#!/usr/bin/env python3
"""Per-persona breakdown of the CYCLE_ENGINE=shadow samples collected by probe.py (T-M9-48).

  scp 'root@89.251.8.115:/tmp/shadowprobe/*' ./probe/ && python3 breakdown.py ./probe [report0.txt]

report0.txt = `api engine-shadow-report -n 0` taken right before probe.py ran (the count baseline). Persona i's samples
are the newest (compared_i - compared_{i-1}) samples of r<i>.txt. Prints, per persona: classes with the date offsets
they cover, the compared values of every class, and the field names hidden in `other`.
"""
import collections, json, re, sys

PERSONAS = ['regular', 'late 8', 'late 15', 'pill', 'postpartum', 'onboarding-only', 'stale anchor', 'skip']


def report(path):
    lines = open(path).read().splitlines()
    n = int(re.search(r'(\d+) comparisons', lines[0]).group(1))
    samples = [json.loads(l) for l in lines if l.strip().startswith('{')]
    return n, samples


def ranges(xs):
    out, start, prev = [], None, None
    for x in sorted(xs):
        if start is None:
            start = prev = x
        elif x == prev + 1:
            prev = x
        else:
            out.append(f'{start}..{prev}' if start != prev else str(start))
            start = prev = x
    if start is not None:
        out.append(f'{start}..{prev}' if start != prev else str(start))
    return ','.join(out)


def main():
    d = sys.argv[1].rstrip('/')
    prev = report(sys.argv[2])[0] if len(sys.argv) > 2 else None
    for i, name in enumerate(PERSONAS, 1):
        try:
            n, samples = report(f'{d}/r{i}.txt')
        except FileNotFoundError:
            continue
        mine = samples[: n - prev] if prev is not None else [s for s in samples if s['user'] == samples[0]['user']]
        prev = n
        print(f'===== {i} {name}: {len(mine)} comparisons, user hash {mine[0]["user"] if mine else "-"}')
        offs, vals, other = collections.defaultdict(list), collections.defaultdict(collections.Counter), collections.Counter()
        for s in mine:
            for c in s['classes']:
                offs[c].append(s['date_offset'])
            for k, v in (s.get('fields') or {}).items():
                if k == 'other':
                    for f in v:
                        other[f] += 1
                        offs['other:' + f].append(s['date_offset'])
                else:
                    vals[k][json.dumps(v, ensure_ascii=False)] += 1
        for c, o in sorted(offs.items(), key=lambda kv: -len(kv[1])):
            print(f'  {c:45} {len(o):4}  offsets {ranges(o)}')
        for k, c in vals.items():
            print(f'  values {k}: {c.most_common(5)}')


if __name__ == '__main__':
    main()
