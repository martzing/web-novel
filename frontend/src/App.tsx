import { Navigate, Route, Routes } from "react-router-dom";

import Shell from "./layout/Shell";
import { Login, Register } from "./routes/Auth";
import Browse from "./routes/Browse";
import Checkout from "./routes/Checkout";
import Coins from "./routes/Coins";
import Comments from "./routes/Comments";
import Home from "./routes/Home";
import Library from "./routes/Library";
import Novel from "./routes/Novel";
import Onboarding from "./routes/Onboarding";
import Reader from "./routes/Reader";
import Series from "./routes/Series";
import Stats from "./routes/Stats";
import Works from "./routes/Works";
import Write from "./routes/Write";

export default function App() {
  return (
    <Routes>
      {/* The reader owns the whole viewport, so it sits outside the shell. */}
      <Route path="/read/:id" element={<Reader />} />

      <Route element={<Shell />}>
        <Route path="/" element={<Home />} />
        <Route path="/browse" element={<Browse />} />
        <Route path="/novels/:slug" element={<Novel />} />
        <Route path="/series/:id" element={<Series />} />
        <Route path="/chapters/:id/comments" element={<Comments />} />
        <Route path="/library" element={<Library />} />
        <Route path="/coins" element={<Coins />} />
        <Route path="/checkout/:packId" element={<Checkout />} />
        <Route path="/works" element={<Works />} />
        <Route path="/write" element={<Write />} />
        <Route path="/stats" element={<Stats />} />
        <Route path="/onboarding" element={<Onboarding />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/account" element={<Navigate to="/library" replace />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  );
}

function NotFound() {
  return (
    <section>
      <h1 className="page-title">ไม่พบหน้านี้</h1>
      <p className="muted" style={{ fontSize: 13.5, marginTop: 14, lineHeight: 2 }}>
        ลิงก์อาจหมดอายุหรือพิมพ์ผิด ลองกลับไปที่หน้าแรก
      </p>
    </section>
  );
}
