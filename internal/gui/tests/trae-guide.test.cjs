// An app magpie can't write at all — Trae and Trae CN keep their models on
// the vendor's own cloud (#4365, #2907) — is a row like the others, but its
// one control opens the values that app's own Add Model dialog is given,
// each with a copy button, where the others' model picker stands. Nothing is
// written, so the row never says magpie is added, in any language.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

const ENDPOINT = "http://127.0.0.1:3425/v1/chat/completions";
const guide = (endpoint, model) => [
  { label: "Endpoint", value: endpoint },
  { label: "API key", value: "magpie-trae-cn" },
  model ? { label: "Model", value: model } : { label: "Model", note: "Any model id magpie lists" },
];
// Trae CN as magpie lists it, its models on the vendor's cloud; Trae with
// nothing listed for it, so its model entry is the note
const traeCN = { id: "trae-cn", name: "Trae CN", path: "/test/Trae CN", icon: "trae", guide: guide(ENDPOINT, "deepseek/deepseek-v4.1-flash"), fields: [] };
const trae = { id: "trae", name: "Trae", path: "/test/Trae", icon: "trae", guide: guide(ENDPOINT, ""), fields: [] };
// connected, so the guided rows aren't the only ones in sight
const crush = {
  id: "crush", name: "Crush", path: "/test/crush.json", icon: "crush", wired: true,
  fields: [{ key: "model", label: "model", value: "magpie/deepseek-flash", options: [{ value: "magpie/deepseek-flash", label: "DeepSeek Flash", ref: "deepseek/deepseek-flash" }] }],
};

async function serve(route, lang) {
  const url = new URL(route.request().url());
  const json = (data) => route.fulfill({ json: data });
  if (url.pathname === "/boot.js") {
    return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false,textSize:100};` });
  }
  if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
  if (url.pathname === "/api/state") return json({ agents: [crush, traeCN, trae], profiles: [], settings: { lang, theme: "light", textSize: 100 } });
  if (url.pathname === "/api/providers") return json({ providers: [], gateway: { running: true, window: true } });
  if (url.pathname === "/api/plugins") return json({ plugins: [] });
  if (url.pathname.startsWith("/api/")) return json({});
  const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
  const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
  await route.fulfill({ body: await fs.readFile(file), contentType });
}

// the row's line, and the button, in the reader's language
const SAID = {
  en: [/^Added by hand · Trae CN keeps its models on the vendor's servers/],
  zh: [/^手动添加 · Trae CN 的模型保存在其厂商的服务器上/],
  "zh-TW": [/^手動新增 · Trae CN 的模型保存在其廠商的伺服器上/],
  ja: [/^手動で追加 · Trae CN のモデルはベンダーのサーバーにあります/],
  de: [/^Von Hand hinzugefügt · Trae CN hält seine Modelle auf den Servern des Anbieters/],
};
const BUTTON = { en: "Add by hand", zh: "手动添加", "zh-TW": "手動新增", ja: "手動で追加", de: "Von Hand hinzufügen" };
// the dialog's own words
const TITLE = { en: "Add magpie to Trae CN", zh: "为 Trae CN 添加 magpie", "zh-TW": "為 Trae CN 新增 magpie", ja: "Trae CN に magpie を追加", de: "magpie zu Trae CN hinzufügen" };
const STEP = {
  en: "Trae CN keeps its models on its own servers, so magpie can't write them. In Trae CN, open Add Model, pick Custom (OpenAI Compatible), and paste these:",
  zh: "Trae CN 的模型保存在它自己的服务器上，magpie 无法写入。在 Trae CN 中打开「添加模型」，选择 Custom (OpenAI Compatible)，然后粘贴以下内容：",
  "zh-TW": "Trae CN 的模型保存在它自己的伺服器上，magpie 無法寫入。在 Trae CN 中開啟「新增模型」，選擇 Custom (OpenAI Compatible)，然後貼上以下內容：",
  ja: "Trae CN のモデルはベンダー自身のサーバーにあり、magpie からは書き込めません。Trae CN で「モデルを追加」を開き、Custom (OpenAI Compatible) を選んで次を貼り付けてください：",
  de: "Trae CN hält seine Modelle auf den eigenen Servern des Anbieters, magpie kann sie nicht schreiben. Öffnen Sie in Trae CN „Modell hinzufügen“, wählen Sie Custom (OpenAI Compatible) und fügen Sie dies ein:",
};
const NOTE = { en: "Any model id magpie lists", zh: "magpie 列表中任意模型的 id", "zh-TW": "magpie 列表中任意模型的 id", ja: "magpie が一覧するモデル id のいずれか", de: "Eine beliebige Modell-Kennung aus magpies Liste" };

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of Object.keys(SAID)) {
    test(`${engine} ${lang}: a guided Trae row opens what its Add Model dialog is given`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      // the row's and the dialog's own motion settled: a shape still moving
      // is one a click waits on
      const page = await (await browser.newContext({ viewport: { width: 960, height: 600 }, reducedMotion: "reduce" })).newPage();
      t.after(async () => {
        if (process.env.ARTIFACT_DIR) {
          await fs.mkdir(process.env.ARTIFACT_DIR, { recursive: true });
          await page.screenshot({ path: path.join(process.env.ARTIFACT_DIR, `${engine}-${lang}-trae-guide.png`) });
        }
        await browser.close();
      });
      page.setDefaultTimeout(15000); // a busy machine takes seconds over a click
      page.setDefaultNavigationTimeout(60000);
      await page.route("**/*", (route) => serve(route, lang));
      await page.goto("http://magpie.test/");
      await page.waitForLoadState("networkidle");
      const more = page.locator('.agent-more[aria-expanded="false"]');
      if (await more.count()) await more.click();

      const row = page.locator('.row.agent[data-id="trae-cn"]');
      await row.locator(".ag-st-t").waitFor();
      // its line says added by hand, and never that magpie is added
      const said = await row.locator(".ag-st-t").innerText();
      assert.match(said, SAID[lang][0]);
      assert.equal(await row.locator(".ag-st").evaluate((e) => e.classList.contains("on")), false);
      // nothing to switch: a guided row has no on/off and no picker
      assert.equal(await row.locator('.lib-switch[role="switch"]').count(), 0, "a guided row has no switch");
      // its button stands where the others' picker does, as wide
      for (const width of [960, 700]) {
        await page.setViewportSize({ width, height: 600 });
        const box = (sel) => page.locator(sel).evaluate((e) => { const r = e.getBoundingClientRect(); return { x: r.x, right: r.right, width: r.width }; });
        let pick, btn;
        for (const until = Date.now() + 3000; ;) {
          pick = await box('.row.agent[data-id="crush"] > .ag-start');
          btn = await box('.row.agent[data-id="trae-cn"] > .ag-start');
          if ((Math.abs(pick.right - btn.right) < 1 && Math.abs(pick.width - btn.width) < 1) || Date.now() > until) break;
          await page.waitForTimeout(50);
        }
        assert(Math.abs(pick.right - btn.right) < 1 && Math.abs(pick.width - btn.width) < 1, `the guided button lines up with the model picker at ${width}px: ${JSON.stringify({ pick, btn })}`);
      }
      await page.setViewportSize({ width: 960, height: 600 });

      // the button opens the dialog, in the reader's language, with the
      // values Trae's own Add Model dialog asks for — and no request to
      // any app: nothing is written anywhere
      let opened = null;
      page.on("request", (r) => { if (r.url().includes("/api/open") || r.url().includes("/api/agents/")) opened = r.url(); });
      await row.locator(".ag-start").click();
      const dialog = page.locator("#modal .guide-ask");
      await dialog.waitFor();
      assert.equal(await dialog.locator(".ehead b").innerText(), TITLE[lang]);
      assert.equal(await dialog.locator(".lib-confirm").innerText(), STEP[lang]);
      const kvs = dialog.locator(".ag-kv");
      assert.equal(await kvs.count(), 3, "endpoint, key and model");
      assert.equal(await kvs.nth(0).locator("code").innerText(), ENDPOINT);
      assert.equal(await kvs.nth(1).locator("code").innerText(), "magpie-trae-cn");
      assert.equal(await kvs.nth(2).locator("code").innerText(), "deepseek/deepseek-v4.1-flash");
      assert.equal(opened, null, "opening the guide writes to nothing");
      // each value copies
      const copied = page.waitForRequest((r) => r.url().includes("/api/copy"));
      await kvs.nth(0).locator("button").click();
      assert.equal(JSON.parse((await copied).postData()).text, ENDPOINT);
      // Done closes it
      await dialog.locator(".bar button").click();
      await page.locator("#modal").waitFor({ state: "hidden" });

      // Trae with no model listed for it names none: a note instead, since
      // a model the gateway doesn't know is one it turns away
      const intl = page.locator('.row.agent[data-id="trae"]');
      await intl.locator(".ag-st-t").waitFor();
      assert.match(await intl.locator(".ag-st-t").innerText(), new RegExp(SAID[lang][0].source.replace("Trae CN", "Trae")));
      await intl.locator(".ag-start").click();
      const second = page.locator("#modal .guide-ask");
      await second.waitFor();
      assert.equal(await second.locator(".ag-kv").nth(2).locator("code").count(), 0, "no model listed: no name");
      assert.equal(await second.locator(".ag-kv").nth(2).locator(".ag-hint").innerText(), NOTE[lang]);
      await second.locator(".bar button").click();
      await page.locator("#modal").waitFor({ state: "hidden" });

      // narrow: no page scrolls sideways and the button stays inside
      await page.setViewportSize({ width: 420, height: 700 });
      await page.waitForFunction(() => document.documentElement.scrollWidth <= window.innerWidth, null, { timeout: 3000 }).catch(async () => {
        assert.fail(`sideways scroll at 420px: ${await page.evaluate(() => document.documentElement.scrollWidth)}`);
      });
      for (const id of ["trae", "trae-cn"]) {
        const r = page.locator(`.row.agent[data-id="${id}"]`);
        const b = await r.locator(".ag-start").evaluate((e) => { const x = e.getBoundingClientRect(); return { left: x.left, right: x.right, width: x.width }; });
        assert(b.width > 0 && b.left >= 0 && b.right <= 420.5, `${id}'s button inside the window at 420px: ${JSON.stringify(b)}`);
      }
    });
  }
}
