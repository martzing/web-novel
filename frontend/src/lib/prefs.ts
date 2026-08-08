import { useCallback, useEffect, useState } from "react";

import { api, type Prefs } from "./api";
import { useAuth } from "./auth";

const STORAGE_KEY = "mokchan.prefs";

/** Matches the column defaults in 0001_init.sql. */
export const DEFAULT_PREFS: Prefs = {
  theme: "light",
  font: "loop",
  font_size: 20,
  line_height: 2,
  column_width: "normal",
};

/**
 * Reader settings.
 *
 * They are written to localStorage immediately so the reader never flickers,
 * and synced to the server for signed-in users so a second device picks up the
 * same settings. An anonymous reader keeps working entirely locally.
 */
export function usePrefs() {
  const { user } = useAuth();
  const [prefs, setPrefs] = useState<Prefs>(readLocal);

  // Server settings win on sign-in: they are the cross-device source of truth.
  useEffect(() => {
    if (!user) return;
    let cancelled = false;

    api
      .getPrefs()
      .then((remote) => {
        if (cancelled) return;
        setPrefs(remote);
        writeLocal(remote);
      })
      .catch(() => {
        // Offline or a transient failure: the local copy is still good.
      });

    return () => {
      cancelled = true;
    };
  }, [user]);

  const setPref = useCallback(
    <K extends keyof Prefs>(key: K, value: Prefs[K]) => {
      setPrefs((current) => {
        const next = { ...current, [key]: value };
        writeLocal(next);
        if (user) {
          api.setPrefs(next).catch(() => {});
        }
        return next;
      });
    },
    [user],
  );

  return { prefs, setPref };
}

function readLocal(): Prefs {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_PREFS;
    // Merge over the defaults so a stored object from an older version, or one
    // missing a key, still yields a complete settings object.
    return { ...DEFAULT_PREFS, ...(JSON.parse(raw) as Partial<Prefs>) };
  } catch {
    return DEFAULT_PREFS;
  }
}

function writeLocal(prefs: Prefs) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // Private browsing can refuse writes; settings simply do not persist.
  }
}
