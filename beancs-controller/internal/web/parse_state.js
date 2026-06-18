const fs = require('fs');
const content = fs.readFileSync('src/App.jsx', 'utf8');

const appBodyStart = content.indexOf('function App() {');
const returnStart = content.indexOf('return (', appBodyStart);
const appBody = content.substring(appBodyStart, returnStart);

const lines = appBody.split('\n');
const states = [];
const refs = [];
const funcs = [];
const variables = [];

for (const line of lines) {
  if (line.includes('const [') && line.includes('useState')) {
    const match = line.match(/const \[\s*([a-zA-Z0-9_]+)/);
    if (match) states.push(match[1]);
  } else if (line.includes('useRef')) {
    const match = line.match(/const\s+([a-zA-Z0-9_]+)\s*=/);
    if (match) refs.push(match[1]);
  } else if (line.includes('useMemo')) {
    const match = line.match(/const\s+([a-zA-Z0-9_]+)\s*=/);
    if (match) variables.push(match[1]);
  } else if (line.includes('function ') || line.includes('const ') && line.includes('=')) {
    const matchFunc = line.match(/async\s+function\s+([a-zA-Z0-9_]+)/) || line.match(/function\s+([a-zA-Z0-9_]+)/);
    if (matchFunc) funcs.push(matchFunc[1]);
  }
}

console.log("States:", states.join(', '));
console.log("Refs:", refs.join(', '));
console.log("Variables:", variables.join(', '));
console.log("Functions:", funcs.join(', '));
