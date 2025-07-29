import { promises as fs } from 'fs';

async function globalTeardown() {
  console.log('🧹 Cleaning up Playwright E2E test environment...');
  
  // Clean up test project directory
  const testProjectDir = process.env.E2E_TEST_PROJECT_DIR;
  if (testProjectDir) {
    try {
      await fs.rm(testProjectDir, { recursive: true, force: true });
      console.log(`✅ Cleaned up test project: ${testProjectDir}`);
    } catch (error) {
      console.warn(`⚠️  Failed to clean up test project: ${error}`);
    }
  }
  
  console.log('✅ Global teardown completed');
}

export default globalTeardown;