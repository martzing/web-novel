import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useAuth } from "@/features/identity";
import { numberTH } from "@/shared/lib/format";
import { Cover, Empty, ErrorNote, Loading, Tabs } from "@/shared/ui";

import type { WriterNovel, WriterSeries } from "../api";
import { ChaptersTab } from "../components/tabs/ChaptersTab";
import { CoverTab } from "../components/tabs/CoverTab";
import { GlossaryTab } from "../components/tabs/GlossaryTab";
import { InfoTab } from "../components/tabs/InfoTab";
import { PricingTab } from "../components/tabs/PricingTab";
import { SeriesTab } from "../components/tabs/SeriesTab";
import { NewSeriesSheet } from "../components/sheets/NewSeriesSheet";
import { NewWorkSheet } from "../components/sheets/NewWorkSheet";
import { TABS, statusLabel, type WorkTab } from "../constants";
import { useWriterNovels, useWriterSeries } from "../queries";

export default function Works() {
  const { user, isTranslator } = useAuth();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [tab, setTab] = useState<WorkTab>("info");
  const [sheet, setSheet] = useState<"work" | "series" | null>(null);

  const novels = useWriterNovels(Boolean(user && isTranslator));
  const seriesList = useWriterSeries(Boolean(user && isTranslator));

  const works = useMemo(() => novels.data?.data ?? [], [novels.data]);

  // Selecting the first work on load is what makes the master/detail layout
  // usable — the mockup's right pane is never empty.
  useEffect(() => {
    if (!selectedId && works.length > 0) setSelectedId(works[0].id);
  }, [selectedId, works]);

  if (!user) {
    return (
      <section>
        <h1 className="page-title">จัดการผลงาน</h1>
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อจัดการผลงานของคุณ
        </Empty>
      </section>
    );
  }
  if (!isTranslator) {
    return (
      <section>
        <h1 className="page-title">จัดการผลงาน</h1>
        <Empty>บัญชีนี้ยังไม่ได้รับสิทธิ์นักแปล</Empty>
      </section>
    );
  }

  const selected = works.find((w) => w.id === selectedId) ?? null;
  const groups = groupWorks(works, seriesList.data?.data ?? []);

  return (
    <section>
      <div className="page-head">
        <div>
          <h1 className="page-title">จัดการผลงาน</h1>
          <div className="muted" style={{ fontSize: 12.5, marginTop: 6 }}>
            {numberTH(works.length)} เรื่อง · {numberTH(seriesList.data?.data.length ?? 0)} ชุดหนังสือ
          </div>
        </div>
        <button className="btn btn--primary" onClick={() => setSheet("work")}>
          + เพิ่มเรื่องใหม่
        </button>
      </div>

      {novels.isError ? (
        <ErrorNote message={(novels.error as Error).message} />
      ) : novels.isLoading ? (
        <Loading rows={4} />
      ) : (
        <div className="works">
          <aside className="works__tree card">
            <div className="eyebrow">ชุดและเรื่อง</div>

            {groups.map((group) => (
              <div key={group.key} style={{ marginTop: 16 }}>
                <div className="works__group">
                  <span>{group.label}</span>
                  <span className="mono muted">{group.works.length}</span>
                </div>
                {group.works.map((work) => (
                  <button
                    key={work.id}
                    className={`works__item${work.id === selectedId ? " is-active" : ""}`}
                    onClick={() => setSelectedId(work.id)}
                  >
                    <Cover
                      url={work.cover_url}
                      titleCN={work.title_cn}
                      style={work.cover_style}
                      color={work.cover_color}
                      text={work.cover_text}
                      width={30}
                      height={42}
                    />
                    <span style={{ minWidth: 0 }}>
                      <span className="works__item-title">{work.title_th}</span>
                      <span className="works__item-sub muted">
                        {statusLabel(work.status)} · แปลแล้ว {numberTH(work.chapters_count)}
                      </span>
                    </span>
                  </button>
                ))}
              </div>
            ))}

            <button
              className="btn btn--ghost btn--block"
              style={{ marginTop: 18 }}
              onClick={() => setSheet("series")}
            >
              + สร้างชุดหนังสือใหม่
            </button>
          </aside>

          <div className="works__detail card">
            {!selected ? (
              <Empty>ยังไม่มีผลงาน · กด “เพิ่มเรื่องใหม่” เพื่อเริ่ม</Empty>
            ) : (
              <>
                <Tabs<WorkTab> tabs={TABS} active={tab} onChange={setTab} />
                <div style={{ marginTop: 22 }}>
                  {tab === "info" && <InfoTab key={selected.id} novel={selected} />}
                  {tab === "cover" && <CoverTab key={selected.id} novel={selected} />}
                  {tab === "chapters" && <ChaptersTab key={selected.id} novel={selected} />}
                  {tab === "glossary" && <GlossaryTab key={selected.id} novel={selected} />}
                  {tab === "series" && (
                    <SeriesTab
                      key={selected.id}
                      novel={selected}
                      seriesList={seriesList.data?.data ?? []}
                      works={works}
                    />
                  )}
                  {tab === "pricing" && <PricingTab key={selected.id} novel={selected} />}
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {sheet === "work" && (
        <NewWorkSheet
          onClose={() => setSheet(null)}
          onCreated={(id) => {
            setSelectedId(id);
            setTab("info");
            setSheet(null);
          }}
        />
      )}
      {sheet === "series" && <NewSeriesSheet onClose={() => setSheet(null)} />}
    </section>
  );
}

interface WorkGroup {
  key: string;
  label: string;
  works: WriterNovel[];
}

/** Groups the work tree by series, with the unfiled works last. */
function groupWorks(works: WriterNovel[], series: WriterSeries[]): WorkGroup[] {
  const groups: WorkGroup[] = series.map((s) => ({
    key: s.id,
    label: s.title,
    works: works.filter((w) => w.series_id === s.id),
  }));

  const unfiled = works.filter((w) => !w.series_id || !series.some((s) => s.id === w.series_id));
  if (unfiled.length > 0) {
    groups.push({ key: "none", label: "ไม่สังกัดชุด", works: unfiled });
  }
  return groups.filter((g) => g.works.length > 0);
}
