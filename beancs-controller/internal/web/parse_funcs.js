const fs = require('fs');
const content = fs.readFileSync('src/App.jsx', 'utf8');

const appBodyStart = content.indexOf('function App() {');
const returnStart = content.indexOf('return (', appBodyStart);
const appBody = content.substring(appBodyStart, returnStart);

const lines = appBody.split('\n');
const funcs = [];

for (const line of lines) {
  const matchFunc = line.match(/async\s+function\s+([a-zA-Z0-9_]+)/) || line.match(/function\s+([a-zA-Z0-9_]+)/);
  if (matchFunc) funcs.push(matchFunc[1]);
}

console.log("Functions:", funcs.join(', '));
