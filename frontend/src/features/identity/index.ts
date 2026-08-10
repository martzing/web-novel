export { identityApi } from "./api";
export type { AuthResponse, GenrePref, Prefs, User } from "./api";
export { AuthProvider, useAuth } from "./auth";
export { DEFAULT_PREFS, usePrefs } from "./prefs";
export { identityKeys, useGenrePrefs, useSaveGenrePrefs } from "./queries";
export { Login, Register } from "./pages/Auth";
export { default as Onboarding } from "./pages/Onboarding";
