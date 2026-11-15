---
id: bug_a5b0
title: [iOS native] saveBody drops the fetched content_hash, so the next page sync discards a current body
type: bug
status: open
priority: 2
labels: []
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T10:09:25Z
updated_at: 2026-11-15T10:09:25Z
---


## Problem
`ArticleStore.saveBody(id:body:)` wrote only `body`. `ArticlePrefetch` and `ReaderModel.loadBody` fetch a detail carrying `contentHash` and dropped it, so a body fetched for changed server content was stored next to the old hash and the next `syncPage` (hash differs) threw it away. Raised in review of PR #468.

## Plan
- [x] `saveBody(id:body:contentHash:)` writes `content_hash` with the body
- [x] `ArticlePrefetch` fetch closure returns body and hash; pass both
- [x] `ReaderModel.loadBody` passes `detail.contentHash`
- [x] `ArticleStoreTests`: a body saved with a new hash survives a `syncPage` carrying that hash; existing hash tests save with the row's hash
- [x] iOS App Checks green on the PR

## Review
PR andrew-craig/cairn-reader#469 (into feature/ios-native-migration). The store, prefetch and reader tests passed in every run. Two runs went red on unrelated tests: run 52 on the three `StubURLProtocol` timeout tests seen on #468, run 53 on Feed/Reader tests while the runner stalled (18–88 s per test). The re-run of 53 is green. The suite is still unstable on CI; the suspected cause is `StubURLProtocol` calling back only in `commonModes`.
