const path = require('path');

// Path to global playwright installation
const playwrightPath = path.join('/usr/local/lib/node_modules/@playwright/mcp/node_modules/playwright');
console.log('Looking for playwright at:', playwrightPath);

try {
    const playwright = require(playwrightPath);
    console.log('Playwright module loaded successfully');
    
    // Check what browsers are available
    console.log('\nAvailable browser launchers:');
    console.log('chromium:', typeof playwright.chromium);
    console.log('firefox:', typeof playwright.firefox);
    console.log('webkit:', typeof playwright.webkit);
    
    // Check if chromium is a valid browser type
    if (playwright.chromium && playwright.chromium.name) {
        console.log('\nchromium browser name:', playwright.chromium.name);
    }
    
    // Try to get version info
    if (playwright._initializer && playwright._initializer.version) {
        console.log('Playwright version:', playwright._initializer.version);
    }
    
} catch (error) {
    console.error('Failed to load playwright:', error.message);
}