-- +goose Up
-- +goose StatementBegin

-- อ่านจบต่อบท — the writer KPI the design asks for.
--
-- Nothing in the schema recorded whether a read reached the end: read events
-- were a bare "someone opened this chapter". Reading progress does track a
-- paragraph anchor, but it is per novel and mutable, so it cannot say how many
-- of last Tuesday's reads finished. The completion flag has to live on the
-- event, beside the read it describes.
ALTER TABLE chapter_read_events
  ADD COLUMN completed BOOLEAN NOT NULL DEFAULT false;

-- Rolled up beside `reads` so the stats page keeps reading one pre-aggregated
-- table rather than scanning the event stream, which is what keeps it fast.
-- Both tables get the column because the rollup flows chapter → novel, and the
-- KPI tiles read the novel table.
ALTER TABLE chapter_daily_stats
  ADD COLUMN completions INT NOT NULL DEFAULT 0;

ALTER TABLE novel_daily_stats
  ADD COLUMN completions INT NOT NULL DEFAULT 0;

-- Existing rows keep completed = false rather than being guessed at. A
-- back-filled guess would put an invented number in front of translators, and
-- the rate self-corrects within a day of real traffic.

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE novel_daily_stats   DROP COLUMN IF EXISTS completions;
ALTER TABLE chapter_daily_stats DROP COLUMN IF EXISTS completions;
ALTER TABLE chapter_read_events DROP COLUMN IF EXISTS completed;

-- +goose StatementEnd
