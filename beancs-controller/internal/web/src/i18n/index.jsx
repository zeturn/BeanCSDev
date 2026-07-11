import React, {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  useMemo,
} from "react";
import { zh } from "./translations";

const STORAGE_KEY = "beancs.lang";

// Module-level language state. Kept in sync by <I18nProvider> so that plain
// (non-component) helper functions such as formatters in utils/index.js can
// call `t()` during render and still pick up the active language.
let currentLang = "en";

function interpolate(text, vars) {
  if (!vars) return text;
  return String(text).replace(/\{(\w+)\}/g, (match, key) =>
    vars[key] !== undefined && vars[key] !== null ? String(vars[key]) : match,
  );
}

// Translate `key`. English source strings are used as the fallback key, so any
// key absent from the `zh` dictionary renders in English.
export function t(key, vars) {
  const value =
    currentLang === "zh" && zh[key] !== undefined ? zh[key] : key;
  return interpolate(value, vars);
}

const I18nContext = createContext({ lang: "en", setLang: () => {}, t });

export function I18nProvider({ children }) {
  const [lang, setLangState] = useState(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved === "en" || saved === "zh") return saved;
    } catch (err) {
      // ignore storage access errors
    }
    return (navigator.language || "en").toLowerCase().startsWith("zh")
      ? "zh"
      : "en";
  });

  useEffect(() => {
    currentLang = lang;
    try {
      localStorage.setItem(STORAGE_KEY, lang);
    } catch (err) {
      // ignore storage access errors
    }
    document.documentElement.setAttribute(
      "lang",
      lang === "zh" ? "zh-CN" : "en",
    );
  }, [lang]);

  const setLang = useCallback((next) => setLangState(next), []);
  const value = useMemo(() => ({ lang, setLang, t }), [lang, setLang]);
  return (
    <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
  );
}

export function useI18n() {
  return useContext(I18nContext);
}
