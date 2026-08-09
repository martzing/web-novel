import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { api, type SeriesBook } from "../lib/api";
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
      </div>

      {s.description && (
        <p style={{ marginTop: 16, fontSize: 14, lineHeight: 2, maxWidth: 720 }}>{s.description}</p>
      )}

      <div className="grid grid--kpi" style={{ marginTop: 24 }}>
        <SeriesStat value={numberTH(s.books.length)} label="เล่มในชุด" />
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
    </section>
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
      </div>
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
