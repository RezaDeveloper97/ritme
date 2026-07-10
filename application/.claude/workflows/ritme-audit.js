export const meta = {
  name: 'ritme-audit',
  description: 'Full multi-agent audit of the Ritme app: security, architecture, dependencies, performance/UI rules, crash-resilience — findings adversarially verified',
  whenToUse: 'Before a release or after a large feature: run a comprehensive audit across all project rules.',
  phases: [
    { title: 'Audit', detail: 'parallel auditors per dimension' },
    { title: 'Verify', detail: 'adversarial verification of each finding' },
  ],
}

const FINDINGS_SCHEMA = {
  type: 'object',
  properties: {
    findings: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          file: { type: 'string' },
          line: { type: 'number' },
          severity: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
          rule: { type: 'string' },
          summary: { type: 'string' },
          fix: { type: 'string' },
        },
        required: ['file', 'severity', 'summary'],
      },
    },
  },
  required: ['findings'],
}

const VERDICT_SCHEMA = {
  type: 'object',
  properties: {
    real: { type: 'boolean' },
    reason: { type: 'string' },
  },
  required: ['real', 'reason'],
}

const ROOT = '/Users/rezataheri/AndroidStudioProjects/Ritme'

const DIMENSIONS = [
  {
    key: 'security',
    agentType: 'security-auditor',
    prompt: `Audit the app at ${ROOT} per your security checklist. Report only real, evidence-backed findings with file:line.`,
  },
  {
    key: 'architecture',
    agentType: 'arch-reviewer',
    prompt: `Audit ${ROOT}/app for hexagonal-architecture violations per your checklist. Grep for evidence; report file:line.`,
  },
  {
    key: 'deps',
    agentType: 'deps-guard',
    prompt: `Audit all Gradle files and imports under ${ROOT} for third-party dependency violations.`,
  },
  {
    key: 'performance',
    prompt: `Review Compose UI code under ${ROOT}/app/src for CLAUDE.md §5/§5b violations: Column+forEach instead of LazyColumn, missing stable keys, heavy work on the main thread (JSON/SQLite/bitmap off Dispatchers.IO?), unstable Composable params, missing SwipeBackContainer on non-root screens, heavy Application.onCreate. Report file:line findings only.`,
  },
  {
    key: 'crash-resilience',
    prompt: `Review ${ROOT}/app/src against CLAUDE.md §7: uncaught-exception handler installed and non-blocking, last-safe-screen writes on each screen, crash reports written locally then uploaded with delete-after-2xx, breadcrumbs bounded and PII-free, sealed Result across ports (no throw across port boundaries, no empty catch). Report file:line findings only.`,
  },
]

phase('Audit')
const results = await pipeline(
  DIMENSIONS,
  d => agent(d.prompt, {
    label: `audit:${d.key}`,
    phase: 'Audit',
    schema: FINDINGS_SCHEMA,
    ...(d.agentType ? { agentType: d.agentType } : {}),
  }),
  (res, d) => parallel((res?.findings ?? []).map(f => () =>
    agent(
      `Adversarially verify this ${d.key} finding in the Ritme Android project (${ROOT}). ` +
      `Read the actual code. Finding: ${JSON.stringify(f)}. ` +
      `Is it real and correctly located? Default real=false if the evidence is weak.`,
      { label: `verify:${f.file}`, phase: 'Verify', schema: VERDICT_SCHEMA }
    ).then(v => ({ ...f, dimension: d.key, verified: v?.real === true, verifyReason: v?.reason }))
  ))
)

const all = results.filter(Boolean).flat().filter(Boolean)
const confirmed = all.filter(f => f.verified)
const rejected = all.length - confirmed.length
log(`${confirmed.length} confirmed findings (${rejected} rejected in verification)`)
return { confirmed, rejectedCount: rejected }
