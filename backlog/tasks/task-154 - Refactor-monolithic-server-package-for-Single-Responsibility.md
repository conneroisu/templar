---
id: task-154
title: Refactor monolithic server package for Single Responsibility
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies:
  - task-85
  - task-131
---

## Description

Dave (Architecture Agent) identified HIGH-severity God Object pattern in PreviewServer with 600+ lines handling HTTP routing WebSocket management file watching build coordination and business logic violating SRP.

## Acceptance Criteria

- [ ] Extract HTTPRouter for route handling
- [ ] Extract WebSocketManager for connection management
- [ ] Extract MiddlewareChain for request processing
- [ ] Extract ServiceOrchestrator for component coordination
- [ ] Use dependency injection for all dependencies
- [ ] Each component has single clear responsibility
- [ ] Improved unit testing for individual concerns
