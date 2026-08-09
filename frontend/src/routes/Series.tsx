import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, type RelatedNovel, type SeriesBook } from "../lib/api";
import { useAuth } from "../lib/auth";
import { numberTH } from "../lib/format";
import { Empty, ErrorNote, Loading, NovelCover } from "../components";

/**
 * The public ชุดหนังสือ page.
 *
 * A series is a reading order first and a list second: the translator's note on
 * each book ("อ่านเล่มนี้ก่อนได้ ไม่สปอยล์") is the reason this page exists
 * rather than a genre filter, so the notes lead and the metadata follows.
 */
export default function Series() {
  const { id = "" } = useParams();

  const series = useQuery({ queryKey: ["series", id], queryFn: () => api.getSeries(id) });

  if (series.isLoading) return <Loading rows={4} />;
  if (series.isError) return <ErrorNote message={(series.error as Error).message} />;
  if (!series.data) return <Empty>ไม่พบชุดหนังสือนี้</Empty>;

  const s = series.data;

  return (
    <section>
      <Link to="/browse" className="muted" style={{ fontSize: 12.5 }}>
        ← กลับ
      </Link>

      <div className="page-head" style={{ marginTop: 20 }}>
        <div>
          <div className="eyebrow">ชุดหนังสือ</div>
          <h1 className="page-title" style={{ marginTop: 6 }}>
            {s.title}
          </h1>
        </div>
        <FollowSeries seriesId={id} />
      </div>

      {s.description && (
        <p style={{ marginTop: 16, fontSize: 14, lineHeight: 2, maxWidth: 720 }}>{s.description}</p>
      )}

      <div className="grid grid--kpi" style={{ marginTop: 24 }}>
        <SeriesStat value={numberTH(s.books.length)} label="เล่มในชุด" />
        <SeriesStat value={numberTH(s.arcs_count)} label="ภาค" />
        <SeriesStat value={numberTH(s.chapters_count)} label="บทที่แปลแล้ว" />
        {s.source_chapters_count > 0 && (
          <SeriesStat value={numberTH(s.source_chapters_count)} label="บทในต้นฉบับ" />
        )}
      </div>

      <div className="section-head">
        <div className="eyebrow">ลำดับการอ่าน</div>
        <span className="muted" style={{ fontSize: 12 }}>
          เรียงตามที่ผู้แปลแนะนำ
        </span>
      </div>

      {s.books.length === 0 ? (
        <Empty>ยังไม่มีเล่มในชุดนี้</Empty>
      ) : (
        <ol className="reading-order">
          {s.books.map((book, index) => (
            <SeriesBookCard key={book.id} book={book} order={index + 1} />
          ))}
        </ol>
      )}

      {s.books.length > 0 && <SeriesRelated novelId={s.books[0].id} />}
    </section>
  );
}

/**
 * ติดตามทั้งชุด.
 *
 * The state is three-valued because following a series is a fan-out over the
 * per-novel follows, not a row of its own: a reader can hold some books and not
 * others, and a book that joins later does not follow itself. "ติดตามบางเล่ม"
 * is therefore a real state the button has to be able to say.
 */
function FollowSeries({ seriesId }: { seriesId: string }) {
  const { user } = useAuth();
  const qc = useQueryClient();

  const state = useQuery({
    queryKey: ["series-follow", seriesId],
    queryFn: () => api.seriesFollowState(seriesId),
    enabled: Boolean(user),
  });

  const toggle = useMutation({
    mutationFn: () =>
      state.data?.state === "all" ? api.unfollowSeries(seriesId) : api.followSeries(seriesId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["series-follow", seriesId] });
      // Per-novel follow badges elsewhere are now stale.
      qc.invalidateQueries({ queryKey: ["following"] });
    },
  });

  if (!user) return null;

  const current = state.data?.state ?? "none";
  const label =
    current === "all"
      ? "ติดตามอยู่ทั้งชุด"
      : current === "partial"
        ? `ติดตามที่เหลือ (${numberTH(state.data?.following ?? 0)}/${numberTH(state.data?.total ?? 0)})`
        : "ติดตามทั้งชุด";

  return (
    <div>
      <button
        className={`btn${current === "all" ? "" : " btn--primary"}`}
        onClick={() => toggle.mutate()}
        disabled={toggle.isPending || state.isLoading}
      >
        {label}
      </button>
      {toggle.isError && <ErrorNote message={(toggle.error as Error).message} />}
    </div>
  );
}

/** เรื่องเกี่ยวเนื่อง, read from the series' first book and grouped by kind. */
function SeriesRelated({ novelId }: { novelId: string }) {
  const related = useQuery({
    queryKey: ["related", novelId],
    queryFn: () => api.listRelated(novelId),
  });

  const items = related.data?.data ?? [];
  if (items.length === 0) return null;

  const groups = new Map<string, RelatedNovel[]>();
  for (const item of items) {
    const bucket = groups.get(item.kind_label) ?? [];
    bucket.push(item);
    groups.set(item.kind_label, bucket);
  }

  return (
    <>
      <div className="section-head">
        <div className="eyebrow">เรื่องเกี่ยวเนื่อง</div>
      </div>
      {[...groups.entries()].map(([label, novels]) => (
        <div key={label} style={{ marginTop: 14 }}>
          <div className="muted" style={{ fontSize: 12 }}>
            {label}
          </div>
          <div className="grid grid--cards" style={{ marginTop: 8 }}>
            {novels.map((novel) => (
              <Link
                key={novel.id}
                to={`/novels/${novel.slug}`}
                className="card"
                style={{ display: "flex", gap: 14, color: "inherit" }}
              >
                <NovelCover novel={novel} width={54} height={76} />
                <div style={{ minWidth: 0 }}>
                  <div className="serif" style={{ fontSize: 15, fontWeight: 600 }}>
                    {novel.title_th}
                  </div>
                  <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
                    {numberTH(novel.chapters_count)} บทที่แปลแล้ว
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </div>
      ))}
    </>
  );
}

function SeriesStat({ value, label }: { value: string; label: string }) {
  return (
    <div>
      <div className="serif" style={{ fontSize: 22, fontWeight: 600 }}>
        {value}
      </div>
      <div className="muted" style={{ fontSize: 11.5, marginTop: 3 }}>
        {label}
      </div>
    </div>
  );
}

function SeriesBookCard({ book, order }: { book: SeriesBook; order: number }) {
  return (
    <li className="card reading-order__item">
      <div className="reading-order__no mono" aria-hidden="true">
        {order}
      </div>

      <Link to={`/novels/${book.slug}`} className="reading-order__cover">
        <NovelCover novel={book} width={62} height={88} />
      </Link>

      <div style={{ minWidth: 0, flex: 1 }}>
        <Link to={`/novels/${book.slug}`} className="serif reading-order__title">
          {book.title_th}
        </Link>
        <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
          {book.genres.map((g) => g.name_th).join(" · ") || "—"}
        </div>
        <div className="muted" style={{ fontSize: 12, marginTop: 6 }}>
          {chapterProgress(book)}
        </div>
        {book.note && <p className="reading-order__note">{book.note}</p>}

        {book.arcs.length > 0 && (
          <ul className="arc-list">
            {book.arcs.map((arc) => (
              <li key={arc.id} className="arc-list__row">
                <span className="mono muted">ภาคที่ {arc.arc_no}</span>
                <span className="arc-list__name">{arc.name}</span>
                <span className="mono muted">
                  {numberTH(arc.from_chapter_no)}–{numberTH(arc.to_chapter_no)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>

      <Link to={`/novels/${book.slug}`} className="btn btn--accent reading-order__cta">
        รายละเอียดและสารบัญ
      </Link>
    </li>
  );
}

/**
 * "แปลแล้ว 87 จาก 412 บท" where the translator has entered a source length,
 * and just the translated count where they have not — "จาก 0" would be worse
 * than saying nothing.
 */
function chapterProgress(book: SeriesBook): string {
  const translated = numberTH(book.chapters_count);
  if (book.source_chapters_count > 0) {
    return `แปลแล้ว ${translated} จาก ${numberTH(book.source_chapters_count)} บท`;
  }
  return `แปลแล้ว ${translated} บท`;
}
