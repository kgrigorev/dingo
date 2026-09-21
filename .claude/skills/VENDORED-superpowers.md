# Vendored: Superpowers skills

The skill directories below are vendored (copied verbatim) from the
[Superpowers](https://github.com/obra/superpowers) project as plain project
skills under `.claude/skills/`. They are not installed as a plugin: the
`superpowers-marketplace` plugin declaration cannot be used in headless/cloud
sessions because registering an `extraKnownMarketplaces` source requires an
interactive folder-trust dialog that those sessions never receive, so the
marketplace never registers and no skills load. Vendoring the skill files
directly bypasses that trust gate — they load the same way any other project
skill does.

## Source

- Repository: https://github.com/obra/superpowers
- Commit: `b36e0829c6d0140e93cfef2ca599b1b07d4a7797` (tag `v6.3.0`)
- License: MIT (full text below)

Only the contents of the upstream `skills/` directory were copied. The
plugin scaffolding (`.claude-plugin/`, `.codex-plugin/`, `.agents/`, `hooks/`,
etc.) was intentionally left out — in particular the `SessionStart` hook that
auto-injects the `using-superpowers` skill as context on every session start
was not brought over, so `using-superpowers` here loads like any other skill
instead of being force-injected.

## Vendored skills (15)

- `brainstorming`
- `dispatching-parallel-agents`
- `executing-plans`
- `finishing-a-development-branch`
- `receiving-code-review`
- `requesting-code-review`
- `subagent-driven-development`
- `systematic-debugging`
- `test-driven-development`
- `using-git-worktrees`
- `using-superpowers`
- `verification-before-completion`
- `writing-plans`
- `writing-skills`

## No auto-update

These files are a point-in-time copy, not a live dependency — there is no
Renovate/Dependabot-style mechanism keeping them in sync with upstream.
To re-sync: clone the upstream repo at the desired ref, diff its `skills/`
directory against `.claude/skills/`, and manually re-copy the directories
you want to update (re-checking for plugin-scaffolding leakage and
`CLAUDE_PLUGIN_ROOT`/`PLUGIN_ROOT` references each time).

## License

```
MIT License

Copyright (c) 2025 Jesse Vincent

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
