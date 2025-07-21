---
id: task-85
title: Split monolithic server package into focused modules
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
---

## Description

Code quality analysis identified the server package as handling too many responsibilities: HTTP serving, WebSocket management, security policies, and middleware. This violates Single Responsibility Principle and makes testing and maintenance difficult.

## Acceptance Criteria

- [ ] HTTP server logic extracted to internal/http package
- [ ] WebSocket functionality moved to internal/websocket package
- [ ] Security policies isolated in internal/security package
- [ ] Middleware separated into internal/middleware package
- [ ] All tests pass after refactoring
- [ ] No functionality regression
