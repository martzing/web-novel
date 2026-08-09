-- +goose Up
-- +goose StatementBegin

-- Seed corrections. Nothing here changes the schema; it fixes demo data that
-- contradicted the design and, worse, hid the feature it was supposed to show.

-- 0002 seeded chapters_count = 214 for the featured novel, and 0007 then set
-- source_chapters_count = 214 as well. Two identical figures make the whole
-- "บทที่แปลแล้ว vs บทในต้นฉบับ" split invisible on every surface that renders
-- it, and they leave the table of contents with no dimmed ยังไม่แปล rows,
-- because nothing sits beyond the translated count. The design says 88 of 214.
UPDATE novels
   SET chapters_count = 88
 WHERE slug = 'nine-streams-sword-immortal';

-- The second seeded novel (76 of 143) already reads correctly and is left alone.

-- Coin packs: align the ladder with the design's price points. Keyed on the old
-- coin amount rather than on id, so this is safe whatever identity values the
-- inserts in 0002 happened to produce.
UPDATE coin_packs SET coins =  100, bonus_coins =   0, price_satang =  3500 WHERE coins =   60;
UPDATE coin_packs SET coins =  300, bonus_coins =  15, price_satang =  9900 WHERE coins =  240;
UPDATE coin_packs SET coins =  700, bonus_coins =  70, price_satang = 21900 WHERE coins =  600;
UPDATE coin_packs SET coins = 1500, bonus_coins = 200, price_satang = 44900 WHERE coins = 1500;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE novels
   SET chapters_count = 214
 WHERE slug = 'nine-streams-sword-immortal';

-- Reverse order: 1500 keeps its coin count across the change, so restoring it
-- first would make the 700 → 600 statement below match it a second time.
UPDATE coin_packs SET coins = 1500, bonus_coins = 300, price_satang = 79900 WHERE coins = 1500 AND bonus_coins = 200;
UPDATE coin_packs SET coins =  600, bonus_coins = 100, price_satang = 34900 WHERE coins =  700;
UPDATE coin_packs SET coins =  240, bonus_coins =  20, price_satang = 14900 WHERE coins =  300;
UPDATE coin_packs SET coins =   60, bonus_coins =   0, price_satang =  3900 WHERE coins =  100;

-- +goose StatementEnd
