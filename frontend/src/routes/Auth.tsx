import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { ApiError } from "../lib/api";
import { useAuth } from "../lib/auth";

/**
 * Sign-in and sign-up.
 *
 * No mockup exists for these screens, so they are built from the same tokens
 * and component set as the rest of the platform.
 */
export function Login() {
  return <AuthForm mode="login" />;
}

export function Register() {
  return <AuthForm mode="register" />;
}

function AuthForm({ mode }: { mode: "login" | "register" }) {
  const { login, register } = useAuth();
  const navigate = useNavigate();

  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const isRegister = mode === "register";

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      if (isRegister) {
        await register(username.trim(), email.trim(), password);
        navigate("/onboarding");
      } else {
        await login(email.trim(), password);
        navigate("/");
      }
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section style={{ maxWidth: 400 }}>
      <div className="eyebrow">หมอกจันทร์</div>
      <h1 className="page-title" style={{ marginTop: 8 }}>
        {isRegister ? "สมัครสมาชิก" : "เข้าสู่ระบบ"}
      </h1>

      <form onSubmit={onSubmit} className="grid" style={{ gap: 14, marginTop: 24 }}>
        {isRegister && (
          <label className="field">
            <span className="field__label">ชื่อผู้ใช้</span>
            <input
              className="input"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              required
            />
          </label>
        )}

        <label className="field">
          <span className="field__label">อีเมล</span>
          <input
            className="input"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
            required
          />
        </label>

        <label className="field">
          <span className="field__label">รหัสผ่าน</span>
          <input
            className="input"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete={isRegister ? "new-password" : "current-password"}
            required
          />
          {isRegister && (
            <span className="muted" style={{ fontSize: 11.5 }}>
              อย่างน้อย 8 ตัวอักษร
            </span>
          )}
        </label>

        {error && <div className="form-error">{error}</div>}

        <button className="btn btn--primary btn--block" type="submit" disabled={busy}>
          {busy ? "กำลังดำเนินการ…" : isRegister ? "สมัครสมาชิก" : "เข้าสู่ระบบ"}
        </button>
      </form>

      <div className="muted" style={{ fontSize: 13, marginTop: 20 }}>
        {isRegister ? (
          <>
            มีบัญชีอยู่แล้ว? <Link to="/login">เข้าสู่ระบบ</Link>
          </>
        ) : (
          <>
            ยังไม่มีบัญชี? <Link to="/register">สมัครสมาชิก</Link>
          </>
        )}
      </div>
    </section>
  );
}

/** Maps the server's error codes to Thai copy the reader can act on. */
function message(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case "INVALID_CREDENTIALS":
        return "อีเมลหรือรหัสผ่านไม่ถูกต้อง";
      case "EMAIL_TAKEN":
        return "อีเมลนี้ถูกใช้งานแล้ว";
      case "USERNAME_TAKEN":
        return "ชื่อผู้ใช้นี้ถูกใช้งานแล้ว";
      case "WEAK_PASSWORD":
        return "รหัสผ่านต้องยาวอย่างน้อย 8 ตัวอักษร";
      case "INVALID_USERNAME":
        return "ชื่อผู้ใช้ต้องยาว 3–32 ตัว ใช้ได้เฉพาะ a-z 0-9 . _ -";
      case "INVALID_EMAIL":
        return "รูปแบบอีเมลไม่ถูกต้อง";
      case "RATE_LIMITED":
        return "พยายามบ่อยเกินไป กรุณารอสักครู่";
      default:
        return err.message;
    }
  }
  return "เกิดข้อผิดพลาด กรุณาลองใหม่";
}
