---
name: casa-secrets
description: Operate casa's secrets and encryption keys — age-encrypted files that travel in the repo, opened by plain-file keys in ~/.config/casa/keys that never touch a repo. Covers casa secrets add/edit/remove/keys/list, registry-free keys, per-file key choice, orphan-safe key deletion, passphrase-sealed backups, and doppler via CASA_AGE_KEY_<NAME>. Use when adding, editing, or listing encrypted files, managing or moving age keys, restoring keys on a fresh machine, or debugging decrypt failures in a casa repo. Skip for non-secret casa work (files, tools, machine) and repos not managed by casa.
license: MIT
---

# casa secrets — age encryption, registry-free keys

casa manages your machines — files, tools, and secrets — from one git repo,
with two verbs: push and pull. This skill covers the secrets third: files
sealed with [age](https://age-encryption.org) that travel in the repo as
armored ciphertext, and the keys that open them, which are plain files on the
machine and **never** enter any repo.

Prerequisite: the `age` CLI (`brew install age`). casa errors with a clear
message if `age-keygen` is missing.

```
usage:
  casa secrets add [path]     encrypt and start managing a file
  casa secrets edit [name]    pick a secret, decrypt, edit, re-seal
  casa secrets remove         stop managing a secret (file stays on disk)
  casa secrets keys           key screen: create, default, backup, delete, doppler
  casa secrets list           list encrypted files (plain output, pipeable)
```

## AGENT GUARDRAILS (read first)

- **All `casa secrets` flows except `list` are interactive** — pickers,
  editors, confirmations, passphrase prompts. **Leave running them to the
  human.** Tell the user the exact command to run; do not drive these flows
  yourself, do not pipe input into them.
- **NEVER generate, guess, suggest, or handle a backup passphrase.** If a flow
  asks for one, that prompt is for the human at the terminal, full stop.
- **Never write key material anywhere.** Do not `cat`, read, copy, echo, log,
  or quote the contents of `~/.config/casa/keys/*.txt`, and never create or
  move identity files yourself. Public recipients (`age1...` strings from
  `age-keygen -y`) are safe to display; private identities are not.
- **Never decrypt a secret to disk or into your output.** If a user asks
  what's inside a secret, point them at `casa secrets edit <name>`.
- **Never hand-edit `.casa/keys/` or commit generated `run_*` scripts.**
  Backups under `.casa/keys/` are repo content casa writes and removes
  (deleting a key removes its backup); the run scripts are generated,
  gitignored, and must stay uncommitted.

What you CAN safely do: run the read-only commands below, inspect repo
ciphertexts (they are armored text), explain the model, and prepare the exact
interactive command for the user to run.

## Read-only inspection (agent-safe, no prompts)

```bash
casa secrets list                    # encrypted files by readable target path (~/.aws/credentials, ...)
ls ~/.config/casa/keys               # key inventory: <name>.txt files + .default marker
cat ~/.config/casa/keys/.default     # name of the default key (marker may be absent)
age-keygen -y ~/.config/casa/keys/main.txt   # PUBLIC recipient of a key — safe to show
casa machine info                    # repo path (default ~/.local/share/casa)
ls <repo>/.casa/keys                 # passphrase-sealed backups: <name>.key.age (may not exist)
casa machine doctor                  # health checks, incl. encryption setup
```

`casa secrets list` prints one target path per line and nothing else — use it
to enumerate secrets. Empty output means no secrets are managed yet.

## The registry-free key model

A key IS a file. Every age identity lives at:

```
~/.config/casa/keys/<name>.txt      (dir 0700, files 0600)
```

- **Name = filename.** No registry, no config entry, no metadata file
  anywhere. Names match `^[a-z][a-z0-9-]*$` (lowercase letters, digits,
  dashes).
- **Recipient = derived.** The public recipient comes from the private file on
  demand via `age-keygen -y` — it is never stored.
- **Default = local marker.** `~/.config/casa/keys/.default` holds the default
  key's name; if the marker is missing or stale, the first key alphabetically
  wins.
- **Zero key metadata in repos.** casa writes one generic block into the
  config template (`.casa.toml.tmpl`) that resolves everything at init time on
  each machine:

  ```toml
  encryption = "age"
  [age]
      identities = [ ... glob of ~/.config/casa/keys/*.txt ... ]
      recipient  = "age1..."   # derived from the default key's file, on the spot
  ```

  The template globs the keys directory and derives the recipient with
  `age-keygen -y` at render time. The repo describes *how to find keys*,
  never *which keys exist*: adding or removing a key never produces a repo
  diff. Two machines with different key sets render different configs from
  the same template.

If no key exists, casa creates one named `main` the first time any secret
operation needs it (a legacy `~/key.txt` is adopted as `main` automatically),
sets it default, and warns the user to back it up.

## Adding a secret (interactive — human runs it)

```bash
casa secrets add ~/.aws/credentials
```

What happens: casa seals the file with the default key and starts managing
it. With more than one key, it first asks which key should seal this file
(default preselected, each row shows name + truncated recipient); choosing a
non-default key re-seals the source with that key after the add. Then it
prints `✓ encrypted with <key> and now managing ~/...` and offers to commit
and push. Run without a path, it prompts for one.

Each secret is independently sealed for exactly one key — a `work` key can
guard work credentials while `main` guards everything else.

To manage a file as an *encrypted template* (secret with templating), use the
storage options on the files side (`casa files storage`).

## Editing a secret (interactive — human runs it)

```bash
casa secrets edit          # picker over all secrets
casa secrets edit aws      # case-insensitive substring match; picker only if ambiguous
```

casa decrypts to a temp file, opens `$EDITOR` (fallback `vi`), then re-seals.
Two guarantees worth knowing when reasoning about state:

- **Same-key re-seal.** casa probes the keys directory for the key that
  actually opens the file and re-encrypts with that same key — editing never
  silently rotates a secret to the default key.
- **Template validation.** For `.tmpl` secrets, casa executes the template
  after each edit and offers to reopen the editor on error, so mistakes
  surface immediately, not at the next apply.

After a successful edit casa runs a scriptless apply (targets rendered from
the secret update immediately) and offers to commit.

## Removing a secret (interactive — human runs it)

```bash
casa secrets remove
```

Picker only — no path argument. Stops managing the selected secret; the
decrypted file stays on disk. The encrypted source leaves the repo.

## The keys screen (interactive — human runs it)

```bash
casa secrets keys
```

Lists keys as `name  age1abcdef...  ★ default` plus a "new key" entry.
Selecting a key offers:

- **make default** — updates the `.default` marker and re-renders the config
  so *new* secrets on this machine seal with this key. Existing secrets keep
  their keys.
- **backup to repo (passphrase)** — see below.
- **delete** — with orphan protection, see below.
- **push to doppler / pull from doppler** — shown only when the `doppler` CLI
  is installed.

### Deleting a key — orphan re-encryption

Deleting a key deletes its identity file, which would make anything sealed
only for it unreadable forever. Before deleting, casa scans every encrypted
source for *orphans* — files this key opens but no surviving key can. If any
exist, casa lists them by target path, asks which surviving key to use, and
re-encrypts them first. Deleting the only key is refused outright
(`create another first`). After a final confirmation casa also removes the
key's passphrase backup from `.casa/keys/` (a fresh machine must never try
to restore a dead key) and re-renders the config.

### Passphrase-sealed repo backups (opt-in)

"backup to repo (passphrase)" seals the *private* identity with a passphrase
the human chooses (`age --encrypt --passphrase --armor`) and writes:

```
<repo>/.casa/keys/<name>.key.age
```

Armored base64 text, safe to commit — protected by the passphrase alone, not
by any key. The repo carries only that sealed data: the matching
`run_once_before` restore script is GENERATED from the installed casa binary,
gitignored, and never committed. **Never type, invent, or store the
passphrase — it exists only in the human's head and at their terminal.**

### Doppler transfer

With a doppler project set up (`doppler setup`), the keys screen can push a
private identity to doppler as env var:

```
CASA_AGE_KEY_<NAME>       # name uppercased, dashes → underscores (work-vault → WORK_VAULT)
```

"pull from doppler" restores the identity to `~/.config/casa/keys/<name>.txt`
(0600) and re-renders the config. Trade-off: convenient across many machines,
but the plaintext private key lives in a third-party secrets manager.

## Fresh machine — how keys arrive

```bash
curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <github-user>
```

During `casa machine setup`, the generated restore script runs before
anything needs decrypting: it globs `.casa/keys/*.key.age`, and for each
backup asks `Restore age key '<name>' from the repo backup? [Y/n]`, then
prompts for that key's passphrase. Skipping is always allowed — apply
proceeds, and only files that key guards fail to decrypt (restore later via
`casa secrets keys`). The script is fully generic: no key names appear in it,
and it skips cleanly if `age` is missing or the key already exists.

Alternatives, in order of convenience:

| Method | Needs | Trade-off |
| --- | --- | --- |
| Repo backup + restore script | repo + each key's passphrase | zero infrastructure; safety rests on the passphrase |
| Manual copy | access to both machines | `scp ~/.config/casa/keys/main.txt new:~/.config/casa/keys/` — nothing touches the repo; easy to forget new keys |
| Doppler | a doppler project | plaintext key in a third-party manager |

Whichever route, the config's `[age]` glob picks restored keys up
automatically — nothing to register.

## Debugging decrypt failures

1. `casa machine doctor` — surfaces encryption problems.
2. `ls ~/.config/casa/keys` — is the expected `<name>.txt` present? A missing
   key is the usual cause on a new machine; restore it via `casa secrets keys`
   (repo backup or doppler pull) or manual copy.
3. `casa secrets list` errors or an apply failing on one file usually means
   that file is sealed for a key this machine doesn't have — no surviving
   local key opens it. The fix is getting that key here, never re-generating
   one with the same name (a new key is a different identity).
4. `age` missing entirely: `brew install age`, then re-run.

## Under the hood

chezmoi renders and applies the files — encrypted sources are chezmoi
`encrypted_*` files using its native age support, and the `[age]` block above
is standard chezmoi config. The repo stays a valid chezmoi repo, so you can
leave casa at any time and keep everything: your keys still live in
`~/.config/casa/keys` and plain chezmoi decrypts with the same config.

## Threat model, in two lines

**Repo leaked:** attacker sees armored ciphertexts and (optionally)
passphrase-sealed key backups — unreadable without a private identity or the
backup passphrase. **Machine compromised:** anything running as the user can
read `~/.config/casa/keys` and decrypt everything, as with any age setup —
disk encryption and account hygiene are the defense at rest. Losing the keys
directory without a backup makes secrets permanently unreadable.
