# Secrets and keys

Your secrets travel with your repo as age-encrypted files. The keys that open
them never touch a repo — they are plain files on your machine, and casa
manages them: creation, per-file key choice, safe editing, backup, and
transfer. Under the hood, chezmoi's native age support does the sealing and
rendering. The `age` CLI must be installed (`brew install age`).

```console
$ casa secrets add [path]    # encrypt and start managing a file
$ casa secrets edit [name]   # pick a secret, decrypt, edit, re-encrypt
$ casa secrets keys          # encryption keys — create, default, delete, doppler
$ casa secrets list          # list encrypted files
```

## The registry-free key model

A key *is* a file. Every age identity lives at:

```
~/.config/casa/keys/<name>.txt
```

There is no registry, no config entry, no metadata file. The key's name is its
filename, its public recipient is derived on demand with `age-keygen -y`, and
the default key is whichever name is written in a local
`~/.config/casa/keys/.default` marker (falling back to the first key,
alphabetically, if the marker is missing or stale).

Nothing key-related is committed to your repo — no names, no paths, no
recipients. Instead, casa writes one generic `[age]` block into your config
template (`.casa.toml.tmpl`) that resolves everything at `chezmoi init` time on
each machine:

- `identities` is a glob of `~/.config/casa/keys/*.txt`
- `recipient` is derived from the default key's file via `age-keygen -y`

This is deliberate: the repo describes *how to find keys*, never *which keys
exist*. Two machines with different key sets render different configs from the
same template, and adding or removing a key never produces a repo diff. The
only optional exception is passphrase-sealed backups (see below), which are
themselves encrypted.

If you have no key yet, casa creates one named `main` the first time any secret
operation needs it (a legacy `~/key.txt` is adopted as `main` automatically).

## Adding a secret

```console
$ casa secrets add ~/.aws/credentials
```

casa seals the file with the default key (via chezmoi's encrypted add) and
starts managing it. If you have more than one key, casa first asks which key should seal this
file (default preselected, each shown with a truncated recipient); choosing a
non-default key re-seals the source file with that key after the add. Each
secret is therefore independently sealed for exactly one key — a `work` key can
protect work credentials while `main` protects everything else.

## Editing a secret

```console
$ casa secrets edit            # picker over all secrets
$ casa secrets edit aws        # substring match; picker only if ambiguous
```

casa decrypts the secret to a temp file, opens `$EDITOR` (falling back to
`vi`), then re-seals. Two guarantees:

- **Same-key re-seal.** casa probes your keys to find which one actually opens
  the file and re-encrypts with that same key — editing never silently rotates
  a secret to the default key.
- **Template validation.** If the secret is an encrypted template (`.tmpl`),
  casa executes the template after each edit and, on error, offers to reopen
  the editor — so template mistakes surface now, not at the next apply.

After a successful edit casa runs a scriptless apply so any targets rendered
from the secret update immediately, then offers to commit.

To track a file as an encrypted template in the first place, use the storage
options on the files side. See [files](configs.md).

## The keys screen

```console
$ casa secrets keys
```

Lists your keys (name, truncated recipient, `★ default`) plus a "new key"
entry. Key names are lowercase letters, digits, and dashes. Selecting a key
offers:

- **make default** — updates the `.default` marker and re-renders the chezmoi
  config so *new* secrets on this machine encrypt with this key. Existing
  secrets keep their keys.
- **backup to repo (passphrase)** — see below.
- **delete** — with orphan protection, see below.
- **push to doppler** / **pull from doppler** — shown only when the `doppler`
  CLI is installed.

### Deleting a key

Deleting a key means deleting its identity file, which would make anything
sealed only for it unreadable forever. So before deleting, casa scans every
encrypted source and finds *orphans*: files this key can open but no surviving
key can. If there are any, casa lists them by target path, asks which surviving
key to use, and re-encrypts them before the delete proceeds. Deleting your only
key is refused outright.

After confirmation casa also removes the key's passphrase backup from the repo
(a new machine must never try to restore a dead key) and re-renders the config.

### Backing up a key to the repo

"backup to repo (passphrase)" seals the *private* identity with a passphrase
you choose (`age --passphrase --armor`) and writes it to:

```
<source-dir>/.casa/keys/<name>.key.age
```

The armored (base64 text) output is safe to commit — it is protected by the
passphrase, not by any key. The repo carries only that sealed data: the
matching `run_once_before` restore script is generated by the installed casa
binary, gitignored, and never committed. On a fresh machine it runs before
anything needs decrypting — it globs `.casa/keys/*.key.age`, prompts for each
key's passphrase, and restores the identities into `~/.config/casa/keys`. The
script is fully generic — no key names appear in it — and skips cleanly if
`age` is not installed yet or the key already exists.

## Getting keys to a new machine

Three options, in rough order of convenience:

| Method | What you need | Trade-off |
| --- | --- | --- |
| Repo backup + restore script | The repo and each key's passphrase | Zero extra infrastructure; runs automatically during `casa machine setup`. The sealed private key lives in the repo, so its safety rests entirely on the passphrase. |
| Copy the file yourself | Access to both machines | `scp ~/.config/casa/keys/main.txt new:~/.config/casa/keys/` — nothing touches the repo, but it is manual and easy to forget for newly created keys. |
| Doppler | A doppler project (`doppler setup`) | `push to doppler` stores the identity as `CASA_AGE_KEY_<NAME>` (name uppercased, dashes to underscores); `pull from doppler` restores it and re-renders the config. Convenient across many machines, but the plaintext private key lives in a third-party secrets manager. |

Whichever you use, the config's `[age]` glob picks the restored keys up
automatically — there is nothing to register. See [machine](machine.md) for
the full new-machine flow.

## Threat model

**The repo.** Someone with your repo (public or leaked) sees armored age
ciphertexts and, optionally, passphrase-sealed key backups. Secrets are sealed
for specific recipients; without a private identity file they are unreadable.
The backups are additionally independent of your keys — they fall only to a
guessed or reused passphrase, so treat that passphrase like a master password
and skip the repo backup entirely if you prefer to move keys out of band. The
repo never names your keys or recipients beyond what ciphertexts inherently
reveal.

**The machine.** The private identities in `~/.config/casa/keys` are plaintext
files (directory `0700`, keys `0600`). Anyone with your local user account —
malware included — can read them and decrypt every secret, exactly as with
any age-based setup. casa does not attempt to protect keys from your own
account; disk encryption and account hygiene are the defense at rest. Losing
the directory without a backup makes your secrets permanently unreadable, which
is why key creation warns you to back up immediately.
