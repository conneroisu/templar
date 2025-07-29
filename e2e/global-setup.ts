import { chromium, FullConfig } from '@playwright/test';
import { spawn, ChildProcess } from 'child_process';
import { promises as fs } from 'fs';
import path from 'path';

let serverProcess: ChildProcess | null = null;

async function globalSetup(config: FullConfig) {
  console.log('🚀 Starting Playwright E2E test setup...');
  
  // Create test project directory
  const testProjectDir = path.join(process.cwd(), 'e2e-test-project');
  
  try {
    await fs.rm(testProjectDir, { recursive: true, force: true });
  } catch (error) {
    // Directory may not exist, which is fine
  }
  
  await fs.mkdir(testProjectDir, { recursive: true });
  await fs.mkdir(path.join(testProjectDir, 'components'), { recursive: true });
  
  // Create basic test components
  const buttonComponent = `package components

templ Button(text string) {
	<button class="btn" id="test-button">{text}</button>
}`;

  const cardComponent = `package components

templ Card(title string, content string) {
	<div class="card" id="test-card">
		<h3 class="card-title">{title}</h3>
		<p class="card-content">{content}</p>
	</div>
}`;

  await fs.writeFile(path.join(testProjectDir, 'components', 'button.templ'), buttonComponent);
  await fs.writeFile(path.join(testProjectDir, 'components', 'card.templ'), cardComponent);
  
  // Create templar config
  const configContent = `server:
  port: 8080
  host: "localhost"
  open: false
  environment: "test"

components:
  scan_paths: ["./components"]
  exclude_patterns: ["*_test.templ"]

development:
  hot_reload: true
  css_injection: true
  error_overlay: true

build:
  command: "templ generate"
  watch: ["**/*.templ"]
  cache_dir: ".templar/cache"
`;

  await fs.writeFile(path.join(testProjectDir, '.templar.yml'), configContent);
  
  // Store test directory path for tests
  process.env.E2E_TEST_PROJECT_DIR = testProjectDir;
  
  console.log(`✅ Test project created at: ${testProjectDir}`);
  
  // Launch browser for global use (optional, for debugging)
  const browser = await chromium.launch();
  process.env.GLOBAL_BROWSER_WS_ENDPOINT = browser.wsEndpoint();
  
  return async () => {
    await browser.close();
  };
}

export default globalSetup;