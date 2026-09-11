import { Article } from '../types';
import { getDb } from './db';

interface ArticleRow {
  id: string;
  url: string;
  title: string;
  description: string | null;
  image_url: string | null;
  author: string | null;
  published_date: string | null;
  reading_time: number | null;
  tags: string;
  is_read: number;
  is_favorite: number;
  added_at: number;
  read_at: number | null;
  scroll_position: number | null;
  scroll_fraction: number | null;
  body: string | null;
  content_hash: string | null;
}

// A list page from the server upserts its rows; existing rows are updated,
// never bulk-deleted. `body` and `content_hash` need special handling because
// list responses carry a hash but never the cleaned HTML itself:
// - content_hash: overwritten when the incoming row has one (COALESCE keeps
//   the stored value on a summary payload that omits it — none do today, but
//   nothing should crash if one ever does).
// - body: kept when the incoming hash is absent, or equal to what's already
//   stored (a plain refresh) — COALESCE also guards the never-carries-a-body
//   case, same as before this task. Cleared when the incoming hash differs
//   from the stored one: a body whose hash no longer matches the server's is
//   stale and must not be served offline. See task_c55c scope clarification,
//   "Selection, staleness and eviction".
//
// task_ebf1 adds two guards against a queued offline write being clobbered
// by a list sync that runs before the outbox drains:
// - A pending `delete` row for this article (an offline archive) skips the
//   insert/update entirely — the WHERE on the SELECT source makes the insert
//   produce zero rows, so ON CONFLICT never even fires. Otherwise the server
//   still listing the article would resurrect it in the store the moment a
//   sync ran, ahead of the queued DELETE actually reaching the backend.
// - Any other pending outbox row for this article (status/is_favorite/
//   scroll_position) freezes the four user-state columns at their current
//   stored value instead of accepting the server's — those are exactly the
//   fields a queued write is waiting to change, and the server hasn't seen
//   the new value yet. Keyed on article_id alone, not per field: simpler SQL,
//   and the over-freezing is transient (cleared the moment the drain
//   succeeds). `scroll_position` (the legacy column, distinct from
//   `scroll_fraction`) is not one of the four and is never guarded.
const UPSERT_SQL = `
  INSERT INTO articles (
    id, url, title, description, image_url, author, published_date,
    reading_time, tags, is_read, is_favorite, added_at, read_at,
    scroll_position, scroll_fraction, body, content_hash
  )
  SELECT
    $id, $url, $title, $description, $image_url, $author, $published_date,
    $reading_time, $tags, $is_read, $is_favorite, $added_at, $read_at,
    $scroll_position, $scroll_fraction, $body, $content_hash
  WHERE NOT EXISTS (
    SELECT 1 FROM outbox WHERE article_id = $id AND field = 'delete'
  )
  ON CONFLICT(id) DO UPDATE SET
    url = excluded.url,
    title = excluded.title,
    description = excluded.description,
    image_url = excluded.image_url,
    author = excluded.author,
    published_date = excluded.published_date,
    reading_time = excluded.reading_time,
    tags = excluded.tags,
    is_read = CASE
      WHEN EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id) THEN articles.is_read
      ELSE excluded.is_read
    END,
    is_favorite = CASE
      WHEN EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id) THEN articles.is_favorite
      ELSE excluded.is_favorite
    END,
    added_at = excluded.added_at,
    read_at = CASE
      WHEN EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id) THEN articles.read_at
      ELSE excluded.read_at
    END,
    scroll_position = excluded.scroll_position,
    scroll_fraction = CASE
      WHEN EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id) THEN articles.scroll_fraction
      ELSE excluded.scroll_fraction
    END,
    content_hash = COALESCE(excluded.content_hash, articles.content_hash),
    body = CASE
      WHEN excluded.content_hash IS NULL THEN COALESCE(excluded.body, articles.body)
      WHEN excluded.content_hash = articles.content_hash THEN COALESCE(excluded.body, articles.body)
      ELSE NULL
    END
`;

function articleToParams(article: Article): Record<string, string | number | null> {
  return {
    $id: article.id,
    $url: article.url,
    $title: article.title,
    $description: article.description ?? null,
    $image_url: article.imageUrl ?? null,
    $author: article.author ?? null,
    $published_date: article.publishedDate ?? null,
    $reading_time: article.readingTime ?? null,
    $tags: JSON.stringify(article.tags ?? []),
    $is_read: article.isRead ? 1 : 0,
    $is_favorite: article.isFavorite ? 1 : 0,
    $added_at: article.addedAt,
    $read_at: article.readAt ?? null,
    $scroll_position: article.scrollPosition ?? null,
    $scroll_fraction: article.scrollFraction ?? null,
    $body: article.content ?? null,
    $content_hash: article.contentHash ?? null,
  };
}

function rowToArticle(row: ArticleRow): Article {
  return {
    id: row.id,
    url: row.url,
    title: row.title,
    description: row.description ?? undefined,
    content: row.body ?? undefined,
    imageUrl: row.image_url ?? undefined,
    author: row.author ?? undefined,
    publishedDate: row.published_date ?? undefined,
    readingTime: row.reading_time ?? undefined,
    tags: JSON.parse(row.tags) as string[],
    isRead: row.is_read === 1,
    isFavorite: row.is_favorite === 1,
    addedAt: row.added_at,
    readAt: row.read_at ?? undefined,
    scrollPosition: row.scroll_position ?? undefined,
    scrollFraction: row.scroll_fraction ?? undefined,
    contentHash: row.content_hash ?? undefined,
  };
}

export const ArticleStore = {
  /** Upsert a page of articles from the server. Never deletes existing rows. */
  async upsertMany(articles: Article[]): Promise<void> {
    if (articles.length === 0) return;
    const db = await getDb();
    for (const article of articles) {
      await db.runAsync(UPSERT_SQL, articleToParams(article));
    }
  },

  /**
   * The most recently added stored articles, for the Read screen's initial
   * render. Reads never reject: a store failure degrades to "nothing
   * cached" (logged via console.error) rather than blocking callers that
   * depend on a subsequent network refresh always firing.
   */
  async listRecent(limit: number): Promise<Article[]> {
    try {
      const db = await getDb();
      const rows = await db.getAllAsync<ArticleRow>(
        'SELECT * FROM articles ORDER BY added_at DESC LIMIT $limit',
        { $limit: limit },
      );
      return rows.map(rowToArticle);
    } catch (error) {
      console.error('Error loading recent articles:', error);
      return [];
    }
  },

  /**
   * Stored articles marked as favorites, for the Bookmarks screen's initial
   * render. Reads never reject — see `listRecent`.
   */
  async listFavorites(): Promise<Article[]> {
    try {
      const db = await getDb();
      const rows = await db.getAllAsync<ArticleRow>(
        'SELECT * FROM articles WHERE is_favorite = 1 ORDER BY added_at DESC',
        {},
      );
      return rows.map(rowToArticle);
    } catch (error) {
      console.error('Error loading favorite articles:', error);
      return [];
    }
  },

  /** Reads never reject — see `listRecent`. Resolves to null on failure. */
  async getById(id: string): Promise<Article | null> {
    try {
      const db = await getDb();
      const row = await db.getFirstAsync<ArticleRow>('SELECT * FROM articles WHERE id = $id', {
        $id: id,
      });
      return row ? rowToArticle(row) : null;
    } catch (error) {
      console.error('Error loading article by id:', error);
      return null;
    }
  },

  /**
   * Unread/reading articles with no cached body, newest first, for the
   * prefetch service (task_c55c). `upsertMany` already clears `body` when a
   * list sync's incoming hash differs from what's stored, so this is the
   * full selection: nothing else needs to compare hashes. Reads never
   * reject — see `listRecent`.
   */
  async listPrefetchCandidates(limit: number): Promise<Article[]> {
    try {
      const db = await getDb();
      const rows = await db.getAllAsync<ArticleRow>(
        'SELECT * FROM articles WHERE is_read = 0 AND body IS NULL ORDER BY added_at DESC LIMIT $limit',
        { $limit: limit },
      );
      return rows.map(rowToArticle);
    } catch (error) {
      console.error('Error loading prefetch candidates:', error);
      return [];
    }
  },

  /** Cache a freshly fetched article body (cleaned HTML) opportunistically. */
  async saveBody(id: string, body: string): Promise<void> {
    const db = await getDb();
    await db.runAsync('UPDATE articles SET body = $body WHERE id = $id', {
      $id: id,
      $body: body,
    });
  },

  /**
   * Drop cached bodies for rows outside the `limit` most recent by
   * `added_at`, e.g. after a prefetch run, so the store never keeps more
   * bodies than the Read screen's own retention window.
   */
  async evictBodiesOutsideCap(limit: number): Promise<void> {
    const db = await getDb();
    await db.runAsync(
      `UPDATE articles SET body = NULL WHERE id NOT IN (
         SELECT id FROM articles ORDER BY added_at DESC LIMIT $limit
       )`,
      { $limit: limit },
    );
  },

  async updateUserState(
    id: string,
    updates: Partial<Pick<Article, 'isRead' | 'isFavorite' | 'scrollFraction' | 'readAt'>>,
  ): Promise<void> {
    const sets: string[] = [];
    const params: Record<string, string | number | null> = { $id: id };

    if (updates.isRead !== undefined) {
      sets.push('is_read = $is_read');
      params.$is_read = updates.isRead ? 1 : 0;
    }
    if (updates.isFavorite !== undefined) {
      sets.push('is_favorite = $is_favorite');
      params.$is_favorite = updates.isFavorite ? 1 : 0;
    }
    if (updates.scrollFraction !== undefined) {
      sets.push('scroll_fraction = $scroll_fraction');
      params.$scroll_fraction = updates.scrollFraction;
    }
    if (updates.readAt !== undefined) {
      sets.push('read_at = $read_at');
      params.$read_at = updates.readAt;
    }
    if (sets.length === 0) return;

    const db = await getDb();
    await db.runAsync(`UPDATE articles SET ${sets.join(', ')} WHERE id = $id`, params);
  },

  /** Remove a single article, e.g. on archive. */
  async remove(id: string): Promise<void> {
    const db = await getDb();
    await db.runAsync('DELETE FROM articles WHERE id = $id', { $id: id });
  },

  /**
   * Drop all stored articles, e.g. on logout. Also clears the outbox
   * (task_ebf1) — a queued write must never replay against a different
   * account.
   */
  async clear(): Promise<void> {
    const db = await getDb();
    await db.execAsync('DELETE FROM articles');
    await db.execAsync('DELETE FROM outbox');
  },
};
