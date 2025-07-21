---
id: task-26
title: Extract business logic from CLI commands
status: To Do
assignee: []
created_date: '2025-07-20'
labels:
  - architecture
  - refactoring
dependencies: []
---

## Description

CLI commands in /cmd/ package contain business logic mixed with CLI concerns, making business logic difficult to test independently

## Acceptance Criteria

- [ ] Move business logic from /cmd/ to /internal/ packages
- [ ] Keep commands as thin wrappers around business logic
- [ ] Maintain all existing CLI functionality
- [ ] Update command tests to focus on CLI concerns
- [ ] Add unit tests for extracted business logic
- [ ] Ensure proper error handling propagation
