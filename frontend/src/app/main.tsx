import React from "react";
import ReactDOM from "react-dom/client";

import "@/shared/styles/tokens.css";
import "@/shared/styles/base.css";
import "@/shared/styles/components.css";
import "@/shared/styles/reader.css";

import Providers from "./providers";
import Router from "./router";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Providers>
      <Router />
    </Providers>
  </React.StrictMode>,
);
