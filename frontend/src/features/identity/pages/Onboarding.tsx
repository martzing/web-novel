import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { useGenres } from "@/features/catalog";
import { Empty, ErrorNote, Loading } from "@/shared/ui";

import { useAuth } from "../auth";
import { useSaveGenrePrefs } from "../queries";

type Length = "short" | "medium" | "long";

/**
 * How many genres onboarding asks for.
 *
 * Three is the design's number, and it is the point at which the home ranking
 * has enough to personalise with: one pick produces a page that looks like a
 * genre filter rather than a home page.
 */
const MIN_GENRES = 3;

/** R-18 — favourite genres feed the home ranking personalisation. */
export default function Onboarding() {
  const { user } = useAuth();
  const navigate = useNavigate();

  const [step, setStep] = useState(1);
  const [picked, setPicked] = useState<string[]>([]);
  const [length, setLength] = useState<Length | null>(null);

  const genres = useGenres();
  const save = useSaveGenrePrefs(() => navigate("/"));

  const savePicks = () =>
    save.mutate(
      picked.map((id) => ({
        genre_id: id,
        // The preferred length is expressed as a weight bump on the picks
        // rather than a separate field, which the schema has no room for.
        weight: length === "long" ? 3 : length === "medium" ? 2 : 1,
      })),
    );

  if (!user) {
    return (
      <section>
        <h1 className="page-title">ตั้งค่าแนวที่ชอบ</h1>
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อบันทึกแนวที่ชอบ
        </Empty>
      </section>
    );
  }

  return (
    <section style={{ maxWidth: 620 }}>
      <div className="eyebrow">ขั้นที่ {step} จาก 2</div>

      {step === 1 ? (
        <>
          <h1 className="page-title" style={{ marginTop: 8 }}>
            ชอบอ่านแนวไหน
          </h1>
          <div className="muted" style={{ fontSize: 13, marginTop: 8, lineHeight: 1.9 }}>
            เลือกอย่างน้อย {MIN_GENRES} แนว เราจะใช้จัดหน้าแรกให้ตรงกับที่คุณอ่านจริง
            ปรับเปลี่ยนภายหลังได้ตลอด
          </div>

          {genres.isLoading ? (
            <Loading rows={2} />
          ) : (
            <div className="chips" style={{ marginTop: 22 }}>
              {genres.data?.data.map((g) => (
                <button
                  key={g.id}
                  className={`chip${picked.includes(g.id) ? " is-active" : ""}`}
                  onClick={() =>
                    setPicked((current) =>
                      current.includes(g.id)
                        ? current.filter((id) => id !== g.id)
                        : [...current, g.id],
                    )
                  }
                >
                  {g.name_th}
                </button>
              ))}
            </div>
          )}

          <div style={{ display: "flex", gap: 10, marginTop: 30, alignItems: "center" }}>
            <button
              className="btn btn--primary"
              disabled={picked.length < MIN_GENRES}
              onClick={() => setStep(2)}
            >
              ถัดไป
            </button>
            <Link to="/" className="btn btn--ghost">
              ข้ามไปก่อน
            </Link>
            {/* A disabled button with no explanation reads as broken. The
                counter says what is missing. */}
            <span className="muted" style={{ fontSize: 12.5 }}>
              เลือกแล้ว {picked.length} / {MIN_GENRES}
            </span>
          </div>
        </>
      ) : (
        <>
          <h1 className="page-title" style={{ marginTop: 8 }}>
            ความยาวที่ชอบ
          </h1>

          <div className="grid" style={{ marginTop: 22 }}>
            {(
              [
                ["short", "สั้น", "ต่ำกว่า 100 บท"],
                ["medium", "กลาง", "100–400 บท"],
                ["long", "ยาว", "เกิน 400 บท"],
              ] as [Length, string, string][]
            ).map(([key, label, hint]) => (
              <button
                key={key}
                className="card"
                style={{
                  textAlign: "left",
                  borderColor: length === key ? "var(--red)" : undefined,
                }}
                onClick={() => setLength(key)}
              >
                <div className="serif" style={{ fontSize: 17, fontWeight: 600 }}>
                  {label}
                </div>
                <div className="muted" style={{ fontSize: 12.5, marginTop: 4 }}>
                  {hint}
                </div>
              </button>
            ))}
          </div>

          {save.isError && <ErrorNote message={(save.error as Error).message} />}

          <div style={{ display: "flex", gap: 10, marginTop: 30 }}>
            <button className="btn" onClick={() => setStep(1)}>
              ย้อนกลับ
            </button>
            <button
              className="btn btn--primary"
              disabled={save.isPending}
              onClick={savePicks}
            >
              เสร็จสิ้น
            </button>
          </div>
        </>
      )}
    </section>
  );
}
