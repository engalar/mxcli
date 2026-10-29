# Working Outside Studio Pro

A project edited with mxcli, git and an editor is sometimes opened in Studio Pro
afterwards. Several kinds of state that Studio Pro depends on are invisible to
`mx check`, and mxcli checks them.

## The ContentsHash index: `mxcli fix hashes`

In an MPR v2 project (Mendix 10.18 and later) the `.mpr` file indexes every
`mprcontents/**/*.mxunit` file by its hash: `Unit.ContentsHash` is
`base64(SHA-256(file))`. mxcli's writer and Studio Pro keep the two in step.
Anything else that changes a `.mxunit` does not: restoring a unit with
`git checkout`, `git restore`, a revert or a merge leaves the index describing
bytes that are no longer on disk. `mx check` and MxBuild read the files and do
not notice; Studio Pro uses the index to decide what changed.

```bash
mxcli fix hashes -p app.mpr            # verify; exits 1 when something is off
mxcli fix hashes -p app.mpr --repair   # rewrite mismatched hashes from the files
```

| Reported | Meaning | `--repair` |
|----------|---------|------------|
| `MISMATCH` | the file's hash differs from the stored one | rewrites the stored hash from the file |
| `MISSING` | the `.mpr` indexes a unit whose file does not exist | reported only — restore the file |
| `ORPHAN` | a `.mxunit` file no unit in the `.mpr` indexes | reported only — `mxcli diag --check-units --fix` removes it |

The repair treats the files as the source of truth and never changes them. It
updates every hash in one SQLite transaction, and, like every mxcli write, is
refused while Studio Pro has the project open (the `.mpr.lock` beside it). An
MPR v1 project keeps unit contents inside the `.mpr`, has no index to drift,
and the command says so.

`mxcli exec` and `mxcli docker check` run the verify too and print a one-line
warning when it finds a mismatch or a missing file. It never fails them;
measured on a 900-unit project, the whole verify takes about 0.2 s.

## Property names Studio Pro's merge compares: `mxcli fix property-sets`

Mendix's diff and merge engine compares two revisions of an element by its
property **names** before any value, and throws when they differ:

```
Objects with ID … of type Microflows$RestOperationCallAction do not have the same properties.
```

The model is valid — `mx check`, the build and the runtime fill an absent
property with its default — but `mx diff`, merging and Studio Pro's Version
Control panel fail on it once Studio Pro has saved the document with its full
property set. Every mxcli write gives a unit the property set Studio Pro writes
for the project's Mendix version. A unit written by an older mxcli and not
rewritten since keeps its gaps; this command reaches those.

```bash
mxcli fix property-sets -p app.mpr --dry-run   # list the units that would change; exits 1 when one would
mxcli fix property-sets -p app.mpr             # write them
```

Each unit's report counts the keys added (with the value Mendix gives an absent
one) and the empty keys stripped (properties the project's version does not
declare yet). A project user role with no `GUID` — older mxcli wrote them
without one — gets its own `$ID` as its GUID, the value a role mxcli creates
now gets; every mxcli write of the security settings does the same. Without
it the build invents a random GUID each time, so the runtime can only match
the role's database row by name. A role that already has a GUID keeps it.
Nothing else changes: no element `$ID` moves, and no existing `GUID` is
touched. A second run writes nothing.

The filled GUID protects a later **rename** of the role only in a database
first synchronised after the fill. A database synchronised before it keeps
the old random GUID and still finds the role by name, so renaming the role
there still deletes its row and the user assignments that are not demo users. A unit holding a *value* in a
key its version does not declare is reported and left alone, and, like every
mxcli write, the command is refused while Studio Pro has the project open.

## Git states that crash Studio Pro

Studio Pro 11.13 fails to open a project with *"Unable to find 'system'
property in 'system'"* when the project folder is a git repository and

- the checked-out branch has **no upstream** — remedy:
  `git push -u origin <branch>` before opening the project (or
  `git remote add origin <url>` first, if the repository has no remote); or
- git reports **"detected dubious ownership"** — typical for a folder shared
  between the host and a devcontainer, owned by a different user on each side.
  Remedy, on the machine that runs Studio Pro:
  `git config --global --add safe.directory <path>`.

`mxcli docker check`, `mxcli run --local` and `mxcli diag -p app.mpr` warn about
both. The warnings are advice and never fail a command. They are silent:

- outside a git repository, or when git is not installed;
- on a **detached HEAD** — how CI systems and `git checkout <sha>` leave a
  repository; there is no branch to push, and nobody opens such a checkout in
  Studio Pro;
- when the `CI` environment variable is set, or `MXCLI_NO_GIT_WARNINGS=1`.

Dubious ownership is judged by the git mxcli runs. Inside a devcontainer that is
the container's git, which may see ownership differently from the host's Studio
Pro — so a clean report from inside the container does not rule it out on the
host.

## One report: `mxcli diag -p`

```bash
mxcli diag -p app.mpr
```

prints mxcli's diagnostics followed by a *Project* section with both checks.

## One commit, all the steps: `mxcli git commit`

Studio Pro's *Commit* does more than `git commit`: it keeps the ContentsHash
index in step, stages exactly the files a Mendix repository should ship, and
attaches the mx_metadata note. `mxcli git commit` is that sequence for commits
made outside Studio Pro:

```bash
mxcli git commit -m "Add order history page" -p app.mpr
mxcli git commit -m "Fix stock filter" -p app.mpr --push
mxcli git commit -m "WIP" -p app.mpr --dry-run
```

In order it:

1. repairs ContentsHash mismatches from the files on disk (`MISSING` and
   `ORPHAN` entries have no automatic repair and refuse the commit);
2. stages everything, then unstages the files Studio Pro never commits:
   `project-settings.user.json`, `*.mpr.lock`, `*.mpr.bak`, `.mendix-cache/`,
   `deployment/`, `theme-cache/`, `.mxcli/`;
3. runs `git commit` with your message and your git identity;
4. computes the mx_metadata note from the new commit and attaches it, and
   prints the `git push origin <branch> refs/notes/mx_metadata` command —
   `--push` runs it for you, because the branch and the notes ref must travel
   together.

The commit is refused while Studio Pro has the project open (the `.mpr.lock`
beside it), and during an unresolved merge — `git add -A` would otherwise
commit the conflict markers. `--dry-run` verifies and reports (including
ContentsHash drift, which it does not repair — a dry run writes nothing),
stages, reports what it would commit, and unstages again without committing.

## Merging branches: `mxcli git merge`

`git merge` cannot merge Mendix models: the `.mpr` is SQLite and every
`.mxunit` is BSON, so two sides changing the model always meet a binary
`CONFLICT (content)` — there is nothing for git's textual merge to work on.
Studio Pro's *Merge Changes Here* resolves it with a model-aware merge, and
that engine ships as a command: `mx git-merge BASE MINE THEIRS …`.

`mxcli git merge <ref>` puts the two together:

```bash
mxcli git merge feature/orders -p app.mpr
mxcli git merge origin/main -p app.mpr --push
```

`git merge <ref>` runs first; a merge git resolves itself (disjoint files,
a fast-forward) is annotated with the mx_metadata note like `mxcli git
commit` does. On a model-file conflict the base and theirs revisions are
checked out into temporary worktrees and `mx git-merge` runs (resolved for
the project's Mendix version):

- what mx auto-resolves is staged and committed as the normal two-parent
  merge commit, with the note attached;
- a conflict mx reports as `ManualResolutionRequired` is never merged
  silently: the merge is aborted, and **both sides' changes are printed for
  review** — unit by unit, ours and theirs. Then decide:
  `--ours` or `--theirs` takes one side of every conflicted model file
  (deterministic; mx is not consulted again), or resolve for real in
  Studio Pro (*Version Control › Merge Changes Here*), which stays the tool
  for mixed resolutions;
- a conflict in non-model files (text sources) is left in place for manual
  resolution, like plain git.

Merging is still never done by plain `git merge` on a Mendix repository: use
`mxcli git merge` or Studio Pro, never a textual merge of `mprcontents/`.
Like every mxcli write, the command is refused while Studio Pro has the
project open.

## Auditing a repository: `mxcli git doctor`

A repository that has known only plain git misses several things Studio Pro
wrote when it committed, and they are invisible one by one. `mxcli git doctor`
reports all of them in one read-only pass; `--fix` repairs:

```bash
mxcli git doctor                       # report only; exits 1 while an issue remains
mxcli git doctor -p app.mpr --fix      # repair, computing real notes from the model
mxcli git doctor --fix --version 10.21.0
```

| Check | `--fix` |
|-------|---------|
| local git config: `core.autocrlf=False` (a checkout must not rewrite line endings inside `.mxunit` files; any other value is rewritten), `mendix.commits-since-gc`, `mendix.lineEndingResetDone` — the `mendix.*` values are Studio Pro's own bookkeeping and are only backfilled when missing | sets/backs them |
| the remote URL: Studio Pro can only use HTTPS | rewrites SSH (`git@host:org/app.git`, `ssh://…`) to HTTPS; other non-HTTPS URLs are reported unchanged |
| the remote's `refs/notes/mx_metadata` | reported only — push the notes ref together with the branch |
| each commit on the checked-out branch and its mx_metadata note: missing notes make Studio Pro show `(unknown)`; a malformed note it cannot read at all | computes real notes from the model (`-p`), falling back to a template note with a `--version` ModelerVersion where the model at a commit cannot be read (MPR v1 history, a revision without the `.mpr`); when no version is known the newest valid note in history supplies it |

Notes Studio Pro wrote are never touched, with or without `--fix`. `--fix`
does not push.

## Commit metadata: `mxcli git note`

Studio Pro attaches a git note under `refs/notes/mx_metadata` to every commit
it makes:

```json
{"BranchName":"feature-x","ModelerVersion":"11.13.0",
 "ModelChanges":[{"Status":"Modified","UnitID":"…","UnitType":"Forms$Page",
                  "UnitName":"Home_Web","Module":"MyModule"}],
 "RelatedStories":[],"SolutionVersion":"","MPRFormatVersion":"Version2","HasModelerVersion":true}
```

It reads the note to check that a revision's Mendix version is compatible and to
show what the revision changed. A commit made with plain git has none, and
Studio Pro 11.13 back-fills one with `"(unknown)"` values and no changes.

```bash
mxcli git note -p app.mpr            # preview the notes for commits not yet pushed
mxcli git note -p app.mpr --write    # attach them
git push origin <branch> refs/notes/mx_metadata
```

The note is computed from the commit: the `.mpr` unit index and the
`mprcontents` units at the commit and at its parent. With no commit argument it
covers `@{upstream}..HEAD`, or `HEAD` when the branch has no upstream. A note
Studio Pro wrote is kept unless you pass `--force`; a placeholder is replaced.

| Status | Meaning |
|--------|---------|
| `Added` / `Deleted` | the unit is new / gone. A deleted module is listed once, as its `Projects$ModuleImpl` |
| `Modified` | the unit's content changed. A re-serialisation that only mints new element IDs is not a change |
| `Moved` | the unit moved to another folder (its container in the `.mpr` changed) |

Measured against Studio Pro's own notes on a Team Server repository: for
ordinary commits the note is identical, header and change list. For a Mendix
**version upgrade** the header is identical but the change list is longer than
Studio Pro's, because an upgrade re-serialises units in ways only Studio Pro
knows to ignore. Upgrades are done in Studio Pro, which writes its own note. MPR
v1 projects are not supported.

**Push the notes ref soon.** `git push` does not push notes, and while the
project is open Studio Pro fetches every few minutes and force-replaces the local
`refs/notes/mx_metadata` with the Team Server's copy, discarding notes that were
not pushed. If that happens, `mxcli git note --write` again: the discarded notes
have become placeholders, which it replaces.

The whole workflow (server branches, what to stage, never merging model history
with git) is in the `teamserver-git` skill that `mxcli init` installs.
