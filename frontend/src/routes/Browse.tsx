import { useState } from "react";
import { useQuery, keepPreviousData } from "@tanstack/react-query";

import { api } from "../lib/api";
import { numberTH } from "../lib/format";
import { Empty, ErrorNote, Loading, NovelCard } from "../components";

type Sort = "popular" | "latest";

export default function Browse() {
  const [query, setQuery] = useState("");
  const [submitted, setSubmitted] = useState("");
  const [genre, setGenre] = useState("");
  const [sort, setSort] = useState<Sort>("popular");

  const genres = useQuery({ queryKey: ["genres"], queryFn: api.listGenres });

  const novels = useQuery({
    queryKey: ["novels", { q: submitted, genre, sort }],
    queryFn: () => api.listNovels({ q: submitted, genre, sort, limit: 30 }),
    // Keeping the previous page visible avoids a flash of empty state while a
    // new filter loads.
    placeholderData: keepPreviousData,
  });

  const results = novels.data?.data ?? [];

  return (
    <section>
      <div className="page-head">
        <h1 className="page-title">หมวดหมู่และค้นหา</h1>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          setSubmitted(query.trim());
        }}
        style={{ marginTop: 22, display: "flex", gap: 10 }}
      >
        <input
          className="input"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="ค้นหาชื่อเรื่อง ชื่อจีน ผู้แต่ง หรือชื่อตัวละคร"
          aria-label="ค้นหานิยาย"
        />
        <button className="btn btn--primary" type="submit">
          ค้นหา
        </button>
      </form>

      <div className="chips" style={{ marginTop: 18 }}>
        <button
          className={`chip${genre === "" ? " is-active" : ""}`}
          onClick={() => setGenre("")}
        >
          ทั้งหมด
        </button>
        {genres.data?.data.map((g) => (
          <button
            key={g.id}
            className={`chip${genre === g.slug ? " is-active" : ""}`}
            onClick={() => setGenre(genre === g.slug ? "" : g.slug)}
          >
            {g.name_th}
          </button>
        ))}
      </div>

      <div
        style={{
          marginTop: 24,
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 16,
          flexWrap: "wrap",
        }}
      >
        <div className="muted" style={{ fontSize: 12.5 }}>
          {novels.isLoading ? "กำลังค้นหา…" : `พบ ${numberTH(results.length)} เรื่อง`}
          {submitted && ` สำหรับ “${submitted}”`}
        </div>
        <div style={{ display: "flex", gap: 6 }}>
          {(
            [
              ["popular", "เรียงตามความนิยม"],
              ["latest", "อัปเดตล่าสุด"],
            ] as [Sort, string][]
          ).map(([key, label]) => (
            <button
              key={key}
              className={`chip${sort === key ? " is-active" : ""}`}
              onClick={() => setSort(key)}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {novels.isError ? (
        <ErrorNote message={(novels.error as Error).message} />
      ) : novels.isLoading ? (
        <Loading rows={4} />
      ) : results.length === 0 ? (
        <Empty>ไม่พบนิยายที่ตรงกับเงื่อนไข ลองเปลี่ยนคำค้นหรือหมวดหมู่</Empty>
      ) : (
        <div className="grid grid--cards" style={{ marginTop: 18 }}>
          {results.map((novel) => (
            <NovelCard key={novel.id} novel={novel} />
          ))}
        </div>
      )}
    </section>
  );
}
