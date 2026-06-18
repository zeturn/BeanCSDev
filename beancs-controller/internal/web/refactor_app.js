const fs = require('fs');
let content = fs.readFileSync('src/App.jsx', 'utf8');

// Replace view state and add Router
content = content.replace(
  'import { createRoot } from "react-dom/client";',
  'import { createRoot } from "react-dom/client";\nimport { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from "react-router-dom";'
);

content = content.replace(
  'const [view, setView] = useState("dashboard");',
  'const navigate = useNavigate();\n  const location = useLocation();\n  const view = location.pathname === "/" ? "dashboard" : location.pathname.substring(1);\n  function setView(v) { navigate("/" + v); }'
);

// We need to wrap <App /> in <BrowserRouter> when rendering
content = content.replace(
  'createRoot(document.getElementById("root")).render(<App />);',
  'createRoot(document.getElementById("root")).render(<BrowserRouter><App /></BrowserRouter>);'
);

// We also need to change `{view === "dashboard" && ...}` into `<Routes><Route path="/dashboard" element={<DashboardView ... />} />...`
// But wait, the prompt says "拆分为多个页面文件" (split into multiple page files).
// I need to create page files. I will do that via bash tools.
