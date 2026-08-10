import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import type { RelationKind } from "@/shared/api/types";
import { Empty, Modal } from "@/shared/ui";

import { writerApi, type WriterNovel } from "../../api";
import { RELATION_KINDS } from "../../constants";
import { writerKeys } from "../../queries";

export function LinkSheet({
  novel,
  candidates,
  onClose,
}: {
  novel: WriterNovel;
  candidates: WriterNovel[];
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [relatedId, setRelatedId] = useState(candidates[0]?.id ?? "");
  const [kind, setKind] = useState<RelationKind>("sequel");
  const [note, setNote] = useState("");

  const link = useMutation({
    mutationFn: () =>
      writerApi.linkNovels(novel.id, { related_novel_id: relatedId, kind, note }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: writerKeys.relations(novel.id) });
      onClose();
    },
  });

  return (
    <Modal title="ผูกเรื่องเกี่ยวเนื่อง" onClose={onClose}>
      {candidates.length === 0 ? (
        <Empty>ต้องมีผลงานอย่างน้อยสองเรื่องจึงจะผูกกันได้</Empty>
      ) : (
        <>
          <div className="grid" style={{ gap: 14 }}>
            <label className="field">
              <span className="field__label">เรื่องที่จะผูก</span>
              <select
                className="select"
                value={relatedId}
                onChange={(e) => setRelatedId(e.target.value)}
              >
                {candidates.map((w) => (
                  <option key={w.id} value={w.id}>
                    {w.title_th}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              <span className="field__label">ความสัมพันธ์ · {novel.title_th} เป็น…</span>
              <select
                className="select"
                value={kind}
                onChange={(e) => setKind(e.target.value as RelationKind)}
              >
                {RELATION_KINDS.map((k) => (
                  <option key={k.key} value={k.key}>
                    {k.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              <span className="field__label">โน้ต (ไม่บังคับ)</span>
              <input className="input" value={note} onChange={(e) => setNote(e.target.value)} />
            </label>
          </div>
          {link.isError && <div className="form-error">{(link.error as Error).message}</div>}
          <button
            className="btn btn--primary btn--block"
            style={{ marginTop: 18 }}
            disabled={relatedId === "" || link.isPending}
            onClick={() => link.mutate()}
          >
            ผูกเรื่อง
          </button>
        </>
      )}
    </Modal>
  );
}
