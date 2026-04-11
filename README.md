# quickswitch (qs)

A CLI tool for managing and launching AI coding tools (Claude Code, Codex) with different API profiles. Each terminal window can use a completely different provider configuration without affecting other running instances.

## The Problem

Claude Code reads credentials and settings from `~/.claude/settings.json` — a single global file. Switching providers (e.g. from one API proxy to another) affects every running instance. Tools like cc-switch work by overwriting this file, making per-window isolation impossible.

## How It Works

quickswitch solves this with `CLAUDE_CONFIG_DIR`, an environment variable that tells Claude Code to use a different directory instead of `~/.claude/`. For each profile, quickswitch:

1. Reads `~/.config/quickswitch/claude/default-settings.json` (shared config: hooks, permissions, model, etc.)
2. Merges the profile's env vars (`ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`) into the `env` section
3. Writes the merged result to `~/.config/quickswitch/claude/runtime/<profile>/settings.json`
4. Copies `~/.claude.json` (theme, onboarding state) so the setup wizard doesn't appear
5. Symlinks all data directories (`sessions`, `history`, `plans`, `tasks`, etc.) back to `~/.claude/` so data is shared across profiles
6. Strips auth env vars from the shell to prevent the current session's credentials from leaking in
7. Launches `claude` with `CLAUDE_CONFIG_DIR` pointing to the runtime directory

The result: each window has isolated credentials but shared sessions, history, and all other data.

## Config Directory Structure

```
~/.config/quickswitch/claude/
├── default-settings.json          # shared settings (hooks, permissions, model, language, etc.)
├── env.json                       # all profiles and their API credentials
└── runtime/
    ├── packycode/
    │   ├── settings.json          # default-settings.json + packycode env merged
    │   ├── .claude.json           # copied from ~/.claude.json (preferences)
    │   ├── sessions -> ~/.claude/sessions
    │   ├── history.jsonl -> ~/.claude/history.jsonl
    │   ├── plans -> ~/.claude/plans
    │   └── ...                    # all other data symlinked to ~/.claude/
    └── zhipu-luo/
        ├── settings.json
        ├── .claude.json
        └── ...
```

### env.json format

```json
{
  "profiles": {
    "packycode": {
      "ANTHROPIC_AUTH_TOKEN": "sk-...",
      "ANTHROPIC_BASE_URL": "https://..."
    },
    "personal": {
      "ANTHROPIC_AUTH_TOKEN": "sk-..."
    }
  }
}
```

### default-settings.json

A standard Claude Code `settings.json` without the `env` section. Credentials are managed by profiles; everything else (hooks, permissions, model, language, etc.) lives here and is shared across all profiles.

## Commands

### Launch Claude

```bash
qs claude <profile>              # launch with a specific profile
qs claude <profile> --resume     # extra args are passed through to claude
qs claude                        # launch using only default-settings.json, no credentials
```

### Manage Profiles

```bash
qs add                           # interactive: select tool, enter profile name and credentials
qs list                          # list all profiles for all tools
qs list claude                   # list claude profiles only
qs delete                        # interactive: select tool and profile to delete
```

## Installation

```bash
git clone https://github.com/nofrish/quickswitch
cd quickswitch
go build -o qs .
mv qs /usr/local/bin/
```

## Setup

1. Copy your current `~/.claude/settings.json` to `~/.config/quickswitch/claude/default-settings.json` and remove the `env` section from it (credentials go in profiles instead)
2. Run `qs add` to create your first profile

## Key Design Decisions

**Why symlinks for shared data?**
Sessions, history, plans, and tasks are valuable and should persist regardless of which profile you use. Symlinks let the runtime directory act as a full `CLAUDE_CONFIG_DIR` while keeping all data in the original `~/.claude/` location.

**Why strip shell auth env vars?**
Claude Code gives shell environment variables higher priority than `settings.json`. If your shell already has `ANTHROPIC_AUTH_TOKEN` set (e.g. from cc-switch), it would override the profile's credentials. Stripping them ensures the profile's `settings.json` values are used.

**Why not just modify `~/.claude/settings.json`?**
That's exactly what cc-switch does, and it means every window shares the same credentials. `CLAUDE_CONFIG_DIR` gives each window a fully isolated config directory.
