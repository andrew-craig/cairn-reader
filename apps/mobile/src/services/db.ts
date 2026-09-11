import * as SQLite from 'expo-sqlite';

const DB_NAME = 'cairnreader.db';

let dbPromise: Promise<SQLite.SQLiteDatabase> | null = null;

/**
 * Opens (and migrates) the single on-device database shared by ArticleStore
 * and Outbox — one file, one migration ladder, so the two tables can
 * reference each other (the outbox guard in articleStore.ts's UPSERT_SQL)
 * without either module reaching into the other's internals.
 */
export async function getDb(): Promise<SQLite.SQLiteDatabase> {
  if (!dbPromise) {
    dbPromise = SQLite.openDatabaseAsync(DB_NAME)
      .then(async (db) => {
        // Kept at its original (v1) shape. Installs from before task_a8a4
        // already have this table; widening this statement would be a no-op
        // against them, so schema changes since v1 are applied by migrate()
        // instead, tracked via PRAGMA user_version.
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
        await migrate(db);
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

/**
 * Applies schema migrations in order, tracked via PRAGMA user_version, so a
 * fresh install (CREATE TABLE above, then every step here) and an upgraded
 * install (already at some version, then only the remaining steps) converge
 * on the same schema. A DB already at the latest version runs no statements —
 * safe to call on every open.
 */
async function migrate(db: SQLite.SQLiteDatabase): Promise<void> {
  const row = await db.getFirstAsync<{ user_version: number }>('PRAGMA user_version');
  const startVersion = row?.user_version ?? 0;
  let version = startVersion;

  // Step 2 (task_c55c): content_hash, for diffing cached bodies against the
  // server's hash so unchanged bodies are never re-downloaded.
  if (version < 2) {
    await db.execAsync('ALTER TABLE articles ADD COLUMN content_hash TEXT');
    version = 2;
  }

  // Step 3 (task_ebf1): the offline mutation outbox. Keyed on (article_id,
  // field) so a repeated write to the same field coalesces into one row —
  // see Outbox.enqueue. Left with an implicit rowid (no WITHOUT ROWID) so
  // Outbox.drain can break created_at ties deterministically.
  if (version < 3) {
    await db.execAsync(`
      CREATE TABLE IF NOT EXISTS outbox (
        article_id TEXT NOT NULL,
        field TEXT NOT NULL,
        payload TEXT NOT NULL,
        created_at INTEGER NOT NULL,
        attempts INTEGER NOT NULL DEFAULT 0,
        PRIMARY KEY (article_id, field)
      );
    `);
    version = 3;
  }

  if (version !== startVersion) {
    await db.execAsync(`PRAGMA user_version = ${version}`);
  }
}
