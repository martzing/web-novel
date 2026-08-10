import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import type { CoverStyle } from "@/shared/api/types";
import { COVER_COLORS, COVER_STYLES, Cover } from "@/shared/ui";

import { writerApi, type WriterNovel } from "../../api";
import { useSaveNovel, writerKeys } from "../../queries";
import { SaveRow } from "../SaveRow";

// ── Tab 2 · หน้าปก ──────────────────────────────────────────────────────────

export function CoverTab({ novel }: { novel: WriterNovel }) {
  const qc = useQueryClient();
  const save = useSaveNovel(novel.id);

  const [style, setStyle] = useState<CoverStyle>(novel.cover_style);
  const [color, setColor] = useState(novel.cover_color ?? COVER_COLORS[0]);
  const [text, setText] = useState(novel.cover_text ?? "");

  const upload = useMutation({
    mutationFn: (file: File) => writerApi.uploadCover(novel.id, file),
    onSuccess: () => qc.invalidateQueries({ queryKey: writerKeys.novels() }),
  });

  return (
    <div className="cover-editor">
      <div className="cover-editor__preview">
        {/* The preview renders from the working state, not the saved novel, so
            picking a swatch shows the result before committing to it. */}
        <Cover
          url={novel.cover_url}
          titleCN={novel.title_cn}
          style={style}
          color={color}
          text={text || novel.title_cn}
          width={198}
          height={264}
        />
        <div className="muted" style={{ fontSize: 11.5, marginTop: 10, textAlign: "center" }}>
          ตัวอย่างหน้าปก 600 × 800
        </div>
      </div>

      <div className="cover-editor__controls">
        <div className="field">
          <span className="field__label">อัปโหลดภาพปก</span>
          <div className="dropzone">
            <div style={{ fontSize: 13 }}>ลากไฟล์มาวาง หรือเลือกจากเครื่อง</div>
            <div className="muted" style={{ fontSize: 11.5, marginTop: 6 }}>
              JPG หรือ PNG อัตราส่วน 3:4 อย่างน้อย 600 × 800 px
            </div>
            <label className="btn btn--sm" style={{ marginTop: 12, cursor: "pointer" }}>
              เลือกไฟล์
              <input
                type="file"
                accept="image/jpeg,image/png,image/webp"
                hidden
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) upload.mutate(file);
                }}
              />
            </label>
          </div>
          {upload.isPending && <span className="muted" style={{ fontSize: 12 }}>กำลังอัปโหลด…</span>}
          {upload.isError && <span className="form-error">{(upload.error as Error).message}</span>}
        </div>

        <div className="field" style={{ marginTop: 18 }}>
          <span className="field__label">หรือสร้างปกจากแม่แบบ</span>
          <div className="chips">
            {COVER_STYLES.map((s) => (
              <button
                key={s.key}
                className={`chip${style === s.key ? " is-active" : ""}`}
                onClick={() => setStyle(s.key)}
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>

        <div className="field" style={{ marginTop: 18 }}>
          <span className="field__label">สีพื้นปก</span>
          <div className="swatches">
            {COVER_COLORS.map((c) => (
              <button
                key={c}
                className={`swatch${color === c ? " is-active" : ""}`}
                style={{ background: c }}
                aria-label={`สี ${c}`}
                onClick={() => setColor(c)}
              />
            ))}
          </div>
        </div>

        <label className="field" style={{ marginTop: 18 }}>
          <span className="field__label">ตัวอักษรบนปก</span>
          <input
            className="input"
            value={text}
            maxLength={40}
            onChange={(e) => setText(e.target.value)}
            placeholder={novel.title_cn ?? novel.title_th}
          />
        </label>

        <SaveRow saving={save.isPending} error={save.error} saved={save.isSuccess}>
          <button
            className="btn btn--primary"
            disabled={save.isPending}
            onClick={() =>
              save.mutate({ cover_style: style, cover_color: color, cover_text: text })
            }
          >
            ใช้ปกนี้
          </button>
          <button
            className="btn"
            onClick={() => {
              setStyle(novel.cover_style);
              setColor(novel.cover_color ?? COVER_COLORS[0]);
              setText(novel.cover_text ?? "");
            }}
          >
            คืนค่าปกเดิม
          </button>
        </SaveRow>
      </div>
    </div>
  );
}

