---
id: T-M12-03
title: Supply-chain guard on the claude/** → stage auto-merge and root auto-deploy
milestone: M12
type: release
status: todo
depends_on: []
parallel_group: M12-SEC
touches: [.github/workflows/claude-to-stage.yml, deploy/stage-autodeploy.sh, deploy/systemd/ritme-stage-autodeploy.service, docs/quality-2026-11/autodeploy.md]
skills: []
verify: bash -n deploy/stage-autodeploy.sh && grep -q "verify-commit" deploy/stage-autodeploy.sh
---

# T-M12-03 — Supply-chain guard on the claude/** → stage auto-merge and root auto-deploy
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-06: any push to `claude/**` reaches the production host as root within minutes, with no review, and the deploy script it runs comes from the pushed commit.

## Scope
1. Step 1 — ask the user (AskUserQuestion) which guard they want: (a) signed commits from an allow-listed key, verified on the server; (b) a reviewed PR into `stage` (branch protection); (c) both. Record the answer in `docs/quality-2026-11/autodeploy.md`.
2. Server side: `stage-autodeploy.sh` runs `git verify-commit` (or checks the PR merge) against a pinned key and exits non-zero otherwise; the unit `ExecStart` points to a copy installed outside the checkout (e.g. `/usr/local/sbin`), updated only by a manual step.
3. Workflow: vitest and `go test` with integration (T-M12-11) must pass before the auto-merge; no `continue-on-error` on gates.
4. Document in the doc how to install/rotate the key and how to roll back.

## Out of scope
- Moving image builds off the prod host (write it as a follow-up idea in the doc).
- Any change to production compose.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- An unsigned / unknown-key commit on `stage` is refused by the script (show the test run in PROGRESS.md).
- The user has applied the server-side unit change (report the exact commands; do not run them over ssh without their OK).
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
