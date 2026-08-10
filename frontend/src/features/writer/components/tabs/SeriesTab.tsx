import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { moveItem, useReorder } from "@/shared/lib/reorder";
import { Empty, Loading } from "@/shared/ui";

import {
  writerApi,
  type WriterNovel,
  type WriterRelation,
  type WriterSeries,
  type WriterSeriesBook,
} from "../../api";
import { useSaveNovel, useWriterRelations, useWriterSeriesBooks, writerKeys } from "../../queries";
import { LinkSheet } from "../sheets/LinkSheet";

// ── Tab 4 · ชุดและเรื่องเกี่ยวเนื่อง ────────────────────────────────────────

export function SeriesTab({
  novel,
  seriesList,
  works,
}: {
  novel: WriterNovel;
  seriesList: WriterSeries[];
  works: WriterNovel[];
}) {
  const qc = useQueryClient();
  const save = useSaveNovel(novel.id);
  const [linking, setLinking] = useState(false);

  const books = useWriterSeriesBooks(novel.series_id);
  const relations = useWriterRelations(novel.id);

  const [order, setOrder] = useState<WriterSeriesBook[]>([]);
  useEffect(() => setOrder(books.data?.data ?? []), [books.data]);

  const reorder = useMutation({
    mutationFn: (ids: string[]) => writerApi.reorderWriterSeries(novel.series_id!, ids),
    onSuccess: () => qc.invalidateQueries({ queryKey: writerKeys.seriesBooksOf(novel.series_id ?? "") }),
  });

  const unlink = useMutation({
    mutationFn: (relatedId: string) => writerApi.unlinkNovels(novel.id, relatedId),
    onSuccess: () => qc.invalidateQueries({ queryKey: writerKeys.relations(novel.id) }),
  });

  const applyMove = (from: number, to: number) => {
    const next = moveItem(order, from, to);
    setOrder(next);
    reorder.mutate(next.map((b) => b.novel_id));
  };

  const { handlersFor, classFor } = useReorder(applyMove);

  return (
    <div>
      <label className="field">
        <span className="field__label">ชุดหนังสือที่เรื่องนี้สังกัด</span>
        <select
          className="select"
          value={novel.series_id ?? ""}
          onChange={(e) =>
            // The empty option means "leave the series", which the API expresses
            // as an explicit null rather than an omitted key.
            save.mutate({ series_id: e.target.value === "" ? null : e.target.value })
          }
        >
          <option value="">ไม่สังกัดชุดใด</option>
          {seriesList.map((s) => (
            <option key={s.id} value={s.id}>
              {s.title}
            </option>
          ))}
        </select>
      </label>
      {save.isError && <div className="form-error">{(save.error as Error).message}</div>}

      {novel.series_id && (
        <>
          <div className="section-head">
            <div className="eyebrow">ลำดับการอ่านในชุด</div>
            <span className="muted" style={{ fontSize: 12 }}>
              ลากเพื่อจัดลำดับ ผู้อ่านจะเห็นลำดับนี้ในหน้าชุดหนังสือ
            </span>
          </div>

          {books.isLoading ? (
            <Loading rows={2} />
          ) : (
            <div className="rows" style={{ marginTop: 12 }}>
              {order.map((book, index) => (
                <div key={book.novel_id} className={`row drag-row${classFor(index)}`} {...handlersFor(index)}>
                  <span className="drag-handle" aria-hidden="true">
                    ⣿
                  </span>
                  <span className="mono muted" style={{ minWidth: 24, fontSize: 12.5 }}>
                    {index + 1}
                  </span>
                  <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{book.title_th}</span>
                  <SeriesNote novelId={book.novel_id} note={book.note ?? ""} />
                  {/* Drag is pointer-only, so the same move is always reachable
                      from the keyboard through these. */}
                  <button
                    className="btn btn--ghost btn--sm"
                    disabled={index === 0}
                    aria-label="เลื่อนขึ้น"
                    onClick={() => applyMove(index, index - 1)}
                  >
                    ↑
                  </button>
                  <button
                    className="btn btn--ghost btn--sm"
                    disabled={index === order.length - 1}
                    aria-label="เลื่อนลง"
                    onClick={() => applyMove(index, index + 1)}
                  >
                    ↓
                  </button>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      <div className="section-head">
        <div className="eyebrow">เรื่องเกี่ยวเนื่อง</div>
        <button className="btn btn--sm" onClick={() => setLinking(true)}>
          + ผูกเรื่องเกี่ยวเนื่อง
        </button>
      </div>

      {relations.isLoading ? (
        <Loading rows={2} />
      ) : (relations.data?.data ?? []).length === 0 ? (
        <Empty>ยังไม่มีเรื่องเกี่ยวเนื่อง</Empty>
      ) : (
        <div className="rows" style={{ marginTop: 12 }}>
          {relations.data!.data.map((rel: WriterRelation) => (
            <div key={rel.related_novel_id} className="row">
              <span className="pill">{rel.kind_label}</span>
              <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{rel.title_th}</span>
              {rel.mirrored ? (
                // Declared on the other novel: shown for context, but its note
                // and order belong over there.
                <span className="muted" style={{ fontSize: 11.5 }}>
                  ผูกจากอีกเรื่อง
                </span>
              ) : (
                <button
                  className="btn btn--ghost btn--sm"
                  disabled={unlink.isPending}
                  onClick={() => unlink.mutate(rel.related_novel_id)}
                >
                  ปลด
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {linking && (
        <LinkSheet
          novel={novel}
          candidates={works.filter((w) => w.id !== novel.id)}
          onClose={() => setLinking(false)}
        />
      )}
    </div>
  );
}

/** The per-book note in a series' reading order, saved on blur. */
function SeriesNote({ novelId, note }: { novelId: string; note: string }) {
  const qc = useQueryClient();
  const [value, setValue] = useState(note);
  useEffect(() => setValue(note), [note]);

  const save = useMutation({
    mutationFn: (next: string) => writerApi.setSeriesNote(novelId, next),
    onSuccess: () => qc.invalidateQueries({ queryKey: writerKeys.seriesBooks() }),
  });

  return (
    <input
      className="input series-note"
      value={value}
      placeholder="โน้ตสำหรับผู้อ่าน"
      onChange={(e) => setValue(e.target.value)}
      onBlur={() => value !== note && save.mutate(value)}
    />
  );
}

