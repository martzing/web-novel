-- +goose Up
-- +goose StatementBegin

-- Keep novels.search_tsv populated. The column and its GIN index exist from
-- 0001 but nothing ever wrote to them.
--
-- Note this is only one input to search ranking. Thai has no word boundaries,
-- so `เซียนดาบ` is a substring of the single lexeme `เซียนดาบเก้าสายธาร` and
-- tsvector matching alone misses it; the repository blends this with trigram
-- similarity and ILIKE. Do not "simplify" search down to the tsvector.
CREATE OR REPLACE FUNCTION novels_search_tsv_refresh() RETURNS TRIGGER AS $$
BEGIN
  NEW.search_tsv :=
      setweight(to_tsvector('simple', coalesce(NEW.title_th, '')), 'A')
   || setweight(to_tsvector('simple', coalesce(NEW.title_cn, '')), 'A')
   || setweight(to_tsvector('simple', coalesce(NEW.author_name, '')), 'B')
   || setweight(to_tsvector('simple', coalesce(NEW.description, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER novels_search_tsv_trg
BEFORE INSERT OR UPDATE OF title_th, title_cn, author_name, description ON novels
FOR EACH ROW EXECUTE FUNCTION novels_search_tsv_refresh();

-- Backfill existing rows through the trigger.
UPDATE novels SET title_th = title_th;

-- Trigram similarity is used in the search ranking expression.
CREATE INDEX IF NOT EXISTS novels_title_cn_trgm ON novels USING GIN (title_cn gin_trgm_ops);
CREATE INDEX IF NOT EXISTS glossary_entries_title_trgm ON glossary_entries USING GIN (title_th gin_trgm_ops);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS glossary_entries_title_trgm;
DROP INDEX IF EXISTS novels_title_cn_trgm;
DROP TRIGGER IF EXISTS novels_search_tsv_trg ON novels;
DROP FUNCTION IF EXISTS novels_search_tsv_refresh();
-- +goose StatementEnd
