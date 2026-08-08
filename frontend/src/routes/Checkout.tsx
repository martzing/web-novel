import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, newIdempotencyKey } from "../lib/api";
import { useAuth } from "../lib/auth";
import { baht, numberTH } from "../lib/format";
import { Empty, ErrorNote, Loading, Tabs } from "../components";

type Method = "card" | "promptpay" | "truemoney";

/**
 * Mock checkout.
 *
 * Phase 2 has no payment gateway: the form is a faithful shell over
 * POST /purchases followed by /mock-complete, which credits the wallet through
 * the same ledger helper a real webhook would use. No card data is transmitted
 * or stored — the fields exist so the flow can be reviewed end to end.
 */
export default function Checkout() {
  const { packId = "" } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const qc = useQueryClient();

  const [method, setMethod] = useState<Method>("card");
  const [done, setDone] = useState<{ coins: number; balance: number } | null>(null);

  const packs = useQuery({ queryKey: ["packs"], queryFn: api.listPacks });
  const pack = packs.data?.data.find((p) => p.id === packId);

  const pay = useMutation({
    mutationFn: async () => {
      const purchase = await api.createPurchase(packId);
      // The completion key is generated once per attempt, so a retry of the
      // same attempt replays rather than double-crediting.
      return api.completePurchase(purchase.purchase_id, newIdempotencyKey());
    },
    onSuccess: (receipt) => {
      qc.invalidateQueries({ queryKey: ["wallet"] });
      qc.invalidateQueries({ queryKey: ["ledger"] });
      setDone({
        coins: (pack?.coins ?? 0) + (pack?.bonus_coins ?? 0),
        balance: receipt.balance_after + receipt.bonus_balance_after,
      });
    },
  });

  if (!user) {
    return (
      <section>
        <h1 className="page-title">ชำระเงิน</h1>
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อเติมเหรียญ
        </Empty>
      </section>
    );
  }

  if (packs.isLoading) return <Loading rows={3} />;
  if (!pack) return <Empty>ไม่พบแพ็กเหรียญนี้</Empty>;

  if (done) {
    return (
      <section style={{ maxWidth: 460 }}>
        <div className="eyebrow">เติมเหรียญสำเร็จ</div>
        <h1 className="page-title" style={{ marginTop: 8 }}>
          ได้รับ {numberTH(done.coins)} เหรียญ
        </h1>
        <div className="muted" style={{ fontSize: 13.5, marginTop: 10, lineHeight: 1.9 }}>
          ยอดคงเหลือตอนนี้ {numberTH(done.balance)} เหรียญ
        </div>
        <div style={{ display: "flex", gap: 10, marginTop: 22, flexWrap: "wrap" }}>
          <Link to="/coins" className="btn btn--primary">
            ดูยอดเหรียญ
          </Link>
          <button className="btn" onClick={() => navigate(-1)}>
            กลับไปอ่านต่อ
          </button>
        </div>
      </section>
    );
  }

  return (
    <section style={{ maxWidth: 720 }}>
      <Link to="/coins" className="muted" style={{ fontSize: 12.5 }}>
        ← กลับไปเลือกแพ็ก
      </Link>
      <h1 className="page-title" style={{ marginTop: 14 }}>
        ชำระเงิน
      </h1>

      <div className="card" style={{ marginTop: 22 }}>
        <div className="eyebrow">สรุปคำสั่งซื้อ</div>
        <div style={{ display: "flex", justifyContent: "space-between", marginTop: 12, fontSize: 14 }}>
          <span>
            {numberTH(pack.coins)} เหรียญ
            {pack.bonus_coins > 0 && (
              <span className="muted"> +{numberTH(pack.bonus_coins)} โบนัส</span>
            )}
          </span>
          <span>{baht(pack.price_satang)}</span>
        </div>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            marginTop: 14,
            paddingTop: 14,
            borderTop: "1px solid var(--line)",
            fontSize: 16,
          }}
        >
          <span>ยอดชำระ</span>
          <span className="serif" style={{ fontWeight: 600 }}>
            {baht(pack.price_satang)}
          </span>
        </div>
      </div>

      <div style={{ marginTop: 26 }}>
        <div className="eyebrow" style={{ marginBottom: 10 }}>
          ช่องทางชำระเงิน
        </div>
        <Tabs<Method>
          active={method}
          onChange={setMethod}
          tabs={[
            { key: "card", label: "บัตรเครดิต / เดบิต" },
            { key: "promptpay", label: "พร้อมเพย์ (QR)" },
            { key: "truemoney", label: "ทรูมันนี่ วอลเล็ท" },
          ]}
        />
      </div>

      <div className="card" style={{ marginTop: 18 }}>
        {method === "card" && (
          <div className="grid" style={{ gap: 14 }}>
            <div className="muted" style={{ fontSize: 11.5 }}>
              VISA · MASTERCARD · JCB
            </div>
            <label className="field">
              <span className="field__label">หมายเลขบัตร</span>
              <input className="input" inputMode="numeric" placeholder="0000 0000 0000 0000" />
            </label>
            <div style={{ display: "flex", gap: 12 }}>
              <label className="field" style={{ flex: 1 }}>
                <span className="field__label">วันหมดอายุ</span>
                <input className="input" placeholder="MM/YY" />
              </label>
              <label className="field" style={{ flex: 1 }}>
                <span className="field__label">รหัสหลังบัตร</span>
                <input className="input" inputMode="numeric" placeholder="CVV" />
              </label>
            </div>
            <label className="field">
              <span className="field__label">ชื่อบนบัตร</span>
              <input className="input" />
            </label>
          </div>
        )}

        {method === "promptpay" && (
          <div style={{ textAlign: "center" }}>
            <div
              className="cover"
              style={{ width: 176, height: 176, margin: "0 auto", borderRadius: "var(--r2)" }}
            >
              <span className="muted" style={{ fontSize: 12 }}>
                QR พร้อมเพย์
              </span>
            </div>
            <div className="muted" style={{ fontSize: 12.5, marginTop: 14, lineHeight: 1.9 }}>
              สแกนด้วยแอปธนาคารเพื่อชำระเงิน
              <br />
              ชื่อบัญชี บจก. หมอกจันทร์ ดิจิทัล
            </div>
          </div>
        )}

        {method === "truemoney" && (
          <label className="field">
            <span className="field__label">เบอร์โทรที่ผูกกับวอลเล็ท</span>
            <input className="input" inputMode="tel" placeholder="08X-XXX-XXXX" />
            <span className="muted" style={{ fontSize: 11.5, marginTop: 6 }}>
              ยืนยันด้วย OTP
            </span>
          </label>
        )}
      </div>

      {pay.isError && <ErrorNote message={(pay.error as Error).message} />}

      <button
        className="btn btn--primary btn--block"
        style={{ marginTop: 20 }}
        onClick={() => pay.mutate()}
        disabled={pay.isPending}
      >
        {pay.isPending ? "กำลังดำเนินการ…" : `ชำระ ${baht(pack.price_satang)}`}
      </button>

      <div className="muted" style={{ fontSize: 11.5, marginTop: 14, lineHeight: 1.9 }}>
        ระบบชำระเงินจำลองสำหรับการพัฒนา ยังไม่เชื่อมต่อผู้ให้บริการจริง
        ข้อมูลบัตรในหน้านี้ไม่ถูกส่งหรือจัดเก็บ
      </div>
    </section>
  );
}
