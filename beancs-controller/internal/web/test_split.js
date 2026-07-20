const fs = require('fs');
const content = fs.readFileSync('src/App.jsx', 'utf8');

const appBodyStart = content.indexOf('function App() {');
const returnStart = content.indexOf('return (\n    <div className="app-shell">');

const imports = content.substring(0, appBodyStart);
const logic = content.substring(appBodyStart + 'function App() {'.length, returnStart);
const render = content.substring(returnStart);

fs.writeFileSync('test_logic.js', logic);
fs.writeFileSync('test_render.js', render);
console.log("Split test successful.");
