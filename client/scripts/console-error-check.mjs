#!/usr/bin/env node
/**
 * Console error checker — opens the app in headless Chromium,
 * navigates key UI paths, and reports any console.error / console.warn messages.
 *
 * Usage:
 *   npx puppeteer browsers install chrome
 *   node scripts/console-error-check.mjs [--base-url https://localhost:3000]
 */

import puppeteer from 'puppeteer';

const BASE_URL = process.argv.find((a) => a.startsWith('--base-url='))?.split('=')[1]
  || process.env.ARSENALE_BASE_URL
  || 'https://localhost:3000';

const ADMIN_EMAIL = process.env.ARSENALE_ADMIN_EMAIL || 'admin@example.com';
const ADMIN_PASSWORD = process.env.ARSENALE_ADMIN_PASSWORD || 'ArsenaleTemp91Qx';

const consoleMessages = [];
const delay = (ms) => new Promise((r) => setTimeout(r, ms));

async function run() {
  console.log(`[check] Base URL: ${BASE_URL}`);

  const browser = await puppeteer.launch({
    headless: true,
    ignoreHTTPSErrors: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--ignore-certificate-errors'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  page.on('console', (msg) => {
    const level = msg.type();
    if (level === 'error' || level === 'warning') {
      consoleMessages.push({ level, text: msg.text(), url: page.url() });
    }
  });

  page.on('pageerror', (err) => {
    consoleMessages.push({ level: 'pageerror', text: err.message, url: page.url() });
  });

  try {
    // 1. Login
    console.log('[check] Navigating to login...');
    await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle2', timeout: 30000 });
    await delay(3000);

    await page.screenshot({ path: '/tmp/arsenale-login-debug.png' });
    console.log(`[check] URL after load: ${page.url()}`);

    // Handle passkey-first flow — click "Use email and password instead" if present
    const passwordLink = await page.evaluateHandle(() => {
      const links = [...document.querySelectorAll('a, button')];
      return links.find((el) => el.textContent?.toLowerCase().includes('email and password')) || null;
    });
    if (passwordLink && passwordLink.asElement()) {
      console.log('[check] Clicking "email and password" link...');
      await passwordLink.asElement().click();
      await delay(2000);
    }

    // Try to find email input
    const emailInput = await page.$('#login-email')
      || await page.$('input[name="email"]')
      || await page.$('input[type="email"]');

    if (emailInput) {
      await emailInput.click({ clickCount: 3 });
      await emailInput.type(ADMIN_EMAIL, { delay: 20 });

      // Submit email step
      const btn1 = await page.$('button[type="submit"]');
      if (btn1) { await btn1.click(); await delay(2000); }

      // Look for password field
      const pwdInput = await page.$('#login-password') || await page.$('input[type="password"]');
      if (pwdInput) {
        await pwdInput.type(ADMIN_PASSWORD, { delay: 20 });
        const btn2 = await page.$('button[type="submit"]');
        if (btn2) {
          await btn2.click();
          await page.waitForNavigation({ waitUntil: 'networkidle2', timeout: 15000 }).catch(() => {});
        }
      }
    } else {
      console.log('[check] No email input found — may be already logged in or different flow');
    }

    console.log(`[check] After login: ${page.url()}`);
    await delay(3000);

    // 2. Dashboard
    console.log('[check] Checking dashboard...');
    await page.screenshot({ path: '/tmp/arsenale-dashboard-debug.png' });
    await delay(2000);

    // 3. Command palette
    console.log('[check] Opening command palette...');
    await page.keyboard.down('Control');
    await page.keyboard.press('k');
    await page.keyboard.up('Control');
    await delay(1500);
    await page.keyboard.press('Escape');
    await delay(500);

    // 4. Zoom
    console.log('[check] Testing zoom...');
    await page.keyboard.down('Control');
    await page.keyboard.press('Equal');
    await page.keyboard.up('Control');
    await delay(500);
    await page.keyboard.down('Control');
    await page.keyboard.press('0');
    await page.keyboard.up('Control');
    await delay(500);

    // 5. Final wait
    await delay(2000);
    console.log('[check] Done navigating.');
  } catch (err) {
    console.error(`[check] Navigation error: ${err.message}`);
  } finally {
    await browser.close();
  }

  // Report
  const errors = consoleMessages.filter((m) => m.level === 'error' || m.level === 'pageerror');
  const warnings = consoleMessages.filter((m) => m.level === 'warning');

  console.log('\n========== RESULTS ==========');
  console.log(`Errors:   ${errors.length}`);
  console.log(`Warnings: ${warnings.length}`);

  if (errors.length > 0) {
    console.log('\n--- ERRORS ---');
    for (const e of errors) {
      console.log(`  [${e.level}] ${e.text}`);
      console.log(`    at ${e.url}`);
    }
  }

  if (warnings.length > 0) {
    console.log('\n--- WARNINGS ---');
    for (const w of warnings) {
      console.log(`  [${w.level}] ${w.text}`);
      console.log(`    at ${w.url}`);
    }
  }

  if (errors.length > 0) {
    process.exit(1);
  }

  console.log('\n[check] No console errors found.');
}

run().catch((err) => {
  console.error(`[check] Fatal: ${err.message}`);
  process.exit(2);
});
