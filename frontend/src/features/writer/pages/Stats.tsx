import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { useAuth } from "@/features/identity";
import { baht, numberTH, thaiDate, trend } from "@/shared/lib/format";
import { Empty, ErrorNote, Loading, Tabs } from "@/shared/ui";

import { writerApi } from "../api";
import { useEarnings, useWriterNovels, useWriterStats, writerKeys } from "../queries";

type Period = "14d" | "30d" | "all";
type View = "stats" | "earnings";

export default function Stats() {
  const { user, isTranslator } = useAuth();
  const [novelId, setNovelId] = useState("");
  const [period, setPeriod] = useState<Period>("14d");
  // Earnings live here as a second view rather than a fourth sidebar entry:
  // the design fixes the writer navigation at three items, and money is
  // something a translator checks while looking at performance anyway.
  const [view, setView] = useState<View>("stats");

  const novels = useWriterNovels(isTranslator);

  useEffect(() => {
    if (!novelId && novels.data?.data.length) setNovelId(novels.data.data[0].id);
  }, [novels.data, novelId]);

  const stats = useWriterStats(novelId, period);

  if (!user || !isTranslator) {
    return (
      <section>
        <h1 className="page-title">สถิติผลงาน</h1>
        <Empty>
          {user ? (
            "บัญชีนี้ยังไม่มีสิทธิ์นักแปล"
          ) : (
            <>
              <Link to="/login">เข้าสู่ระบบ</Link> ด้วยบัญชีนักแปลเพื่อดูสถิติ
            </>
          )}
        </Empty>
      </section>
    );
  }

  const novel = novels.data?.data.find((n) => n.id === novelId);
  const s = stats.data;
  const peak = Math.max(1, ...(s?.series.map((p) => p.reads) ?? [1]));

  return (
    <section>
      <div className="page-head">
        <h1 className="page-title">สถิติผลงาน</h1>
        {novel && (
          <div className="muted" style={{ fontSize: 12.5 }}>
            {novel.title_th}
          </div>
        )}
      </div>

      <div style={{ marginTop: 20 }}>
        <Tabs<View>
          active={view}
          onChange={setView}
          tabs={[
            { key: "stats", label: "ผลงาน" },
            { key: "earnings", label: "รายได้" },
          ]}
        />
      </div>

      {view === "earnings" ? (
        <Earnings />
      ) : (
        <>
          {(novels.data?.data.length ?? 0) > 1 && (
            <select
              className="select"
              style={{ marginTop: 18, maxWidth: 320 }}
              value={novelId}
              onChange={(e) => setNovelId(e.target.value)}
            >
              {novels.data!.data.map((n) => (
                <option key={n.id} value={n.id}>
                  {n.title_th}
                </option>
              ))}
            </select>
          )}

          <div style={{ marginTop: 20 }}>
            <Tabs<Period>
              active={period}
              onChange={setPeriod}
              tabs={[
                { key: "14d", label: "14 วันล่าสุด" },
                { key: "30d", label: "30 วัน" },
                { key: "all", label: "ทั้งหมด" },
              ]}
            />
          </div>

          {stats.isError ? (
        <ErrorNote message={(stats.error as Error).message} />
      ) : stats.isLoading || !s ? (
        <Loading rows={2} />
      ) : (
        <>
          <div className="grid grid--kpi" style={{ marginTop: 22 }}>
            <Kpi label="ยอดอ่าน" value={numberTH(s.reads)} delta={trend(s.reads_trend_pct)} />
            <Kpi label="ผู้ติดตามใหม่" value={numberTH(s.followers)} />
            <Kpi
              label="เหรียญที่ได้รับ"
              value={numberTH(s.coins_earned)}
              delta={trend(s.coins_trend_pct)}
            />
            {/* The window itself is already in the tabs above, so this tile
                carries อ่านจบต่อบท instead: how many opens actually reached
                the end, which is the number that tells a translator whether a
                chapter landed. */}
            <Kpi
              label="อ่านจบต่อบท"
              value={`${s.completion_rate_pct.toFixed(0)}%`}
              sub={`${thaiDate(s.period_from)} – ${thaiDate(s.period_to)}`}
            />
          </div>

          <div className="section-head">
            <div className="eyebrow">ยอดอ่านรายวัน</div>
          </div>

          {s.series.length === 0 ? (
            <Empty>ยังไม่มีข้อมูลในช่วงนี้</Empty>
          ) : (
            <div className="card scroll-x" style={{ marginTop: 14 }}>
              <div
                style={{
                  display: "flex",
                  alignItems: "flex-end",
                  gap: 6,
                  height: 150,
                  minWidth: s.series.length * 22,
                }}
                role="img"
                aria-label={`ยอดอ่านรายวัน สูงสุด ${numberTH(peak)} ครั้ง`}
              >
                {s.series.map((point) => (
                  <div
                    key={point.day}
                    title={`${thaiDate(point.day)} · ${numberTH(point.reads)} ครั้ง`}
                    style={{
                      flex: 1,
                      minWidth: 14,
                      height: `${Math.max(2, (point.reads / peak) * 100)}%`,
                      background: "var(--red)",
                      opacity: 0.75,
                      borderRadius: "2px 2px 0 0",
                    }}
                  />
                ))}
              </div>
              <div className="muted" style={{ fontSize: 11.5, marginTop: 10 }}>
                สูงสุด {numberTH(peak)} ครั้งต่อวัน
              </div>
            </div>
          )}

          <div className="section-head">
            <div className="eyebrow">บทที่ทำผลงานดีที่สุด</div>
          </div>

          {s.top_chapters.length === 0 ? (
            <Empty>ยังไม่มีข้อมูลรายบท</Empty>
          ) : (
            <div className="card scroll-x" style={{ marginTop: 14, padding: 0 }}>
              <table className="table">
                <thead>
                  <tr>
                    <th>บท</th>
                    <th>ชื่อบท</th>
                    <th>ยอดอ่าน</th>
                    <th>เหรียญ</th>
                  </tr>
                </thead>
                <tbody>
                  {s.top_chapters.map((c) => (
                    <tr key={c.chapter_id}>
                      <td className="mono">{c.chapter_no}</td>
                      <td>{c.title}</td>
                      <td className="mono">{numberTH(c.reads)}</td>
                      <td className="mono">{numberTH(c.coins_earned)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
            </>
          )}
        </>
      )}
    </section>
  );
}

/**
 * รายได้และการถอนเงิน (W-10).
 *
 * `GET /writer/earnings` and `POST /writer/payouts` have existed since phase 3
 * with tests behind them, but nothing in the app ever called them — a
 * translator could earn and never see it. The design never drew this screen, so
 * it reuses the card and table treatment from the stats view above.
 *
 * `available_satang` is the authority on what can be withdrawn: it is the net
 * earnings minus payouts already requested, so a second request cannot spend
 * money the first one has claimed.
 */
function Earnings() {
  const qc = useQueryClient();
  const [amount, setAmount] = useState("");

  const earnings = useEarnings();

  const payout = useMutation({
    mutationFn: () => writerApi.requestPayout(Math.round(Number(amount) * 100)),
    onSuccess: () => {
      setAmount("");
      qc.invalidateQueries({ queryKey: writerKeys.earnings() });
    },
  });

  if (earnings.isLoading) return <Loading rows={3} />;
  if (earnings.isError) return <ErrorNote message={(earnings.error as Error).message} />;

  const available = earnings.data?.available_satang ?? 0;
  const rows = earnings.data?.data ?? [];
  const requested = Math.round(Number(amount) * 100);
  const canRequest = requested > 0 && requested <= available && !payout.isPending;

  return (
    <>
      <div className="card" style={{ marginTop: 22 }}>
        <div className="eyebrow">ยอดที่ถอนได้</div>
        <div className="serif" style={{ fontSize: 36, fontWeight: 600, marginTop: 6 }}>
          {baht(available)}
        </div>
        <div className="muted" style={{ fontSize: 12.5, marginTop: 8, lineHeight: 1.9 }}>
          หักคำขอถอนที่ยังรอดำเนินการแล้ว · ยอดนี้มาจากส่วนแบ่งสุทธิหลังหักค่าธรรมเนียมแพลตฟอร์ม
        </div>

        <div className="payout-form">
          <label className="field" style={{ flex: "1 1 200px" }}>
            <span className="field__label">จำนวนที่ต้องการถอน (บาท)</span>
            <input
              className="input"
              inputMode="decimal"
              value={amount}
              placeholder="0.00"
              onChange={(e) => setAmount(e.target.value)}
            />
          </label>
          <button
            className="btn btn--primary"
            disabled={!canRequest}
            onClick={() => payout.mutate()}
          >
            {payout.isPending ? "กำลังส่งคำขอ…" : "ขอถอนเงิน"}
          </button>
        </div>

        {requested > available && amount !== "" && (
          <div className="muted" style={{ fontSize: 12, marginTop: 8, color: "var(--red)" }}>
            ยอดที่ขอถอนเกินยอดที่ถอนได้
          </div>
        )}
        {payout.isError && <ErrorNote message={(payout.error as Error).message} />}
        {payout.isSuccess && (
          <div className="muted" style={{ fontSize: 12.5, marginTop: 10 }}>
            ส่งคำขอแล้ว · รอผู้ดูแลอนุมัติ
          </div>
        )}
      </div>

      <div className="section-head">
        <div className="eyebrow">รายการที่ได้รับ</div>
      </div>

      {rows.length === 0 ? (
        <Empty>ยังไม่มีรายได้ · รายได้จะปรากฏเมื่อมีผู้อ่านปลดล็อกหรือให้ทิป</Empty>
      ) : (
        <div className="card scroll-x" style={{ marginTop: 14, padding: 0 }}>
          <table className="table">
            <thead>
              <tr>
                <th>เมื่อ</th>
                <th>เหรียญที่ผู้อ่านจ่าย</th>
                <th>ส่วนแบ่งสุทธิ</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <td>{thaiDate(row.created_at)}</td>
                  <td className="mono">{numberTH(row.gross_coins)}</td>
                  <td className="mono">{numberTH(row.net_coins)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

function Kpi({
  label,
  value,
  delta,
  sub,
}: {
  label: string;
  value: string;
  delta?: string;
  sub?: string;
}) {
  const positive = delta?.startsWith("+");
  return (
    <div className="card">
      <div className="eyebrow">{label}</div>
      <div className="serif" style={{ fontSize: 28, fontWeight: 600, marginTop: 8 }}>
        {value}
      </div>
      {delta && (
        <div
          style={{ fontSize: 12.5, marginTop: 6, color: positive ? "var(--gold)" : "var(--red)" }}
        >
          {delta}
        </div>
      )}
      {sub && (
        <div className="muted" style={{ fontSize: 11.5, marginTop: 6 }}>
          {sub}
        </div>
      )}
    </div>
  );
}
