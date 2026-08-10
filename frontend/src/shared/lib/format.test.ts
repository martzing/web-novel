import { afterEach, describe, expect, it, vi } from "vitest";

import { baht, greeting, percent, relativeTime, thaiDate, trend } from "./format";

afterEach(() => {
  vi.useRealTimers();
});

describe("thaiDate", () => {
  it("renders the Buddhist era year Thai readers expect", () => {
    // 2026 CE + 543 = 2569 BE.
    expect(thaiDate("2026-08-10T00:00:00Z")).toContain("2569");
  });

  it("returns an empty string rather than 'Invalid Date'", () => {
    expect(thaiDate(undefined)).toBe("");
    expect(thaiDate("not a date")).toBe("");
  });
});

describe("relativeTime", () => {
  it("steps through the phrasing the mockups use", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-10T12:00:00Z"));

    const ago = (minutes: number) =>
      relativeTime(new Date(Date.now() - minutes * 60_000).toISOString());

    expect(ago(1)).toBe("เมื่อสักครู่");
    expect(ago(30)).toBe("30 นาทีที่แล้ว");
    expect(ago(3 * 60)).toBe("3 ชั่วโมงที่แล้ว");
    expect(ago(24 * 60)).toBe("เมื่อวาน");
    expect(ago(3 * 24 * 60)).toBe("3 วันที่แล้ว");
    expect(ago(14 * 24 * 60)).toBe("2 สัปดาห์ที่แล้ว");
  });

  it("falls back to an absolute date past a month", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-10T12:00:00Z"));
    expect(relativeTime("2026-01-01T00:00:00Z")).toContain("2569");
  });
});

describe("baht", () => {
  it("converts satang to baht", () => {
    expect(baht(9900)).toBe("฿99");
    expect(baht(0)).toBe("฿0");
  });
});

describe("percent and trend", () => {
  it("rounds a percentage to a whole number", () => {
    expect(percent(66.6)).toBe("67%");
  });

  it("signs a positive trend so a gain reads as one", () => {
    expect(trend(12.34)).toBe("+12.3%");
    expect(trend(-4)).toBe("-4.0%");
  });
});

describe("greeting", () => {
  it("picks the greeting from the hour", () => {
    expect(greeting(new Date(2026, 7, 10, 8))).toBe("สวัสดียามเช้า");
    expect(greeting(new Date(2026, 7, 10, 14))).toBe("สวัสดียามบ่าย");
    expect(greeting(new Date(2026, 7, 10, 20))).toBe("สวัสดียามค่ำ");
  });
});
