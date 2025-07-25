---
id: task-156
title: Implement Command Query Separation in build pipeline
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies:
  - task-95
  - task-132
---

## Description

Dave (Architecture Agent) identified HIGH-severity violation of Command Query Separation in build pipeline. Methods mix command and query operations with unclear goroutine lifecycle making the system unpredictable and prone to race conditions.

## Acceptance Criteria

- [ ] Create separate BuildCommand and BuildQuery interfaces
- [ ] Extract CacheManager as separate component
- [ ] Extract WorkerPoolManager with clear lifecycle
- [ ] Extract MetricsCollector as separate concern
- [ ] Implement proper error handling and result reporting
- [ ] Build operations return status and errors properly
- [ ] Clear separation between mutations and queries
