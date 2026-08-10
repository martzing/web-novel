import { useState } from "react";
import { Link } from "react-router-dom";

import { useGenres } from "@/features/catalog";
import type { Genre } from "@/features/catalog";

import type { WriterNovel } from "../../api";
import { useSaveNovel } from "../../queries";
import { SaveRow } from "../SaveRow";

// ── Tab 1 · ข้อมูลเรื่อง ────────────────────────────────────────────────────

const MAX_GENRES = 3;

export function InfoTab({ novel }: { novel: WriterNovel }) {
  const save = useSaveNovel(novel.id);
  const genres = useGenres();

  const [titleTH, setTitleTH] = useState(novel.title_th);
  const [titleCN, setTitleCN] = useState(novel.title_cn ?? "");
  const [author, setAuthor] = useState(novel.author_name ?? "");
  const [description, setDescription] = useState(novel.description ?? "");
  const [sourceCount, setSourceCount] = useState(novel.source_chapters_count);

  // Genre ids are numbers on the wire while `Genre.id` from /genres is a
  // string, so every id crossing this boundary is converted once, here. Mixing
  // the two silently breaks `includes`: 1 and "1" are different keys, so a chip
  // would refuse to deselect and could be added twice.
  const [genreIds, setGenreIds] = useState<number[]>(novel.genre_ids);

  // Genres are only sent when the translator actually changed them. The patch
  // treats an absent key as "leave alone" and a present array as the complete
  // new set, so sending the current state unconditionally would overwrite the
  // novel's genres with whatever this form happened to be seeded with.
  const [genresEdited, setGenresEdited] = useState(false);

  const toggleGenre = (rawID: string) => {
    const id = Number(rawID);
    setGenresEdited(true);
    setGenreIds((prev) => {
      if (prev.includes(id)) return prev.filter((g) => g !== id);
      // The cap is enforced here rather than by disabling the unselected
      // chips, so a translator can always see the full list.
      if (prev.length >= MAX_GENRES) return prev;
      return [...prev, id];
    });
  };

  return (
    <div>
      <div className="form-grid">
        <label className="field">
          <span className="field__label">ชื่อเรื่องภาษาไทย</span>
          <input className="input" value={titleTH} onChange={(e) => setTitleTH(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">ชื่อต้นฉบับ</span>
          <input className="input" value={titleCN} onChange={(e) => setTitleCN(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">ผู้แต่ง</span>
          <input className="input" value={author} onChange={(e) => setAuthor(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">บทในต้นฉบับ</span>
          <input
            className="input"
            type="number"
            min={0}
            value={sourceCount}
            onChange={(e) => setSourceCount(Math.max(0, Number(e.target.value) || 0))}
          />
        </label>
      </div>

      <label className="field" style={{ marginTop: 18 }}>
        <span className="field__label">เรื่องย่อ</span>
        <textarea
          className="textarea"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </label>

      <div style={{ marginTop: 18 }}>
        <div className="field__label" style={{ marginBottom: 8 }}>
          หมวดหมู่ เลือกได้สูงสุด {MAX_GENRES} หมวด
        </div>
        <div className="chips">
          {(genres.data?.data ?? []).map((g: Genre) => (
            <button
              key={g.id}
              className={`chip${genreIds.includes(Number(g.id)) ? " is-active" : ""}`}
              onClick={() => toggleGenre(g.id)}
            >
              {g.name_th}
            </button>
          ))}
        </div>
      </div>

      <SaveRow saving={save.isPending} error={save.error} saved={save.isSuccess}>
        <button
          className="btn btn--primary"
          disabled={save.isPending || titleTH.trim() === ""}
          onClick={() =>
            save.mutate({
              title_th: titleTH,
              title_cn: titleCN,
              author_name: author,
              description,
              source_chapters_count: sourceCount,
              ...(genresEdited ? { genre_ids: genreIds } : {}),
            })
          }
        >
          บันทึกการแก้ไข
        </button>
        <Link to={`/novels/${novel.slug}`} className="btn">
          ดูตัวอย่างหน้าเรื่อง
        </Link>
      </SaveRow>
    </div>
  );
}

