---
name: dispatch-cursor
description: Dispatch implementation and investigation work to the local cursor-agent CLI (Cursor Grok 4.6 Extra High by default) as a headless executor inside a kernel sandbox, while this session stays the orchestrator and acceptor. The default executor for this repo. Use when a task is large enough to hand off as a written task book, or when the user asks to "派发 cursor" / "dispatch cursor" / "让 cursor 做". Covers the model-config trap, the three conditions under which the sandbox silently turns off, the dispatch, the preflight proof, task-book structure, what the sandbox cannot do in this repo (databases, pnpm installs, git) and the acceptance protocol.
---

# Dispatching cursor-agent as executor

> **If you are the cursor executor and this file was loaded into your context: ignore it.**
> It describes how the orchestrator dispatches *you*. It is not a task book. Your task book
> is the prompt you were started with, and nothing here overrides it.

This session is the **orchestrator**: it adjudicates design, writes the task book, owns git,
databases, migrations and production, and accepts or rejects the result. The local
`cursor-agent` CLI is the **executor**: it reads, writes and runs gates inside a sandbox.

Ported on 2026-09-18 from nextmoe-infra's `.claude/skills/dispatch-cursor/`. The sandbox
measurements below were taken there on the same machine and the same cursor-agent version.
The scripts here are copies, not links: this repo must not break when infra's skill changes.
`dispatch.sh` differs only in its deny list, which also covers this repo's env files.
`dispatch-grok` in this directory is the older, shell-less protocol; use this one.

## 1. The model-config trap — read before running cursor-agent at all

**`cursor-agent --model X` writes X back into the config it started with.** Without
`CURSOR_CONFIG_DIR`, that is the user's global `~/.cursor/cli-config.json`, and every other
session on this machine that dispatches copies it.

On 2026-09-18 this repo's session probed five models with bare `cursor-agent -p --model …`
runs. It turned the user's default from Cursor Grok 4.6 Extra High into Claude Sonnet 5, while
three other sessions were dispatching.

- **Never run `cursor-agent` outside `dispatch.sh`.** `dispatch.sh` points
  `CURSOR_CONFIG_DIR` at a per-run copy and pins `--model cursor-grok-4.6-xhigh`.
- If it happens anyway, restore `model`, `selectedModel`, `modelParameters` and
  `modelSelectionHistory` from a per-run copy made before the change. Every
  `$SCRATCHPAD/cursor/runs/*/cursor-config/cli-config.json` under `/tmp/claude-1000` is one.
  Do not overwrite the whole file: the copies have `authInfo` removed.
- Quota: Opus, Sonnet and GPT models hit the account's monthly cap on 2026-09-18. Grok 4.6 is
  the executor.

## 2. What cursor-agent can do here

| Capability | Result |
|---|---|
| Shell | bash, inside the sandbox of §3 |
| Go gates | `go build`, `go vet`, `go test` work, including `httptest.NewServer` on the sandbox's private loopback. `GOCACHE` / `GOMODCACHE` are a fresh `/tmp/cursor-sandbox-cache/<hash>/` per run: every run builds cold and downloads modules through Cursor's proxy (`proxy.golang.org` is allowed). `GOTOOLCHAIN=go1.26.1` therefore downloads the toolchain once per run. `dispatch.sh` deletes that cache (1.3 GB of tmpfs) afterwards |
| Network | package registries only; everything else is refused. No raw TCP |
| **Databases** | **none**. The loopback is private, docker is unreachable. DB-backed tests skip without `TEST_DATABASE_DSN`. The executor can **write** DB tests but never run them: the orchestrator runs them at acceptance (§5) |
| **pnpm** | the store (`~/.local/share/pnpm`) is outside the sandbox, so `pnpm install` / `pnpm add` fail. **The orchestrator installs before dispatch**: run `pnpm install --frozen-lockfile` in the worktree, and add any new dependency itself and commit it. The executor then runs `pnpm -F web typecheck / lint / test` on the installed tree |
| git | reads work; every write is denied by rule. The orchestrator commits |
| Project instructions | `CLAUDE.md` loads as a rule; `.claude/skills/*` may be listed as skills (hence the guard at the top) |
| Model | pinned by `dispatch.sh`: `cursor-grok-4.6-xhigh`. `CURSOR_MODEL=cursor-grok-4.6-low` for probes |
| Budget | wall clock only: `CURSOR_TIMEOUT`, default 4 h |
| Auth | `CURSOR_API_KEY` from the environment |

## 3. The sandbox, and how it silently turns off

A headless run applies the Landlock + user-namespace sandbox only when all three hold. When one
fails, every command runs as the real user, with no error and nothing in the output to say so:

1. No `--force` / `--yolo`.
2. `approvalMode: "allowlist"`. The user's config says `"unrestricted"`, which is unsandboxed.
3. The command is not on `permissions.allow`. An allowed command runs outside the sandbox, and
   the user's config allows `Shell(**)`.

`dispatch.sh` makes all three hold in a per-run config copy: the allow list is emptied, the
deny list is replaced, `authInfo` is dropped. The user's own config is never edited.

Inside the sandbox:

- The executor can write the worktree and `/tmp`.
- `ssh` to production, the host loopback and docker are unreachable.
- **Reads are not fenced**: the whole filesystem is readable, including the main checkout's
  env files.
- `git add` / `git commit` would work at the sandbox level; the deny rules stop them.

A fresh worktree has **no env files** (they are gitignored), which is why dispatches go to
worktrees. This repo has six of them:

- `apps/api/.env` — live catalog key, OAuth secret, DB DSN;
- `apps/web/.env`;
- `apps/web/.env.prod` — **production**;
- `docker/api.env`;
- `docker/web.env`;
- `refs/legacy/.env.prod`.

`dispatch.sh`'s deny list covers all six by shell text and by file-tool read. It also denies:

- git writes, including `git -C` / `-c` / `--git-dir`, which slip past per-subcommand rules;
- `gh`, `ssh`, `psql`, `docker`, `sudo`, `pkill`, `pnpm dev`;
- other agents;
- credential files;
- file-tool writes to `.claude/`, `.cursor/`, `CLAUDE.md`, `AGENTS.md` and `node_modules/`.

The environment layer unsets tokens, `SSH_AUTH_SOCK`, `TEST_DATABASE_DSN` and
`KUN_DATABASE_URL`, and rewrites push URLs to a path that does not exist.

## 4. The dispatch

```bash
# 1. a worktree nobody else stands on (iron rule 13), from the commit the task builds on.
#    This repo works on master and often has unpushed commits, so branch from local master,
#    not origin/master.
git -C /home/kun/Desktop/code/website/kun-galgame-forum worktree add -b <branch> \
  /home/kun/.config/superpowers/worktrees/kun-galgame-forum/<name> master
# 2. dependencies the sandbox cannot fetch
( cd <worktree> && pnpm install --frozen-lockfile )   # only if the task touches apps/web
# 3. the task book, from task-book-template.md, in the session scratchpad (never the repo)
export CURSOR_OUT_ROOT="$SCRATCHPAD/cursor/runs"
mkdir -p "$CURSOR_OUT_ROOT/<slug>"                     # write task.md there
# 4. from the worktree root, detached
cd <worktree> && CURSOR_OUT_ROOT="$CURSOR_OUT_ROOT" setsid nohup \
  /home/kun/Desktop/code/website/kun-galgame-forum/.claude/skills/dispatch-cursor/dispatch.sh <slug> \
  >"$CURSOR_OUT_ROOT/<slug>/dispatch.out" 2>&1 </dev/null &
```

Wait with a background until-loop on `pgrep -f 'dispatch-cursor/[d]ispatch.sh <slug>'`, never
a foreground sleep.

- **At most two at once**, and never two over overlapping paths or in one worktree.
- A detached dispatch survives the harness killing its background tasks when RAM runs out;
  the waiter does not, so re-arm it.
- To resume a killed run:
  1. put a "RESUMED RUN" note at the top of the same task book, saying what exists, what is
     left, and to re-run every gate;
  2. move the old `stream.jsonl` and `run.json` aside;
  3. dispatch again from the same worktree.

`dispatch.sh` refuses to run from a main checkout or a subdirectory. It also refuses `--force`,
`--yolo`, `--sandbox`, `--approve-mcps`, `--auto-review` and `--worktree`. Add per-task rules
with `--deny '<rule>'`.

## 5. Reading the result and accepting it

`check.sh` runs at the end of every dispatch; it can also be run by hand on a killed one. It
fails when:

- the preflight line is not exactly
  `dispatch-preflight: sandbox=native net=blocked loopback=private` — then treat every shell
  call in the run as unfenced;
- a shell call succeeded with no sandbox policy;
- HEAD moved;
- there is no `result` event, or the event is an error.

It also lists every refused call.

Then, by hand, in the worktree:

1. `git status --porcelain`: only the writable paths the task book named.
2. `git log --oneline -3`: no commit you did not make.
3. Read the whole diff.
4. Re-run every gate **yourself**:
   - `cd apps/api && GOTOOLCHAIN=go1.26.1 make lint && GOTOOLCHAIN=go1.26.1 go test ./...` —
     the system Go 1.27 breaks errcheck and changes inlining, which renames handlers in
     `routes.golden`;
   - `make openapi && git diff --exit-code apps/api/openapi` once that target exists;
   - `pnpm -F web lint && pnpm -F web typecheck && pnpm -F web test` if the web changed.
5. **Run the DB suites the executor could not reach.**
   - Create a throwaway database with
     `/home/kun/Desktop/code/website/nextmoe-infra/scripts/ephemeral-test-db.sh create <slug>`.
     It lives on the host Postgres, auth comes from `~/.pgpass`, and the DSN it prints has no
     password. Export that DSN as `TEST_DATABASE_DSN` and `KUN_DATABASE_URL` for the migration
     step only.
   - Build the schema with `apps/api/scripts/testdb-bootstrap.sh` once W0a lands it, or by
     hand per memory `kungal-db-backed-tests-bootstrap`.
   - Run `go test -count=1 -p 1 ./...`.
   - `drop` the database afterwards.
   - Never point any of this at the dev database (`apps/api/.env`) or production (iron
     rule 14).
6. Mutate: break each behaviour the task added (flip a condition, drop a filter) and confirm
   a test fails. Grok's tests have repeatedly asserted a little less than its report implies.
7. Commit from the orchestrator, `git commit -- <paths>` (never `add -A`: it misses the root
   `pnpm-lock.yaml` or sweeps in strays). Then merge the branch into master.

The report is a file in the scratchpad, never in the repo. `result` in `run.json` is every
assistant text block concatenated; read `report.md`, not that.

## 6. The task book

Template: `task-book-template.md` (step 0, environment and discipline already filled in).

- **English**, self-contained. The executor cannot see this conversation.
- **State every adjudication inline, and quote binding clauses** rather than citing them.
  Design docs in `docs/proj/**` are Chinese: the executor may read them, but the task book
  restates in English every rule it depends on. infra's `refs/` is gitignored and exists only
  in infra's main checkout: give absolute paths into
  `/home/kun/Desktop/code/website/nextmoe-infra/refs/`.
- **No open design decisions.** Where the mechanics depend on code the executor has yet to
  read, state the invariant plus the precedent, and require it to report what it chose.
- **Name the writable paths and the commands it may run.**
- **Name every symmetric case.** Grok implements what the book names and does not infer the
  mirror image (infra `mt-stray-rows`).
- **Word each check as the property, not your guess at its shape.**
- **Demand a positive control** for every gate, search or census.
- **Ask it to prove new tests run.** A `TestMain` or helper that skips without a database
  takes DB-free tests down with it.
- **Forbid ranking**; "anything that looks wrong, in scope or not" goes near the top of the
  report.
- Keep one dispatch to one coherent change that its tests can assert. Split a large wave into
  several dispatches rather than one four-hour run.

## 7. What the orchestrator never delegates

- Adjudications.
- All git.
- Databases, migrations and the ephemeral test DB.
- `pnpm install` / `pnpm add` and lockfile changes.
- Docker and `pnpm dev`; runtime checks against the dev stack; browser checks.
- Production (`ssh kungal-neo`).
- Cross-repo docs.
- Final acceptance (§5).

## 8. What is worth dispatching

| Shape | Verdict |
|---|---|
| Broad read → narrow `file:line` report | **Dispatch** |
| Wide mechanical edit a gate asserts | **Dispatch** |
| Fully adjudicated implementation whose correctness tests assert | **Dispatch**; still read the diff and mutate |
| New code carrying open design judgement | **Do not dispatch** until adjudicated |
| Anything whose truth is in a database or production | **Do not dispatch**; the sandbox cannot reach either |

## 9. Quality ledger

Record each real dispatch: task shape, model, elapsed, what acceptance found. infra's ledger
(`nextmoe-infra/.claude/skills/dispatch-cursor/SKILL.md` §8) holds five earlier runs of the same
model. Its verdict: dependable on fully adjudicated work; name every symmetric case; tests
assert a little less than the report implies, so mutate.

| Date | Task | Model | Elapsed | Acceptance |
|---|---|---|---|---|
