import { describe, expect, it } from "vitest";

import { writerKeys } from "./queries";

describe("writerKeys", () => {
  const keys = [
    writerKeys.novels(),
    writerKeys.series(),
    writerKeys.seriesBooksOf("series-1"),
    writerKeys.relations("novel-1"),
    writerKeys.arcs("novel-1"),
    writerKeys.chapters("novel-1"),
    writerKeys.chapter("chapter-1"),
    writerKeys.glossary("novel-1"),
    writerKeys.stats("novel-1", "14d"),
    writerKeys.earnings(),
  ];

  it("roots every key at writerKeys.all", () => {
    for (const key of keys) {
      expect(key.slice(0, writerKeys.all.length)).toEqual([...writerKeys.all]);
    }
  });

  it("gives each query its own key", () => {
    const serialised = keys.map((key) => JSON.stringify(key));
    expect(new Set(serialised).size).toBe(keys.length);
  });

  /**
   * The regression this factory was introduced for.
   *
   * `SeriesNote` invalidated the un-suffixed key while `SeriesTab` read the
   * suffixed one. Saving a note therefore refreshed nothing. Building both from
   * the same root makes the broad key a true prefix of the narrow one, which is
   * what React Query's prefix matching needs.
   */
  it("makes the broad series-books key a prefix of the per-series one", () => {
    const broad = writerKeys.seriesBooks();
    const narrow = writerKeys.seriesBooksOf("series-1");
    expect(narrow.slice(0, broad.length)).toEqual([...broad]);
  });

  it("keeps two stats periods of one novel apart", () => {
    expect(writerKeys.stats("novel-1", "14d")).not.toEqual(writerKeys.stats("novel-1", "30d"));
  });
});
