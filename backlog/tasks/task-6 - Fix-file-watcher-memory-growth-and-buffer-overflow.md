---
id: task-6
title: Fix file watcher memory growth and buffer overflow
status: To Do
assignee: []
created_date: '2025-07-20'
labels:
  - performance
  - reliability
dependencies: []
---

## Description

File watcher shows linear memory growth (86KB → 3MB) and queue overflow issues under concurrent operations. Optimize memory usage and buffer management.

## Acceptance Criteria

- [ ] Memory growth eliminated or made logarithmic
- [ ] Queue overflow issues resolved
- [ ] Event buffer sizing optimized for scale
- [ ] Memory pooling implemented for ChangeEvent structs
- [ ] Batch processing added for multiple file events
