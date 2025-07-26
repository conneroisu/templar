package monitoring

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"runtime"
	"time"

	"github.com/conneroisu/templar/internal/logging"
)

// DashboardData represents data for the health dashboard
type DashboardData struct {
	Health          HealthResponse              `json:"health"`
	RecoveryHistory map[string]*RecoveryHistory `json:"recovery_history"`
	SystemMetrics   SystemMetrics               `json:"system_metrics"`
	Timestamp       time.Time                   `json:"timestamp"`
}

// SystemMetrics provides additional system metrics for the dashboard
type SystemMetrics struct {
	CPUUsage       float64       `json:"cpu_usage"`
	MemoryUsage    float64       `json:"memory_usage"`
	DiskUsage      float64       `json:"disk_usage"`
	GoroutineCount int           `json:"goroutine_count"`
	HeapSize       uint64        `json:"heap_size"`
	GCCount        uint32        `json:"gc_count"`
	Uptime         time.Duration `json:"uptime"`
}

// HealthDashboard provides a web interface for monitoring system health
type HealthDashboard struct {
	healthMonitor     *HealthMonitor
	selfHealingSystem *SelfHealingSystem
	logger            logging.Logger
}

// NewHealthDashboard creates a new health dashboard
func NewHealthDashboard(
	healthMonitor *HealthMonitor,
	selfHealingSystem *SelfHealingSystem,
	logger logging.Logger,
) *HealthDashboard {
	return &HealthDashboard{
		healthMonitor:     healthMonitor,
		selfHealingSystem: selfHealingSystem,
		logger:            logger.WithComponent("health_dashboard"),
	}
}

// ServeHTTP handles HTTP requests for the health dashboard
func (hd *HealthDashboard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health-dashboard":
		hd.handleDashboardHTML(w, r)
	case "/health-dashboard/api/data":
		hd.handleDashboardAPI(w, r)
	case "/health-dashboard/api/recovery":
		hd.handleRecoveryAPI(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleDashboardHTML serves the HTML dashboard page
func (hd *HealthDashboard) handleDashboardHTML(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Templar Health Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #2c3e50; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .status-healthy { color: #27ae60; font-weight: bold; }
        .status-degraded { color: #f39c12; font-weight: bold; }
        .status-unhealthy { color: #e74c3c; font-weight: bold; }
        .metric { display: flex; justify-content: space-between; margin: 10px 0; padding: 5px 0; border-bottom: 1px solid #eee; }
        .recovery-item { margin: 10px 0; padding: 10px; border-left: 4px solid #3498db; background: #f8f9fa; }
        .critical { border-left-color: #e74c3c; }
        .auto-refresh { text-align: center; margin: 20px 0; }
        button { padding: 10px 20px; margin: 5px; border: none; border-radius: 4px; cursor: pointer; }
        .btn-primary { background: #3498db; color: white; }
        .btn-success { background: #27ae60; color: white; }
        .btn-danger { background: #e74c3c; color: white; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🩺 Templar Health Dashboard</h1>
            <p>Real-time system health monitoring and self-healing status</p>
        </div>

        <div class="auto-refresh">
            <button class="btn-primary" onclick="refreshData()">🔄 Refresh Now</button>
            <button class="btn-success" id="autoRefreshBtn" onclick="toggleAutoRefresh()">⏱️ Auto-refresh: OFF</button>
            <span id="lastUpdated">Last updated: Never</span>
        </div>

        <div class="grid">
            <div class="card">
                <h2>🏥 Overall Health</h2>
                <div id="overallHealth">Loading...</div>
            </div>

            <div class="card">
                <h2>📊 System Metrics</h2>
                <div id="systemMetrics">Loading...</div>
            </div>

            <div class="card">
                <h2>🔍 Health Checks</h2>
                <div id="healthChecks">Loading...</div>
            </div>

            <div class="card">
                <h2>🔧 Recovery History</h2>
                <div id="recoveryHistory">Loading...</div>
            </div>
        </div>
    </div>

    <script>
        let autoRefreshInterval = null;
        let autoRefreshEnabled = false;

        function refreshData() {
            fetch('/health-dashboard/api/data')
                .then(response => response.json())
                .then(data => updateDashboard(data))
                .catch(error => console.error('Error fetching data:', error));
        }

        function updateDashboard(data) {
            updateOverallHealth(data.health);
            updateSystemMetrics(data.system_metrics);
            updateHealthChecks(data.health.checks);
            updateRecoveryHistory(data.recovery_history);
            document.getElementById('lastUpdated').textContent = 
                'Last updated: ' + new Date(data.timestamp).toLocaleTimeString();
        }

        function updateOverallHealth(health) {
            const statusClass = 'status-' + health.status;
            document.getElementById('overallHealth').innerHTML = 
                '<div class="metric"><span>Status:</span><span class="' + statusClass + '">' + 
                health.status.toUpperCase() + '</span></div>' +
                '<div class="metric"><span>Uptime:</span><span>' + formatDuration(health.uptime) + '</span></div>' +
                '<div class="metric"><span>Total Checks:</span><span>' + health.summary.total + '</span></div>' +
                '<div class="metric"><span>Healthy:</span><span class="status-healthy">' + health.summary.healthy + '</span></div>' +
                '<div class="metric"><span>Degraded:</span><span class="status-degraded">' + health.summary.degraded + '</span></div>' +
                '<div class="metric"><span>Unhealthy:</span><span class="status-unhealthy">' + health.summary.unhealthy + '</span></div>';
        }

        function updateSystemMetrics(metrics) {
            document.getElementById('systemMetrics').innerHTML =
                '<div class="metric"><span>Memory Usage:</span><span>' + formatBytes(metrics.heap_size) + '</span></div>' +
                '<div class="metric"><span>Goroutines:</span><span>' + metrics.goroutine_count + '</span></div>' +
                '<div class="metric"><span>GC Runs:</span><span>' + metrics.gc_count + '</span></div>' +
                '<div class="metric"><span>Uptime:</span><span>' + formatDuration(metrics.uptime) + '</span></div>';
        }

        function updateHealthChecks(checks) {
            let html = '';
            for (const [name, check] of Object.entries(checks)) {
                const statusClass = 'status-' + check.status;
                const criticalClass = check.critical ? ' critical' : '';
                html += '<div class="recovery-item' + criticalClass + '">' +
                        '<strong>' + name + '</strong> ' +
                        '<span class="' + statusClass + '">' + check.status.toUpperCase() + '</span><br>' +
                        '<small>' + check.message + '</small><br>' +
                        '<small>Duration: ' + formatDuration(check.duration) + 
                        ' | Last checked: ' + new Date(check.last_checked).toLocaleTimeString() + '</small>' +
                        '</div>';
            }
            document.getElementById('healthChecks').innerHTML = html || 'No health checks available';
        }

        function updateRecoveryHistory(history) {
            let html = '';
            for (const [name, h] of Object.entries(history)) {
                if (h.recovery_attempts > 0) {
                    const successClass = h.recovery_successful ? 'status-healthy' : 'status-unhealthy';
                    html += '<div class="recovery-item">' +
                            '<strong>' + name + '</strong><br>' +
                            '<small>Consecutive failures: ' + h.consecutive_failures + '</small><br>' +
                            '<small>Recovery attempts: ' + h.recovery_attempts + '</small><br>' +
                            '<small class="' + successClass + '">Last recovery: ' + 
                            (h.recovery_successful ? 'Successful' : 'Failed') + '</small>' +
                            '</div>';
                }
            }
            document.getElementById('recoveryHistory').innerHTML = html || 'No recovery attempts recorded';
        }

        function formatDuration(nanoseconds) {
            const seconds = Math.floor(nanoseconds / 1000000000);
            if (seconds < 60) return seconds + 's';
            const minutes = Math.floor(seconds / 60);
            if (minutes < 60) return minutes + 'm ' + (seconds % 60) + 's';
            const hours = Math.floor(minutes / 60);
            return hours + 'h ' + (minutes % 60) + 'm';
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function toggleAutoRefresh() {
            autoRefreshEnabled = !autoRefreshEnabled;
            const btn = document.getElementById('autoRefreshBtn');
            
            if (autoRefreshEnabled) {
                autoRefreshInterval = setInterval(refreshData, 10000); // Refresh every 10 seconds
                btn.textContent = '⏱️ Auto-refresh: ON';
                btn.className = 'btn-danger';
            } else {
                clearInterval(autoRefreshInterval);
                btn.textContent = '⏱️ Auto-refresh: OFF';
                btn.className = 'btn-success';
            }
        }

        // Initial load
        refreshData();
    </script>
</body>
</html>`

	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		hd.logger.Error(context.Background(), err, "Failed to parse dashboard template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, nil); err != nil {
		hd.logger.Error(context.Background(), err, "Failed to execute dashboard template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleDashboardAPI serves JSON data for the dashboard
func (hd *HealthDashboard) handleDashboardAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := hd.getDashboardData()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		hd.logger.Error(context.Background(), err, "Failed to encode dashboard data")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleRecoveryAPI handles recovery-related API requests
func (hd *HealthDashboard) handleRecoveryAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get recovery history
		history := make(map[string]*RecoveryHistory)
		if hd.selfHealingSystem != nil {
			history = hd.selfHealingSystem.GetRecoveryHistory()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"recovery_history": history,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getDashboardData collects all data for the dashboard
func (hd *HealthDashboard) getDashboardData() DashboardData {
	health := hd.healthMonitor.GetHealth()

	var recoveryHistory map[string]*RecoveryHistory
	if hd.selfHealingSystem != nil {
		recoveryHistory = hd.selfHealingSystem.GetRecoveryHistory()
	} else {
		recoveryHistory = make(map[string]*RecoveryHistory)
	}

	systemMetrics := hd.getSystemMetrics()

	return DashboardData{
		Health:          health,
		RecoveryHistory: recoveryHistory,
		SystemMetrics:   systemMetrics,
		Timestamp:       time.Now(),
	}
}

// getSystemMetrics collects current system metrics
func (hd *HealthDashboard) getSystemMetrics() SystemMetrics {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return SystemMetrics{
		GoroutineCount: runtime.NumGoroutine(),
		HeapSize:       mem.HeapAlloc,
		GCCount:        mem.NumGC,
		Uptime:         time.Since(startTime),
	}
}
