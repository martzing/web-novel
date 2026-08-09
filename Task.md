# Task — แผนงานปรับ Design ↔ Document ↔ Codebase ให้สอดคล้อง

อ้างอิงผลตรวจสอบใน [Check-diff.md](Check-diff.md) · Branch `main` @ `5b3bf4a` · จัดทำ 2026-08-09

> ## ✅ ดำเนินการครบทั้ง 24 tasks แล้ว (2026-08-10)
>
> Branch: `chore/align-design-docs-code`
>
> | ตัวชี้วัด | ผล |
> | --- | --- |
> | Backend tests | 587 PASS / 0 SKIP / 0 FAIL (เดิม 575) |
> | `gofmt -l .` · `go vet ./...` | ไม่มี output |
> | `npm run typecheck` · `npm run build` | ผ่าน |
> | Redis | ถอดออกจาก compose/config/เอกสารครบ |
> | Migrations | เพิ่ม `0008_seed_fixes.sql`, `0009_read_completion.sql` |
> | Routes | 89 → 94 |
>
> ไฟล์ใหม่: `backend/migrations/0008_seed_fixes.sql`,
> `backend/migrations/0009_read_completion.sql`,
> `backend/internal/handler/library/seriesfollow_integration_test.go`,
> `backend/internal/handler/writer/glossary_integration_test.go`,
> `frontend/src/layout/NotificationBell.tsx`
>
> รายละเอียดผลลัพธ์อยู่ใน [Check-diff.md §8](Check-diff.md#8-สถานะหลังแก้ไข-2026-08-10)

---

## 0. สรุปการตัดสินใจ (Decision Log)

ตารางนี้คือแหล่งอ้างอิงเดียวว่า "แต่ละความต่าง จะยึดอะไรเป็นหลัก" ทุก task ด้านล่างสืบมาจากตารางนี้

### 0.1 ตัดสินใจแล้ว — ต้องลงมือทำ

| Check-diff ID | ประเด็น | ทิศทางที่เลือก | ยึดอะไรเป็นหลัก |
| --- | --- | --- | --- |
| H-01 | Redis ประกาศแต่ไม่ใช้ | ถอด Redis ออกให้หมด (เอกสาร + compose + config) | **Codebase** |
| H-02 | seed `chapters_count` = 214 | ตั้งเป็น 88 / 214 | **Design** |
| H-03 | `รอบปล่อยบทใหม่` ไม่แสดง (W-22) | เพิ่ม UI บนหน้า detail | **Document** |
| H-04 | KPI เงินบาท (THB) | ตัด THB ออกจาก PRD | **Codebase** |
| H-05 | README `0001 … 0006` | แก้เป็น `0001 … 0007` | **Codebase** |
| M-01 | ตาราง migration ตก 0007 | เติมแถว | **Codebase** |
| M-02 | schema doc ขัดกันเอง (Redis ZSET) | ลบประโยค Redis | **Codebase** |
| M-03 | Glossary CRUD ไม่ครบ (W-05) | ทำ DELETE endpoint + UI เต็ม | **Document** |
| M-04 | Payout ไม่มี UI (W-10) | ทำ UI รายได้ / ถอนเงิน | **Document** |
| M-05 | Notification ไม่มี UI (R-17) | ทำกระดิ่งแจ้งเตือนใน Shell | **Document** |
| M-06 | หน้า series ขาด 3 ส่วน | ทำครบทั้ง 3 ส่วน | **Design** |
| M-07 | `ติดตามทั้งชุด` | ทำแบบ fan-out (ไม่แก้ schema) | **Design** |
| M-09 | KPI `อ่านจบต่อบท` | ทำ completion rate | **Design** |
| M-10 | บรรทัดสรุปราคาบนหน้า detail | เพิ่มทั้ง API + UI | **Design** |
| M-11 | README "Every novel surface…" | แก้ถ้อยคำ README | **Codebase + PRD** |
| M-12 | README ลิสต์ jobs 5/9 | เติมให้ครบ 9 | **Codebase** |
| L-01 | api-spec ตก 4 routes | เติมให้ครบ | **Codebase** |
| L-02 | `/search?type=` ไม่ถูกอ่าน | ตัด `type` ออกจากเอกสาร | **Codebase** |
| L-03 | ตัวอย่าง composition root ล้าสมัย | อัปเดตตัวอย่าง | **Codebase** |
| L-04 | ตาราง dependency rule ไม่มี `middleware` | เติม | **Codebase** |
| L-05 | "PostgreSQL 15+" | แก้เป็น 16 | **Codebase** |
| L-06 | "the featured novel" (เอกพจน์) | แก้เป็น 2 เรื่อง | **Codebase** |
| L-07 | test-cases "Postgres/Redis" | ตัด Redis | **Codebase** |
| L-08 | user-stories ไม่มีคอลัมน์สถานะ | เพิ่มคอลัมน์ Status ทุกตาราง | — |
| L-09 | PRD ไม่พูดถึง `ความยาวที่ชอบ` | เขียนเพิ่มใน PRD | **Design + Codebase** |
| L-10, L-11 | label แม่แบบปก + ป้ายขั้นตอน wizard | แก้ให้ตรง Design | **Design** |
| L-13, D-16 | Onboarding | คง 2 step แต่บังคับ ≥ 3 แนว | **ผสม** |
| L-15 | popover `ดูในอภิธานศัพท์ →` | ทำ | **Design** |
| L-18 | 2 บรรทัดบนหน้าเหรียญ | ทำทั้งสองบรรทัด | **Design** |
| L-19 | ราคาแพ็กเหรียญใน seed | แก้ seed ให้ตรง Design | **Design** |
| L-20, D-02, D-04, D-05 | KPI หน้า detail 5 ช่อง + ทำลิงก์ | ทำตาม Design | **Design** |

### 0.2 ตัดสินใจแล้ว — **ไม่ทำ** (บันทึกเป็น backlog ในเอกสาร)

| Check-diff ID | ประเด็น | เหตุผล |
| --- | --- | --- |
| M-08 / D-01 | `อ่านสะสมสัปดาห์นี้ N ชั่วโมง` | ต้องเก็บ duration ต่อ session = schema + job ใหม่ทั้งชุด ไม่คุ้มกับข้อความเดียวบนหน้าแรก |
| L-22 / D-27 | ปุ่ม `ดูรายได้ที่คาดการณ์` | ต้องอาศัยสมมติฐานหลายชั้นที่คลาดเคลื่อนง่าย และหน้ารายได้จริงจะมีอยู่แล้ว (T-18) |
| L-17 / D-09…D-12 | รายละเอียด checkout (accordion, QR countdown, บันทึกบัตร, โค้ดส่วนลด, VAT) | รอ Phase 7 — จะทำพร้อม payment provider จริง |
| L-16 / D-23 | pinch-to-zoom | ชนกับ browser zoom และมีปุ่ม A− / A+ อยู่แล้ว |
| L-12 / D-25 | ตัวเลือก `รอบปล่อยบทใหม่` | คงของโค้ดที่มีตัวเลือกมากกว่า Design |
| L-14 / D-17 | อักษรจีนในชิปแนว onboarding | เลี่ยงการแก้ genres API เพื่อ cosmetic |
| D-08 | รูปแบบ tabs ของ checkout | ส่วนหนึ่งของ Phase 7 |

---

## 1. กลุ่ม A — เอกสารล้วน (ไม่แตะโค้ด, ความเสี่ยงต่ำ, ทำได้ทันที)

### T-01 · ถอด Redis ออกจากทั้งโปรเจกต์ 🔴

**ยึด:** Codebase · **สืบจาก:** H-01, M-02, L-07

- [x] `README.md:3` — ลบ `+ Redis 7` ออกจากบรรทัด stack
- [x] `AGENT.md` §Project Snapshot — ลบ Redis ออกจากรายการ
- [x] `AGENT.md` §Project Snapshot → "Runtime" — แก้เป็น `docker-compose.yml starts Postgres, API, and web services`
- [x] `docs/architecture.md` — ตรวจและลบการอ้างถึง Redis (ถ้ามี)
- [x] `docs/database-schema.md` §`ranking_snapshots` — ลบประโยค *"Live ranking is computed in Redis (`ZSET`) and snapshotted here for history"* แล้วเขียนใหม่ให้ตรงกับ `jobs.RankingJob` ที่เขียน `ranking_snapshots` ตรงๆ ทุกวันจันทร์ 04:00 และ read path fallback ไป live popularity เมื่อยังไม่มี snapshot
- [x] `docs/test-cases.md:6` — `**I** — integration, real Postgres` (ตัด `/Redis`)
- [x] `docker-compose.yml` — ลบ service `redis`, ลบ `REDIS_URL` จาก service `api`, ลบ `depends_on: redis`
- [x] `backend/internal/config/config.go` — ลบฟิลด์ `RedisURL` และบรรทัดที่อ่าน `REDIS_URL`
- [x] `backend/.env.example` — ลบ `REDIS_URL`
- [x] `backend/internal/ratelimit/limiter.go:3` — ปรับ comment ให้ยังอ่านรู้เรื่องหลังถอด Redis

**Acceptance:** `grep -ri redis` ทั้ง repo เหลือเฉพาะบันทึกเชิงประวัติ/เหตุผล (เช่น comment ใน `repository/identity/sessions.go` ที่อธิบายว่าทำไมใช้ Postgres แทน Redis) · `go build ./...` และ `docker compose config` ผ่าน

---

### T-02 · แก้ข้อเท็จจริงใน `README.md` 🔴

**ยึด:** Codebase · **สืบจาก:** H-05, M-11, M-12

- [x] บรรทัด 112 — `go run ./cmd/migrate -cmd up   # applies 0001 … 0008` (นับ migration ใหม่จาก T-09 ด้วย)
- [x] บรรทัด 124 — เติม job ที่ตกหล่นให้ครบ 9: bonus expiry, glossary re-render, scheduled publishing, **auto-unlock**, stats rollups, weekly ranking, **session sweep**, **wallet reconcile**, **read-event partitions**
- [x] บรรทัด 13–14 — แก้ถ้อยคำ "Every novel surface now shows two chapter counts" ให้ตรงกับ PRD: หน้ารายละเอียดเรื่องและหน้าชุดหนังสือแสดงสองตัวเลข ส่วนชั้นหนังสือแสดงความคืบหน้าเทียบบทที่แปลแล้ว

---

### T-03 · แก้ `docs/database-schema.md` 🟡

**ยึด:** Codebase · **สืบจาก:** M-01, L-05, L-06

- [x] เติมแถว `0007_monetization.sql` ในตาราง "Migration & seed layout" (สรุปสั้น: novels +12 คอลัมน์, `chapters.public_at`, tip kind, series `owner_user_id`/`slug`, 3 ตารางใหม่)
- [x] เติมแถว `0008_seed_fixes.sql` หลังทำ T-09
- [x] แก้หัวเอกสารจาก `PostgreSQL 15+` → `PostgreSQL 16`
- [x] แก้คำอธิบาย 0002 จาก "the featured novel" → ระบุว่า seed มี **2 เรื่อง** (`nine-streams-sword-immortal`, `return-to-nineteen`)
- [x] เติม `novel_relations`, `auto_unlock_subscriptions`, `auto_unlock_attempts` ลงในผัง "Domain overview" (ตอนนี้ผังยังเป็นยุค 0001)

---

### T-04 · แก้ `docs/architecture.md` 🟢

**ยึด:** Codebase · **สืบจาก:** L-03, L-04

- [x] อัปเดตตัวอย่าง composition root ให้ตรงของจริง: `catalogsvc.New(catalogRepo, walletRepo)` พร้อมหมายเหตุว่า wallet repo ทำหน้าที่เป็น `catalog.Entitlements`
- [x] เติม `httpx`, `middleware` ในแถว `handler/...` ของตาราง "Dependency rule" (`wallet` และ `writer` handler import `middleware` เพื่อใช้ `RequireRole`)

---

### T-05 · แก้ `docs/api-spec.md` 🟢

**ยึด:** Codebase · **สืบจาก:** L-01, L-02

- [x] เติม 4 routes ที่ทำแล้วแต่ไม่อยู่ในเอกสาร:
  - `GET /users/me/genre-prefs`
  - `GET /me/follows/{novel_id}`
  - `GET /writer/novels`
  - `GET /writer/novels/{id}/arcs`
- [x] §Discovery utility — ตัด `type=novel|chapter|character` ออก เหลือ `GET /search?q=` และคงหมายเหตุว่าเป็น alias ของ `GET /novels` ที่ใช้ ranking แบบผสมเดียวกัน

---

### T-06 · แก้ `docs/prd.md` 🟡

**ยึด:** ผสม · **สืบจาก:** H-04, L-09, M-08, L-22, L-17

- [x] §Writer workspace — ตัด `THB` ออกจาก "KPI tiles (reads, followers, coins, THB)" เพราะอัตราแปลงจริงขึ้นกับแพ็กที่ผู้อ่านซื้อ ไม่ใช่ค่าคงที่
- [x] §Writer workspace — เพิ่ม KPI `อ่านจบต่อบท` (จาก T-14)
- [x] §Onboarding — เขียนเพิ่มว่ามีขั้น `ความยาวที่ชอบ` (สั้น / กลาง / ยาว) และอธิบายว่าเก็บเป็น `weight` บน `user_genre_prefs` (สั้น=1 กลาง=2 ยาว=3) ไม่มีคอลัมน์แยก
- [x] §Onboarding — ระบุว่าต้องเลือกอย่างน้อย 3 แนว
- [x] เพิ่มหัวข้อ **"Deferred / backlog"** บันทึกสิ่งที่ Design มีแต่ตัดสินใจไม่ทำ: เวลาอ่านสะสมรายสัปดาห์, ปุ่มรายได้คาดการณ์, pinch-to-zoom, ตัวเลือกรอบปล่อยบทตาม Design, อักษรจีนในชิปแนว
- [x] §Rollout phases — เพิ่มหมายเหตุใต้ Phase 7 ว่ารายละเอียด checkout ตาม Design (accordion, QR countdown, บันทึกบัตร, โค้ดส่วนลด, VAT) จะทำพร้อม payment provider จริง

---

### T-07 · เพิ่มคอลัมน์ Status ใน `docs/user-stories.md` 🟢

**สืบจาก:** L-08

- [x] เพิ่มคอลัมน์ `Status` ในตาราง Reader / Writer / Admin ทุกตาราง ใช้ ✅ done · ⚠️ partial · ❌ not implemented
- [x] กำกับตามความจริงหลังจบงานทั้งหมด: `A-01`, `A-02`, `A-04`, `A-05` = ❌ · ที่เหลือ = ✅
- [x] แก้ข้อความ `W-22` ให้สอดคล้องกับ UI ใหม่จาก T-15

---

## 2. กลุ่ม B — Backend (migration + API)

### T-08 · Migration `0008_seed_fixes.sql` 🔴

**ยึด:** Design · **สืบจาก:** H-02, L-19

> ⚠️ ต้องเป็น migration **ใหม่** ห้ามแก้ `0002_seed.sql` ที่ apply ไปแล้ว (กฎใน `AGENT.md` §Editing Guardrails)

- [x] `UPDATE novels SET chapters_count = 88 WHERE slug = 'nine-streams-sword-immortal';`
      คู่กับ `source_chapters_count = 214` ที่ 0007 ตั้งไว้ ให้ตรงตัวเลขใน Design
- [x] อัปเดต `coin_packs` ให้ตรง Design: `100/฿35`, `300 +15/฿99`, `700 +70/฿219 (best value)`, `1500 +200/฿449`
      (เขียนเป็น `UPDATE ... WHERE coins = <ค่าเดิม>` หรือ `DELETE` + `INSERT` ให้ idempotent)
- [x] เขียน `-- +goose Down` ย้อนกลับเป็นค่าเดิม
- [x] ตรวจว่าเรื่องที่สอง (`return-to-nineteen`, 76 / 143) ไม่ต้องแก้ — ตัวเลขสื่อฟีเจอร์อยู่แล้ว

**Acceptance:** `go run ./cmd/migrate -cmd up` แล้ว `GET /novels/nine-streams-sword-immortal` คืน `chapters_count: 88, source_chapters_count: 214` · หน้า ToC แสดงแถว dim `ยังไม่แปล` · `go test ./...` ยังผ่าน

> ✅ **ตรวจแล้ว:** เทส wallet ที่อ้าง "seeded 240-coin pack" จริงๆ สร้าง pack เองผ่าน `makeme.ANewCoinPack()` ([handler_integration_test.go:99-103](backend/internal/handler/wallet/handler_integration_test.go#L99-L103)) การแก้ seed จึงไม่ทำให้เทสพัง — แต่ควรรันยืนยันซ้ำ และแก้ comment ที่เขียนว่า "seeded" ให้ตรงความจริง

---

### T-09 · เพิ่มข้อมูลราคาใน `NovelDetailResponse` 🟡

**ยึด:** Design · **สืบจาก:** M-10, D-03

- [x] `backend/internal/domain/catalog` — เพิ่ม `PricePerChapter`, `FreeUntilChapter` ใน `NovelDetail`
- [x] `backend/internal/repository/catalog/repository.go` — map สองคอลัมน์นี้จาก `entities.Novel`
- [x] `backend/internal/handler/catalog/dto.go` — เพิ่ม `price_per_chapter`, `free_until_chapter` ใน `NovelDetailResponse`
- [x] `docs/api-spec.md` — เติมสองฟิลด์นี้ในหัวข้อ Catalog

**Acceptance:** `GET /novels/{id}` คืนสองฟิลด์ · เพิ่มเทส integration ยืนยันค่า

---

### T-10 · เพิ่ม DELETE ให้ glossary 🟡

**ยึด:** Document (W-05) · **สืบจาก:** M-03

- [x] `DELETE /writer/glossary-entries/:id` — ลบ entry (ตรวจ ownership ของ novel ก่อน)
- [x] `DELETE /writer/glossary-groups/:id` — ลบกลุ่ม (ปฏิเสธถ้ายังมี entry เหลือ หรือ cascade ตามที่ตัดสินใจใน service)
- [x] ยืนยันว่า trigger `glossary_entries_bump` ทำงานตอน DELETE ด้วย (trigger ประกาศ `AFTER INSERT OR UPDATE OR DELETE` — ต้องมีเทสยืนยันว่า `novels.glossary_rev` เด้ง และ re-render job ล้าง `<span data-k>` ที่ชี้ไปคำที่ถูกลบ)
- [x] `docs/api-spec.md` §Writer — เติมสอง endpoint

> ⚠️ **จุดเสี่ยง:** การลบคำที่ยังถูกอ้างใน `chapter_glossary_refs` / `body_html` ต้องกำหนดพฤติกรรมชัดเจน — แนะนำให้ re-render คืนเป็นข้อความธรรมดา (renderer เก็บ marker ที่ไม่รู้จักไว้อยู่แล้ว ตามเทส `glossaryrender`)

**Acceptance:** เทส integration: ลบคำแล้ว `glossary_rev` เด้ง, body ที่อ้างคำนั้นถูก re-render, ผู้แปลคนอื่นลบไม่ได้ (403)

---

### T-11 · ขยาย `GET /series/{id}` ให้รองรับหน้า series ใหม่ 🟡

**ยึด:** Design · **สืบจาก:** M-06, D-18, D-19

- [x] `SeriesBookResponse` — เพิ่ม `arcs[]` (arc_no, name, from/to chapter) ต่อเล่ม เพื่อทำ "รายการภาคย่อย" ตาม Design
- [x] `SeriesResponse` — เพิ่ม `arcs_count` (ผลรวมภาคทั้งชุด) สำหรับ KPI ช่องที่ Design มีแต่โค้ดไม่มี
- [x] `backend/internal/repository/catalog/series.go` — โหลด arcs แบบ bulk query เดียว (อย่า N+1 ต่อเล่ม)
- [x] `docs/api-spec.md` — อัปเดต shape ของ `GET /series/{id}`

**Acceptance:** เทส `TestPublicSeries_...` เดิมยังผ่าน + เทสใหม่ยืนยันว่า arcs มาครบและ query ไม่เพิ่มตามจำนวนเล่ม

---

### T-12 · `ติดตามทั้งชุด` แบบ fan-out 🟡

**ยึด:** Design · **สืบจาก:** M-07, D-21

- [x] `POST /series/:id/follow` และ `DELETE /series/:id/follow` — วนตามเล่มในชุดแล้วเขียน/ลบ `follows` รายเล่ม ในทรานแซกชันเดียว
- [x] `GET /series/:id/follow` — คืนสถานะ (`all` / `partial` / `none`) เพื่อให้ปุ่มแสดงผลถูก
- [x] Register ใน `library` handler; wire ใน `backend/internal/server/server.go`
- [x] `docs/api-spec.md` §Library/Follows — เติม 3 endpoint นี้
- [x] `docs/user-stories.md` — เพิ่ม story ใหม่ (เช่น R-29 "ติดตามทั้งชุดในคลิกเดียว")

> ⚠️ **กฎ gin wildcard (AGENT.md §Rules With Teeth):** ต้องใช้ชื่อ `:id` ให้ตรงกับ `/series/:id` ที่ catalog ลงทะเบียนไว้แล้ว — ใช้ `:series_id` จะทำให้ engine panic ตอน start และ `TestServerNew_RegistersAllRoutesWithoutPanic` จะจับได้ ห้ามวางไว้ใต้ `/me/follows/...` เพราะจะชนกับ `:novel_id`

**Acceptance:** `TestServerNew_RegistersAllRoutesWithoutPanic` ผ่าน · เทส: follow ชุด → ทุกเล่มมีแถว `follows` · เล่มที่เพิ่มเข้าชุดทีหลังไม่ถูกติดตามอัตโนมัติ (ระบุพฤติกรรมนี้ในเอกสาร) · unfollow ชุดลบเฉพาะเล่มในชุด

---

### T-13 · Completion rate (`อ่านจบต่อบท`) 🟡

**ยึด:** Design · **สืบจาก:** M-09, D-14

> โครงสร้างปัจจุบันไม่มีสัญญาณ "อ่านจบ" เลย — `chapter_read_events` เก็บแค่ event, `chapter_daily_stats` มี `reads / unique_readers / coins_earned`

- [x] Migration `0009_read_completion.sql`:
  - `ALTER TABLE chapter_read_events ADD COLUMN completed BOOLEAN NOT NULL DEFAULT false;`
  - `ALTER TABLE chapter_daily_stats ADD COLUMN completions INT NOT NULL DEFAULT 0;`
- [x] `POST /chapters/{id}/read-event` — รับ body `{completed: bool}` (ค่าเริ่มต้น false, ยังคงตอบ `202` เหมือนเดิม)
- [x] `frontend/src/routes/Reader.tsx` — ยิง read-event พร้อม `completed: true` เมื่อผู้อ่านถึงย่อหน้าสุดท้าย (ใช้ตรรกะเดียวกับที่ track `currentParagraph` อยู่แล้ว)
- [x] `jobs.StatsRollupJob` — รวม `completions` ลง `chapter_daily_stats`
- [x] `StatsResponse` — เพิ่ม `completion_rate_pct` (completions ÷ reads)
- [x] `docs/database-schema.md`, `docs/api-spec.md` — อัปเดต

> 📌 **ขอบเขต:** ทำเฉพาะตัวเลขของเรื่องนั้น **ไม่ทำ** "ค่าเฉลี่ยหมวด 71%" ที่ Design แสดงคู่กัน — ต้อง aggregate ข้ามทุกเรื่องในหมวดซึ่งเป็นงานคนละขนาด บันทึกเป็น backlog ใน PRD

**Acceptance:** เทสหน่วยของสูตร (หารศูนย์ = 0%) + เทส integration ยืนยันว่า rollup นับ completions ตรงกับ fixture

---

### T-14 · เทสสำหรับของใหม่ทั้งหมด 🟡

- [x] เขียนเทสครอบ T-09 ถึง T-13 ตามรูปแบบเดิม (domain/service = unit + fake, handler/repository = integration ผ่าน `test/apitest`)
- [x] เพิ่มแถวใหม่ในตาราง mapping ของ `docs/test-cases.md` พร้อม ID ใหม่ (เช่น `I-SER-05` ติดตามทั้งชุด, `I-WR-05` ลบคำอภิธาน, `I-WR-06` completion rate)
- [x] อัปเดตตัวเลขรวมในหัว `docs/test-cases.md` (ตอนนี้ 575) ให้ตรงกับผลรันจริงหลังเพิ่มเทส

---

## 3. กลุ่ม C — Frontend

### T-15 · `routes/Novel.tsx` — หน้ารายละเอียดเรื่อง 🔴

**ยึด:** Design + Document · **สืบจาก:** H-03, M-10, L-20, D-02, D-04, D-05

- [x] คืน KPI เป็น **5 ช่อง** ตาม Design: เรตติ้ง / บทที่แปลแล้ว / บทในต้นฉบับ / **ภาค** / ผู้ติดตาม
- [x] แสดง `รอบปล่อยบทใหม่` (`release_schedule`) — ทำให้ user story **W-22** เป็นจริง
- [x] เพิ่มบรรทัดสรุปราคาเหนือสารบัญ: `บทที่ 1–{free_until_chapter} อ่านฟรี · บทหลังจากนั้น {price_per_chapter} เหรียญต่อบท` (ต้องรอ T-09) — ซ่อนบรรทัดนี้เมื่อ `price_per_chapter = 0`
- [x] ทำ `อภิธานศัพท์ N คำ` เป็นลิงก์ไปพาเนลอภิธานศัพท์ในหน้าอ่าน
- [x] ทำ `ความเห็น N` เป็นลิงก์ไป `/chapters/{id}/comments`
- [x] เปลี่ยน pill `อยู่ในชุดหนังสือ →` เป็นปุ่ม `ดูทั้งชุด · N เรื่อง M ภาค` พร้อมตัวเลข (ใช้ `arcs_count` จาก T-11)

---

### T-16 · `routes/Series.tsx` — หน้าชุดหนังสือ 🟡

**ยึด:** Design · **สืบจาก:** M-06, M-07, D-18, D-19, D-20, D-21, L-21

- [x] เพิ่ม KPI ช่องที่ 4: จำนวน `ภาค` รวมทั้งชุด
- [x] section `เรื่องหลัก` — การ์ดเล่มพร้อม progress bar, ปุ่ม CTA และรายการภาคย่อยของแต่ละเล่ม (ใช้ `arcs[]` จาก T-11)
- [x] section `เรื่องเกี่ยวเนื่อง` — เรียก `GET /novels/{id}/related` ของเล่มแรกในชุด แล้วจัดกลุ่มตาม kind
- [x] ปุ่ม `ติดตามทั้งชุด` — เรียก API จาก T-12 พร้อมสถานะ `all` / `partial` / `none`
- [x] คงคำอธิบายเหตุผลของหน้านี้ (โน้ตต่อเล่มนำหน้า metadata) ตาม comment ที่มีอยู่ใน `Series.tsx:8-14`

---

### T-17 · `layout/Shell.tsx` — กระดิ่งแจ้งเตือน 🟡

**ยึด:** Document (R-17) · **สืบจาก:** M-05

> API และ client function พร้อมหมดแล้ว (`api.listNotifications`, `api.unreadCount`, `api.markNotificationsRead` ที่ [api.ts:818-821](frontend/src/lib/api.ts#L818-L821)) แต่ยังไม่มี component ใดเรียก

- [x] กระดิ่ง + badge จำนวนที่ยังไม่อ่านใน topbar และ sidebar (polling ด้วย TanStack Query `staleTime` เหมือน wallet/shelf)
- [x] Dropdown แสดงรายการ พร้อม deep link ตาม `kind`: `new_chapter` → หน้าอ่าน · `reply` → หน้าความเห็น · `bonus_expiring` → หน้าเหรียญ
- [x] กด "อ่านแล้วทั้งหมด" → `POST /me/notifications/read` ด้วย `ids: []`
- [x] จัดสไตล์ใน `frontend/src/styles/components.css` ตาม design token (ห้าม inline style ตามกฎใน `AGENT.md`)
- [x] Design ไม่ได้วาดหน้านี้ไว้ — ออกแบบให้กลมกลืนกับ sidebar/topbar ที่มีอยู่

---

### T-18 · หน้ารายได้และการถอนเงิน 🟡

**ยึด:** Document (W-10) · **สืบจาก:** M-04

> `GET /writer/earnings` (มี `available_satang`) และ `POST /writer/payouts` พร้อมใช้แล้ว แต่ `frontend/src` ไม่มีที่ไหนเรียกเลย

- [x] เพิ่ม client function `listEarnings` / `requestPayout` ใน `frontend/src/lib/api.ts`
- [x] ทำเป็นแท็บในหน้า `สถิติผลงาน` หรือ route ใหม่ `/earnings` (แนะนำแท็บ เพื่อไม่เพิ่มรายการใน sidebar ที่ Design กำหนดไว้ 3 รายการ)
- [x] แสดง: เหรียญสะสม, ยอดที่ถอนได้ (`available_satang`), ประวัติการถอน, ฟอร์มขอถอน
- [x] Design ไม่ได้วาดหน้านี้ไว้ — ใช้ pattern การ์ด/ตารางเดิมจาก `Stats.tsx`

---

### T-19 · แท็บจัดการอภิธานศัพท์ใน `routes/Works.tsx` 🟡

**ยึด:** Document (W-05) · **สืบจาก:** M-03

- [x] เพิ่ม client function: `createGlossaryGroup`, `createGlossaryEntry`, `updateGlossaryEntry`, `deleteGlossaryEntry`, `deleteGlossaryGroup`
- [x] ทำ UI จัดการกลุ่ม + คำ — วางในแท็บ `ภาคและบท` หรือแท็บที่ 6 ใหม่
- [x] แสดง `term_key` ให้ชัด เพราะเป็นสิ่งที่ผู้แปลพิมพ์เป็น `{{term_key}}` ใน editor
- [x] ยืนยันการลบด้วย dialog พร้อมเตือนว่าบทที่อ้างคำนี้จะถูก re-render

> 📌 Design ไม่ได้วาดหน้านี้ — ให้ยึด layout master/detail ของ `จัดการผลงาน` ที่มีอยู่

---

### T-20 · `routes/Stats.tsx` — KPI อ่านจบต่อบท 🟡

**ยึด:** Design · **สืบจาก:** M-09, D-14

- [x] แทน KPI ช่องที่ 4 `ช่วงเวลา` ด้วย `อ่านจบต่อบท {completion_rate_pct}%`
- [x] ข้อมูลช่วงเวลาไม่หายไป — หน้านี้มี Tabs `14 วันล่าสุด / 30 วัน / ทั้งหมด` อยู่แล้ว ([Stats.tsx:80-90](frontend/src/routes/Stats.tsx#L80-L90))
- [x] คงตัวเลข `เหรียญที่ได้รับ` เป็นเหรียญล้วน **ไม่แสดงเงินบาท** ตามที่ตัดสินใจใน H-04

---

### T-21 · `routes/Coins.tsx` — สองบรรทัดจาก Design 🟢

**ยึด:** Design · **สืบจาก:** L-18, D-06, D-07

- [x] `ปลดล็อกได้อีกราว N บท` — คำนวณฝั่ง client จากยอดคงเหลือ ÷ ราคากลางต่อบท ใช้คำว่า "ราว" ให้ชัดว่าเป็นการประมาณ เพราะราคาต่อบทต่างกันได้ในแต่ละเรื่อง
- [x] `ใช้ไปเดือนนี้ N เหรียญ` — รวมจาก ledger ที่ดึงมาแสดงอยู่แล้ว (นับเฉพาะรายการติดลบในเดือนปัจจุบัน)

---

### T-22 · `routes/Onboarding.tsx` 🟢

**ยึด:** ผสม · **สืบจาก:** L-13, D-16

- [x] คงโครง 2 step (ดีกว่าหน้าเดียวบนมือถือ) แต่เปลี่ยนเงื่อนไขปุ่ม `ถัดไป` เป็นบังคับ **≥ 3 แนว**
- [x] แก้ copy ให้ตรง Design: `เลือกอย่างน้อย 3 แนว เราจะใช้จัดหน้าแรกให้ตรงกับที่คุณอ่านจริง ปรับเปลี่ยนภายหลังได้ตลอด`
- [x] แสดงตัวนับความคืบหน้า (เช่น `เลือกแล้ว 2 / 3`) เพื่อไม่ให้ปุ่มที่ disabled ดูเหมือนพัง
- [x] **ไม่** เพิ่มอักษรจีนในชิปแนว (ตัดสินใจแล้วใน L-14)

---

### T-23 · `routes/Reader.tsx` — ปุ่มใน popover 🟢

**ยึด:** Design · **สืบจาก:** L-15, D-22

- [x] เพิ่มปุ่ม `ดูในอภิธานศัพท์ →` ใน note popover — ปิด popover แล้วเปิดพาเนลอภิธานศัพท์พร้อมเลื่อนไปยังคำนั้น
- [x] **ไม่** ทำ pinch-to-zoom (ตัดสินใจแล้วใน L-16) — ตรวจว่าข้อความแนะนำในแผงตั้งค่าไม่ได้กล่าวถึง gesture ที่ไม่มีจริง

---

### T-24 · แก้ copy ให้ตรง Design 🟢

**ยึด:** Design · **สืบจาก:** L-10, L-11

- [x] `components/index.tsx:72-75` — label แม่แบบปก: `ink` → **หมึกจีนแนวตั้ง** · `seal` → **ตราประทับ** · `brush` → **ลายพู่กัน** · `plain` → **เรียบ ตัวอักษรอย่างเดียว`**
      (ค่า key ไม่เปลี่ยน — ตรงกับ CHECK constraint ใน migration 0007 อยู่แล้ว)
- [x] `routes/Write.tsx:172` — ป้ายขั้นตอน wizard: `เลือกเรื่อง` / `เลือกบทหรือสร้างบทใหม่` / `เขียน`
- [x] **ไม่** แก้ตัวเลือก `รอบปล่อยบทใหม่` (คงของโค้ดที่มี 5 ตัวเลือก — ตัดสินใจแล้วใน L-12)

---

## 4. ลำดับการทำงานที่แนะนำ

```
รอบที่ 1 (เอกสารล้วน — merge ได้ทันที ไม่กระทบโค้ด)
  T-01 Redis · T-02 README · T-03 schema · T-04 architecture · T-05 api-spec

รอบที่ 2 (Backend — ปลดล็อกงาน frontend)
  T-08 migration seed  ──┐
  T-09 ราคาใน detail    ──┼──→ T-15
  T-11 series arcs      ──┼──→ T-16
  T-12 follow series    ──┘
  T-10 glossary DELETE  ─────→ T-19
  T-13 completion rate  ─────→ T-20

รอบที่ 3 (Frontend)
  T-15 Novel · T-16 Series · T-17 Notification · T-18 Payout
  T-19 Glossary · T-20 Stats · T-21 Coins · T-22 Onboarding · T-23 Reader · T-24 Copy

รอบที่ 4 (ปิดงาน)
  T-14 เทส · T-06 PRD · T-07 user-stories status · อัปเดต Check-diff.md
```

**Critical path:** T-08 → T-09/T-11/T-12 → T-15/T-16
**ทำคู่ขนานได้ทันที:** T-01…T-05, T-17, T-18, T-21, T-22, T-23, T-24

---

## 5. จุดเสี่ยงที่ต้องระวัง

| # | ความเสี่ยง | มาจาก | วิธีป้องกัน |
| --- | --- | --- | --- |
| 1 | Route ใหม่ทำ gin panic ตอน start | `AGENT.md` §Rules With Teeth | T-12 ต้องใช้ `:id` ให้ตรงกับ `/series/:id` เดิม · `TestServerNew_RegistersAllRoutesWithoutPanic` จะจับได้ |
| 2 | แก้ migration เก่าแทนที่จะเพิ่มใหม่ | `AGENT.md` §Editing Guardrails | T-08 / T-13 ต้องเป็นไฟล์ `0008`, `0009` เท่านั้น |
| 3 | N+1 query ตอนโหลด arcs ต่อเล่ม | T-11 | โหลด arcs ของทุกเล่มด้วย query เดียวแล้ว group ใน Go |
| 4 | ลบคำอภิธานทิ้ง `<span data-k>` ค้างใน `body_html` | T-10 | ยืนยันว่า trigger เด้ง `glossary_rev` ตอน DELETE และ re-render job ทำงาน — ต้องมีเทสคุม |
| 5 | เทสสถิติพังจากคอลัมน์ใหม่ | T-13 | `chapter_daily_stats` มี composite PK — ถ้าเพิ่ม entity ต้องตรวจว่า tag `gorm:"primaryKey"` ครบทุกคีย์ |
| 6 | เทสอ้าง coin pack เดิม | T-08 | ✅ ตรวจแล้วว่าใช้ `makeme` fixture ไม่ใช่ seed — แต่ให้รันยืนยันซ้ำ |
| 7 | Integration test skip เงียบ | ทุก task | ต้อง `export DOCKER_HOST` ก่อนรัน และยืนยันว่า SKIP = 0 |
| 8 | inline style แทนที่จะใช้ token | T-17, T-18, T-19 | สไตล์ต้องอยู่ใน `frontend/src/styles/` ตาม `CLAUDE.md` §Coding Rules |

---

## 6. Definition of Done

งานทั้งหมดถือว่าเสร็จเมื่อผ่านทุกข้อนี้:

```bash
# Backend
cd backend
gofmt -l .                    # ไม่มี output
go vet ./...                  # ไม่มี output
export DOCKER_HOST=unix://$HOME/.rd/docker.sock
export TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -v ./... | grep -c -- '--- FAIL'   # 0
go test -count=1 -v ./... | grep -c -- '--- SKIP'   # 0
docker rm -f $(docker ps -aq)

# Frontend
cd ../frontend
npm run typecheck
npm run build

# Redis ต้องหายไปจริง (เหลือได้เฉพาะ comment เชิงเหตุผล)
cd .. && grep -rin redis --include='*.go' --include='*.md' --include='*.yml' \
  --exclude-dir=node_modules .
```

เช็กลิสต์ปิดงาน:

- [x] ทุก checkbox ในหัวข้อ 1–3 ถูกติ๊ก
- [x] `docs/test-cases.md` มีตัวเลขรวมและ mapping ตรงกับผลรันจริง
- [x] `docs/user-stories.md` มีคอลัมน์ Status ครบ และ W-05 / W-10 / W-22 / R-17 = ✅
- [x] `docs/api-spec.md` ลิสต์ routes ครบทุกตัวที่ลงทะเบียนจริง (ตรวจด้วยคำสั่ง grep ในภาคผนวก `Check-diff.md`)
- [x] รัน `Check-diff.md` ซ้ำอีกรอบ — ประเด็นที่ตัดสินใจว่าจะแก้ ต้องหายไปหมด และประเด็นที่ตัดสินใจว่าไม่ทำ ต้องถูกย้ายไปหัวข้อ backlog ใน PRD
- [x] อัปเดต `Check-diff.md` ให้สะท้อนสถานะใหม่ พร้อมวันที่ตรวจรอบล่าสุด
