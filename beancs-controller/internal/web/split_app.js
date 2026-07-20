const fs = require('fs');

let appJsx = fs.readFileSync('src/App.jsx', 'utf8');

appJsx = appJsx.replace(
  'import { createRoot } from "react-dom/client";',
  'import { createRoot } from "react-dom/client";\nimport { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from "react-router-dom";'
);

appJsx = appJsx.replace('function App() {', 'function AppShell() {\n  const navigate = useNavigate();\n  const location = useLocation();\n  const view = location.pathname === "/" ? "dashboard" : location.pathname.substring(1);\n  function setView(v) { navigate("/" + v); }');

appJsx = appJsx.replace('  const [view, setView] = useState("dashboard");\n', '');

const skeletonStart = appJsx.indexOf('{shouldShowSkeleton(view, dashboard, network) ? (');
const skeletonEnd = appJsx.indexOf(') : (', skeletonStart);
const mainEnd = appJsx.indexOf('</main>', skeletonEnd);

if (skeletonStart > 0 && mainEnd > 0) {
  let mainContent = appJsx.substring(skeletonEnd + 5, mainEnd);

  // Clean replacement for `<> ... </>` -> `<Routes> ... </Routes>`
  mainContent = mainContent.replace('<>', '<Routes>');

  // Dashboard is a bit different in formatting, replace it directly:
  mainContent = mainContent.replace(
    '{view === "dashboard" && (\n              <DashboardView dashboard={dashboard} />\n            )}',
    '<Route path="/dashboard" element={<DashboardView dashboard={dashboard} />} />'
  );

  // Settings
  mainContent = mainContent.replace(
    '{view === "settings" && <SettingsView version={appVersion} />}',
    '<Route path="/settings" element={<SettingsView version={appVersion} />} />'
  );

  // Domains
  mainContent = mainContent.replace(
    '{view === "domains" && <DomainsView domains={domains} />}',
    '<Route path="/domains" element={<DomainsView domains={domains} />} />'
  );

  // For the standard multi-line `{view === "..." && (\n  <...View ... /> \n)}`
  // We can use a regex that matches exactly `\n            )}`
  mainContent = mainContent.replace(/\{view === "([a-zA-Z]+)" && \(\s+([\s\S]*?)\s+\)\}/g, '<Route path="/$1" element={$2} />');

  // The array includes for runtime tables
  mainContent = mainContent.replace(
    '{["namespaces", "pods", "nodes", "ingresses", "services"].includes(\n              view,\n            ) && (\n              <RuntimeTable',
    '{["namespaces", "pods", "nodes", "ingresses", "services"].map(kind => (\n<Route key={kind} path={`/${kind}`} element={\n              <RuntimeTable'
  );
  mainContent = mainContent.replace(
    'onDetail={setRuntimeDetail}\n              />\n            )}',
    'onDetail={setRuntimeDetail}\n              />\n            } />\n))}'
  );

  mainContent = mainContent.replace('</>', '<Route path="*" element={<Navigate to="/dashboard" replace />} />\n          </Routes>');

  appJsx = appJsx.substring(0, skeletonEnd + 5) + mainContent + appJsx.substring(mainEnd);
}

appJsx = appJsx.replace(
  'createRoot(document.getElementById("root")).render(<App />);',
  'function App() { return <BrowserRouter><AppShell /></BrowserRouter>; }\ncreateRoot(document.getElementById("root")).render(<App />);'
);

fs.writeFileSync('src/App.jsx', appJsx);
console.log("App.jsx refactored.");
