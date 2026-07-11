import React from "react";
import { useI18n } from "./index";

export function LanguageSwitcher() {
  const { lang, setLang } = useI18n();
  return (
    <div className="lang-switcher" role="group" aria-label="Language">
      <button
        type="button"
        className={lang === "en" ? "active" : ""}
        aria-pressed={lang === "en"}
        onClick={() => setLang("en")}
      >
        EN
      </button>
      <button
        type="button"
        className={lang === "zh" ? "active" : ""}
        aria-pressed={lang === "zh"}
        onClick={() => setLang("zh")}
      >
        中文
      </button>
    </div>
  );
}
