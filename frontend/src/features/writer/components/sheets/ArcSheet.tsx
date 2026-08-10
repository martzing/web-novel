import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Modal } from "@/shared/ui";

import { writerApi, type WriterArc } from "../../api";
import { writerKeys } from "../../queries";

export function ArcSheet({
  novelId,
  arc,
  onClose,
}: {
  novelId: string;
  arc: WriterArc | null;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [arcNo, setArcNo] = useState(arc?.arc_no ?? 1);
  const [name, setName] = useState(arc?.name ?? "");
  const [from, setFrom] = useState(arc?.from_chapter_no ?? 1);
  const [to, setTo] = useState(arc?.to_chapter_no ?? 1);

  const submit = useMutation({
    mutationFn: () => {
      const body = { arc_no: arcNo, name, from_chapter_no: from, to_chapter_no: to };
      return arc ? writerApi.updateWriterArc(arc.id, body) : writerApi.createWriterArc(novelId, body);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: writerKeys.arcs(novelId) });
      onClose();
    },
  });

  return (
    <Modal title={arc ? "แก้ไขภาค" : "เพิ่มภาคใหม่"} onClose={onClose}>
      <div className="form-grid">
        <label className="field">
          <span className="field__label">ภาคที่</span>
          <input
            className="input"
            type="number"
            min={1}
            value={arcNo}
            onChange={(e) => setArcNo(Math.max(1, Number(e.target.value) || 1))}
          />
        </label>
        <label className="field">
          <span className="field__label">ชื่อภาค</span>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">ตั้งแต่บทที่</span>
          <input
            className="input"
            type="number"
            min={1}
            value={from}
            onChange={(e) => setFrom(Math.max(1, Number(e.target.value) || 1))}
          />
        </label>
        <label className="field">
          <span className="field__label">ถึงบทที่</span>
          <input
            className="input"
            type="number"
            min={1}
            value={to}
            onChange={(e) => setTo(Math.max(1, Number(e.target.value) || 1))}
          />
        </label>
      </div>
      {to < from && (
        <div className="form-error" style={{ marginTop: 10 }}>
          บทสุดท้ายต้องไม่น้อยกว่าบทแรก
        </div>
      )}
      {submit.isError && <div className="form-error">{(submit.error as Error).message}</div>}
      <button
        className="btn btn--primary btn--block"
        style={{ marginTop: 18 }}
        disabled={name.trim() === "" || to < from || submit.isPending}
        onClick={() => submit.mutate()}
      >
        {arc ? "บันทึกภาค" : "เพิ่มภาค"}
      </button>
    </Modal>
  );
}

