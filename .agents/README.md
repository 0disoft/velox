# Agent Workspace

- Status: Active
- Owner: Project maintainer

## Purpose

This directory routes coding agents to the smallest procedure and checklist
that owns a Velox change. It does not replace product, architecture, command,
or validation sources of truth.

## Read Order

1. AGENTS.md
2. VALIDATION.md
3. CHECKLIST.md
4. .agents/context-map.md
5. The selected skill and checklist
6. The primary product or architecture contract
7. Relevant implementation and tests when they exist

## Primary Routes

- Product scope: docs/product/02-spec.md
- Architecture: docs/architecture/ and docs/adr/
- CLI: .agents/skills/cli-tool/SKILL.md
- Feature work: .agents/skills/feature/SKILL.md
- Bug fixes: .agents/skills/bugfix/SKILL.md
- Refactoring: .agents/skills/refactor/SKILL.md
- Test hardening: .agents/skills/test-hardening/SKILL.md
- Dependency changes: .agents/skills/dependency-upgrade/SKILL.md
- Operational changes: .agents/skills/ops-change/SKILL.md

## Required Checklists

- Security-sensitive changes: .agents/checklists/security.md
- Performance-sensitive changes: .agents/checklists/performance.md
- CLI changes: .agents/checklists/cli-tool.md
- Dependency changes: .agents/checklists/dependency.md
- Operational changes: .agents/checklists/ops-change.md

## Rules

- Product scope comes from docs/product/02-spec.md, not generated suggestions.
- Architecture changes use an ADR and synchronized derived documents.
- Planned behavior is not reported as implemented.
- Unconfigured validation is reported as skipped with a reason.
- Agents do not invent build, test, release, or deployment commands.
- Generated output, caches, and external AI responses are not source truth.
- Secrets, credentials, private paths, and raw user data do not enter agent
  artifacts.

## Current Stage

The repository is pre-implementation. Agent work should prioritize M0
feasibility, contract consistency, benchmark honesty, security boundaries, and
small reversible changes over feature breadth.
