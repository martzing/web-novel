export default function Stub({ title }: { title: string }) {
  return (
    <section style={{ maxWidth: 720 }}>
      <h1
        className="serif"
        style={{ fontSize: 28, fontWeight: 600, margin: 0 }}
      >
        {title}
      </h1>
      <p
        style={{
          marginTop: 18,
          color: "var(--soft)",
          fontSize: 13.5,
          lineHeight: 2,
        }}
      >
        หน้านี้ยังไม่พร้อมใช้งานในระยะที่ 1 จะเปิดใช้งานพร้อมระบบสมาชิก / เหรียญ
        / เขียนบท ในระยะถัดไป
      </p>
    </section>
  );
}
