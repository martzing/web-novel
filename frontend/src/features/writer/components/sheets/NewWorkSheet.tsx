import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Modal } from "@/shared/ui";

import { writerApi } from "../../api";
import { writerKeys } from "../../queries";

export function NewWorkSheet({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (id: string) => void;
}) {
  const qc = useQueryClient();
  const [titleTH, setTitleTH] = useState("");
  const [titleCN, setTitleCN] = useState("");
  const [author, setAuthor] = useState("");

  const create = useMutation({
    mutationFn: () =>
      writerApi.createWriterNovel({ title_th: titleTH, title_cn: titleCN, author_name: author }),
    onSuccess: (novel) => {
      qc.invalidateQueries({ queryKey: writerKeys.novels() });
      onCreated(novel.id);
    },
  });

  return (
    <Modal title="เพิ่มเรื่องใหม่" onClose={onClose}>
      <div className="grid" style={{ gap: 14 }}>
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
      </div>
      {create.isError && <div className="form-error">{(create.error as Error).message}</div>}
      <button
        className="btn btn--primary btn--block"
        style={{ marginTop: 18 }}
        disabled={titleTH.trim() === "" || create.isPending}
        onClick={() => create.mutate()}
      >
        สร้างเรื่อง
      </button>
    </Modal>
  );
}

