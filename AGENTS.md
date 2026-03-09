# Repository Guidelines

## Project Structure & Module Organization
This repository is documentation-first and currently contains architecture and planning artifacts.
- `README.md`: entry point and project consensus summary.
- `docs/ARCHITECTURE_V0.md`: runtime architecture and event-flow design.
- `docs/AGENT_SANDBOX_INTEGRATION_V0.md`: sandbox integration details.
- `docs/NEXT_STEPS.md`: execution roadmap.
- `docs/CONVERSATION_LOG.md`: decision history.

Keep top-level docs focused and atomic. Prefer updating an existing file over creating overlapping documents.

## Build, Test, and Development Commands
There is no compiled application in this repository yet, so contributor workflows are doc-review focused.
- `rg --files`: list tracked files quickly.
- `rg -n "TODO|FIXME" *.md`: find open edits before submitting.
- `git diff -- *.md`: review only markdown changes.
- `git status --short`: confirm final change set.

If you use a Markdown linter locally, run it before opening a PR.

## Coding Style & Naming Conventions
Use concise technical writing with stable terminology.
- Headings: clear, task-oriented (`## Redis Stream 设计`, `## 验收标准`).
- Filenames: uppercase snake-style with version suffix when needed (example: `docs/ARCHITECTURE_V0.md`).
- Keep examples executable and explicit (keys like `stream:session:{session_key}`, API paths, JSON fields).
- Avoid introducing new terms for existing concepts; reuse repository vocabulary.

## Testing Guidelines
Testing here means consistency validation.
- Verify links, commands, and key names across files.
- Ensure new decisions do not conflict with `README.md` consensus.
- When changing architecture behavior, update both design and execution docs in the same PR.
- For substantial changes, include a short “before/after” note in the PR description.

## Commit & Pull Request Guidelines
Git history favors short, typed messages such as `docs: update project docs`.
- Commit format: `<type>: <summary>` (for example, `docs: clarify session_key rules`).
- Keep one logical change per commit.
- PRs should include: purpose, affected files, key decision changes, and reviewer checklist items.
- Link related issue/task IDs when available.

## Security & Configuration Notes
Do not commit secrets, tokens, tenant IDs, or internal endpoints in examples. Use placeholders and redact sensitive identifiers.
