const fs = require('fs');
const path = require('path');

function replaceFile(filePath, replacer) {
    let content = fs.readFileSync(filePath, 'utf8');
    let newContent = replacer(content);
    if (content !== newContent) {
        fs.writeFileSync(filePath, newContent);
        console.log(`Updated ${filePath}`);
    }
}

// Fix Button variant
function fixButtonVariant(content) {
    return content.replace(/<Button\s+className="([^"]*?)(primary|danger|ghost|icon)([^"]*?)"/g, (match, before, variant, after) => {
        let classes = `${before}${after}`.trim();
        if (classes) {
            return `<Button variant="${variant}" className="${classes}"`;
        }
        return `<Button variant="${variant}"`;
    });
}

function processDir(dir) {
    for (const file of fs.readdirSync(dir)) {
        const fullPath = path.join(dir, file);
        if (fs.statSync(fullPath).isDirectory()) {
            processDir(fullPath);
        } else if (fullPath.endsWith('.jsx')) {
            replaceFile(fullPath, fixButtonVariant);
        }
    }
}

processDir('beancs-controller/internal/web/src/views');
processDir('beancs-controller/internal/web/src/components');
