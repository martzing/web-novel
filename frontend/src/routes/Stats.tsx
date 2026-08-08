import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { numberTH, thaiDate, trend } from "../lib/format";
import { Empty, ErrorNote, Loading, Tabs } from "../components";

type Period = "14d" | "30d" | "all";

export default function Stats() {
  const { user, isTranslator } = useAuth();
  const [novelId, setNovelId] = useState("");
  const [period, setPeriod] = useState<Period>("14d");

  const novels = useQuery({
    queryKey: ["writer", "novels"],
    queryFn: api.listWriterNovels,
    enabled: isTranslator,
  });

  useEffect(() => {
    if (!novelId && novels.data?.data.length) setNovelId(novels.data.data[0].id);
  }, [novels.data, novelId]);

  const stats = useQuery({
    queryKey: ["writer", "stats", novelId, period],
    queryFn: () => api.getStats(novelId, period),
    enabled: Boolean(novelId),
  });

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
            <Kpi
              label="ช่วงเวลา"
              value={`${thaiDate(s.period_from)}`}
              sub={`ถึง ${thaiDate(s.period_to)}`}
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
    </section>
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
