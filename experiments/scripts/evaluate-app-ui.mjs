#!/usr/bin/env node
import { spawn } from "node:child_process";
import { createRequire } from "node:module";
import { once } from "node:events";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { setTimeout as delay } from "node:timers/promises";

const workspace = process.argv[2];
const thirdArg = process.argv[3];
const artifactDir = thirdArg && !/^\d+$/.test(thirdArg) ? thirdArg : null;
const port = Number((artifactDir ? process.argv[4] : thirdArg) || "4300");

if (!workspace) {
  console.error("usage: evaluate-app-ui.mjs /path/to/workspace [port]");
  process.exit(2);
}

const result = {
  workspace,
  artifact_dir: artifactDir,
  port,
  started_at: new Date().toISOString(),
  server_started: false,
  browser_available: false,
  passed: false,
  pages: [],
  workflows: [],
  screenshot_dir: artifactDir ? path.join(artifactDir, "browser-screenshots") : "",
  failures: [],
  notes: [],
};

let server;

function finish(exitCode = 0) {
  result.finished_at = new Date().toISOString();
  console.log(JSON.stringify(result, null, 2));
  if (server && !server.killed) {
    server.kill("SIGTERM");
  }
  process.exit(exitCode);
}

async function waitForHttp(url, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (response.ok || response.status < 500) {
        return true;
      }
    } catch {
      // Server is not ready yet.
    }
    await delay(500);
  }
  return false;
}

try {
  if (result.screenshot_dir) {
    await mkdir(result.screenshot_dir, { recursive: true });
  }

  server = spawn(
    "pnpm",
    [
      "--filter",
      "@salescraft/web",
      "exec",
      "next",
      "dev",
      "--hostname",
      "127.0.0.1",
      "--port",
      String(port),
    ],
    {
      cwd: workspace,
      env: {
        ...process.env,
        NEXT_TELEMETRY_DISABLED: "1",
        TURBO_TELEMETRY_DISABLED: "1",
        TURBO_NO_UPDATE_NOTIFIER: "1",
      },
      stdio: ["ignore", "pipe", "pipe"],
    },
  );

  let serverOutput = "";
  server.stdout.on("data", (chunk) => {
    serverOutput += chunk.toString();
  });
  server.stderr.on("data", (chunk) => {
    serverOutput += chunk.toString();
  });

  server.once("exit", (code) => {
    if (!result.server_started) {
      result.failures.push(`web server exited before readiness with code ${code}`);
    }
  });

  result.server_started = await waitForHttp(`http://127.0.0.1:${port}/login`, 30000);
  if (!result.server_started) {
    result.server_output = serverOutput.slice(-4000);
    finish(1);
  }

  let chromium;
  try {
    const workspaceRequire = createRequire(`${workspace}/package.json`);
    ({ chromium } = workspaceRequire("@playwright/test"));
    result.browser_available = true;
  } catch (error) {
    result.failures.push(`Playwright import failed: ${error.message}`);
    result.server_output = serverOutput.slice(-4000);
    finish(1);
  }

  let browser;
  try {
    browser = await chromium.launch({ headless: true });
  } catch (error) {
    result.failures.push(`Playwright browser launch failed: ${error.message}`);
    result.server_output = serverOutput.slice(-4000);
    finish(1);
  }

  const context = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
  });
  const page = await context.newPage();
  const routes = [
    { path: "/login", expected: ["Salescraft", "Email", "Password"] },
    { path: "/", expected: ["Pipeline", "Relationships"] },
    { path: "/contacts", expected: ["Contacts"] },
    { path: "/pipeline", expected: ["Pipeline"] },
    { path: "/bids", expected: ["Bids"] },
    { path: "/estimates", expected: ["Estimate"] },
    { path: "/projects", expected: ["Projects"] },
    { path: "/relationships", expected: ["Relationships"] },
    { path: "/intelligence", expected: ["Intelligence"] },
    { path: "/products", expected: ["Products"] },
    { path: "/reports", expected: ["Reports"] },
    { path: "/users", expected: ["Users"] },
    { path: "/settings", expected: ["Settings"] },
  ];

  for (const route of routes) {
    const started = Date.now();
    const url = `http://127.0.0.1:${port}${route.path}`;
    const pageResult = {
      path: route.path,
      status: 0,
      title: "",
      duration_ms: 0,
      expected_text_present: true,
      visible_text_sample: "",
      errors: [],
    };
    try {
      const response = await page.goto(url, { waitUntil: "networkidle", timeout: 15000 });
      pageResult.status = response ? response.status() : 0;
      pageResult.title = await page.title();
      const text = (await page.locator("body").innerText({ timeout: 5000 })).replace(/\s+/g, " ").trim();
      pageResult.visible_text_sample = text.slice(0, 500);
      for (const expected of route.expected) {
        if (!text.toLowerCase().includes(expected.toLowerCase())) {
          pageResult.expected_text_present = false;
          pageResult.errors.push(`missing expected text: ${expected}`);
        }
      }
      const buttons = await page.locator("button").count();
      const links = await page.locator("a").count();
      pageResult.interactive_controls = { buttons, links };
      if (pageResult.status >= 400) {
        pageResult.errors.push(`HTTP status ${pageResult.status}`);
      }
      if (result.screenshot_dir) {
        const name = route.path === "/" ? "home" : route.path.replace(/^\//, "").replaceAll("/", "-");
        const screenshotPath = path.join(result.screenshot_dir, `route-${name}.png`);
        await page.screenshot({ path: screenshotPath, fullPage: true });
        pageResult.screenshot = screenshotPath;
      }
    } catch (error) {
      pageResult.expected_text_present = false;
      pageResult.errors.push(error.message);
    }
    pageResult.duration_ms = Date.now() - started;
    result.pages.push(pageResult);
  }

  async function runWorkflow(name, path, check) {
    const started = Date.now();
    const workflow = {
      name,
      path,
      duration_ms: 0,
      passed: false,
      assertions: [],
      errors: [],
    };
    try {
      await page.goto(`http://127.0.0.1:${port}${path}`, {
        waitUntil: "networkidle",
        timeout: 15000,
      });
      await check(workflow);
      if (result.screenshot_dir) {
        const screenshotName = name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
        const screenshotPath = path.join(result.screenshot_dir, `workflow-${screenshotName}.png`);
        await page.screenshot({ path: screenshotPath, fullPage: true });
        workflow.screenshot = screenshotPath;
      }
      workflow.passed = workflow.errors.length === 0;
    } catch (error) {
      workflow.errors.push(error.message);
    }
    workflow.duration_ms = Date.now() - started;
    result.workflows.push(workflow);
  }

  async function assertText(workflow, text) {
    const bodyText = (await page.locator("body").innerText({ timeout: 5000 })).toLowerCase();
    const passed = bodyText.includes(text.toLowerCase());
    workflow.assertions.push({ type: "text", value: text, passed });
    if (!passed) {
      workflow.errors.push(`missing workflow text: ${text}`);
    }
  }

  async function assertVisible(workflow, selector, label) {
    const visible = await page.locator(selector).first().isVisible({ timeout: 3000 }).catch(() => false);
    workflow.assertions.push({ type: "visible", value: label, passed: visible });
    if (!visible) {
      workflow.errors.push(`missing visible control: ${label}`);
    }
  }

  await runWorkflow("login form basics", "/login", async (workflow) => {
    await assertVisible(workflow, 'input[type="email"], input[name="email"]', "email input");
    await assertVisible(workflow, 'input[type="password"], input[name="password"]', "password input");
    await assertVisible(workflow, 'button:has-text("Sign in")', "sign in button");
    await page.locator('input[type="email"], input[name="email"]').first().fill("alex@salescraft.local");
    await page.locator('input[type="password"], input[name="password"]').first().fill("not-a-real-password");
    await assertText(workflow, "Accept invite");
  });

  await runWorkflow("shell navigation integrity", "/", async (workflow) => {
    const navTargets = [
      ["Contacts", "/contacts"],
      ["Pipeline", "/pipeline"],
      ["Bids", "/bids"],
      ["Projects", "/projects"],
      ["Intelligence", "/intelligence"],
      ["Products", "/products"],
      ["Reports", "/reports"],
      ["Users", "/users"],
      ["Settings", "/settings"],
    ];
    for (const [label, target] of navTargets) {
      await page.goto(`http://127.0.0.1:${port}/`, {
        waitUntil: "networkidle",
        timeout: 15000,
      });
      const link = page.locator(`a:has-text("${label}")`).first();
      const visible = await link.isVisible({ timeout: 3000 }).catch(() => false);
      workflow.assertions.push({ type: "nav_link", value: `${label} ${target}`, passed: visible });
      if (!visible) {
        workflow.errors.push(`missing nav link: ${label} ${target}`);
        continue;
      }
      const response = await page.goto(`http://127.0.0.1:${port}${target}`, {
        waitUntil: "networkidle",
        timeout: 15000,
      });
      const status = response ? response.status() : 0;
      const passed = status < 400;
      workflow.assertions.push({ type: "nav_status", value: `${label} ${status}`, passed });
      if (!passed) {
        workflow.errors.push(`nav target failed: ${label} ${target} returned ${status}`);
      }
    }
  });

  await runWorkflow("estimate builder acceptance surface", "/estimates", async (workflow) => {
    for (const text of ["Estimate", "Running total", "Upload", "Scale", "Rooms", "Products", "Review", "Alternates"]) {
      await assertText(workflow, text);
    }
    await assertVisible(workflow, 'button:has-text("Save draft")', "save draft");
    await assertVisible(workflow, 'button:has-text("Submit review")', "submit review");
  });

  await runWorkflow("relationship intelligence acceptance surface", "/relationships", async (workflow) => {
    for (const text of ["Relationship intelligence", "Briefings", "interest", "Log interaction", "Mike Johnson"]) {
      await assertText(workflow, text);
    }
  });

  await runWorkflow("bid response acceptance surface", "/bids", async (workflow) => {
    for (const text of ["Bids", "Validate package", "Active bids", "Due", "Checklist"]) {
      await assertText(workflow, text);
    }
  });

  await browser.close();
  result.passed =
    result.pages.every((entry) => entry.errors.length === 0) &&
    result.workflows.every((entry) => entry.passed);
  if (!result.passed) {
    result.failures.push("one or more browser route or workflow checks failed");
  }
  result.server_output = serverOutput.slice(-4000);
  finish(result.passed ? 0 : 1);
} catch (error) {
  result.failures.push(error.stack || error.message);
  finish(1);
}

await once(process, "beforeExit");
