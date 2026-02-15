# Optimize N+1 Query in Recommendation Flow

**Priority:** P3
**Status:** pending
**Task ID:** 19

## Problem

Recording recommendations happens in a loop, creating N database calls instead of 1 batch operation.

## Impact

Poor performance when recommending multiple articles. Database becomes bottleneck with N sequential round trips instead of single batched operation.

**Performance impact:**
- Before: N queries (5 INSERTs + 5 UPDATEs = 10 queries for 5 articles)
- After: 2 queries (1 batch INSERT + 1 batch UPDATE)
- 5x reduction in database round trips

## Current Implementation

Location: `services/explore/recommender/internal/db/article_repository.go:327`

```go
for _, article := range recommendations {
    if err := r.RecordRecommendation(ctx, userID, article.ID); err != nil {
        // Individual INSERT for each article
    }
}
```

## Proposed Solution

Add batch method to `article_repository.go`:

```go
func (r *ArticleRepository) RecordRecommendationsBatch(ctx context.Context, userID string, articleIDs []string) error {
    if len(articleIDs) == 0 {
        return nil
    }

    // Build batch INSERT
    query := `
        INSERT INTO user_article_recommendations (user_id, article_id, recommended_at)
        VALUES `

    values := make([]interface{}, 0, len(articleIDs)*2)
    placeholders := make([]string, 0, len(articleIDs))

    for i, articleID := range articleIDs {
        placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, NOW())", i*2+1, i*2+2))
        values = append(values, userID, articleID)
    }

    query += strings.Join(placeholders, ", ")
    query += " ON CONFLICT (user_id, article_id) DO NOTHING"

    _, err := r.db.ExecContext(ctx, query, values...)
    if err != nil {
        return fmt.Errorf("failed to record recommendations batch: %w", err)
    }

    // Also batch increment recommends counter
    return r.incrementRecommendsCountBatch(ctx, articleIDs)
}

func (r *ArticleRepository) incrementRecommendsCountBatch(ctx context.Context, articleIDs []string) error {
    query := `
        UPDATE articles
        SET recommends = recommends + 1
        WHERE id = ANY($1)`

    _, err := r.db.ExecContext(ctx, query, articleIDs)
    return err
}
```

Use in recommendation engine:
```go
articleIDs := make([]string, len(recommendations))
for i, article := range recommendations {
    articleIDs[i] = article.ID
}

if err := r.articleRepo.RecordRecommendationsBatch(ctx, userID, articleIDs); err != nil {
    return nil, fmt.Errorf("failed to record recommendations: %w", err)
}
```

## Files to Modify

- `services/explore/recommender/internal/db/article_repository.go`
- `services/explore/recommender/internal/recommend/engine.go` (or wherever recommendations are recorded)

## Testing

- Verify batch method inserts all recommendations
- Verify ON CONFLICT handling (duplicates are ignored)
- Verify recommends counter incremented correctly
- Benchmark: compare loop vs batch performance
- Test with various batch sizes (1, 5, 10, 100 items)

## Performance Impact

**Query reduction:** 5x fewer database round trips
**Latency improvement:** ~80-90% reduction for typical 5-article recommendations
**Scalability:** Improves linearly with batch size

## Notes

- Use pgx placeholders ($1, $2, etc.) for parameterization
- ON CONFLICT handles duplicate recommendations gracefully
- Consider adding metrics to track batch operation performance
