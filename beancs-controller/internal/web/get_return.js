const fs = require('fs');
const content = fs.readFileSync('src/App.jsx', 'utf8');
const returnIndex = content.lastIndexOf('  return (\n    <div className="app-shell">');
console.log(content.substring(returnIndex));
