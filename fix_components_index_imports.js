const fs = require('fs');
let content = fs.readFileSync('beancs-controller/internal/web/src/components/index.jsx', 'utf8');
content = content.replace(
  'export * from "./ui";',
  'export * from "./ui";\nimport { Button, Input, Select, Textarea, Checkbox, Modal, Drawer } from "./ui";'
);
fs.writeFileSync('beancs-controller/internal/web/src/components/index.jsx', content);
console.log('Fixed index.jsx imports');
