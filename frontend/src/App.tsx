import { Route, Routes } from "react-router-dom";

import Shell from "./layout/Shell";
import Browse from "./routes/Browse";
import Home from "./routes/Home";
import Novel from "./routes/Novel";
import Reader from "./routes/Reader";
import Stub from "./routes/Stub";

export default function App() {
  return (
    <Routes>
      <Route path="/read/:id" element={<Reader />} />
      <Route element={<Shell />}>
        <Route path="/" element={<Home />} />
        <Route path="/browse" element={<Browse />} />
        <Route path="/novels/:slug" element={<Novel />} />
        <Route path="/library" element={<Stub title="ชั้นหนังสือ" />} />
        <Route path="/coins" element={<Stub title="เหรียญ" />} />
        <Route path="/write" element={<Stub title="เขียนบท" />} />
        <Route path="/stats" element={<Stub title="สถิติผลงาน" />} />
        <Route path="*" element={<Stub title="ไม่พบหน้านี้" />} />
      </Route>
    </Routes>
  );
}
