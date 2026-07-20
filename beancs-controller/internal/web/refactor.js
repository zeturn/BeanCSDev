const fs = require('fs');
const content = fs.readFileSync('src/App.jsx', 'utf8');

const returnIndex = content.lastIndexOf('return (');
const returnBlock = content.substring(returnIndex);

const viewNames = [
  "dashboard", "deploy", "dependencies", "progress", "projects",
  "applications", "deployments", "apiKeys", "registries", "workloadImage",
  "alerts", "events", "logs", "metrics", "settings", "github", "domains",
  "networking", "cloudflare"
];

console.log("Return block length:", returnBlock.length);
