# Check-diff — ผลการตรวจสอบความสอดคล้อง Design ↔ Document ↔ Codebase

โครงการ **หมอกจันทร์ (Mokchan)** — Thai-first web novel platform

| หัวข้อ | ค่า |
| --- | --- |
| วันที่ตรวจสอบ | 2026-08-09 |
| Branch / Commit | `main` @ `5b3bf4a` (working tree สะอาด) |
| ขอบเขต | ทุกไฟล์ใน `design/`, `README.md`, `AGENT.md`, `CLAUDE.md`, `docs/*.md`, `backend/`, `frontend/`, `docker-compose.yml` |

> ## ⚠️ สถานะเอกสารฉบับนี้ — แก้ไขไปแล้ว (2026-08-10)
>
> รายงานนี้เป็น **ภาพ ณ วันที่ตรวจ** เก็บไว้เป็นบันทึกว่าพบอะไรบ้าง
> ประเด็นที่ตัดสินใจแก้ทั้งหมดถูกดำเนินการแล้วบน branch
> `chore/align-design-docs-code` ตามแผนใน [Task.md](Task.md) — ดูสรุปที่
> [§8 สถานะหลังแก้ไข](#8-สถานะหลังแก้ไข-2026-08-10) ท้ายไฟล์
>
> **อย่าใช้ตาราง §2–§5 เป็นสถานะปัจจุบัน** ให้อ่าน §8 คู่กันเสมอ

---

## 0. บทสรุปผู้บริหาร

ภาพรวมคือ **โครงการนี้มีความสอดคล้องสูงผิดปกติ** เมื่อเทียบกับโปรเจกต์ทั่วไป โดยเฉพาะ
เอกสารฝั่งวิศวกรรม (`api-spec.md`, `architecture.md`, `test-cases.md`) ที่ตรวจสอบแล้วตรงกับโค้ดจริง
เกือบทั้งหมด รวมถึงคำกล่าวอ้างเชิงตัวเลขที่ยืนยันได้จากการรันจริง

| มิติที่เปรียบเทียบ | ผลลัพธ์ |
| --- | --- |
| Design ↔ Frontend (โครงหน้าจอหลัก) | ตรงกัน 13/13 หน้าจอ |
| Design ↔ Frontend (design tokens, ธีม, ฟอนต์) | ตรงกัน 100% (ค่าสีตรงทุกตัวอักษร) |
| Design ↔ Frontend (รายละเอียดใน component) | ต่างกัน 18 จุด (ส่วนใหญ่เป็น element ที่ยังไม่ทำ) |
| `docs/api-spec.md` ↔ Routes จริง | 89 routes ตรงกัน, ไม่มีเอกสาร 4, มีเอกสารแต่ไม่ทำ 5 (ประกาศไว้แล้ว) |
| `docs/architecture.md` ↔ โครงสร้างโค้ด | กฎ dependency ถูกบังคับใช้จริง 100% |
| `docs/database-schema.md` ↔ `backend/migrations/` | ตรงกัน, ยกเว้นตาราง migration ตกหล่น 0007 |
| `docs/test-cases.md` ↔ ไฟล์เทสจริง | ตรงกัน 76/76 ฟังก์ชัน, ตัวเลข 575 tests / 0 skips **ยืนยันแล้ว** |
| `README.md` / `AGENT.md` / `CLAUDE.md` ↔ โค้ด | ตรงกัน ยกเว้น 4 จุด |

**ประเด็นที่ควรแก้ก่อน** (รายละเอียดในหัวข้อ 5):

1. **Redis** ถูกประกาศเป็นส่วนหนึ่งของ stack ใน README/AGENT/docker-compose แต่ backend ไม่มี Redis client เลย
2. **Seed data ขัดกับ Design** — `chapters_count` ของเรื่องหลัก = 214 เท่ากับ `source_chapters_count` ทำให้ฟีเจอร์ "สองตัวเลขบท" มองไม่เห็นในข้อมูลตั้งต้น (Design ระบุ 88 / 214)
3. **User story W-22 ไม่เป็นจริง** — `รอบปล่อยบทใหม่` ไม่ถูกแสดงบนหน้ารายละเอียดเรื่อง
4. **PRD ระบุ KPI "THB"** แต่ API สถิติไม่มีฟิลด์เงินบาท
5. **README ระบุ migration `0001 … 0006`** แต่ของจริงมีถึง `0007`

---

## 1. รายการทั้งหมดที่นำมาเปรียบเทียบ

### 1.1 Design (`design/`)

| ไฟล์ | ขนาด | เนื้อหา |
| --- | --- | --- |
| `design/Xianxia Platform.dc.html` | 1,769 บรรทัด | 12 หน้าจอหลัก + bottom sheets 4 แบบ + unlock modal |
| `design/Xianxia Reader.dc.html` | 667 บรรทัด | หน้าอ่าน + 3 side panels + settings sheet + note popover + resume toast |
| `design/support.js` | 1,911 บรรทัด | runtime ของ design-compiler (ไม่ใช่ spec ของผลิตภัณฑ์) |
| `design/.thumbnail` | — | artifact, อยู่ใน "Do Not Touch" ของ `CLAUDE.md` |

**หน้าจอใน Design (นับจาก `data-s`)**: `home`, `browse`, `detail`, `library`, `coins`, `comments`,
`write`, `stats`, `onboard`, `works`, `series`, `checkout` (+ `reader` ในอีกไฟล์)

### 1.2 Document

| ไฟล์ | บรรทัด | บทบาท |
| --- | --- | --- |
| `README.md` | 197 | setup, project layout, quick start, testing, "Not implemented" |
| `AGENT.md` | 229 | playbook สำหรับ AI agent, architecture rules, "Rules With Teeth" |
| `CLAUDE.md` | 91 | entrypoint เฉพาะ Claude, ชี้ไป `AGENT.md` |
| `docs/README.md` | 18 | index ของเอกสาร |
| `docs/prd.md` | 201 | vision, scope, non-goals, metrics, rollout phases, locked decisions |
| `docs/user-stories.md` | 79 | R-01…R-28, W-01…W-23, A-01…A-05 |
| `docs/architecture.md` | 282 | hexagonal architecture, bounded contexts, coin write path, jobs |
| `docs/api-spec.md` | 305 | REST catalogue + implementation status + deviations |
| `docs/database-schema.md` | 379 | schema อ่านง่าย + migration layout |
| `docs/test-cases.md` | 258 | U/I/E/L cases + mapping ไปยังไฟล์เทสจริง |
| `backend/test/makeme/docs.md` | — | คู่มือ test builder |

### 1.3 Codebase

| ส่วน | จำนวน | หมายเหตุ |
| --- | --- | --- |
| Backend Go packages | 8 bounded contexts (`identity`, `catalog`, `reading`, `library`, `wallet`, `social`, `writer`, `notification`) | ครบตาม `AGENT.md` |
| HTTP routes | 89 routes + `/health` | ใน `internal/handler/*` |
| Migrations | `0001` – `0007` | `backend/migrations/` |
| Background jobs | 9 jobs | `internal/jobs/registry.go` |
| Frontend routes | 16 route entries | `frontend/src/App.tsx` |
| Frontend route pages | 14 ไฟล์ | `frontend/src/routes/` |
| Stylesheets | 4 ไฟล์ (`tokens`, `base`, `components`, `reader`) | `frontend/src/styles/` |
| Go test functions | 302 ฟังก์ชัน → 575 test cases | รวม subtests |

### 1.4 คำสั่งที่รันจริงเพื่อยืนยัน

| คำสั่ง | ผลลัพธ์ |
| --- | --- |
| `gofmt -l .` | ไม่มี output — ผ่าน |
| `go vet ./...` | ไม่มี output — ผ่าน |
| `go test ./...` (ไม่มี Docker) | ผ่านทุก package แต่ **196 SKIP** — ยืนยันคำเตือนในเอกสารว่า skip เงียบ |
| `DOCKER_HOST=unix://$HOME/.rd/docker.sock TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -v ./...` | **575 PASS / 0 SKIP / 0 FAIL** — ตรงกับ `docs/test-cases.md` เป๊ะ |
| `npm run typecheck` | ผ่าน |
| `npm run build` | ผ่าน (105 modules, 322 kB JS / 20 kB CSS) |

---

## 2. Design ↔ Codebase

### 2.1 ตารางเทียบหน้าจอ (โครงสร้างหลัก)

| # | Design screen | Frontend route / ไฟล์ | ผลลัพธ์ |
| --- | --- | --- | --- |
| 1 | `home` — หน้าแรก | `/` → `routes/Home.tsx` | ✅ ตรงกัน |
| 2 | `browse` — หมวดหมู่และค้นหา | `/browse` → `routes/Browse.tsx` | ✅ ตรงกัน |
| 3 | `detail` — รายละเอียดเรื่อง | `/novels/:slug` → `routes/Novel.tsx` | ✅ ตรงกัน (ต่างรายละเอียด ดู 2.3) |
| 4 | `library` — ชั้นหนังสือ | `/library` → `routes/Library.tsx` | ✅ ตรงกัน |
| 5 | `coins` — เหรียญ | `/coins` → `routes/Coins.tsx` | ✅ ตรงกัน |
| 6 | `checkout` — ชำระเงิน | `/checkout/:packId` → `routes/Checkout.tsx` | ⚠️ ตรงบางส่วน |
| 7 | `comments` — ความเห็น | `/chapters/:id/comments` → `routes/Comments.tsx` | ✅ ตรงกัน |
| 8 | `works` — จัดการผลงาน | `/works` → `routes/Works.tsx` | ✅ ตรงกัน (5 แท็บครบ) |
| 9 | `write` — เขียนบท | `/write` → `routes/Write.tsx` | ✅ ตรงกัน (wizard 3 ขั้นครบ) |
| 10 | `stats` — สถิติผลงาน | `/stats` → `routes/Stats.tsx` | ⚠️ ตรงบางส่วน |
| 11 | `onboard` — ตั้งค่าแนวที่ชอบ | `/onboarding` → `routes/Onboarding.tsx` | ⚠️ ตรงบางส่วน |
| 12 | `series` — ชุดหนังสือ | `/series/:id` → `routes/Series.tsx` | ⚠️ ตรงบางส่วน (ขาดหลายส่วน) |
| 13 | Reader (ไฟล์แยก) | `/read/:id` → `routes/Reader.tsx` | ✅ ตรงกัน |
| — | *(ไม่มีใน Design)* | `/login`, `/register` → `routes/Auth.tsx` | ➕ โค้ดมีเกิน Design |
| — | *(ไม่มีใน Design)* | `/account` → redirect `/library` | ➕ โค้ดมีเกิน Design |
| — | *(ไม่มีใน Design)* | `*` → NotFound | ➕ โค้ดมีเกิน Design |

### 2.2 Design tokens, ธีม และ typography — ตรงกัน 100%

ตรวจค่าต่อค่าระหว่าง `design/*.dc.html` กับ `frontend/src/styles/tokens.css` และ `routes/Reader.tsx`:

| กลุ่ม | Design | Codebase | ผล |
| --- | --- | --- | --- |
| สีหลัก (light) | `bg #FAF8F3` · `ink #23201B` · `red #A8382B` · `gold #A9803F` · `panel #FFFDF8` · `soft #6B6357` | เหมือนกันทุกค่า ([tokens.css:13-25](frontend/src/styles/tokens.css#L13-L25)) | ✅ |
| ธีม sepia | `#F1E6D0` / `#3A3025` / `#7A6A52` / `#F8F0DE` / `#9C3A25` / `#96762F` | เหมือนกันทุกค่า ([tokens.css:57-74](frontend/src/styles/tokens.css#L57-L74)) | ✅ |
| ธีม dark | `#121211` / `#CFC8BA` / `#8B8478` / `#1C1B19` / `#C4574A` / `#BE9758` | เหมือนกันทุกค่า ([tokens.css:76-93](frontend/src/styles/tokens.css#L76-L93)) | ✅ |
| ฟอนต์ | `IBM Plex Sans Thai` / `Trirong` / `Sarabun` / `IBM Plex Mono` / `Noto Serif Thai` | เหมือนกัน + โหลดครบใน `index.html` | ✅ |
| Reader font map | `loop→Sarabun`, `serif→Noto Serif Thai`, `sans→IBM Plex Sans Thai` | เหมือนกัน ([Reader.tsx:754-763](frontend/src/routes/Reader.tsx#L754-L763)) | ✅ |
| Reader width map | `narrow 540px` / `normal 640px` / `wide 760px` | เหมือนกัน ([Reader.tsx:765-774](frontend/src/routes/Reader.tsx#L765-L774)) | ✅ |
| ค่าเริ่มต้น | `theme light`, `font loop`, `fs 20`, `lh 2.0`, `width normal` | เหมือนกัน ([prefs.ts:9-15](frontend/src/lib/prefs.ts#L9-L15)) | ✅ |
| Sidebar width | `244px` | `--sidebar-w: 244px` | ✅ |

> `frontend/src/styles/tokens.css` มี comment กำกับไว้ว่า *"Design tokens, lifted from design/…"* ซึ่งตรวจแล้วเป็นความจริง

### 2.3 ความแตกต่างระดับรายละเอียด (Design มี — Codebase ยังไม่มี)

| # | หน้าจอ | Design | Codebase | ระดับ |
| --- | --- | --- | --- | --- |
| D-01 | home | บรรทัด `อ่านสะสมสัปดาห์นี้ 4 ชั่วโมง 12 นาที` | ไม่มี metric เวลาอ่านสะสมทั้งระบบ (ทั้ง API และ UI) | กลาง |
| D-02 | detail | KPI 5 ช่อง (เรตติ้ง / แปลแล้ว / ต้นฉบับ / **ภาค** / ผู้ติดตาม) | 4 ช่อง — ย้าย `ภาค` ไปบรรทัด meta ([Novel.tsx:153-169](frontend/src/routes/Novel.tsx#L153-L169)) | ต่ำ |
| D-03 | detail | บรรทัดสรุปราคา `บทที่ 1–48 (ภาคที่ 1) อ่านฟรี · บทหลังจากนั้น 5 เหรียญต่อบท` | ไม่มี — และ `NovelDetailResponse` ไม่ส่ง `price_per_chapter` / `free_until_chapter` ออกมาเลย | กลาง |
| D-04 | detail | ปุ่ม `ดูทั้งชุด · 5 เรื่อง 8 ภาค` | มีแค่ pill `อยู่ในชุดหนังสือ →` ไม่มีตัวเลขสรุป | ต่ำ |
| D-05 | detail | `อภิธานศัพท์ 27 คำ` และ `ความเห็น · 1,208` เป็นลิงก์กดได้ | แสดงเป็นข้อความธรรมดา ไม่ใช่ลิงก์ | ต่ำ |
| D-06 | coins | `ปลดล็อกได้อีกราว 48 บท` | ไม่มี | ต่ำ |
| D-07 | coins | `ใช้ไปเดือนนี้ 165 เหรียญ` | ไม่มี | ต่ำ |
| D-08 | coins | แพ็ก `100/฿35`, `300+15/฿99`, `700+70/฿219`, `1500+200/฿449` | seed เป็น `60/฿39`, `240+20/฿149`, `600+100/฿349`, `1500+300/฿799` | ต่ำ (ข้อมูล) |
| D-09 | checkout | accordion + radio dot ต่อช่องทาง | ใช้ `Tabs` component แทน | ต่ำ |
| D-10 | checkout | checkbox `บันทึกบัตรใบนี้ไว้…` | ไม่มี | ต่ำ |
| D-11 | checkout | QR countdown `QR หมดอายุใน 09:58 นาที` + ปุ่ม `บันทึกภาพ QR` | ไม่มี | ต่ำ |
| D-12 | checkout | `โค้ดส่วนลด`, `ส่วนลดสมาชิก −฿0`, `ภาษีมูลค่าเพิ่ม รวมแล้ว` | ไม่มี | ต่ำ |
| D-13 | checkout | `ชำระผ่านช่องทางที่เข้ารหัสตามมาตรฐาน PCI DSS` | แทนด้วยข้อความ *"ระบบชำระเงินจำลอง… ข้อมูลบัตรไม่ถูกส่งหรือจัดเก็บ"* | ✅ เหมาะสมกว่า |
| D-14 | stats | KPI ที่ 4 = `อ่านจบต่อบท 86% (ค่าเฉลี่ยหมวด 71%)` | แทนด้วย `ช่วงเวลา` (วันที่เริ่ม–สิ้นสุด) — ไม่มี completion rate ใน API | กลาง |
| D-15 | stats | `เหรียญที่ได้รับ 41,650 · ราว ฿12,900` | ไม่มีตัวเลขเงินบาท (DTO มีแค่ `coins_earned`) | **สูง** (ขัด PRD ด้วย) |
| D-16 | onboard | หน้าเดียว: เลือกแนว + `ความยาวที่ชอบ` พร้อมกัน, กำกับ `เลือกอย่างน้อย 3 แนว` | แยกเป็น 2 step จริง และบังคับแค่ ≥ 1 แนว | ต่ำ |
| D-17 | onboard | ชิปแนวแสดงอักษรจีน + ชื่อไทย (`{{ g.cn }}` + `{{ g.name }}`) | แสดงชื่อไทยอย่างเดียว | ต่ำ |
| D-18 | series | สถิติ 4 ช่อง (เรื่อง / **ภาค** / แปลแล้ว / ต้นฉบับ) | 3 ช่อง ไม่มีจำนวนภาค | ต่ำ |
| D-19 | series | section `เรื่องหลัก` — การ์ดเล่มพร้อม progress bar, CTA และรายการภาคย่อย | มีแค่ reading-order list ธรรมดา | กลาง |
| D-20 | series | section `เรื่องเกี่ยวเนื่อง` บนหน้า series | ไม่มี (มีเฉพาะบนหน้า novel detail) | กลาง |
| D-21 | series | ปุ่ม `ติดตามทั้งชุด` | ไม่มี — และ backend ไม่มี follow-by-series (follow เป็นราย novel เท่านั้น) | กลาง |
| D-22 | reader | popover มีปุ่ม `ดูในอภิธานศัพท์ →` | popover มีแค่ปุ่มปิด ([Reader.tsx:367-374](frontend/src/routes/Reader.tsx#L367-L374)) | ต่ำ |
| D-23 | reader | คำแนะนำ `บีบนิ้วเพื่อย่อ–ขยายตัวอักษร` (pinch-to-zoom) | ไม่มี gesture นี้ | ต่ำ |
| D-24 | works | label แม่แบบปก `หมึกจีนแนวตั้ง` / `ตราประทับ` / `ลายพู่กัน` / `เรียบ ตัวอักษรอย่างเดียว` | key ตรงกัน (`ink`/`seal`/`brush`/`plain`) แต่ label เป็น `ลายทแยง` / `ตราประทับ` / `ไล่สี` / `สีพื้น` | ต่ำ (copy) |
| D-25 | works | `รอบปล่อยบทใหม่` 4 ตัวเลือก รวม `สัปดาห์ละ 2 บท จันทร์และพฤหัส` | 5 ตัวเลือก, เพิ่ม `เดือนละครั้ง`, ตัดชื่อวันออก | ต่ำ |
| D-26 | write | ป้ายขั้นตอน `เลือกเรื่อง` / `เลือกบทหรือสร้างบทใหม่` / `เขียน` | `เลือกผลงาน` / `เลือกบท` / `เขียนและเผยแพร่` | ต่ำ (copy) |
| D-27 | works | ปุ่ม `ดูรายได้ที่คาดการณ์` | ไม่มี | ต่ำ |

### 2.4 Codebase มีเกิน Design (เพิ่มเข้ามาอย่างสมเหตุสมผล)

| # | รายการ | หมายเหตุ |
| --- | --- | --- |
| C-01 | หน้า `/login` และ `/register` | Design ไม่มีหน้า auth เลย (grep `เข้าสู่ระบบ`/`รหัสผ่าน` = 0 ครั้ง) แต่ระบบต้องมี |
| C-02 | ปุ่ม `ส่งทิป` ท้ายบทใน Reader ([Reader.tsx:474-529](frontend/src/routes/Reader.tsx#L474-L529)) | Design มีเฉพาะ checkbox `เปิดรับทิปจากผู้อ่านท้ายบท` ในแท็บราคา ไม่ได้วาด UI ฝั่งผู้อ่าน |
| C-03 | `ซื้อทั้งภาค` modal พร้อมใบเสนอราคา (gross / ส่วนลด / total) | Design ไม่ได้วาด — ระบุแค่ checkbox ขายเป็นภาค |
| C-04 | รีวิว/ให้ดาว บนหน้า novel detail | Design แสดงแค่ตัวเลข `2,140 รีวิว` ไม่มีฟอร์มเขียนรีวิว |
| C-05 | drag-and-drop reorder (`lib/reorder.ts`) | Design บอกแค่ "ลากเพื่อจัดลำดับ" |
| C-06 | สถานะ loading / empty / error ทุกหน้า | Design เป็น static mock |
| C-07 | responsive shell 3 แบบ (sidebar / icon rail / bottom tab bar) | Design มีแค่ sidebar |

---

## 3. Document ↔ Codebase

### 3.1 `README.md`

| # | ข้อความในเอกสาร | สถานะจริง | ผล |
| --- | --- | --- | --- |
| R-a | "React (Vite + TS + TanStack Query) + Go 1.25 (Gin + GORM + pgx) + PostgreSQL 16 + **Redis 7**" | `go.mod` = `go 1.25.0` ✅, postgres:16-alpine ✅, **Redis: `go.mod` ไม่มี redis client ใดๆ; grep ทั้ง backend เจอแต่ comment ที่บอกว่า *ไม่* ใช้ Redis** | ❌ **ไม่ตรง** |
| R-b | "PRD phases 1–6 are implemented end to end" | ตรงในระดับ API/domain; ระดับ UI มีช่องว่าง (notification UI, payout UI, glossary CRUD UI) | ⚠️ ตรงบางส่วน |
| R-c | `go run ./cmd/migrate -cmd up  # applies 0001 … 0006` | ของจริงมี `0001`–**`0007`** | ❌ **ไม่ตรง** |
| R-d | "Background jobs (bonus expiry, glossary re-render, scheduled publishing, stats rollups, weekly ranking)" | มี **9** jobs — ตกหล่น `auto_unlock`, `session_sweep`, `wallet_reconcile`, `read_event_partitions` | ⚠️ ไม่ครบ |
| R-e | "`JWT_SECRET` must be at least 16 characters or the API refuses to start" | `auth.MinSecretLen = 16`, `NewIssuer` คืน `ErrWeakSecret`, `server.New` panic | ✅ ตรง |
| R-f | Project layout tree | ตรงกับ directory จริงทุกบรรทัด | ✅ ตรง |
| R-g | "testcontainers **skips** rather than fails without Docker" | ยืนยันแล้ว: ไม่มี Docker → 196 SKIP แต่ exit 0 | ✅ ตรง |
| R-h | "Not implemented: Phase 7 / most `/admin/*` / E และ L tiers" | ตรงทุกข้อ | ✅ ตรง |
| R-i | "Every novel surface now shows two chapter counts" | หน้า Library และการ์ด "อ่านค้างไว้" บนหน้าแรกแสดงเฉพาะ `บทที่แปลแล้ว` | ⚠️ กล่าวเกินจริงเล็กน้อย (PRD เขียนถูกต้องกว่า) |

### 3.2 `AGENT.md`

| # | ข้อความในเอกสาร | สถานะจริง | ผล |
| --- | --- | --- | --- |
| A-a | Repository Map (23 รายการ) | มีครบทุก path | ✅ ตรง |
| A-b | Bounded contexts 8 ตัว "never import one another" | ตรวจด้วย `go list -deps` — ไม่มี cross-context import | ✅ ตรง |
| A-c | "Keep domain packages framework-free" | ตรวจแล้ว: ไม่มี domain package ใด import gorm / gin / entities / repository / handler | ✅ **ยืนยันแล้ว** |
| A-d | "Every repository method begins `db := dbctx.From(ctx, r.db)`" | สแกน 114 repository methods — **0 method** แตะ `r.db` โดยไม่ผ่าน `dbctx.From` | ✅ **ยืนยันแล้ว** |
| A-e | Gin wildcard: "Every `/novels/...` route uses `:id`" | ตรวจ 89 routes — ใช้ `:id` ทั้งหมด, มี `TestServerNew_RegistersAllRoutesWithoutPanic` เฝ้า | ✅ ตรง |
| A-f | "One coin write path … `wallet.Repository.Apply`" | ตรง; ledger/balance/unlock/earnings เขียนผ่าน `Apply` เท่านั้น | ✅ ตรง |
| A-g | "Rate limiters are per-engine, never package globals" | สร้างใน `server.Build` ([server.go:119-124](backend/internal/server/server.go#L119-L124)) | ✅ ตรง |
| A-h | "`SetTrustedProxies` must stay explicit" | [server.go:101](backend/internal/server/server.go#L101) | ✅ ตรง |
| A-i | "Advisory locks in jobs are transaction-scoped" | `pg_try_advisory_xact_lock` ใน `jobs.withJobLock` | ✅ ตรง |
| A-j | Redis อยู่ใน "Project Snapshot" / "Runtime" | เหมือน R-a — ไม่มีการใช้งานจริง | ❌ **ไม่ตรง** |

### 3.3 `CLAUDE.md`

| # | ข้อความในเอกสาร | สถานะจริง | ผล |
| --- | --- | --- | --- |
| C-a | "PRD phases 1–6 are implemented" (ลิสต์ฟีเจอร์) | ตรงกับ README/AGENT | ✅ สอดคล้องภายใน |
| C-b | "Phase 7 and most `/admin/*` deliberately not implemented" | ตรง (มีเฉพาะ `POST /admin/wallet-adjust`) | ✅ ตรง |
| C-c | Coding Rules (layer, route registration, styles) | โค้ดปฏิบัติตามครบ | ✅ ตรง |
| C-d | Common Commands | คำสั่งทั้งหมดใช้งานได้จริง (รันแล้ว) | ✅ ตรง |
| C-e | Do Not Touch list | ตรงกับ `.gitignore` และไฟล์จริง | ✅ ตรง |

### 3.4 `docs/prd.md`

| # | ข้อความในเอกสาร | สถานะจริง | ผล |
| --- | --- | --- | --- |
| P-a | Reader: 3 ธีม / 3 ฟอนต์ / 3 ความกว้าง / ปรับขนาด + line height | ครบทุกตัว | ✅ ตรง |
| P-b | Reader: immersive tap, term popover, 3 side panels, sync prefs | ครบทุกตัว | ✅ ตรง |
| P-c | Novel detail: chapter filter pills `ทั้งหมด/อ่านฟรี/ปลดล็อกแล้ว/ยังไม่ปลดล็อก` | ตรงเป๊ะ ([Novel.tsx:320-323](frontend/src/routes/Novel.tsx#L320-L323)) | ✅ ตรง |
| P-d | Novel detail: บทเกินจำนวนที่แปลแล้วแสดง dimmed `ยังไม่แปล` | มี ([Novel.tsx:602](frontend/src/routes/Novel.tsx#L602)) | ✅ ตรง |
| P-e | Library: `รายละเอียดและสารบัญ` + accent CTA + progress เป็นจำนวนบท | ตรงเป๊ะ ([components/index.tsx:341-366](frontend/src/components/index.tsx#L341-L366)) | ✅ ตรง |
| P-f | Arc bundles ลด 15% (platform constant) | `discount_percent` เป็นค่าคงที่ ไม่มีคอลัมน์ต่อ novel | ✅ ตรง |
| P-g | Tips 1–1000, paid coins only, `INSUFFICIENT_PAID_COINS` (402) | ตรง มีเทส I-TIP-01…03 | ✅ ตรง |
| P-h | Auto-unlock + 24h early access, teaser สำหรับคนอื่น | ตรง มีเทส I-EA-01…06, I-AU-01…08 | ✅ ตรง |
| P-i | Works management 5 แท็บ | ตรงเป๊ะ ([Works.tsx:36-40](frontend/src/routes/Works.tsx#L36-L40)) | ✅ ตรง |
| P-j | `ซ่อนจากหน้าร้าน` เป็นสถานะจริง | มี `hidden` ใน CHECK constraint + เทส I-HID-01/02 | ✅ ตรง |
| P-k | Relation kinds 5 แบบ | ตรงทั้ง 5 (`sequel`/`prequel`/`spinoff`/`side_story`/`same_world`) | ✅ ตรง |
| P-l | `เขียนบท` เป็น wizard 3 ขั้น derive จาก URL | ตรง ([Write.tsx:164](frontend/src/routes/Write.tsx#L164)) | ✅ ตรง |
| P-m | **"Stats page: KPI tiles (reads, followers, coins, **THB**)"** | `StatsResponse` ไม่มีฟิลด์เงินบาท และ UI ไม่แสดง | ❌ **ไม่ตรง** |
| P-n | "Coin earnings include tips as well as unlocks" | `writer_earnings.kind` แยก `unlock`/`tip` และรวมใน stats | ✅ ตรง |
| P-o | Onboarding: "Pick favorite genres" | มี — แต่ `ความยาวที่ชอบ` (มีใน Design + โค้ด) ไม่ถูกกล่าวถึงใน PRD เลย | ⚠️ เอกสารขาด |
| P-p | Rollout table: phase 1–6 `done`, phase 7 `not implemented` | ตรง | ✅ ตรง |
| P-q | Locked decisions ทั้ง 9 ข้อ | ตรวจแล้วตรงกับโค้ดทุกข้อ | ✅ ตรง |
| P-r | `รอบปล่อยบทใหม่` เป็น display-only metadata | ตรงในแง่ที่ไม่ขับ scheduler — แต่ไม่มีที่ไหน "display" เลย (ดู 3.5 W-22) | ⚠️ ตรงครึ่งเดียว |

### 3.5 `docs/user-stories.md`

| ID | Story | สถานะจริง | ผล |
| --- | --- | --- | --- |
| R-01…R-19 | Reader core | ครบทุกข้อ | ✅ |
| R-20…R-28 | series / bundles / tips / auto-unlock | ครบทุกข้อ | ✅ |
| W-01…W-04 | สร้างเรื่อง / arc / เขียน+autosave / footnote+glossary bind | ครบ | ✅ |
| **W-05** | "Manage per-novel glossary (groups + entries). **CRUD**" | Backend มีแค่ `POST /writer/novels/{id}/glossary` + `PATCH /writer/glossary-entries/{id}` — **ไม่มี DELETE** และ **frontend ไม่มี UI จัดการ glossary เลย** (`listWriterGlossary` ถูกประกาศแต่ไม่มีใครเรียก) | ❌ **ไม่ครบ** |
| W-06…W-09 | draft/schedule/publish, ราคา, สถิติ, ตอบคอมเมนต์ | ครบ | ✅ |
| **W-10** | "Request payout of coins to fiat" | `POST /writer/payouts` + `GET /writer/earnings` มีใน backend แต่ **ไม่มี UI ใดเรียก** (grep `payout` ใน `frontend/src` = 0) | ⚠️ **backend-only** |
| W-11…W-21, W-23 | works workspace, source count, cover template, series, relations, pricing, arc sale, tips, early access, hidden, deep link | ครบทุกข้อ | ✅ |
| **W-22** | "`รอบปล่อยบทใหม่` **is shown on the detail page**" | API ส่ง `release_schedule` ออกมาและ TypeScript type มีฟิลด์นี้ แต่ **`Novel.tsx` ไม่เคย render** — ใช้เฉพาะในแท็บตั้งค่าของ `Works.tsx` | ❌ **ไม่ตรง** |
| A-01, A-02, A-04, A-05 | Admin moderate / coin packs / payouts / genres | ไม่ได้ทำ (`api-spec.md` และ `README.md` ประกาศไว้แล้ว) | ⚠️ `user-stories.md` ไม่มีคอลัมน์สถานะกำกับ |
| A-03 | Adjust wallet balance | มี `POST /admin/wallet-adjust` + เทส I-COIN-06 | ✅ |

### 3.6 `docs/architecture.md`

| # | ข้อความในเอกสาร | สถานะจริง | ผล |
| --- | --- | --- | --- |
| Ar-a | Hexagonal layers (domain → service → repository → handler → server) | ตรง | ✅ |
| Ar-b | Dependency rule table | domain สะอาด ✅, service ✅, repository ✅ — แต่ **handler import `internal/middleware`** (`wallet`, `writer`) ซึ่งไม่อยู่ในรายการ "May import" | ⚠️ ตารางไม่ครบ |
| Ar-c | ตัวอย่าง composition root: `catalogsvc.New(catalogRepo)` | ของจริงเป็น `catalogsvc.New(catalogRepo, walletRepo)` ([server.go:193](backend/internal/server/server.go#L193)) | ⚠️ ตัวอย่างล้าสมัย |
| Ar-d | Bounded-context table (8 contexts + ports) | ตรง — `reading.Entitlements`, `catalog.Entitlements`, `notification.Followers` มีจริง | ✅ |
| Ar-e | Transactions: repository-owned + ambient (`dbctx`) | ตรง (ยืนยันจาก 114 methods) | ✅ |
| Ar-f | Single coin write path 5 ขั้นตอน | ตรงกับ `repository/wallet/apply.go` | ✅ |
| Ar-g | **"`internal/jobs` holds a scheduler and nine jobs"** | นับได้ **9 jobs** พอดี: `bonus_expiry`, `glossary_rerender`, `publish_scheduled`, `auto_unlock`, `stats_rollup`, `weekly_ranking`, `session_sweep`, `wallet_reconcile`, `read_event_partitions` | ✅ **ยืนยันแล้ว** |
| Ar-h | Two-axis visibility (`See` beside `Decide`, 3 states) | ตรง (`domain/reading/visibility.go`) | ✅ |
| Ar-i | `withJobLock` ใช้ claim batch เท่านั้น + เทสเฝ้า | ตรง — `TestAutoUnlockJob_OneBrokeSubscriberDoesNotRollBackTheOthers` มีจริง | ✅ |
| Ar-j | Known risks 3 ข้อ | ตรงกับโค้ด (ไม่มี sanitiser, ไม่มี token denylist, rate limit in-process) | ✅ |

### 3.7 `docs/api-spec.md`

**สรุป: 89 routes ที่ลงทะเบียนจริง ตรงกับเอกสารเกือบทั้งหมด**

| กลุ่ม | เอกสาร | โค้ด | ผล |
| --- | --- | --- | --- |
| Auth (5) | register, login, refresh, logout, GET /auth/me | ครบ 5 | ✅ |
| Me / prefs | GET+PATCH `/users/me`, GET+PUT `/users/me/prefs`, PUT `/users/me/genre-prefs` | ครบ + **มี `GET /users/me/genre-prefs` เพิ่ม (ไม่อยู่ในเอกสาร)** | ⚠️ |
| Catalog (10) | genres, novels, novel detail, chapters, arcs, glossary, related, series, search, ranking | ครบ 10 | ✅ |
| Reading (6) | chapter, next, prev, read-event, GET+PUT progress | ครบ 6 | ✅ |
| Library/Follows (9) | library×3, bookmarks×3, follows POST+DELETE | ครบ + **มี `GET /me/follows/{novel_id}` เพิ่ม (ไม่อยู่ในเอกสาร)** | ⚠️ |
| Comments/Reviews (7) | ครบ | ครบ 7 | ✅ |
| Wallet (16) | wallet, ledger, packs, purchases, mock-complete/fail, unlock, tip, bundle quote+unlock, auto-unlock×3, earnings, payouts, wallet-adjust | ครบ 16 | ✅ |
| Writer (27) | ตามตาราง | ครบ + **`GET /writer/novels` และ `GET /writer/novels/{id}/arcs` ไม่อยู่ในเอกสาร** | ⚠️ |
| Notifications (3) | ครบ | ครบ 3 | ✅ |
| Admin | 5 กลุ่ม endpoint | ทำจริงแค่ `POST /admin/wallet-adjust` | ✅ **ประกาศไว้แล้วใน "Not implemented"** |

**ประเด็นเพิ่มเติม:**

| # | ประเด็น | รายละเอียด |
| --- | --- | --- |
| Ap-a | `GET /search?q=&type=novel\|chapter\|character` | handler เป็น `h.respondNovelList(c, c.Query("q"))` — **ไม่อ่านพารามิเตอร์ `type` เลย** เอกสารระบุว่าเป็น alias ของ `/novels` แต่ยังคงลิสต์ `type` ในตาราง |
| Ap-b | "Deviations from this document" 8 ข้อ | ตรวจแล้วถูกต้องทุกข้อ (slug-or-id, optional auth + `unlocked`, `my_review`, `available_satang`, keyset cursor, logout, purchases idempotency, namespaced keys, hidden 404) | ✅ |
| Ap-c | Error codes 40 กว่ารหัส | สุ่มตรวจครบ ตรงกับ `httpx` และ handler mapping | ✅ |
| Ap-d | Chapter response shape (`locked_reason`: `paywall` / `early_access`) | ตรงกับ `domain/reading` | ✅ |

### 3.8 `docs/database-schema.md` ↔ `backend/migrations/`

| # | ข้อความในเอกสาร | สถานะจริง | ผล |
| --- | --- | --- | --- |
| Db-a | Extensions `citext`, `pg_trgm`, `pgcrypto` | ตรง ([0001_init.sql:4-6](backend/migrations/0001_init.sql#L4-L6)) | ✅ |
| Db-b | ตารางทั้งหมดใน "Domain overview" | มีครบทุกตาราง | ✅ |
| Db-c | `user_prefs` constraints (theme/font/font_size 14–28/line_height 1.4–2.4/column_width) | ตรง | ✅ |
| Db-d | Glossary trigger `glossary_entries_bump` | ตรง ([0001_init.sql:190](backend/migrations/0001_init.sql#L190)) | ✅ |
| Db-e | 0007: novels ได้ **สิบสอง** คอลัมน์ใหม่ | นับได้ 12 พอดี | ✅ |
| Db-f | 0007: partial unique index `novels_series_position` | ตรง | ✅ |
| Db-g | 0007: `chapters.public_at` snapshot at publish | ตรง + backfill | ✅ |
| Db-h | 0007: 3 ตารางใหม่ (`auto_unlock_subscriptions`, `auto_unlock_attempts`, `novel_relations`) | ตรง | ✅ |
| Db-i | **"PostgreSQL 15+"** | README/AGENT + docker-compose ใช้ **16** | ⚠️ ไม่ขัดกันโดยตรง แต่ไม่สอดคล้อง |
| Db-j | **ตาราง "Migration & seed layout" ลิสต์แค่ `0001`–`0006`** | มี `0007` (มีหัวข้ออธิบายแยกด้านล่าง แต่ตกจากตาราง) | ❌ **ตกหล่น** |
| Db-k | **"Live ranking is computed in Redis (`ZSET`) and snapshotted here"** | `jobs.RankingJob` เขียน `ranking_snapshots` ตรงๆ ไม่มี Redis — และหัวข้อ "Open items → Resolved" ในไฟล์เดียวกันเขียนว่า *"Done **without Redis**"* | ❌ **ขัดกันเองในเอกสารฉบับเดียว** |
| Db-l | "0002 seeds … **the featured novel**, its arcs, chapters 86–88, glossary entries, and 4 coin packs" | seed มี **2 เรื่อง** (`nine-streams-sword-immortal` + `return-to-nineteen`), arcs 4, chapters 86–88 ✅, coin packs 4 ✅ | ⚠️ ระบุจำนวนเรื่องไม่ครบ |
| Db-m | Production caveat เรื่อง `ACCESS EXCLUSIVE` / `CREATE INDEX CONCURRENTLY` | ตรงกับ comment ใน 0007 | ✅ |

### 3.9 `docs/test-cases.md`

| # | ข้อความในเอกสาร | สถานะจริง | ผล |
| --- | --- | --- | --- |
| T-a | **"575 passing tests with 0 skips"** | รันจริงด้วย Docker → **575 PASS / 0 SKIP / 0 FAIL** | ✅ **ยืนยันเป๊ะ** |
| T-b | Mapping table (76 ชื่อฟังก์ชันเทส) | สแกนแล้ว **76/76 มีอยู่จริง** ในโค้ด | ✅ **ยืนยันแล้ว** |
| T-c | "E และ L tiers out of scope" | ไม่มี browser/load toolchain ในโปรเจกต์ | ✅ ตรง |
| T-d | "**I — integration, real Postgres/Redis**" | ไม่มี Redis ในเทสใดๆ | ❌ **ไม่ตรง** |
| T-e | `TestRerenderJob_.../a_stale_body_still_serves_the_old_HTML` | ชื่อจริงคือ `TestRerenderJob_UpdatesBodyHTMLAndLiftsGlossaryRev` + subtest `"a stale body still serves the old HTML"` | ✅ ตรง (แค่ย่อชื่อในเอกสาร) |
| T-f | คำเตือน `docker rm -f $(docker ps -aq)` หลังรัน | จำเป็นจริง — หลังรันมี container ค้าง 3 ตัว | ✅ ตรง |

---

## 4. Design ↔ Document

| # | หัวข้อ | Design | Document | ผล |
| --- | --- | --- | --- | --- |
| DD-01 | หน้าจอทั้งหมด | 12 + reader | PRD §5 ครอบคลุมทุกหน้า | ✅ ตรง |
| DD-02 | Checkout | มี form บัตร / พร้อมเพย์ / ทรูมันนี่ เต็มรูปแบบ + PCI DSS | PRD/README ระบุชัดว่า Phase 7 (payment provider จริง) **ยังไม่ทำ** และ checkout เป็น "UI shell" | ✅ เอกสารอธิบาย gap ไว้แล้ว |
| DD-03 | `ความยาวที่ชอบ` (สั้น/กลาง/ยาว) ใน onboarding | มี | **PRD ไม่กล่าวถึงเลย** (เขียนแค่ "Pick favorite genres") | ❌ เอกสารขาด |
| DD-04 | `ติดตามทั้งชุด` (follow ทั้ง series) | มีปุ่มบนหน้า series | ไม่มีใน PRD, user-stories หรือ api-spec | ❌ เอกสารขาด |
| DD-05 | `อ่านสะสมสัปดาห์นี้ N ชั่วโมง` | มีบนหน้าแรก | ไม่มีใน PRD / schema / metrics | ❌ เอกสารขาด |
| DD-06 | `อ่านจบต่อบท 86%` (completion rate) | มีใน stats | ไม่มีใน PRD (PRD ระบุ reads/followers/coins/THB) | ❌ เอกสารขาด |
| DD-07 | `ราว ฿12,900` (แปลงเหรียญเป็นบาท) | มีใน stats | **PRD ระบุตรงกัน** ("KPI tiles (reads, followers, coins, THB)") | ✅ Design + Doc ตรงกัน (โค้ดไม่ตรง) |
| DD-08 | หน้า login / register | **ไม่มีใน Design** | user-stories ไม่มีเรื่อง auth flow เป็น story แยก แต่ api-spec มี `/auth/*` ครบ | ⚠️ Design ขาด |
| DD-09 | UI แจ้งเตือน (notification) | ไม่มี (พูดถึงคำว่า "แจ้งเตือน" แค่ใน copy) | R-17 + api-spec มี 3 endpoints | ⚠️ Design ขาด → ทำให้ UI ก็ไม่มี |
| DD-10 | แม่แบบปก 4 แบบ | `ink`/`seal`/`brush`/`plain` | `database-schema.md` + migration 0007 CHECK `('image','ink','seal','brush','plain')` | ✅ ตรง |
| DD-11 | `รอบปล่อยบทใหม่` | select 4 ตัวเลือกในแท็บราคา | PRD: display-only; W-22: "shown on the detail page" | ⚠️ Design ไม่ได้วาดบนหน้า detail ด้วย |
| DD-12 | ตัวเลข `88 บทที่แปลแล้ว` / `214 บทในต้นฉบับ` | ระบุชัด | PRD §Discovery ระบุแนวคิดสองตัวเลข | ✅ ตรง (แต่ seed ไม่ตรง — ดู S-01) |

---

## 5. สรุปความแตกต่างทั้งหมด เรียงตามความสำคัญ

### 🔴 ระดับสูง — ควรแก้

| # | ประเด็น | อยู่ที่ | แก้ที่ไหน |
| --- | --- | --- | --- |
| H-01 | **Redis ถูกประกาศเป็นส่วนหนึ่งของ stack แต่ไม่มีการใช้งานจริง** — `README.md:3`, `AGENT.md` "Project Snapshot"/"Runtime", `docker-compose.yml` service `redis`, `docs/database-schema.md` ("Live ranking … in Redis ZSET"), `docs/test-cases.md` ("real Postgres/Redis"). backend ไม่มี redis client ใน `go.mod` และ `internal/ratelimit/limiter.go:3` เขียนไว้เองว่า *"the system has no other Redis dependency"* | Doc + docker-compose ↔ Code | แก้เอกสาร 5 จุด และตัดสินใจว่าจะเก็บ service `redis` ไว้เผื่ออนาคตหรือถอดออก |
| H-02 | **Seed ขัดกับ Design** — `0002_seed.sql` ตั้ง `chapters_count = 214` ให้ `nine-streams-sword-immortal` และ `0007` ตั้ง `source_chapters_count = 214` ด้วย ⇒ ผู้ใช้ที่ติดตั้งใหม่จะเห็น "214 บทที่แปลแล้ว / 214 บทในต้นฉบับ" ขณะที่ Design ระบุ **88 / 214** ทำให้ฟีเจอร์เรือธง ("สองตัวเลขบท") มองไม่เห็น และ logic `ยังไม่แปล` บนหน้า ToC ไม่ทำงาน | Design ↔ Seed data | `backend/migrations/` เพิ่ม migration ใหม่ตั้ง `chapters_count = 88` |
| H-03 | **W-22 ไม่เป็นจริง** — `รอบปล่อยบทใหม่` ไม่ถูกแสดงบนหน้ารายละเอียดเรื่อง ทั้งที่ API ส่งมาแล้วและ type มีฟิลด์ | Doc ↔ Code | เพิ่ม render ใน `frontend/src/routes/Novel.tsx` หรือแก้ user story |
| H-04 | **PRD ระบุ KPI "THB" แต่ไม่มีในระบบ** — `StatsResponse` ([writer/dto.go:139-145](backend/internal/handler/writer/dto.go#L139-L145)) มีแค่ `coins_earned`; Design ก็แสดง `ราว ฿12,900` | Design + Doc ↔ Code | เพิ่ม `coins_earned_satang` ใน stats DTO + แสดงใน `Stats.tsx` |
| H-05 | **README ระบุ migration ผิดช่วง** — `# applies 0001 … 0006` ทั้งที่มี `0007_monetization.sql` (migration ใหญ่ที่สุดรองจาก init) | Doc ↔ Code | `README.md:112` |

### 🟡 ระดับกลาง

| # | ประเด็น | อยู่ที่ |
| --- | --- | --- |
| M-01 | `docs/database-schema.md` ตาราง "Migration & seed layout" ตกแถว `0007` | Doc |
| M-02 | `docs/database-schema.md` ขัดกันเอง — §Ranking บอกใช้ Redis ZSET แต่ §Open items บอก "Done without Redis" | Doc |
| M-03 | **W-05 glossary CRUD ไม่ครบ** — ไม่มี `DELETE` endpoint และไม่มี UI จัดการอภิธานศัพท์เลย (`api.listWriterGlossary` ประกาศไว้แต่ไม่มีผู้เรียก) | Doc ↔ Code |
| M-04 | **W-10 payout เป็น backend-only** — `GET /writer/earnings` + `POST /writer/payouts` ไม่มี UI; PRD phase 3 ระบุ "Writer workspace + stats + **payouts** — done" | Doc ↔ Code |
| M-05 | **Notification UI ไม่มี** — backend 3 endpoints + `api.ts` client function ครบ แต่ไม่มี component ใดเรียก; README/AGENT/PRD นับ "notifications" เป็นส่วนหนึ่งของ phase 4 ที่ done | Doc ↔ Code (Design ก็ไม่มี) |
| M-06 | หน้า `series` ขาด 3 ส่วนจาก Design: การ์ดเล่มพร้อม progress + รายการภาค, section `เรื่องเกี่ยวเนื่อง`, ปุ่ม `ติดตามทั้งชุด` | Design ↔ Code |
| M-07 | `ติดตามทั้งชุด` ไม่มีทั้ง API และเอกสารรองรับ (follow เป็นราย novel เท่านั้น) | Design ↔ Doc ↔ Code |
| M-08 | หน้าแรกไม่มี `อ่านสะสมสัปดาห์นี้ N ชั่วโมง` และไม่มี metric นี้ทั้งระบบ | Design ↔ Code |
| M-09 | Stats ไม่มี KPI `อ่านจบต่อบท` (completion rate) | Design ↔ Code |
| M-10 | หน้ารายละเอียดเรื่องไม่มีบรรทัดสรุปราคา (`บทที่ 1–48 อ่านฟรี · หลังจากนั้น 5 เหรียญ`) และ API ไม่ส่ง `price_per_chapter` / `free_until_chapter` ออกมา | Design ↔ Code |
| M-11 | `README.md` "Every novel surface now shows two chapter counts" — หน้า Library และการ์ด continue reading แสดงเฉพาะจำนวนที่แปลแล้ว | Doc ↔ Code |
| M-12 | `README.md` ลิสต์ background jobs แค่ 5 จาก 9 | Doc ↔ Code |

### 🟢 ระดับต่ำ (copy / รายละเอียด / เอกสารตกหล่นเล็กน้อย)

| # | ประเด็น |
| --- | --- |
| L-01 | `api-spec.md` ไม่ได้ลิสต์ 4 routes ที่ทำแล้ว: `GET /users/me/genre-prefs`, `GET /me/follows/{novel_id}`, `GET /writer/novels`, `GET /writer/novels/{id}/arcs` |
| L-02 | `GET /search` ไม่อ่านพารามิเตอร์ `type` ที่ยังลิสต์อยู่ในเอกสาร |
| L-03 | `architecture.md` ตัวอย่าง composition root ล้าสมัย (`catalogsvc.New(catalogRepo)` → จริงมี 2 อาร์กิวเมนต์) |
| L-04 | `architecture.md` ตาราง dependency rule ไม่ได้อนุญาต `internal/middleware` ให้ handler ทั้งที่โค้ดใช้จริง |
| L-05 | `database-schema.md` ระบุ "PostgreSQL 15+" ขณะที่ README/AGENT/compose ใช้ 16 |
| L-06 | `database-schema.md` บอก 0002 seed "the featured novel" (เอกพจน์) จริงๆ มี 2 เรื่อง |
| L-07 | `test-cases.md` เขียน "real Postgres/Redis" — ไม่มี Redis |
| L-08 | `user-stories.md` ไม่มีคอลัมน์สถานะ ทำให้ A-01/A-02/A-04/A-05 ดูเหมือนทำแล้ว (ต้องไปอ่าน README/api-spec ถึงจะรู้ว่าไม่ได้ทำ) |
| L-09 | PRD ไม่กล่าวถึง `ความยาวที่ชอบ` ใน onboarding ที่มีทั้งใน Design และโค้ด (โค้ดเก็บเป็น `weight` บน `user_genre_prefs`) |
| L-10 | Copy แม่แบบปกต่างจาก Design (`ลายทแยง`/`ไล่สี`/`สีพื้น` แทน `หมึกจีนแนวตั้ง`/`ลายพู่กัน`/`เรียบ ตัวอักษรอย่างเดียว`) |
| L-11 | ป้ายขั้นตอน wizard ต่างจาก Design (`เลือกผลงาน`/`เลือกบท`/`เขียนและเผยแพร่`) |
| L-12 | ตัวเลือก `รอบปล่อยบทใหม่` เพิ่ม `เดือนละครั้ง` และตัดชื่อวันออก |
| L-13 | Onboarding ไม่บังคับ "อย่างน้อย 3 แนว" ตาม Design (บังคับแค่ ≥ 1) |
| L-14 | ชิปแนวใน onboarding ไม่แสดงอักษรจีนตาม Design |
| L-15 | note popover ในหน้าอ่านไม่มีปุ่ม `ดูในอภิธานศัพท์ →` |
| L-16 | ไม่มี pinch-to-zoom ตามที่ Design เขียนไว้ในคำแนะนำ |
| L-17 | Checkout ขาด: saved-card checkbox, QR countdown + save-QR, โค้ดส่วนลด, บรรทัด VAT/ส่วนลดสมาชิก |
| L-18 | หน้า Coins ขาด `ปลดล็อกได้อีกราว N บท` และ `ใช้ไปเดือนนี้ N เหรียญ` |
| L-19 | ราคา coin pack ใน seed ต่างจาก Design (60/240/600/1500 @ ฿39/149/349/799 vs 100/300/700/1500 @ ฿35/99/219/449) |
| L-20 | หน้า detail: KPI 4 ช่องแทน 5 (ย้าย `ภาค` ไปบรรทัด meta), ลิงก์ glossary/comments เป็นข้อความธรรมดา |
| L-21 | หน้า series ขาด KPI จำนวน `ภาค` |
| L-22 | ปุ่ม `ดูรายได้ที่คาดการณ์` ในแท็บราคาไม่มีในโค้ด |

---

## 6. สิ่งที่ตรวจแล้ว "ตรงกัน" และยืนยันด้วยการรันจริง

รายการเหล่านี้เป็นคำกล่าวอ้างในเอกสารที่ **พิสูจน์ได้** และผ่านการพิสูจน์แล้ว:

| # | คำกล่าวอ้าง | วิธีพิสูจน์ | ผล |
| --- | --- | --- | --- |
| V-01 | "575 passing tests with 0 skips" (`test-cases.md`) | รัน `go test -count=1 -v ./...` พร้อม `DOCKER_HOST` | **575 / 0 / 0** ✅ |
| V-02 | Mapping table 76 ฟังก์ชันเทส | สคริปต์ดึงชื่อจากเอกสารแล้ว grep ในโค้ด | 76/76 พบ ✅ |
| V-03 | "Domain packages framework-free" (`AGENT.md`, `architecture.md`) | `go list -deps` กรอง gorm/gin/entities/repository/handler/service | 0 ละเมิด ✅ |
| V-04 | "Every repository method begins `dbctx.From`" (`AGENT.md`) | สแกน 114 repository methods | 0 ละเมิด ✅ |
| V-05 | "nine jobs" (`architecture.md`) | นับจาก `registry.go` + `Name()` | 9 ✅ |
| V-06 | "Gin wildcard names must match" (`AGENT.md`) | ตรวจ 89 routes + `TestServerNew_RegistersAllRoutesWithoutPanic` | ✅ |
| V-07 | "testcontainers skips rather than fails" (README/AGENT) | รันโดยไม่ตั้ง `DOCKER_HOST` | 196 SKIP, exit 0 ✅ |
| V-08 | `gofmt`/`go vet` สะอาด | รันจริง | ไม่มี output ✅ |
| V-09 | `npm run typecheck` / `npm run build` ผ่าน | รันจริง | ผ่าน ✅ |
| V-10 | Design tokens "lifted from design/" | เทียบค่าสีทีละค่า 3 ธีม | ตรงทุกค่า ✅ |
| V-11 | Reader font/width map ตาม Design | เทียบ `FONTS`/`WIDTHS` ใน design กับ `readerFont`/`readerWidth` | ตรงเป๊ะ ✅ |
| V-12 | "Locked design decisions" 9 ข้อใน PRD | ตรวจกับ migration + domain code | ตรงทุกข้อ ✅ |
| V-13 | "Rules With Teeth" 15 ข้อใน AGENT.md | ตรวจกับโค้ดและเทสที่อ้างถึง | ตรงทุกข้อ ✅ |

---

## 7. ข้อเสนอแนะ

### แก้เอกสาร (เร็ว, ความเสี่ยงต่ำ)

1. `README.md:3` และ `AGENT.md` — ตัด "Redis 7" ออกจากคำอธิบาย stack หรือเปลี่ยนเป็น
   *"Redis 7 (provisioned in docker-compose, not yet consumed by the API)"*
2. `README.md:112` — `# applies 0001 … 0007`
3. `README.md:124` — เติม `auto-unlock`, `session sweep`, `wallet reconcile`, `partition creation` ในรายการ jobs
4. `README.md:13-14` — ปรับถ้อยคำ "Every novel surface" ให้ตรงกับความจริง (หน้า Library แสดงเฉพาะบทที่แปลแล้ว)
5. `docs/database-schema.md` — เติมแถว `0007_monetization.sql` ในตาราง, ลบประโยค Redis ZSET, แก้ "PostgreSQL 15+" → 16, แก้ "the featured novel" → 2 เรื่อง
6. `docs/test-cases.md:6` — "real Postgres" (ตัด Redis)
7. `docs/api-spec.md` — เติม 4 GET routes ที่ตกหล่น, และตัด `type=` ออกจาก `/search` หรือทำให้ handler รองรับ
8. `docs/architecture.md` — อัปเดตตัวอย่าง composition root, เติม `middleware` ในตาราง dependency rule
9. `docs/user-stories.md` — เพิ่มคอลัมน์ Status เพื่อไม่ให้ A-01/02/04/05, W-05, W-10 ดูเหมือนเสร็จแล้ว
10. `docs/prd.md` — เพิ่ม `ความยาวที่ชอบ` ในหัวข้อ Onboarding

### แก้โค้ด (จัดลำดับตามผลกระทบต่อผู้ใช้)

1. **migration ใหม่**: `UPDATE novels SET chapters_count = 88 WHERE slug = 'nine-streams-sword-immortal'`
   — คืนความหมายให้ฟีเจอร์สองตัวเลขบทในข้อมูลตั้งต้น (H-02)
2. **`Novel.tsx`**: render `release_schedule` บนหน้า detail (H-03, ทำให้ W-22 เป็นจริง)
3. **stats**: เพิ่มฟิลด์เงินบาทใน `StatsResponse` และแสดงในการ์ด `เหรียญที่ได้รับ` (H-04)
4. **หน้า series**: เติม section `เรื่องเกี่ยวเนื่อง` (ใช้ `GET /novels/{id}/related` ที่มีอยู่แล้ว) และ KPI จำนวนภาค (M-06)
5. **Notification UI**: มี API + client function ครบแล้ว เหลือแค่ bell + dropdown ใน `Shell.tsx` (M-05)
6. **Writer earnings/payout UI**: API พร้อมแล้ว เหลือหน้าจอ (M-04)
7. **Glossary CRUD UI + DELETE endpoint** เพื่อให้ W-05 เป็นจริง (M-03)

### ไม่แนะนำให้แก้

- **Checkout ที่ต่างจาก Design** — Design วาด flow การชำระเงินจริง แต่ Phase 7 ตั้งใจเป็นเฟสสุดท้าย
  โค้ดปัจจุบันแทนบรรทัด PCI DSS ด้วยคำเตือนว่าเป็นระบบจำลอง ซึ่ง **ถูกต้องกว่า Design** ในบริบทนี้
- **หน้า login/register ที่ไม่มีใน Design** — จำเป็นต่อระบบ, Design ขาดเอง
- **ความต่างของ copy ระดับคำ** (L-10 ถึง L-14, L-20) — ควรให้ผู้ออกแบบตัดสินใจก่อนแก้

---

## 8. สถานะหลังแก้ไข (2026-08-10)

ทุกประเด็นถูกตัดสินใจทีละข้อกับเจ้าของโปรเจกต์ แล้วดำเนินการตาม [Task.md](Task.md)
บน branch `chore/align-design-docs-code`

### 8.1 ผลการตรวจซ้ำ

| ตัวชี้วัด | ก่อนแก้ | หลังแก้ |
| --- | --- | --- |
| Backend tests | 575 PASS / 0 SKIP | **587 PASS / 0 SKIP / 0 FAIL** (+12 เทสใหม่) |
| HTTP routes | 89 | **94** (+2 glossary DELETE, +3 series follow) |
| Migrations | 0001–0007 | **0001–0009** |
| `gofmt` / `go vet` | ผ่าน | ผ่าน |
| `npm run typecheck` / `build` | ผ่าน | ผ่าน |
| Redis ในโค้ด/compose/เอกสาร | ประกาศแต่ไม่ใช้ | **ถอดออกหมด** (เหลือเฉพาะ comment เชิงเหตุผล) |

### 8.2 ระดับสูง 🔴 — แก้ครบ

| # | ประเด็นเดิม | ทำอะไร |
| --- | --- | --- |
| H-01 | Redis ผี | ลบ service `redis` + `REDIS_URL` ออกจาก compose/config/.env.example และแก้เอกสาร 5 จุด |
| H-02 | seed 214/214 | `0008_seed_fixes.sql` ตั้ง `chapters_count = 88` และปรับราคาแพ็กเหรียญให้ตรง Design |
| H-03 | W-22 ไม่เป็นจริง | `Novel.tsx` แสดง `รอบปล่อยบทใหม่` แล้ว (label ย้ายไป `format.ts` ใช้ร่วมกับ `Works.tsx`) |
| H-04 | PRD ระบุ THB | **ยึดโค้ด** — ตัด THB ออกจาก PRD พร้อมเหตุผล (อัตราแปลงขึ้นกับแพ็กที่ซื้อ) และให้เงินบาทปรากฏเฉพาะที่จริง คือแท็บ `รายได้` |
| H-05 | README `0001…0006` | แก้เป็น `0001 … 0009` |

### 8.3 ระดับกลาง 🟡 — แก้ครบ

| # | ทำอะไร |
| --- | --- |
| M-01, M-02 | `database-schema.md` เติมแถว 0007/0008/0009, ลบประโยค Redis ZSET, แก้ PG16, แก้ "2 เรื่อง", เติม 3 ตารางในผัง |
| M-03 | เพิ่ม `DELETE /writer/glossary-entries/{id}` + `DELETE /writer/glossary-groups/{id}` (409 `GROUP_NOT_EMPTY`) และแท็บ `อภิธานศัพท์` ใน `จัดการผลงาน` |
| M-04 | แท็บ `รายได้` ใน `สถิติผลงาน` — ยอดที่ถอนได้, ประวัติ, ฟอร์มขอถอน |
| M-05 | `NotificationBell` ใน sidebar + topbar พร้อม unread badge และ deep link ตาม `kind` |
| M-06 | หน้า series ได้ KPI `ภาค`, การ์ดเล่มพร้อมรายการภาคย่อยและ CTA, และ section `เรื่องเกี่ยวเนื่อง` |
| M-07 | `GET/POST/DELETE /series/{id}/follow` แบบ fan-out (none/partial/all) |
| M-09 | `chapter_read_events.completed` → rollup → `completion_rate_pct` → KPI `อ่านจบต่อบท` |
| M-10 | `GET /novels/{id}` ส่ง `price_per_chapter`/`free_until_chapter` และหน้า detail มีบรรทัดสรุปราคา |
| M-11, M-12 | แก้ถ้อยคำ README ทั้งสองจุด |

### 8.4 ระดับต่ำ 🟢

แก้: L-01…L-11, L-13, L-15, L-18, L-19, L-20, L-21

**ตัดสินใจไม่ทำ** (บันทึกไว้ใน `docs/prd.md` §8a Deferred): M-08 (เวลาอ่านสะสม),
L-12 (ตัวเลือกรอบปล่อยบทตาม Design), L-14 (อักษรจีนในชิปแนว), L-16 (pinch-to-zoom),
L-17 (รายละเอียด checkout — รอ Phase 7), L-22 (ปุ่มรายได้คาดการณ์),
และ `ค่าเฉลี่ยหมวด` ที่คู่กับ `อ่านจบต่อบท`

### 8.5 สิ่งที่พบระหว่างแก้ (ไม่อยู่ในรายงานเดิม)

| # | เรื่อง |
| --- | --- |
| 1 | `NovelStats.AvgCompletePct` มีอยู่ในโดเมนแล้วแต่ **ไม่เคยถูกเซ็ตและไม่เคยถูกส่งออก** — เป็นฟิลด์ตายมาตั้งแต่ต้น งาน completion rate จึงเป็นการต่อสายให้ฟิลด์ที่มีอยู่แล้ว ไม่ใช่เพิ่มของใหม่ |
| 2 | comment ในเทส wallet เขียนว่า "the seeded 240-coin pack" ทั้งที่ pack มาจาก `makeme` fixture — แก้ comment ให้ตรงความจริงแล้ว |
| 3 | `chapter_glossary_refs.entry_id` เป็น `ON DELETE CASCADE` อยู่แล้ว การลบ entry จึงไม่ต้องเคลียร์ binding เอง |
| 4 | `.topbar` ถูกซ่อนบน desktop — กระดิ่งแจ้งเตือนจึงต้องมีที่อยู่ใน sidebar ด้วย ไม่ใช่แค่ topbar |

---

## ภาคผนวก — คำสั่งที่ใช้ตรวจสอบซ้ำได้

```bash
# Backend
cd backend
gofmt -l . && go vet ./...
export DOCKER_HOST=unix://$HOME/.rd/docker.sock
export TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -v ./... | grep -c -- '--- PASS'   # 575
go test -count=1 -v ./... | grep -c -- '--- SKIP'   # 0
docker rm -f $(docker ps -aq)                        # เก็บกวาด container

# รายการ route ทั้งหมด (89)
grep -rnE '\.(GET|POST|PUT|PATCH|DELETE)\("' internal/handler --include='*.go' | grep -v _test

# ตรวจ domain purity
go list -deps -f '{{.ImportPath}} {{join .Imports " "}}' ./internal/domain/... \
  | grep -E '^github.com/mokchan' | grep -E 'gorm|gin-gonic|/entities|/repository|/handler|/service'

# Frontend
cd ../frontend && npm run typecheck && npm run build
```
