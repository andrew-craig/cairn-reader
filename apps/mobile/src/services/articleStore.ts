import * as SQLite from 'expo-sqlite';
import { Article } from '../types';

const DB_NAME = 'cairnreader.db';

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
}

// A list page from the server upserts its rows; existing rows are updated,
// never bulk-deleted. `body` is the one exception: list responses never carry
// cleaned HTML, so a plain overwrite would erase a previously cached body on
// every refresh. COALESCE keeps whatever body is already stored when the
// incoming row doesn't have one.
const UPSERT_SQL = `
  INSERT INTO articles (
    id, url, title, description, image_url, author, published_date,
    reading_time, tags, is_read, is_favorite, added_at, read_at,
    scroll_position, scroll_fraction, body
  ) VALUES (
    $id, $url, $title, $description, $image_url, $author, $published_date,
    $reading_time, $tags, $is_read, $is_favorite, $added_at, $read_at,
    $scroll_position, $scroll_fraction, $body
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
    is_read = excluded.is_read,
    is_favorite = excluded.is_favorite,
    added_at = excluded.added_at,
    read_at = excluded.read_at,
    scroll_position = excluded.scroll_position,
    scroll_fraction = excluded.scroll_fraction,
    body = COALESCE(excluded.body, articles.body)
`;

let dbPromise: Promise<SQLite.SQLiteDatabase> | null = null;

async function getDb(): Promise<SQLite.SQLiteDatabase> {
  if (!dbPromise) {
    dbPromise = SQLite.openDatabaseAsync(DB_NAME)
      .then(async (db) => {
        await db.execAsync(`
          CREATE TABLE IF NOT EXISTS articles (
            id TEXT PRIMARY KEY,
            url TEXT NOT NULL,
            title TEXT NOT NULL,
            description TEXT,
            image_url TEXT,
            author TEXT,
            published_date TEXT,
            reading_time INTEGER,
            tags TEXT NOT NULL DEFAULT '[]',
            is_read INTEGER NOT NULL DEFAULT 0,
            is_favorite INTEGER NOT NULL DEFAULT 0,
            added_at INTEGER NOT NULL,
            read_at INTEGER,
            scroll_position REAL,
            scroll_fraction REAL,
            body TEXT
          );
        `);
        return db;
      })
      .catch((error) => {
        // Don't leave a rejected promise cached forever — a transient failure
        // (e.g. the OS briefly denying disk access) would otherwise poison
        // every future call. Reset so the next getDb() retries the open.
        dbPromise = null;
        throw error;
      });
  }
  return dbPromise;
}

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

  /** Cache a freshly fetched article body (cleaned HTML) opportunistically. */
  async saveBody(id: string, body: string): Promise<void> {
    const db = await getDb();
    await db.runAsync('UPDATE articles SET body = $body WHERE id = $id', {
      $id: id,
      $body: body,
    });
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

  /** Drop all stored articles, e.g. on logout. */
  async clear(): Promise<void> {
    const db = await getDb();
    await db.execAsync('DELETE FROM articles');
  },
};
