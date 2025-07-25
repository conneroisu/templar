---
id: task-62
title: Optimize WebSocket broadcasting performance
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
---

## Description

WebSocket broadcasting in internal/server/websocket.go has O(n) performance with client count. Need lock-free implementation and parallel message sending for better scalability.

## Acceptance Criteria

- [ ] Implement lock-free client registry with atomic operations
- [ ] Add fan-out goroutines for parallel message sending
- [ ] Implement client connection pooling and reuse
- [ ] Support 10x more concurrent clients than current
- [ ] All WebSocket functionality preserved
