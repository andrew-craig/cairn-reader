DROP TABLE source_routes;

ALTER TABLE contents DROP COLUMN source_sender_id;

DROP INDEX idx_user_contents_user_list_added;

ALTER TABLE user_contents DROP COLUMN list;
