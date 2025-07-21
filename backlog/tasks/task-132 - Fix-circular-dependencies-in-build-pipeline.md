---
id: task-132
title: Fix circular dependencies in build pipeline
status: To Do
assignee: []
created_date: '2025-07-20'
labels:
  - high
  - architecture
  - dependencies
dependencies: []
---

## Description

Build pipeline exhibits circular import patterns between build, registry, scanner, and server packages impacting maintainability

## Acceptance Criteria

- [ ] Event bus mediator pattern implemented
- [ ] Direct dependencies between packages removed
- [ ] Event-driven communication established
- [ ] Eventual consistency for state synchronization
- [ ] No circular imports detected by go mod graph
