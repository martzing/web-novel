-- +goose Up
-- +goose StatementBegin

-- Seed genres shown on the browse chips.
INSERT INTO genres (slug, name_th) VALUES
  ('xianxia',      'เซียน'),
  ('wuxia',        'กำลังภายใน'),
  ('rebirth',      'ย้อนเวลา'),
  ('system',       'ระบบ'),
  ('cultivation',  'บำเพ็ญ'),
  ('romance',      'โรแมนซ์'),
  ('adventure',    'ผจญภัย'),
  ('mystery',      'สืบสวน');

-- Seed a translator account so we can attribute novels/chapters.
INSERT INTO users (username, email, password_hash, display_name, roles)
VALUES ('mokchan', 'mokchan@example.com',
        '$2a$10$placeholderplaceholderplaceholderplaceholderplaceh',
        'สำนักแปลหมอกจันทร์',
        ARRAY['reader','translator']);

INSERT INTO writer_profiles (user_id, pen_name, sect_name)
SELECT id, 'สำนักแปลหมอกจันทร์', 'สำนักแปลหมอกจันทร์'
  FROM users WHERE username = 'mokchan';

-- Seed the featured novel from the design mock.
INSERT INTO novels (slug, title_th, title_cn, author_name, description,
                    status, primary_translator_id,
                    rating_avg, rating_count, followers_count, chapters_count)
SELECT 'nine-streams-sword-immortal',
       'เซียนดาบเก้าสายธาร', '九流剑仙', 'เยี่ยจื่อหลิง',
       'เยี่ยหลิงเฟิงเกิดมาพร้อมเส้นชี่ที่แคบกว่าคนทั่วไปเกือบครึ่ง สามปีในสำนักเมฆาวสันต์เขาได้แต่แบกฟืนขึ้นเขา จนคืนหนึ่งเขาพบคัมภีร์ดาบที่ถูกเผาไปครึ่งเล่มในลำธารใต้หน้าผา',
       'ongoing', u.id,
       4.80, 2140, 38400, 214
  FROM users u WHERE u.username = 'mokchan';

INSERT INTO novel_genres (novel_id, genre_id)
SELECT n.id, g.id
  FROM novels n, genres g
 WHERE n.slug = 'nine-streams-sword-immortal'
   AND g.slug IN ('xianxia', 'wuxia', 'cultivation');

-- A second novel for the "แปลใหม่ประจำสัปดาห์" and grid.
INSERT INTO novels (slug, title_th, title_cn, author_name, description,
                    status, primary_translator_id,
                    rating_avg, rating_count, followers_count, chapters_count)
SELECT 'return-to-nineteen',
       'คืนกลับสู่ปีที่สิบเก้า', '重回十九岁', 'เยว่หลิงชิว',
       'เมื่อผู้บำเพ็ญขั้นแปรวิญญาณตายลงกลางฟ้าลงทัณฑ์ เขาลืมตาขึ้นอีกครั้งในร่างของตัวเองตอนอายุสิบเก้า ในคืนก่อนที่สำนักจะถูกเผา',
       'ongoing', u.id,
       4.60, 812, 12030, 76
  FROM users u WHERE u.username = 'mokchan';

INSERT INTO novel_genres (novel_id, genre_id)
SELECT n.id, g.id
  FROM novels n, genres g
 WHERE n.slug = 'return-to-nineteen'
   AND g.slug IN ('xianxia', 'rebirth');

-- Arcs for the featured novel, matching ARC_DEFS in the mock.
INSERT INTO arcs (novel_id, arc_no, name, from_chapter_no, to_chapter_no)
SELECT n.id, v.arc_no, v.name, v.from_ch, v.to_ch
  FROM novels n,
       (VALUES
         (1::smallint, 'ธุลีเมืองชายแดน',      1,   48),
         (2::smallint, 'สำนักเมฆาวสันต์',       49,  120),
         (3::smallint, 'ทะเลทรายกระดูกขาว',    121, 186),
         (4::smallint, 'นครเซียนเหนือเมฆ',     187, 214)
       ) AS v(arc_no, name, from_ch, to_ch)
 WHERE n.slug = 'nine-streams-sword-immortal';

-- Chapter 87 (the reader mock chapter) as published + a couple of neighbours.
WITH nv AS (SELECT id, primary_translator_id FROM novels WHERE slug = 'nine-streams-sword-immortal')
INSERT INTO chapters (novel_id, arc_id, chapter_no, title, status,
                      price_coins, translator_id, published_at, word_count)
SELECT nv.id, a.id, c.chapter_no, c.title, 'published',
       c.price, nv.primary_translator_id, now() - c.age, c.words
  FROM nv
  JOIN arcs a ON a.novel_id = nv.id AND a.arc_no = 2,
       (VALUES
         (86, 'เสียงระฆังจากยอดเขา'::text, 5::smallint, INTERVAL '3 days',  2900),
         (87, 'ดาบแรกใต้ฟ้าหมอก'::text,   5::smallint, INTERVAL '2 days',  3180),
         (88, 'ลมหวนคืนสู่ตะวันตก'::text, 5::smallint, INTERVAL '1 day',   3050)
       ) AS c(chapter_no, title, price, age, words);

INSERT INTO chapter_bodies (chapter_id, body_html, body_source, revision, glossary_rev)
SELECT c.id,
       '<p>หิมะบางโปรยลงเหนือหุบเขาเมฆาวสันต์มาตั้งแต่ยามดึก พอฟ้าสาง ลานฝึกก็ขาวจนแทบมองไม่เห็นรอยหินใต้ฝ่าเท้า</p>' ||
       '<p><span data-k="ye">เยี่ยหลิงเฟิง</span>ยืนอยู่กลางลาน มือขวากุมด้ามกระบี่ไม้ที่สึกจนมันวาว เขาสูดลมเย็นเข้าเต็มปอด แล้วปล่อยให้<span data-k="qi">ชี่</span>ไหลออกจาก<span data-k="dantian">ตันเถียน</span> ขึ้นไปตาม<span data-k="jingmai">จิงม่าย</span>ทั้งสิบสองสาย</p>',
       'หิมะบางโปรยลงเหนือหุบเขาเมฆาวสันต์...',
       1, 0
  FROM chapters c
  JOIN novels n ON n.id = c.novel_id
 WHERE n.slug = 'nine-streams-sword-immortal';

-- Glossary groups + entries from the reader mock (NOTES + GLOSSARY).
WITH nv AS (SELECT id FROM novels WHERE slug = 'nine-streams-sword-immortal')
INSERT INTO glossary_groups (novel_id, name, sort_no)
SELECT nv.id, g.name, g.sort_no FROM nv,
       (VALUES
         ('ลำดับขั้นการบำเพ็ญ'::text, 1::smallint),
         ('ศัพท์การบำเพ็ญ',           2),
         ('สำนักและตระกูล',           3),
         ('ตัวละคร',                  4)
       ) AS g(name, sort_no);

WITH gg AS (
  SELECT g.id, g.name FROM glossary_groups g
    JOIN novels n ON n.id = g.novel_id
   WHERE n.slug = 'nine-streams-sword-immortal'
)
INSERT INTO glossary_entries (group_id, term_key, title_th, title_cn, body, kind)
SELECT gg.id, e.term_key, e.title_th, e.title_cn, e.body, e.kind
  FROM gg,
       (VALUES
         ('ศัพท์การบำเพ็ญ', 'qi',        'ชี่',            '气',
          'พลังชีวิตที่ไหลเวียนอยู่ในร่างและในฟ้าดิน คุณภาพของชี่กำหนดเพดานของผู้บำเพ็ญมากกว่าปริมาณ',
          'ศัพท์'),
         ('ศัพท์การบำเพ็ญ', 'dantian',   'ตันเถียน',      '丹田',
          'แหล่งกักเก็บชี่ใต้สะดือราวสามนิ้ว หากตันเถียนแตกจะสูญเสียพลังทั้งหมด',
          'ศัพท์'),
         ('ศัพท์การบำเพ็ญ', 'jingmai',   'จิงม่าย',       '经脉',
          'เส้นทางเดินของชี่ทั่วร่าง สิบสองสายหลักและแปดสายพิเศษ',
          'ศัพท์'),
         ('ลำดับขั้นการบำเพ็ญ', 'foundation','ขั้นหลอมรากฐาน','筑基',
          'ขั้นที่สองของการบำเพ็ญ หลอมชี่ให้กลายเป็นรากฐานถาวรในร่าง',
          'ลำดับขั้น'),
         ('ลำดับขั้นการบำเพ็ญ', 'core',      'ขั้นก่อแกนธาตุ','结丹',
          'ขั้นที่สาม ชี่ถูกบีบอัดจนเกิดแกนธาตุกลางตันเถียน',
          'ลำดับขั้น'),
         ('สำนักและตระกูล', 'sect',      'สำนักเมฆาวสันต์','云春宗',
          'สำนักระดับกลางในเขตเขาตะวันตก ขึ้นชื่อเรื่องวิชาดาบและกฎที่เข้มงวด',
          'สำนัก'),
         ('ตัวละคร',         'ye',        'เยี่ยหลิงเฟิง','叶凌风',
          'ศิษย์นอกรุ่นสิบเก้า อายุสิบเจ็ด เส้นชี่แคบกว่าคนทั่วไปเกือบครึ่ง',
          'ตัวละคร')
       ) AS e(group_name, term_key, title_th, title_cn, body, kind)
 WHERE gg.name = e.group_name;

-- Coin packs from the "เหรียญ" mock.
INSERT INTO coin_packs (coins, bonus_coins, price_satang, is_best_value, sort_no) VALUES
  ( 60,   0,   3900, false, 1),
  (240,  20,  14900, false, 2),
  (600, 100,  34900, true,  3),
  (1500,300,  79900, false, 4);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM coin_packs;
DELETE FROM chapter_glossary_refs;
DELETE FROM glossary_entries;
DELETE FROM glossary_groups;
DELETE FROM chapter_bodies;
DELETE FROM chapters;
DELETE FROM arcs;
DELETE FROM novel_genres;
DELETE FROM novels;
DELETE FROM writer_profiles;
DELETE FROM users;
DELETE FROM genres;
-- +goose StatementEnd
