/* eslint-env jest */
// Adapts expo-sqlite's async API onto node:sqlite's DatabaseSync(':memory:')
// so articleStore's real schema and real queries run in tests, instead of a
// hand-rolled fake that can't disagree with the SQL.
const { DatabaseSync } = require('node:sqlite');

const databases = new Map();

function makeDb(nativeDb) {
  return {
    execAsync: async (sql) => {
      nativeDb.exec(sql);
    },
    runAsync: async (sql, params = {}) => {
      const result = nativeDb.prepare(sql).run(params);
      return { changes: result.changes, lastInsertRowId: result.lastInsertRowid };
    },
    getAllAsync: async (sql, params = {}) => nativeDb.prepare(sql).all(params),
    getFirstAsync: async (sql, params = {}) => nativeDb.prepare(sql).get(params) ?? null,
    closeAsync: async () => nativeDb.close(),
  };
}

module.exports = {
  openDatabaseAsync: jest.fn(async (name) => {
    if (!databases.has(name)) {
      databases.set(name, makeDb(new DatabaseSync(':memory:')));
    }
    return databases.get(name);
  }),
};
