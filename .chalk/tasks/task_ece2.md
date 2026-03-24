---
id: task_ece2
title: Migrate Vault to auto-unseal via KMS
type: task
status: open
priority: 4
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:37Z
updated_at: 2026-03-21T04:20:37Z
---
The current vault-unseal watchdog stores all 5 Shamir unseal keys in plaintext. Migrate to Vault's auto-unseal using a cloud KMS.
