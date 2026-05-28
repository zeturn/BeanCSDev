const fs = require('fs');

let content = fs.readFileSync('beancs-controller/internal/web/src/components/index.jsx', 'utf8');

// replace <button, <input, <select, <textarea with uppercase versions, handling cases.
content = content.replace(/<button /g, '<Button ');
content = content.replace(/<\/button>/g, '</Button>');
content = content.replace(/<input /g, '<Input ');
content = content.replace(/<\/input>/g, '</Input>');
content = content.replace(/<select /g, '<Select ');
content = content.replace(/<\/select>/g, '</Select>');
content = content.replace(/<textarea /g, '<Textarea ');
content = content.replace(/<\/textarea>/g, '</Textarea>');
content = content.replace(/<button\n/g, '<Button\n');

// Also fix button variant
content = content.replace(/<Button\s+className="([^"]*?)(primary|danger|ghost|icon)([^"]*?)"/g, (match, before, variant, after) => {
    let classes = `${before}${after}`.trim();
    if (classes) {
        return `<Button variant="${variant}" className="${classes}"`;
    }
    return `<Button variant="${variant}"`;
});


fs.writeFileSync('beancs-controller/internal/web/src/components/index.jsx', content);
console.log('Updated index.jsx');
