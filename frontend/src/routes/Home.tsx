import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { greeting, numberTH, percent, relativeTime } from "../lib/format";
import { Empty, Loading, NovelCard, NovelCover } from "../components";

export default function Home() {
  const { user } = useAuth();

  const latest = useQuery({
    queryKey: ["novels", "latest"],
    queryFn: () => api.listNovels({ sort: "latest", limit: 8 }),
  });

  const ranking = useQuery({
    queryKey: ["ranking", "weekly"],
    queryFn: () => api.weeklyRanking(5),
  });

  const shelf = useQuery({
    queryKey: ["shelf", "reading"],
    queryFn: () => api.getShelf("reading"),
    enabled: Boolean(user),
  });

  const continueReading = shelf.data?.data?.[0];
  const featured = ranking.data?.data?.[0];

  return (
    <section>
      <div className="page-head">
        <h1 className="page-title">{greeting()}</h1>
        <div className="muted" style={{ fontSize: 12.5 }}>
          {user ? `ยินดีต้อนรับ ${user.display_name}` : "เข้าสู่ระบบเพื่อบันทึกตำแหน่งการอ่าน"}
        </div>
      </div>

      {user && (
        <>
          <div className="section-head">
            <div className="eyebrow">อ่านค้างไว้</div>
            <Link to="/library" className="muted" style={{ fontSize: 12 }}>
              ดูทั้งหมด →
            </Link>
          </div>

          {shelf.isLoading ? (
            <Loading rows={1} />
          ) : continueReading ? (
            <Link
              to={
                continueReading.last_chapter_id
                  ? `/read/${continueReading.last_chapter_id}`
                  : `/novels/${continueReading.slug}`
              }
              className="card"
              style={{ display: "flex", gap: 22, marginTop: 14, color: "inherit" }}
            >
              <NovelCover
                novel={{
                  cover_url: continueReading.cover_url,
                  title_cn: continueReading.title_cn,
                  cover_style: continueReading.cover_style,
                  cover_color: continueReading.cover_color,
                  cover_text: continueReading.cover_text,
                }}
                width={74}
                height={104}
              />
              <div style={{ minWidth: 0, flex: 1 }}>
                <div className="muted" style={{ fontSize: 12 }}>
                  {continueReading.title_cn}
                </div>
                <div className="serif" style={{ fontSize: 19, fontWeight: 600, marginTop: 3 }}>
                  {continueReading.title_th}
                </div>
                <div className="muted" style={{ fontSize: 12.5, marginTop: 8 }}>
                  {continueReading.last_chapter_no ? (
                    <>
                      บทที่ {numberTH(continueReading.last_chapter_no)} จาก{" "}
                      {numberTH(continueReading.chapters_count)} บทที่แปลแล้ว · อ่านไปแล้ว{" "}
                      {percent(continueReading.pct)}
                      {continueReading.chapters_count - continueReading.last_chapter_no > 0 && (
                        <span style={{ color: "var(--gold)" }}>
                          {" · "}มีบทใหม่{" "}
                          {numberTH(continueReading.chapters_count - continueReading.last_chapter_no)} บท
                        </span>
                      )}
                    </>
                  ) : (
                    "ยังไม่เริ่มอ่าน"
                  )}
                </div>
              </div>
              <div style={{ alignSelf: "center", color: "var(--red)", fontSize: 13, whiteSpace: "nowrap" }}>
                อ่านต่อ →
              </div>
            </Link>
          ) : (
            <Empty>ยังไม่มีเรื่องที่อ่านค้างไว้ ลองเลือกสักเรื่องจากรายการด้านล่าง</Empty>
          )}
        </>
      )}

      {featured && (
        <>
          <div className="section-head">
            <div className="eyebrow">แปลใหม่ประจำสัปดาห์</div>
          </div>
          <Link
            to={`/novels/${featured.slug}`}
            className="card"
            style={{ display: "block", marginTop: 14, color: "inherit" }}
          >
            <div
              className="cover"
              style={{ width: "100%", height: 190, borderRadius: "var(--r2)" }}
            >
              {featured.cover_url ? (
                <img src={featured.cover_url} alt="" />
              ) : (
                <span className="serif muted" style={{ fontSize: 22 }}>
                  {featured.title_cn ?? featured.title_th}
                </span>
              )}
            </div>
            <div style={{ marginTop: 16, display: "flex", gap: 8, flexWrap: "wrap" }}>
              {featured.genres.map((g) => (
                <span key={g.id} className="pill">
                  {g.name_th}
                </span>
              ))}
            </div>
            <h2 style={{ fontSize: 22, marginTop: 10 }}>{featured.title_th}</h2>
            <div className="muted" style={{ fontSize: 12.5, marginTop: 6 }}>
              {numberTH(featured.followers_count)} ผู้ติดตาม · {numberTH(featured.chapters_count)} บท
            </div>
          </Link>
        </>
      )}

      <div className="section-head">
        <div className="eyebrow">อันดับสัปดาห์นี้</div>
      </div>
      {ranking.isLoading ? (
        <Loading />
      ) : ranking.data && ranking.data.data.length > 0 ? (
        <div className="rows" style={{ marginTop: 14 }}>
          {ranking.data.data.map((novel) => (
            <Link key={novel.id} to={`/novels/${novel.slug}`} className="row">
              <span className="mono" style={{ color: "var(--gold)", minWidth: 26, fontSize: 15 }}>
                {novel.rank}
              </span>
              <span style={{ flex: 1, minWidth: 0 }}>
                <span className="serif" style={{ fontSize: 15, fontWeight: 600 }}>
                  {novel.title_th}
                </span>
                <span className="muted" style={{ display: "block", fontSize: 12, marginTop: 2 }}>
                  {novel.genres.map((g) => g.name_th).join(" · ")} · {numberTH(novel.chapters_count)} บท
                </span>
              </span>
              <span className="mono muted" style={{ fontSize: 13 }}>
                {novel.rating_avg.toFixed(1)}
              </span>
            </Link>
          ))}
        </div>
      ) : (
        <Empty>ยังไม่มีข้อมูลอันดับ</Empty>
      )}

      <div className="section-head">
        <div className="eyebrow">อัปเดตล่าสุด</div>
        <Link to="/browse" className="muted" style={{ fontSize: 12 }}>
          ดูทั้งหมด →
        </Link>
      </div>
      {latest.isLoading ? (
        <Loading />
      ) : latest.data && latest.data.data.length > 0 ? (
        <div className="grid grid--cards" style={{ marginTop: 14 }}>
          {latest.data.data.map((novel) => (
            <NovelCard key={novel.id} novel={novel} />
          ))}
        </div>
      ) : (
        <Empty>ยังไม่มีนิยายในระบบ</Empty>
      )}

      {latest.data?.data?.[0]?.slug && (
        <div className="muted" style={{ fontSize: 11.5, marginTop: 26 }}>
          อัปเดตล่าสุด {relativeTime(new Date().toISOString())}
        </div>
      )}
    </section>
  );
}
