---
title: Agentic-SDD
description: Spec-driven engineering skills and preview-first installer for coding agents, with human approval and bounded verification.
---

Agentic-SDD is an open-source toolkit of **18 portable core engineering skills** (`skills-files/`) plus an **optional OpenCode orchestration pack** (`opencode-pack/`), delivered by a preview-first Go CLI (`asdx-core`) and terminal UI companion (`asdx`). It helps you scope work clearly, keep human approval explicit, verify changes with real checks, and learn as you build.

> **Status note:** This documentation covers the current repository checkout (`unreleased`). Public metadata for release `v0.11.0` (tag and Homebrew cask formula/archive digest) is observed, while archive binary execution, live macOS installation, and public GitHub Pages deployment are not yet verified.

## Who Agentic-SDD is for

- **Individual developers using coding agents:** Turn vague prompts into bounded changes you can actually review, test, and maintain.
- **Learners, interns, and junior engineers:** Practice debugging, scope control, and code review alongside an agent, with optional explanation skills you invoke only when you want them.
- **Experienced engineers and maintainers:** Keep requirements (`spec.md`), design (`plan.md`), and execution state (`tasks.md`) connected across sessions without forcing multi-agent overhead on everyday work.

One core methodology serves all three groups through one main agent by default. Optional multi-profile delegation is available for OpenCode, never mandatory.

## When Agentic-SDD is unnecessary

Skip the toolkit or keep it minimal when ceremony would outweigh the change:

- **One-off questions, experiments, or throwaway scripts:** Your coding agent's normal chat loop is enough.
- **Small, well-understood bug fixes:** Even in an Agentic-SDD repository, a bounded fix with clear scope and a focused test uses the [direct fix](tutorials/direct-fix.md) path—no `spec.md`, `plan.md`, or `tasks.md` files.
- **Hard security containment:** Agentic-SDD skills and profile permissions are workflow guidance, not an OS sandbox, credential boundary, or guarantee that a model will obey every instruction.

If you are weighing this toolkit against alternatives, see [Related approaches](introduction/related-approaches.md) before deciding whether to add anything.

## Choose your path

| What you want to do | Go to |
|---|---|
| Understand skills, profiles, features, direct fixes, batches, and consent | [Core concepts](introduction/concepts.md) |
| Check where files go and what is verified across clients | [Supported clients and destinations](introduction/clients.md) |
| Decide between `asdx-core` (CLI) and `asdx` (TUI), and core vs pack | [Choose your setup](getting-started/choose-setup.md) |
| Install on macOS or build from source on Linux | [Installation](getting-started/installation.md) |
| Run a write-free preview in a temporary `--home` directory | [First five minutes](getting-started/first-five-minutes.md) |
| Walk through a small bug fix with no feature paperwork | [Direct fix tutorial](tutorials/direct-fix.md) |
| Walk through `spec.md` → `plan.md` → `tasks.md`, combined approval, and one batch | [First feature tutorial](tutorials/first-feature.md) |
| Install, update, back up, or restore skills and profiles | [Manage skills](how-to/skills.md) · [Updates and recovery](how-to/updates-and-recovery.md) |
| Look up exact commands, TUI screens, file paths, or skill IDs | [CLI](reference/cli.md) · [TUI](reference/tui.md) · [Files and backups](reference/files-and-backups.md) · [Skills](reference/skills.md) |
