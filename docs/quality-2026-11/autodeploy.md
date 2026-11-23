# Stage auto-deploy: supply-chain guard (T-M12-03, finding S-06)

## Problem

A push to `claude/**` ran the checks, was merged into `stage` by `claude-to-stage.yml`, and within ~2 minutes the
production host rebuilt stage **as root** with no review. The script that did it,
`/opt/ritme-stage-src/deploy/stage-autodeploy.sh`, came from the checkout, so it was itself part of the pushed commit:
whoever could push a branch could replace the gate.

## Decision (user, 2026-11-18)

**(a) Signed commits from an allow-listed key, verified on the server.** No PR review gate (option (b)) and not both
(option (c)).

## What the guard does

| Layer | Where | What it does |
|---|---|---|
| Real gate | `deploy/stage-autodeploy.sh`, installed as `/usr/local/sbin/ritme-stage-autodeploy` | Before any checkout or build, every commit that `stage` gained since the last verified commit must pass (rules below). Otherwise the script logs `REFUSED: …`, exits 1, and deploys nothing. |
| Pinned keys | `/etc/ritme/allowed_signers` (root:root, 0644, outside the checkout) | The only SSH public keys whose signatures count. If the file is missing, empty or group/world-writable, the gate refuses. |
| Trust anchor | `/var/lib/ritme-stage-autodeploy/verified` | The last verified commit. You set it once by hand at install, and the script moves it after each pass. If it is missing, the gate refuses. |
| Pinned runner | `ritme-stage-autodeploy.service` → `ExecStart=/usr/local/sbin/ritme-stage-autodeploy` | The unit runs the installed copy, which only you update. The script never runs or re-execs the copy in the checkout. After verification it runs `./deploy-stage.sh` from the checkout, because by then that file is verified content. |
| CI gates | `.github/workflows/claude-to-stage.yml` | `merge` needs `backend` (ci.yml: `go vet` + `go test -race` with integration on MariaDB/Redis) and `frontend` (audits, typecheck, vitest). No gate step has `continue-on-error`. |
| Early signature check | same workflow, step "Signatures" | This is advisory. It uses the keys in the repo variable `RITME_ALLOWED_SIGNERS` and refuses to merge a branch with unsigned commits, so an unsigned branch fails in CI and does not block stage on the server. It cannot be the real gate, because a push to `claude/**` runs the pushed branch's own copy of the workflow. If the variable is unset, the step prints a warning and moves on. |

### Acceptance rules (`verify_range ANCHOR NEW`)

`git rev-list ANCHOR..NEW` lists every new commit, including commits brought in through merges. Each one must be one of:

1. **Signed by an allowed key.** That means `git verify-commit` with `gpg.format=ssh`,
   `gpg.ssh.allowedSignersFile=/etc/ritme/allowed_signers` and `gpg.minTrustLevel=fully` passes, and `%G?` is `G`. A
   good signature from an unknown key (`U`, "No principal matched") is refused.
2. **An unsigned two-parent merge whose tree is exactly `git merge-tree --write-tree p1 p2`.** This is the merge
   commit the GitHub Action makes (`git merge --no-ff` as `ritme-ci`, not signed by GitHub's web-flow key). Such a
   commit adds no content of its own. Its parents are in the range and are checked by the same rules. An "evil merge"
   with smuggled changes, a merge with conflicts, and an octopus merge are all refused.
3. **Brought in only through a non-first parent of a merge that is itself signed by an allowed key.** This is how you
   approve a branch whose commits were made unsigned, for example by Claude Code on the web. Run a signed
   `git merge --no-ff` into `stage` on the Mac. A signed merge does **not** vouch for its first-parent line: an
   unsigned commit directly on `stage` stays refused.

A refused head is recorded in `last` and is not retried every tick. Later heads stay refused while the bad commits
are in the range, until you approve them (see below).

Requirements on the server: git ≥ 2.38 (SSH signing since 2.34, `merge-tree --write-tree` since 2.38) and
`ssh-keygen`. The script checks the git version and refuses if it is older.

## One-time setup: Mac

The key is used only for signing, never for login. Give it a passphrase and let the macOS keychain hold it.

```bash
ssh-keygen -t ed25519 -C "ritme-signing rezadeveloper97@gmail.com" -f ~/.ssh/ritme_signing
ssh-add --apple-use-keychain ~/.ssh/ritme_signing

cd ~/PhpstormProjects/ritme
# scoped to THIS repo only (--local)
git config --local gpg.format ssh
git config --local user.signingkey ~/.ssh/ritme_signing.pub
git config --local commit.gpgsign true
git config --local tag.gpgsign true
# so `git log --show-signature` / verify-commit work locally too
mkdir -p ~/.config/git
echo "rezadeveloper97@gmail.com namespaces=\"git\" $(cat ~/.ssh/ritme_signing.pub)" > ~/.config/git/ritme_allowed_signers
git config --local gpg.ssh.allowedSignersFile ~/.config/git/ritme_allowed_signers

# check
git commit --allow-empty -m "chore: signing test" && git verify-commit HEAD && git log -1 --format='%G? %GS'
```

Optional, recommended: make the CI pre-check and the GitHub "Verified" badge work.

```bash
gh ssh-key add ~/.ssh/ritme_signing.pub --type signing --title "ritme signing (Mac)"
gh variable set RITME_ALLOWED_SIGNERS --body "rezadeveloper97@gmail.com namespaces=\"git\" $(cat ~/.ssh/ritme_signing.pub)"
```

### Agents' commits

- **Claude Code on this Mac** (CLI, subagents, `/next-task` and the other queue skills) commits through the repo's git
  config. Once `commit.gpgsign=true` is set locally, its commits are signed automatically with the same key, as long
  as ssh-agent has the key unlocked. Without the agent, a commit fails with a signing error. It is never made
  unsigned silently.
- **Claude Code on the web** commits in a cloud sandbox that does not have your key, so those commits are unsigned.
  With `RITME_ALLOWED_SIGNERS` set, `claude-to-stage` refuses to auto-merge such a branch and tells you so. Approve it
  from the Mac:

  ```bash
  git fetch origin
  git worktree add ../ritme-approve origin/stage && cd ../ritme-approve
  git log --oneline origin/stage..origin/claude/<branch>   # review what you are approving
  git merge --no-ff -S origin/claude/<branch> -m "Approve claude/<branch> into stage"
  git push origin HEAD:stage
  cd - && git worktree remove ../ritme-approve
  ```

- If an unsigned branch got onto `stage` anyway (the variable was unset), the server refuses that stage head and
  every later one. Either push a signed revert, or approve the commits on the server by hand after you review them
  (see "Approve by hand" below).

## One-time setup: server (89.251.8.115, as root)

Copy the two files and the public key from the Mac:

```bash
cd ~/PhpstormProjects/ritme
scp deploy/stage-autodeploy.sh deploy/systemd/ritme-stage-autodeploy.service ~/.ssh/ritme_signing.pub root@89.251.8.115:/root/
```

Then on the server:

```bash
git --version   # must be >= 2.38; otherwise install a newer git first (the gate refuses on older ones)

# 1. pinned keys, outside the checkout
install -d -o root -g root -m 0755 /etc/ritme
echo "rezadeveloper97@gmail.com namespaces=\"git\" $(cat /root/ritme_signing.pub)" > /etc/ritme/allowed_signers
chown root:root /etc/ritme/allowed_signers && chmod 0644 /etc/ritme/allowed_signers

# 2. the gate itself (an installed copy)
install -o root -g root -m 0755 /root/stage-autodeploy.sh /usr/local/sbin/ritme-stage-autodeploy
/usr/local/sbin/ritme-stage-autodeploy --self-test        # must end with "self-test: all passed"

# 3. trust anchor = what stage runs now (every older commit is unsigned and stays trusted as history)
cat /var/lib/ritme-stage-autodeploy/ok                     # the last successfully deployed commit
cp /var/lib/ritme-stage-autodeploy/ok /var/lib/ritme-stage-autodeploy/verified

# 4. the unit runs the installed copy
install -o root -g root -m 0644 /root/ritme-stage-autodeploy.service /etc/systemd/system/ritme-stage-autodeploy.service
systemctl daemon-reload
systemctl cat ritme-stage-autodeploy | grep ExecStart      # → /usr/local/sbin/ritme-stage-autodeploy
systemctl start ritme-stage-autodeploy; tail -n 20 /var/log/ritme-stage-autodeploy.log
rm /root/stage-autodeploy.sh /root/ritme-stage-autodeploy.service /root/ritme_signing.pub
```

An update to `deploy/stage-autodeploy.sh` in the repo does **nothing** on the server until you repeat step 2. That is
deliberate.

## Approve by hand (server)

After you have reviewed the commits:

```bash
cd /opt/ritme-stage-src && git fetch origin && git log --format='%h %G? %an %s' "$(cat /var/lib/ritme-stage-autodeploy/verified)..origin/stage"
git rev-parse origin/stage > /var/lib/ritme-stage-autodeploy/verified   # trust everything up to here
rm /var/lib/ritme-stage-autodeploy/last && systemctl start ritme-stage-autodeploy
```

## Key rotation and revocation

1. On the Mac, create the new key (`~/.ssh/ritme_signing_2026b`) and point `user.signingkey` at it.
2. On the server, **add** its line to `/etc/ritme/allowed_signers`. During the overlap both keys are valid.
3. Update `RITME_ALLOWED_SIGNERS` and the GitHub signing key.
4. When nothing unsigned by the new key is still pending, delete the old line on the server and in the variable.

If a key leaks: remove its line from `/etc/ritme/allowed_signers` at once, which makes the gate refuse its commits
immediately. Then rotate. To limit how long a key is valid, use the `valid-after="YYYYMMDD"` and
`valid-before="YYYYMMDD"` options on the allowed_signers line.

## Rollback

- **A bad stage deploy.** Push a signed `git revert` to `stage`, or deploy by hand from the Mac with the
  `deploy-stage` skill. The gate only covers the automatic path.
- **The guard itself.** You can pause it with `systemctl stop ritme-stage-autodeploy.timer`, and stage can then be
  deployed by hand only. Do not point `ExecStart` back at the checkout: that reopens S-06. To go back to the previous
  installed version, install the older file from git (`git show <sha>:deploy/stage-autodeploy.sh`) with step 2.

## Residual risks

- **Downgrade.** Someone with push access can move `stage` to an older commit that is already signed or verified.
  That redeploys old code, but cannot introduce new code.
- **The Mac key is the root of trust.** Anything that can sign as you, such as malware with the agent unlocked,
  passes the gate.
- **Builds still run on the production host as root.** A signed but buggy commit can still break production
  through shared resources such as disk, CPU or the docker daemon.

## Follow-up idea (out of scope)

Build the stage images **off the production host**, for example in CI or on a separate builder, sign them, push them
to a registry, and have the server only `docker pull` and verify the signature, with no `docker build` and no
repo-supplied scripts running as root. That removes the root-on-prod build step entirely.

## Evidence

`ritme-stage-autodeploy --self-test` builds a temporary repo with temporary keys and runs `verify_range`. It passed
with git 2.48.1 / bash 3.2 (macOS) and with git 2.39.5 / bash 5.2 (Debian bookworm, `golang:1.25-bookworm`):

```
  ok    commit signed by the allowed key → accept
  ok    unsigned commit on top → refuse
  ok    unsigned commit, anchor = parent → refuse
  ok    commit signed by an unknown key → refuse
  ok    nothing new (anchor = tip) → accept
  ok    unsigned clean merge of signed commits → accept
  ok    unsigned merge with extra content → refuse
  ok    clean merge bringing an unsigned commit → refuse
  ok    signed merge vouching an unsigned branch → accept
  ok    signed merge over an unsigned stage commit → refuse
  ok    unknown trust anchor → refuse
  ok    world-writable allowed_signers → refuse
  ok    missing allowed_signers → refuse
  ok    signed commit, no allowed_signers file → refuse
self-test: all passed
```

To show that the tests catch a regression, each check was removed in turn from a copy of the script and the
self-test was run again; every mutant failed:

| Mutant | Self-test result |
|---|---|
| Accept any good signature (`%G?` = `G` or `U`) | 2 FAILED (unknown key; no allowed_signers file) |
| Unsigned merge accepted without comparing trees | 1 FAILED (evil merge) |
| Signed merge also vouches for its first parent | 1 FAILED (unsigned stage commit) |
| Writable `allowed_signers` accepted | 1 FAILED |

git 2.48 already fails `verify-commit` on "No principal matched". `minTrustLevel=fully` and the `%G? == G` check are a
second layer against other git builds and configs.

An end-to-end run of `main()` in a container, with a bare repo as origin and a stub `deploy-stage.sh`, gave these
results:

| Case | Result |
|---|---|
| No allowed_signers | refused, exit 1 |
| No anchor | refused, exit 1 |
| Unsigned commits since the anchor | each one listed as `REFUSED`, exit 1 |
| Same head again | not retried, exit 0 |
| Signed tip on top of unsigned commits | refused |
| After approving by hand, signed commits only | verified, stub deploy ran, exit 0 |
