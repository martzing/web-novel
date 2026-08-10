import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import type { GlossaryEntry } from "@/shared/api/types";
import { Empty, ErrorNote, Loading, Modal } from "@/shared/ui";

import { writerApi, type WriterNovel } from "../../api";
import { useWriterGlossary, writerKeys } from "../../queries";

// ── Tab 4 · อภิธานศัพท์ ─────────────────────────────────────────────────────

/**
 * Glossary management (W-05).
 *
 * The editor already binds terms with `{{term_key}}` and the reader already
 * taps them open, but until now there was nowhere to define one — the API had
 * existed since phase 1 with no screen behind it. `term_key` is shown on every
 * row rather than hidden as an implementation detail, because it is literally
 * what the translator types into a chapter body.
 */
export function GlossaryTab({ novel }: { novel: WriterNovel }) {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<{ groupId: string; entry: GlossaryEntry | null } | null>(null);
  const [addingGroup, setAddingGroup] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<GlossaryEntry | null>(null);

  const glossary = useWriterGlossary(novel.id);

  const invalidate = () => qc.invalidateQueries({ queryKey: writerKeys.glossary(novel.id) });

  const removeEntry = useMutation({
    mutationFn: (id: string) => writerApi.deleteGlossaryEntry(id),
    onSuccess: () => {
      setConfirmDelete(null);
      invalidate();
    },
  });

  const removeGroup = useMutation({
    mutationFn: (id: string) => writerApi.deleteGlossaryGroup(id),
    onSuccess: invalidate,
  });

  const groups = glossary.data?.data ?? [];

  return (
    <div>
      <div className="section-head" style={{ marginTop: 0 }}>
        <div className="eyebrow">หมวดและศัพท์</div>
        <button className="btn btn--sm" onClick={() => setAddingGroup(true)}>
          + เพิ่มหมวดใหม่
        </button>
      </div>

      <p className="muted" style={{ fontSize: 12.5, marginTop: 10, lineHeight: 1.9 }}>
        พิมพ์ <code className="mono">{"{{term_key}}"}</code> ในเนื้อบทเพื่อผูกศัพท์
        ผู้อ่านจะแตะคำนั้นแล้วเห็นคำอธิบายทันที · แก้คำอธิบายแล้วบททุกบทที่ผูกไว้จะถูกเรนเดอร์ใหม่ให้เอง
      </p>

      {glossary.isLoading ? (
        <Loading rows={3} />
      ) : groups.length === 0 ? (
        <Empty>ยังไม่มีอภิธานศัพท์ · เริ่มจากสร้างหมวด เช่น ลำดับขั้นการบำเพ็ญ หรือ ตัวละคร</Empty>
      ) : (
        groups.map((group) => (
          <div key={group.id} style={{ marginTop: 22 }}>
            <div className="section-head" style={{ marginTop: 0 }}>
              <div className="eyebrow">{group.name}</div>
              <div style={{ display: "flex", gap: 8 }}>
                <button
                  className="btn btn--sm"
                  onClick={() => setEditing({ groupId: group.id, entry: null })}
                >
                  + เพิ่มศัพท์
                </button>
                {group.entries.length === 0 && (
                  <button
                    className="btn btn--ghost btn--sm"
                    onClick={() => removeGroup.mutate(group.id)}
                    disabled={removeGroup.isPending}
                  >
                    ลบหมวด
                  </button>
                )}
              </div>
            </div>

            {group.entries.length === 0 ? (
              <Empty>หมวดนี้ยังไม่มีศัพท์</Empty>
            ) : (
              <div className="rows" style={{ marginTop: 12 }}>
                {group.entries.map((entry) => (
                  <div key={entry.id} className="row">
                    <span className="mono muted" style={{ minWidth: 92, fontSize: 11.5 }}>
                      {entry.term_key}
                    </span>
                    <span style={{ flex: 1, minWidth: 0 }}>
                      <span style={{ fontSize: 13.5 }}>
                        {entry.title_th}
                        {entry.title_cn && <span className="muted"> {entry.title_cn}</span>}
                      </span>
                      <span
                        className="muted"
                        style={{ display: "block", fontSize: 11.5, marginTop: 2 }}
                      >
                        {entry.body}
                      </span>
                    </span>
                    <button
                      className="btn btn--ghost btn--sm"
                      onClick={() => setEditing({ groupId: group.id, entry })}
                    >
                      แก้ไข
                    </button>
                    <button className="btn btn--ghost btn--sm" onClick={() => setConfirmDelete(entry)}>
                      ลบ
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        ))
      )}

      {removeGroup.isError && <ErrorNote message={(removeGroup.error as Error).message} />}

      {addingGroup && (
        <GlossaryGroupSheet
          novelId={novel.id}
          sortNo={groups.length}
          onClose={() => setAddingGroup(false)}
          onSaved={invalidate}
        />
      )}

      {editing && (
        <GlossaryEntrySheet
          novelId={novel.id}
          groupId={editing.groupId}
          entry={editing.entry}
          onClose={() => setEditing(null)}
          onSaved={invalidate}
        />
      )}

      {confirmDelete && (
        <Modal title="ลบศัพท์นี้" onClose={() => setConfirmDelete(null)}>
          <p style={{ fontSize: 13.5, lineHeight: 1.9 }}>
            ลบ <strong>{confirmDelete.title_th}</strong> ออกจากอภิธานศัพท์
            บททุกบทที่ผูกคำนี้ไว้จะถูกเรนเดอร์ใหม่ และคำจะกลายเป็นข้อความธรรมดา
            เนื้อบทไม่หายไป
          </p>
          {removeEntry.isError && <ErrorNote message={(removeEntry.error as Error).message} />}
          <div style={{ display: "flex", gap: 10, marginTop: 20 }}>
            <button
              className="btn btn--primary"
              onClick={() => removeEntry.mutate(confirmDelete.id)}
              disabled={removeEntry.isPending}
            >
              {removeEntry.isPending ? "กำลังลบ…" : "ลบศัพท์"}
            </button>
            <button className="btn" onClick={() => setConfirmDelete(null)}>
              ยกเลิก
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}

function GlossaryGroupSheet({
  novelId,
  sortNo,
  onClose,
  onSaved,
}: {
  novelId: string;
  sortNo: number;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState("");

  const submit = useMutation({
    mutationFn: () => writerApi.createGlossaryGroup(novelId, name.trim(), sortNo),
    onSuccess: () => {
      onSaved();
      onClose();
    },
  });

  return (
    <Modal title="เพิ่มหมวดศัพท์" onClose={onClose}>
      <label className="field">
        <span className="field__label">ชื่อหมวด</span>
        <input
          className="input"
          value={name}
          placeholder="เช่น ลำดับขั้นการบำเพ็ญ"
          onChange={(e) => setName(e.target.value)}
        />
      </label>
      {submit.isError && <ErrorNote message={(submit.error as Error).message} />}
      <button
        className="btn btn--primary"
        style={{ marginTop: 20 }}
        disabled={!name.trim() || submit.isPending}
        onClick={() => submit.mutate()}
      >
        {submit.isPending ? "กำลังบันทึก…" : "เพิ่มหมวด"}
      </button>
    </Modal>
  );
}

function GlossaryEntrySheet({
  novelId,
  groupId,
  entry,
  onClose,
  onSaved,
}: {
  novelId: string;
  groupId: string;
  entry: GlossaryEntry | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [termKey, setTermKey] = useState(entry?.term_key ?? "");
  const [titleTH, setTitleTH] = useState(entry?.title_th ?? "");
  const [titleCN, setTitleCN] = useState(entry?.title_cn ?? "");
  const [body, setBody] = useState(entry?.body ?? "");
  const [kind, setKind] = useState(entry?.kind ?? "");

  const submit = useMutation({
    mutationFn: () => {
      const payload = {
        term_key: termKey.trim(),
        title_th: titleTH.trim(),
        title_cn: titleCN.trim(),
        body: body.trim(),
        kind: kind.trim(),
      };
      return entry
        ? writerApi.updateGlossaryEntry(entry.id, payload)
        : writerApi.createGlossaryEntry(novelId, { group_id: groupId, ...payload });
    },
    onSuccess: () => {
      onSaved();
      onClose();
    },
  });

  const ready = termKey.trim() !== "" && titleTH.trim() !== "";

  return (
    <Modal title={entry ? "แก้ไขศัพท์" : "เพิ่มศัพท์"} onClose={onClose}>
      <div className="form-grid">
        <label className="field">
          <span className="field__label">คีย์ที่ใช้ในเนื้อบท</span>
          <input
            className="input mono"
            value={termKey}
            placeholder="qi"
            onChange={(e) => setTermKey(e.target.value)}
          />
          <span className="muted" style={{ fontSize: 11.5, marginTop: 6 }}>
            พิมพ์ {`{{${termKey || "key"}}}`} ในเนื้อบทเพื่อผูกคำนี้
          </span>
        </label>
        <label className="field">
          <span className="field__label">หมวดของคำ (ไม่บังคับ)</span>
          <input
            className="input"
            value={kind}
            placeholder="เช่น ลำดับขั้น"
            onChange={(e) => setKind(e.target.value)}
          />
        </label>
        <label className="field">
          <span className="field__label">คำภาษาไทย</span>
          <input className="input" value={titleTH} onChange={(e) => setTitleTH(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">คำต้นฉบับ (ไม่บังคับ)</span>
          <input className="input" value={titleCN} onChange={(e) => setTitleCN(e.target.value)} />
        </label>
      </div>

      <label className="field" style={{ marginTop: 14 }}>
        <span className="field__label">คำอธิบาย</span>
        <textarea
          className="input"
          rows={4}
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
      </label>

      {submit.isError && <ErrorNote message={(submit.error as Error).message} />}

      <button
        className="btn btn--primary"
        style={{ marginTop: 20 }}
        disabled={!ready || submit.isPending}
        onClick={() => submit.mutate()}
      >
        {submit.isPending ? "กำลังบันทึก…" : entry ? "บันทึกการแก้ไข" : "เพิ่มศัพท์"}
      </button>
    </Modal>
  );
}

