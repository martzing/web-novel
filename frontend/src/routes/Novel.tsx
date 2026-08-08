import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, type Arc, type ChapterListItem } from "../lib/api";
import { useAuth } from "../lib/auth";
import { numberTH, relativeTime, thaiDate } from "../lib/format";
import { Cover, Empty, ErrorNote, Loading, StarPicker, Stars } from "../components";

export default function Novel() {
  const { slug = "" } = useParams();
  const { user } = useAuth();
  const qc = useQueryClient();

  const novel = useQuery({ queryKey: ["novel", slug], queryFn: () => api.getNovel(slug) });
  const novelId = novel.data?.id;

  const chapters = useQuery({
    queryKey: ["chapters", novelId],
    queryFn: () => api.listChapters(novelId!),
    enabled: Boolean(novelId),
  });

  const following = useQuery({
    queryKey: ["following", novelId],
    queryFn: () => api.isFollowing(novelId!),
    enabled: Boolean(novelId && user),
  });

  const shelf = useQuery({
    queryKey: ["shelf", "counts"],
    queryFn: () => api.getShelf(),
    enabled: Boolean(user),
  });

  const onShelf = shelf.data?.data.some((item) => item.novel_id === novelId) ?? false;

  const toggleShelf = useMutation({
    mutationFn: async () => {
      if (onShelf) {
        await api.removeFromShelf(novelId!);
        return;
      }
      await api.setShelfStatus(novelId!, "reading");
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["shelf"] }),
  });

  const toggleFollow = useMutation({
    mutationFn: () => (following.data?.following ? api.unfollow(novelId!) : api.follow(novelId!)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["following", novelId] });
      qc.invalidateQueries({ queryKey: ["novel", slug] });
    },
  });

  if (novel.isLoading) return <Loading rows={5} />;
  if (novel.isError) return <ErrorNote message={(novel.error as Error).message} />;
  if (!novel.data) return <Empty>ไม่พบนิยายเรื่องนี้</Empty>;

  const n = novel.data;
  const rows = chapters.data?.data ?? [];
  const firstUnread = rows.find((c) => c.unlocked) ?? rows[0];

  return (
    <section>
      <Link to="/browse" className="muted" style={{ fontSize: 12.5 }}>
        ← กลับ
      </Link>

      <div style={{ display: "flex", gap: 26, marginTop: 20, flexWrap: "wrap" }}>
        <Cover url={n.cover_url} titleCN={n.title_cn} width={168} height={236} />

        <div style={{ flex: "1 1 320px", minWidth: 0 }}>
          <div className="muted" style={{ fontSize: 12.5 }}>
            {n.genres.map((g) => g.name_th).join(" · ")}
            {n.title_cn && ` · แปลจาก ${n.title_cn}`}
          </div>
          <h1 className="page-title" style={{ marginTop: 8 }}>
            {n.title_th}
          </h1>
          <div className="muted" style={{ fontSize: 13, marginTop: 8 }}>
            {n.author_name ? `ผู้แต่ง ${n.author_name}` : "ไม่ระบุผู้แต่ง"}
          </div>

          <div style={{ display: "flex", gap: 10, marginTop: 20, flexWrap: "wrap" }}>
            {firstUnread && (
              <Link to={`/read/${firstUnread.id}`} className="btn btn--primary">
                อ่านต่อบทที่ {firstUnread.chapter_no}
              </Link>
            )}
            {user && (
              <>
                <button
                  className="btn"
                  onClick={() => toggleShelf.mutate()}
                  disabled={toggleShelf.isPending}
                >
                  {onShelf ? "อยู่ในชั้นหนังสือแล้ว" : "เพิ่มลงชั้นหนังสือ"}
                </button>
                <button
                  className="btn"
                  onClick={() => toggleFollow.mutate()}
                  disabled={toggleFollow.isPending}
                >
                  {following.data?.following ? "ติดตามอยู่" : "ติดตาม"}
                </button>
              </>
            )}
          </div>

          <div className="grid grid--kpi" style={{ marginTop: 26 }}>
            <Stat value={n.rating_avg.toFixed(1)} label={`${numberTH(n.rating_count)} รีวิว`}>
              <Stars rating={n.rating_avg} />
            </Stat>
            <Stat value={numberTH(n.chapters_count)} label="บททั้งหมด" />
            <Stat value={String(n.arcs.length)} label="ภาค" />
            <Stat value={numberTH(n.followers_count)} label="ผู้ติดตาม" />
          </div>

          <div className="muted" style={{ fontSize: 12.5, marginTop: 18, display: "flex", gap: 16 }}>
            <span>อภิธานศัพท์ {numberTH(n.glossary_count)} คำ</span>
            <span>ความเห็น {numberTH(n.comments_count)}</span>
          </div>
        </div>
      </div>

      {n.description && (
        <p style={{ marginTop: 26, fontSize: 14, lineHeight: 2, maxWidth: 720 }}>{n.description}</p>
      )}

      <div className="section-head">
        <div className="eyebrow">สารบัญ</div>
        <span className="muted" style={{ fontSize: 12 }}>
          {numberTH(rows.length)} บทที่เผยแพร่แล้ว
        </span>
      </div>

      {chapters.isLoading ? (
        <Loading />
      ) : rows.length === 0 ? (
        <Empty>ยังไม่มีบทที่เผยแพร่</Empty>
      ) : (
        <TableOfContents arcs={n.arcs} chapters={rows} />
      )}

      <Reviews novelId={n.id} />
    </section>
  );
}

function Stat({ value, label, children }: { value: string; label: string; children?: React.ReactNode }) {
  return (
    <div>
      <div className="serif" style={{ fontSize: 22, fontWeight: 600 }}>
        {value}
      </div>
      {children}
      <div className="muted" style={{ fontSize: 11.5, marginTop: 3 }}>
        {label}
      </div>
    </div>
  );
}

/** Groups chapters under their arc, matching the mockup's ToC. */
function TableOfContents({ arcs, chapters }: { arcs: Arc[]; chapters: ChapterListItem[] }) {
  const grouped = arcs
    .map((arc) => ({
      arc,
      items: chapters.filter(
        (c) => c.chapter_no >= arc.from_chapter_no && c.chapter_no <= arc.to_chapter_no,
      ),
    }))
    .filter((g) => g.items.length > 0);

  // Chapters outside every arc still need a home.
  const grouptedIDs = new Set(grouped.flatMap((g) => g.items.map((c) => c.id)));
  const ungrouped = chapters.filter((c) => !grouptedIDs.has(c.id));

  return (
    <div style={{ marginTop: 14 }}>
      {grouped.map(({ arc, items }) => (
        <div key={arc.id} style={{ marginBottom: 26 }}>
          <div className="muted" style={{ fontSize: 12.5, marginBottom: 8 }}>
            ภาคที่ {arc.arc_no} · {arc.name}
            <span className="mono"> ({arc.from_chapter_no}–{arc.to_chapter_no})</span>
          </div>
          <ChapterRows items={items} />
        </div>
      ))}

      {ungrouped.length > 0 && (
        <div>
          {grouped.length > 0 && (
            <div className="muted" style={{ fontSize: 12.5, marginBottom: 8 }}>
              บทอื่น ๆ
            </div>
          )}
          <ChapterRows items={ungrouped} />
        </div>
      )}
    </div>
  );
}

function ChapterRows({ items }: { items: ChapterListItem[] }) {
  return (
    <div className="rows">
      {items.map((c) => (
        <Link key={c.id} to={`/read/${c.id}`} className="row">
          <span className="mono muted" style={{ minWidth: 44, fontSize: 12.5 }}>
            {c.chapter_no}
          </span>
          <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{c.title}</span>
          <span className="muted" style={{ fontSize: 11.5, whiteSpace: "nowrap" }}>
            {thaiDate(c.published_at)}
          </span>
          <span style={{ whiteSpace: "nowrap" }}>
            {c.price_coins === 0 ? (
              <span className="pill">อ่านฟรี</span>
            ) : c.unlocked ? (
              <span className="pill pill--gold">ปลดล็อกแล้ว</span>
            ) : (
              <span className="pill pill--red">◎ {c.price_coins}</span>
            )}
          </span>
        </Link>
      ))}
    </div>
  );
}

function Reviews({ novelId }: { novelId: string }) {
  const { user } = useAuth();
  const qc = useQueryClient();
  const [rating, setRating] = useState(0);
  const [body, setBody] = useState("");

  const reviews = useQuery({
    queryKey: ["reviews", novelId],
    queryFn: () => api.listReviews(novelId),
  });

  const submit = useMutation({
    mutationFn: () => api.upsertReview(novelId, { rating, body }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["reviews", novelId] });
      qc.invalidateQueries({ queryKey: ["novel"] });
      setBody("");
    },
  });

  const mine = reviews.data?.my_review;

  return (
    <>
      <div className="section-head">
        <div className="eyebrow">รีวิว</div>
      </div>

      {user ? (
        <div className="card" style={{ marginTop: 14 }}>
          <div className="muted" style={{ fontSize: 12.5 }}>
            {mine ? "แก้ไขรีวิวของคุณ" : "ให้คะแนนเรื่องนี้"}
          </div>
          <div style={{ marginTop: 8 }}>
            <StarPicker value={rating || mine?.rating || 0} onChange={setRating} />
          </div>
          <textarea
            className="textarea"
            style={{ marginTop: 12, minHeight: 90 }}
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder={mine?.body ?? "เขียนความรู้สึกสั้น ๆ (ไม่บังคับ)"}
          />
          {submit.isError && <div className="form-error">{(submit.error as Error).message}</div>}
          <button
            className="btn btn--primary"
            style={{ marginTop: 12 }}
            disabled={(rating || mine?.rating || 0) === 0 || submit.isPending}
            onClick={() => submit.mutate()}
          >
            {mine ? "บันทึกรีวิว" : "ส่งรีวิว"}
          </button>
        </div>
      ) : (
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อให้คะแนนและเขียนรีวิว
        </Empty>
      )}

      {reviews.data && reviews.data.data.length > 0 && (
        <div className="grid" style={{ marginTop: 16 }}>
          {reviews.data.data.map((r) => (
            <div key={r.id} className="card">
              <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                <div style={{ fontSize: 13.5 }}>{r.author.display_name}</div>
                <Stars rating={r.rating} />
              </div>
              {r.body && (
                <p style={{ fontSize: 13.5, lineHeight: 1.95, marginTop: 8, marginBottom: 0 }}>
                  {r.body}
                </p>
              )}
              <div className="muted" style={{ fontSize: 11.5, marginTop: 8 }}>
                {relativeTime(r.created_at)}
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  );
}
