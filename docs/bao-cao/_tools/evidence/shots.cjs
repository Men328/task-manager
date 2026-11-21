// Chụp ảnh giao diện cho Chương 4 bằng Chromium headless.
// node shots.cjs <token_file> <out_dir>
// Cần puppeteer-core (NODE_PATH trỏ tới node_modules) và Chromium tại CHROME (mặc định /snap/bin/chromium).
const fs = require('fs');
const path = require('path');
const puppeteer = require('puppeteer-core');

const [tokenFile, outDir] = process.argv.slice(2);
const TOKEN = fs.readFileSync(tokenFile, 'utf8').trim();
const BASE = process.env.APP_URL || 'http://localhost:3000';
const PEER = 'bbffa35b-d862-485b-a49f-6e828e10dc76'; // 22010328@st.phenikaa-uni.edu.vn
const FOCUS = process.env.FOCUS_TASK || '';
const PROJECT = process.env.PROJECT_ID || '';

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const ONLY = process.env.ONLY || '';

(async () => {
  const browser = await puppeteer.launch({
    executablePath: process.env.CHROME || '/snap/bin/chromium',
    args: ['--no-sandbox', '--disable-gpu', '--lang=vi-VN'],
    defaultViewport: { width: 1440, height: 900, deviceScaleFactor: 2 },
  });

  async function shot(name, url, { auth = true, action, wait = 2500 } = {}) {
    if (ONLY && !name.includes(ONLY)) return;
    const page = await browser.newPage();
    await page.evaluateOnNewDocument((token, auth, project) => {
      localStorage.setItem('tm-language', 'vi');
      if (project) localStorage.setItem('tm.currentProjectId', project);
      if (auth) localStorage.setItem('tm-session-token', token);
      else localStorage.removeItem('tm-session-token');
    }, TOKEN, auth, PROJECT);
    await page.goto(BASE + url, { waitUntil: 'networkidle2', timeout: 60000 });
    await sleep(wait);
    if (action) await action(page);
    const file = path.join(outDir, `${name}.png`);
    await page.screenshot({ path: file });
    console.log('saved', file);
    await page.close();
  }

  await shot('h4-ui-01-dang-nhap', '/login', { auth: false });
  await shot('h4-ui-02-kanban', '/');
  await shot('h4-ui-03-chi-tiet-task', '/', {
    action: async (page) => {
      await page.click('::-p-text(Viết Chương 4 - Thiết kế hệ thống)');
      await sleep(2500);
    },
  });
  await shot('h4-ui-04-hom-nay', '/today');
  await shot('h4-ui-05-du-an', '/projects');
  await shot('h4-ui-06-lich', '/calendar');
  await shot('h4-ui-07-su-kien', '/events');
  await shot('h4-ui-08-backlog', '/backlog');
  await shot('h4-ui-09-bao-cao', '/reports', { wait: 4000 });
  await shot('h4-ui-10-thong-bao', '/', {
    action: async (page) => {
      await page.click('[aria-label="Thông báo"]');
      await sleep(1500);
    },
  });
  await shot('h4-ui-11-ban-be', '/friends');
  await shot('h4-ui-12-tin-nhan', `/messages/${PEER}`, { wait: 3500 });

  await browser.close();
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
