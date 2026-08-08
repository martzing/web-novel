-- +goose Up
-- +goose StatementBegin

-- Republishing a chapter must not notify its followers a second time.
-- Unpublishing and republishing is a normal editorial action, so the fan-out
-- relies on this index plus ON CONFLICT DO NOTHING rather than on the caller
-- remembering not to repeat itself.
CREATE UNIQUE INDEX notifications_new_chapter_dedupe
    ON notifications (user_id, ((payload->>'chapter_id')))
 WHERE kind = 'new_chapter';

-- Replies are deduplicated the same way, per comment.
CREATE UNIQUE INDEX notifications_reply_dedupe
    ON notifications (user_id, ((payload->>'comment_id')))
 WHERE kind = 'reply';

-- Comment threads are loaded by parent, one query per page of top-level rows.
CREATE INDEX comments_parent_idx ON comments (parent_id) WHERE parent_id IS NOT NULL;

-- Read events are aggregated per chapter and day by the rollup job.
CREATE INDEX chapter_read_events_chapter_time ON chapter_read_events (chapter_id, occurred_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS chapter_read_events_chapter_time;
DROP INDEX IF EXISTS comments_parent_idx;
DROP INDEX IF EXISTS notifications_reply_dedupe;
DROP INDEX IF EXISTS notifications_new_chapter_dedupe;
-- +goose StatementEnd
