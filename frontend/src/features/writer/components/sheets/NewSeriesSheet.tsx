import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Modal } from "@/shared/ui";

import { writerApi } from "../../api";
import { writerKeys } from "../../queries";

export function NewSeriesSheet({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");

  const create = useMutation({
    mutationFn: () => writerApi.createWriterSeries({ title, description }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: writerKeys.series() });
      onClose();
    },
  });

  return (
    <Modal title="สร้างชุดหนังสือใหม่" onClose={onClose}>
      <div className="grid" style={{ gap: 14 }}>
        <label className="field">
          <span className="field__label">ชื่อชุด</span>
          <input className="input" value={title} onChange={(e) => setTitle(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">คำอธิบาย</span>
          <textarea
            className="textarea"
            style={{ minHeight: 90 }}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </label>
      </div>
      {create.isError && <div className="form-error">{(create.error as Error).message}</div>}
      <button
        className="btn btn--primary btn--block"
        style={{ marginTop: 18 }}
        disabled={title.trim() === "" || create.isPending}
        onClick={() => create.mutate()}
      >
        สร้างชุดหนังสือ
      </button>
    </Modal>
  );
}

