---
id: task-126
title: Fix critical CLI flag conflict in host parameter
status: Done
assignee: []
created_date: '2025-07-20'
updated_date: '2025-07-20'
labels:
  - critical
  - cli
  - bug
dependencies: []
---

## Description

The host flag uses -h shorthand which conflicts with built-in help flag causing CLI help functionality to panic

## Acceptance Criteria

- [ ] Host flag uses different shorthand (not -h)
- [ ] CLI help functions work correctly for all commands
- [ ] No breaking changes to existing functionality

## Implementation Notes

Fixed CLI flag conflict by removing -h shorthand from host flag. The host flag now only uses --host (no shorthand) to avoid conflict with built-in -h help flag. Verified help functionality works correctly for all commands.
