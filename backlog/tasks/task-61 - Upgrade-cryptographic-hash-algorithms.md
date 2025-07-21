---
id: task-61
title: Upgrade cryptographic hash algorithms
status: To Do
assignee: []
created_date: '2025-07-20'
labels: []
dependencies: []
---

## Description

MD5 hash algorithm used in internal/scanner/scanner.go line 98 is cryptographically weak and slower than modern alternatives. Should upgrade to SHA-256 for security and performance.

## Acceptance Criteria

- [ ] Replace MD5 with SHA-256 in file scanning
- [ ] Update hash comparison logic throughout codebase
- [ ] Maintain backward compatibility where needed
- [ ] All existing hash functionality preserved
- [ ] Security tests validate new hash implementation
