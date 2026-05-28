const fs = require('fs');

let content = fs.readFileSync('beancs-controller/internal/web/src/components/index.jsx', 'utf8');
content = content.replace(/<select/g, '<Select');
content = content.replace(/<input/g, '<Input');
content = content.replace(/<button/g, '<Button');
content = content.replace(/<textarea/g, '<Textarea');
fs.writeFileSync('beancs-controller/internal/web/src/components/index.jsx', content);
console.log('Fixed select/input/button/textarea');
