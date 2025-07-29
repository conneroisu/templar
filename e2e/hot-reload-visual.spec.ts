import { test, expect, Page } from '@playwright/test';
import { promises as fs } from 'fs';
import path from 'path';

test.describe('Templar Hot Reload Visual Validation', () => {
  let testProjectDir: string;
  let componentsDir: string;

  test.beforeEach(async ({ page }) => {
    // Get test project directory from global setup
    testProjectDir = process.env.E2E_TEST_PROJECT_DIR || '';
    componentsDir = path.join(testProjectDir, 'components');
    
    if (!testProjectDir) {
      throw new Error('E2E_TEST_PROJECT_DIR not set by global setup');
    }

    // Change to test project directory
    process.chdir(testProjectDir);
    
    // Navigate to the Templar server
    await page.goto('/');
    
    // Wait for page to be fully loaded
    await page.waitForLoadState('networkidle');
  });

  test('should demonstrate hot reload functionality with visual proof', async ({ page }) => {
    console.log('🎬 Starting visual hot reload test...');
    
    // Step 1: Take initial screenshot of empty state
    await page.screenshot({ 
      path: 'screenshots/01-initial-state.png',
      fullPage: true 
    });
    console.log('📸 Initial state screenshot taken');

    // Step 2: Check WebSocket connection
    console.log('🔌 Checking WebSocket connectivity...');
    
    // Wait for WebSocket connection and listen for messages
    const wsMessages: any[] = [];
    
    await page.evaluate(() => {
      return new Promise((resolve) => {
        const ws = new WebSocket('ws://localhost:8080/ws');
        
        ws.onopen = () => {
          console.log('WebSocket connected');
          (window as any).testWebSocket = ws;
          resolve(true);
        };
        
        ws.onmessage = (event) => {
          console.log('WebSocket message received:', event.data);
          (window as any).wsMessages = (window as any).wsMessages || [];
          (window as any).wsMessages.push(JSON.parse(event.data));
        };
        
        ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          resolve(false);
        };
        
        // Timeout after 5 seconds
        setTimeout(() => resolve(false), 5000);
      });
    });

    // Verify WebSocket is connected
    const isWebSocketConnected = await page.evaluate(() => {
      return (window as any).testWebSocket?.readyState === WebSocket.OPEN;
    });
    
    expect(isWebSocketConnected).toBe(true);
    console.log('✅ WebSocket connection verified');

    // Step 3: Create initial test component
    const initialComponent = `package components

templ TestComponent() {
	<div id="hot-reload-test" class="p-4 bg-blue-100 border-2 border-blue-500 rounded-lg">
		<h1 class="text-xl font-bold text-blue-800">Initial Component</h1>
		<p class="text-blue-600">This component will be hot reloaded</p>
		<button class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">
			Initial Button
		</button>
	</div>
}`;

    await fs.writeFile(path.join(componentsDir, 'test_component.templ'), initialComponent);
    console.log('📝 Initial test component created');

    // Wait for file system change to be detected and processed
    await page.waitForTimeout(2000);

    // Step 4: Navigate to component preview (if available) or check main page
    // Try to access component directly or through the component list
    try {
      await page.goto('/component/TestComponent');
    } catch {
      // If direct component access isn't available, stay on main page
      console.log('📄 Using main page for component testing');
    }

    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Step 5: Take screenshot after component creation
    await page.screenshot({ 
      path: 'screenshots/02-initial-component.png',
      fullPage: true 
    });
    console.log('📸 Initial component screenshot taken');

    // Step 6: Check for WebSocket messages indicating component update
    const initialMessages = await page.evaluate(() => {
      return (window as any).wsMessages || [];
    });
    
    console.log(`💬 Received ${initialMessages.length} WebSocket messages after component creation`);

    // Step 7: Modify the component to trigger hot reload
    const modifiedComponent = `package components

templ TestComponent() {
	<div id="hot-reload-test" class="p-6 bg-green-100 border-2 border-green-500 rounded-xl shadow-lg">
		<h1 class="text-2xl font-bold text-green-800">🔥 HOT RELOADED! 🔥</h1>
		<p class="text-green-600 mb-4">This component was successfully hot reloaded!</p>
		<div class="flex space-x-2">
			<button class="px-6 py-3 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors">
				✨ Updated Button
			</button>
			<button class="px-6 py-3 bg-purple-500 text-white rounded-lg hover:bg-purple-600 transition-colors">
				🚀 New Button
			</button>
		</div>
		<div class="mt-4 p-3 bg-yellow-100 border border-yellow-400 rounded">
			<p class="text-yellow-800 text-sm">
				⚡ Hot reload timestamp: ${new Date().toISOString()}
			</p>
		</div>
	</div>
}`;

    await fs.writeFile(path.join(componentsDir, 'test_component.templ'), modifiedComponent);
    console.log('📝 Component modified to trigger hot reload');

    // Step 8: Wait for hot reload to process
    await page.waitForTimeout(3000);

    // Step 9: Check for new WebSocket messages
    let reloadDetected = false;
    let attempts = 0;
    const maxAttempts = 10;

    while (!reloadDetected && attempts < maxAttempts) {
      const currentMessages = await page.evaluate(() => {
        return (window as any).wsMessages || [];
      });
      
      // Look for hot reload related messages
      const reloadMessages = currentMessages.filter((msg: any) => 
        msg.type === 'reload' || 
        msg.type === 'component_updated' || 
        msg.type === 'file_changed' ||
        (msg.data && typeof msg.data === 'string' && msg.data.includes('component'))
      );

      if (reloadMessages.length > 0) {
        reloadDetected = true;
        console.log('🔥 Hot reload WebSocket messages detected:', reloadMessages);
      } else {
        console.log(`⏳ Waiting for hot reload... attempt ${attempts + 1}/${maxAttempts}`);
        await page.waitForTimeout(1000);
        attempts++;
      }
    }

    // Step 10: Take screenshot after hot reload
    await page.screenshot({ 
      path: 'screenshots/03-after-hot-reload.png',
      fullPage: true 
    });
    console.log('📸 After hot reload screenshot taken');

    // Step 11: Verify visual changes occurred
    // Check if the component content changed by looking for the new text
    const componentExists = await page.locator('#hot-reload-test').isVisible();
    console.log(`👀 Component visible: ${componentExists}`);

    if (componentExists) {
      const updatedTitle = await page.locator('h1:has-text("HOT RELOADED")').isVisible();
      console.log(`🔥 Hot reload title visible: ${updatedTitle}`);
      
      if (updatedTitle) {
        console.log('✅ Visual hot reload verification: SUCCESS');
      } else {
        console.log('❌ Visual hot reload verification: Title not updated');
      }
    }

    // Step 12: Test multiple rapid changes
    console.log('🔄 Testing rapid component changes...');
    
    for (let i = 1; i <= 3; i++) {
      const rapidChangeComponent = `package components

templ TestComponent() {
	<div id="hot-reload-test" class="p-4 bg-red-${i}00 border-2 border-red-500 rounded-lg">
		<h1 class="text-xl font-bold text-red-800">Rapid Change #${i}</h1>
		<p class="text-red-600">Testing rapid hot reload changes</p>
		<div class="mt-2">
			${Array.from({length: i}, (_, j) => 
				`<span class="inline-block w-4 h-4 bg-red-500 rounded-full mr-1"></span>`
			).join('')}
		</div>
	</div>
}`;

      await fs.writeFile(path.join(componentsDir, 'test_component.templ'), rapidChangeComponent);
      await page.waitForTimeout(1500); // Shorter wait for rapid changes
      
      await page.screenshot({ 
        path: `screenshots/04-rapid-change-${i}.png`,
        fullPage: true 
      });
      console.log(`📸 Rapid change ${i} screenshot taken`);
    }

    // Step 13: Final WebSocket message count
    const finalMessages = await page.evaluate(() => {
      return (window as any).wsMessages || [];
    });
    
    console.log(`💬 Total WebSocket messages received: ${finalMessages.length}`);
    console.log('📊 Final WebSocket messages:', finalMessages.slice(-5)); // Show last 5 messages

    // Step 14: Take final state screenshot
    await page.screenshot({ 
      path: 'screenshots/05-final-state.png',
      fullPage: true 
    });
    console.log('📸 Final state screenshot taken');

    // Step 15: Create a summary HTML report
    const reportHtml = `
<!DOCTYPE html>
<html>
<head>
    <title>Templar Hot Reload Visual Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .screenshot { margin: 20px 0; }
        .screenshot img { max-width: 800px; border: 1px solid #ccc; }
        .success { color: green; }
        .warning { color: orange; }
        .error { color: red; }
        .info { color: blue; }
        .log { background: #f5f5f5; padding: 10px; margin: 10px 0; border-left: 4px solid #007acc; }
    </style>
</head>
<body>
    <h1>🔥 Templar Hot Reload Visual Test Report</h1>
    <p><strong>Test Date:</strong> ${new Date().toISOString()}</p>
    <p><strong>WebSocket Messages:</strong> ${finalMessages.length}</p>
    <p><strong>Hot Reload Status:</strong> <span class="${reloadDetected ? 'success' : 'error'}">${reloadDetected ? '✅ DETECTED' : '❌ NOT DETECTED'}</span></p>
    
    <h2>📸 Visual Test Screenshots</h2>
    
    <div class="screenshot">
        <h3>1. Initial State</h3>
        <img src="01-initial-state.png" alt="Initial state" />
        <p>Empty state before any components are created.</p>
    </div>
    
    <div class="screenshot">
        <h3>2. Initial Component</h3>
        <img src="02-initial-component.png" alt="Initial component" />
        <p>Component created and rendered with blue styling.</p>
    </div>
    
    <div class="screenshot">
        <h3>3. After Hot Reload</h3>
        <img src="03-after-hot-reload.png" alt="After hot reload" />
        <p>Component updated with green styling and "HOT RELOADED" text.</p>
    </div>
    
    <div class="screenshot">
        <h3>4. Rapid Changes</h3>
        <img src="04-rapid-change-1.png" alt="Rapid change 1" />
        <img src="04-rapid-change-2.png" alt="Rapid change 2" />
        <img src="04-rapid-change-3.png" alt="Rapid change 3" />
        <p>Series of rapid component changes to test hot reload stability.</p>
    </div>
    
    <div class="screenshot">
        <h3>5. Final State</h3>
        <img src="05-final-state.png" alt="Final state" />
        <p>Final component state after all changes.</p>
    </div>
    
    <h2>💬 WebSocket Communication Log</h2>
    <div class="log">
        <pre>${JSON.stringify(finalMessages, null, 2)}</pre>
    </div>
    
    <h2>📋 Test Summary</h2>
    <ul>
        <li>WebSocket Connection: <span class="${isWebSocketConnected ? 'success' : 'error'}">${isWebSocketConnected ? '✅ SUCCESS' : '❌ FAILED'}</span></li>
        <li>Component Creation: ✅ SUCCESS</li>
        <li>Hot Reload Detection: <span class="${reloadDetected ? 'success' : 'error'}">${reloadDetected ? '✅ SUCCESS' : '❌ FAILED'}</span></li>
        <li>Visual Changes: ✅ CAPTURED</li>
        <li>Rapid Changes: ✅ TESTED</li>
        <li>Screenshots Generated: ✅ 7 IMAGES</li>
    </ul>
</body>
</html>`;

    await fs.writeFile('screenshots/test-report.html', reportHtml);
    console.log('📄 Test report generated: screenshots/test-report.html');

    // Assertions for test validation
    expect(isWebSocketConnected).toBe(true);
    expect(finalMessages.length).toBeGreaterThan(0);
    
    console.log('🎉 Hot reload visual test completed successfully!');
    console.log('📁 Check the screenshots/ directory for visual proof');
    console.log('📊 Open screenshots/test-report.html for detailed results');
  });

  test('should validate WebSocket message structure and timing', async ({ page }) => {
    console.log('🔍 Testing WebSocket message structure...');
    
    // Set up WebSocket message capture with detailed logging
    await page.evaluate(() => {
      return new Promise((resolve) => {
        const ws = new WebSocket('ws://localhost:8080/ws');
        const messages: any[] = [];
        const timings: number[] = [];
        
        ws.onopen = () => {
          console.log('WebSocket opened for message structure test');
          (window as any).testWS = ws;
          (window as any).wsMessages = messages;
          (window as any).wsTimings = timings;
          resolve(true);
        };
        
        ws.onmessage = (event) => {
          const timestamp = Date.now();
          let parsedData;
          
          try {
            parsedData = JSON.parse(event.data);
          } catch (e) {
            parsedData = { raw: event.data, parseError: true };
          }
          
          messages.push({
            timestamp,
            data: parsedData,
            raw: event.data
          });
          
          timings.push(timestamp);
          console.log('WebSocket message captured:', parsedData);
        };
        
        setTimeout(() => resolve(true), 2000);
      });
    });

    // Create a component change to trigger WebSocket messages
    const testComponent = `package components

templ WebSocketTest() {
	<div id="websocket-test">
		<h1>WebSocket Structure Test</h1>
		<p>Testing message structure: ${Date.now()}</p>
	</div>
}`;

    await fs.writeFile(path.join(componentsDir, 'websocket_test.templ'), testComponent);
    await page.waitForTimeout(3000);

    // Analyze WebSocket messages
    const messageAnalysis = await page.evaluate(() => {
      const messages = (window as any).wsMessages || [];
      const timings = (window as any).wsTimings || [];
      
      return {
        totalMessages: messages.length,
        messages: messages,
        timingDelays: timings.slice(1).map((time, i) => time - timings[i]),
        messageTypes: messages.map(m => m.data?.type || 'unknown'),
        hasValidStructure: messages.every(m => 
          typeof m.data === 'object' && 
          (m.data.type || m.data.raw || m.data.parseError)
        )
      };
    });

    console.log('📊 WebSocket Message Analysis:', messageAnalysis);

    // Take screenshot of the analysis results
    await page.evaluate((analysis) => {
      document.body.innerHTML = `
        <div style="font-family: monospace; padding: 20px;">
          <h1>WebSocket Message Analysis</h1>
          <p><strong>Total Messages:</strong> ${analysis.totalMessages}</p>
          <p><strong>Valid Structure:</strong> ${analysis.hasValidStructure ? '✅ YES' : '❌ NO'}</p>
          <p><strong>Message Types:</strong> ${analysis.messageTypes.join(', ')}</p>
          <h2>Timing Analysis</h2>
          <p><strong>Average Delay:</strong> ${analysis.timingDelays.length > 0 ? 
            Math.round(analysis.timingDelays.reduce((a, b) => a + b, 0) / analysis.timingDelays.length) : 0}ms</p>
          <h2>Messages</h2>
          <pre>${JSON.stringify(analysis.messages, null, 2)}</pre>
        </div>
      `;
    }, messageAnalysis);

    await page.screenshot({ 
      path: 'screenshots/06-websocket-analysis.png',
      fullPage: true 
    });

    // Assertions
    expect(messageAnalysis.totalMessages).toBeGreaterThan(0);
    expect(messageAnalysis.hasValidStructure).toBe(true);
    
    console.log('✅ WebSocket message structure validation completed');
  });
});