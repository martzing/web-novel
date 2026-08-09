import { Link, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { api, type LedgerEntry } from "../lib/api";
import { useAuth } from "../lib/auth";
import { baht, ledgerLabel, numberTH, relativeTime, thaiDate } from "../lib/format";
import { Empty, ErrorNote, Loading } from "../components";

/**
 * The going rate for a paid chapter, used only to turn a balance into
 * "ปลดล็อกได้อีกราว N บท".
 *
 * Deliberately a constant and deliberately labelled ราว: chapter prices differ
 * per novel, so any exact figure would be wrong for most of what the reader is
 * about to open. The estimate answers "roughly how far does this go", which is
 * the question a balance actually raises.
 */
const TYPICAL_CHAPTER_PRICE = 5;

function estimateUnlockableChapters(total?: number): number | null {
  if (!total || total < TYPICAL_CHAPTER_PRICE) return null;
  return Math.floor(total / TYPICAL_CHAPTER_PRICE);
}

/**
 * Coins spent since the first of this month, from the ledger page already on
 * screen. Only debits count, so a top-up does not read as spending.
 */
function coinsSpentThisMonth(entries?: LedgerEntry[]): number {
  if (!entries) return 0;

  const now = new Date();
  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);

  return entries.reduce((sum, entry) => {
    const amount = entry.delta + entry.bonus_delta;
    if (amount >= 0) return sum;
    return new Date(entry.created_at) >= monthStart ? sum + Math.abs(amount) : sum;
  }, 0);
}

export default function Coins() {
  const { user } = useAuth();
  const navigate = useNavigate();

  const wallet = useQuery({ queryKey: ["wallet"], queryFn: api.getWallet, enabled: Boolean(user) });
  const packs = useQuery({ queryKey: ["packs"], queryFn: api.listPacks });
  const ledger = useQuery({
    queryKey: ["ledger"],
    queryFn: () => api.getLedger(),
    enabled: Boolean(user),
  });

  const unlockable = estimateUnlockableChapters(wallet.data?.total);
  const spentThisMonth = coinsSpentThisMonth(ledger.data?.data);

  return (
    <section>
      <div className="page-head">
        <h1 className="page-title">เหรียญ</h1>
      </div>

      {user ? (
        <div className="card" style={{ marginTop: 22 }}>
          <div className="eyebrow">ยอดคงเหลือ</div>
          <div className="serif" style={{ fontSize: 42, fontWeight: 600, marginTop: 6 }}>
            {numberTH(wallet.data?.total ?? 0)}
          </div>
          <div className="muted" style={{ fontSize: 12.5, marginTop: 8, lineHeight: 1.9 }}>
            เหรียญที่ซื้อ {numberTH(wallet.data?.balance ?? 0)} · โบนัส{" "}
            {numberTH(wallet.data?.bonus_balance ?? 0)}
            {unlockable !== null && (
              <>
                <br />
                ปลดล็อกได้อีกราว {numberTH(unlockable)} บท
              </>
            )}
            {spentThisMonth > 0 && (
              <>
                <br />
                ใช้ไปเดือนนี้ {numberTH(spentThisMonth)} เหรียญ
              </>
            )}
            {wallet.data?.bonus_expires_at && (
              <>
                <br />
                เหรียญโบนัสหมดอายุ {thaiDate(wallet.data.bonus_expires_at)}
              </>
            )}
          </div>
        </div>
      ) : (
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อเติมเหรียญและดูประวัติการใช้
        </Empty>
      )}

      <div className="section-head">
        <div className="eyebrow">เติมเหรียญ</div>
      </div>

      {packs.isLoading ? (
        <Loading rows={2} />
      ) : (
        <div className="grid grid--cards" style={{ marginTop: 14 }}>
          {(packs.data?.data ?? []).map((pack) => (
            <button
              key={pack.id}
              className="card"
              style={{ textAlign: "left", position: "relative" }}
              onClick={() => navigate(`/checkout/${pack.id}`)}
            >
              {pack.is_best_value && (
                <span className="pill pill--gold" style={{ position: "absolute", top: 14, right: 14 }}>
                  คุ้มที่สุด
                </span>
              )}
              <div className="serif" style={{ fontSize: 28, fontWeight: 600 }}>
                {numberTH(pack.coins)}
              </div>
              <div className="muted" style={{ fontSize: 12.5, marginTop: 4 }}>
                {pack.bonus_coins > 0 ? `+${numberTH(pack.bonus_coins)} โบนัส` : "เหรียญ"}
              </div>
              <div style={{ fontSize: 17, marginTop: 14 }}>{baht(pack.price_satang)}</div>
            </button>
          ))}
        </div>
      )}

      {user && (
        <>
          <div className="section-head">
            <div className="eyebrow">ประวัติการใช้</div>
          </div>

          {ledger.isError ? (
            <ErrorNote message={(ledger.error as Error).message} />
          ) : ledger.isLoading ? (
            <Loading />
          ) : (ledger.data?.data.length ?? 0) === 0 ? (
            <Empty>ยังไม่มีรายการ</Empty>
          ) : (
            <div className="rows" style={{ marginTop: 14 }}>
              {ledger.data!.data.map((entry) => {
                const amount = entry.delta + entry.bonus_delta;
                return (
                  <div key={entry.id} className="row">
                    <span style={{ flex: 1, minWidth: 0 }}>
                      <span style={{ fontSize: 13.5 }}>
                        {entry.reason || ledgerLabel(entry.kind)}
                      </span>
                      <span className="muted" style={{ display: "block", fontSize: 11.5, marginTop: 2 }}>
                        {relativeTime(entry.created_at)}
                      </span>
                    </span>
                    <span
                      className="mono"
                      style={{ fontSize: 13, color: amount >= 0 ? "var(--gold)" : "var(--red)" }}
                    >
                      {amount >= 0 ? "+" : ""}
                      {numberTH(amount)}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}
    </section>
  );
}
