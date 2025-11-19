const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 }
  });

  const page = await context.newPage();

  try {
    // Navigate to login page
    await page.goto('http://localhost:9000/admin/login');
    await page.waitForTimeout(1000);

    // Login
    await page.fill('input[name=username]', 'admin');
    await page.fill('input[name=password]', 'listmonk');
    await page.click('button[type=submit]');

    // Wait for navigation to admin
    await page.waitForTimeout(3000);

    // Navigate to dashboard
    await page.goto('http://localhost:9000/admin');
    await page.waitForTimeout(2000);

    // Click user dropdown to open menu
    await page.click('.navbar-end .user');
    await page.waitForTimeout(1000);

    // Wait for Dark Mode menu item to be visible
    await page.waitForSelector('text=Dark Mode', { state: 'visible' });

    // Take screenshot with dropdown menu open
    await page.screenshot({
      path: '/home/user/listmonk/screenshots/dark-mode-toggle-menu-playwright.png',
      fullPage: false
    });

    console.log('Screenshot saved successfully!');

  } catch (error) {
    console.error('Error taking screenshot:', error);
  } finally {
    await browser.close();
  }
})();
