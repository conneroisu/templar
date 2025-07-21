---
id: task-131
title: Refactor monolithic server package for maintainability
status: To Do
assignee: []
created_date: '2025-07-20'
labels:
  - high
  - architecture
  - refactoring
dependencies: []
---

## Description

Server package contains 601 lines with multiple responsibilities mixing HTTP routing, WebSocket management, middleware setup, and business logic

## Acceptance Criteria

- [ ] Server.go split into focused modules (HTTP
- [ ] WebSocket
- [ ] middleware)
- [ ] WebSocket functionality extracted to dedicated package
- [ ] HTTP handlers separated by concern
- [ ] Middleware package with composable functions created
- [ ] All existing functionality preserved
- [ ] Test coverage maintained
