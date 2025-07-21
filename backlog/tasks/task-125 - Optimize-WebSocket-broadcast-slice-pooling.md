---
id: task-125
title: Optimize WebSocket broadcast slice pooling
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
priority: high
---

## Description

High-priority performance issue where WebSocket broadcast loop creates new slice for failed clients on every broadcast, causing memory pressure and GC overhead under high connection load.

## Acceptance Criteria

- [ ] Implement pre-allocated slice pools for failed client collection
- [ ] Add batch cleanup for failed connections
- [ ] Reduce GC pressure during broadcast storms by 40-60%
- [ ] Maintain WebSocket connection reliability
- [ ] Add performance benchmarks for broadcast scenarios
