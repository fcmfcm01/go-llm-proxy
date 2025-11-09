// Provider Monitoring JavaScript

const API_BASE = '/admin/api/v1';
let autoRefreshInterval = null;
let refreshIntervalMs = 5000; // 5 seconds

// Initialize monitoring page
$(document).ready(function() {
    console.log('Provider monitoring page initialized');
    loadProviderStatus();

    // Set up auto-refresh
    setupAutoRefresh();
});

// Load provider status from API
function loadProviderStatus() {
    fetch(`${API_BASE}/providers/status`)
        .then(response => response.json())
        .then(data => {
            updateDashboard(data);
            updateProviderTable(data.providers || []);
        })
        .catch(error => {
            console.error('Error loading provider status:', error);
            showAlert('danger', 'Error loading provider status: ' + error.message);
        });
}

// Update dashboard statistics
function updateDashboard(data) {
    const total = data.providers ? data.providers.length : 0;
    const enabled = data.enabled || 0;
    const disabled = data.disabled || 0;
    const healthy = data.providers ? data.providers.filter(p => p.enabled && p.status === 'healthy').length : 0;
    const unhealthy = total - healthy;

    document.getElementById('totalProviders').textContent = total;
    document.getElementById('healthyProviders').textContent = healthy;
    document.getElementById('unhealthyProviders').textContent = unhealthy;
    document.getElementById('avgSuccessRate').textContent = '0%';
}

// Update provider table
function updateProviderTable(providers) {
    const tbody = document.getElementById('providerStatusBody');
    tbody.innerHTML = '';

    if (providers.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="8" class="text-center">No providers found</td>
            </tr>
        `;
        return;
    }

    providers.forEach(provider => {
        const row = document.createElement('tr');

        // Determine status
        const isHealthy = provider.enabled && provider.status !== 'unhealthy';
        const statusClass = isHealthy ? 'success' : 'danger';
        const statusIcon = isHealthy ? 'check-circle' : 'x-circle';
        const statusText = isHealthy ? 'Healthy' : 'Unhealthy';

        row.innerHTML = `
            <td>
                <span class="badge bg-${statusClass}">
                    <i class="bi bi-${statusIcon}"></i> ${statusText}
                </span>
            </td>
            <td>
                <strong>${provider.name}</strong><br>
                <small class="text-muted">${provider.id}</small>
            </td>
            <td>
                <span class="badge bg-info">${provider.priority || 'N/A'}</span>
            </td>
            <td>
                <span id="response-time-${provider.id}">--</span>
            </td>
            <td>
                <div class="progress" style="height: 5px;">
                    <div class="progress-bar bg-success" style="width: 0%"></div>
                </div>
                <small class="text-muted">--%</small>
            </td>
            <td>
                <span id="request-count-${provider.id}">0</span>
            </td>
            <td>
                <small id="last-check-${provider.id}">--</small>
            </td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button type="button" class="btn btn-outline-primary" onclick="editPriority('${provider.id}', '${provider.name}', ${provider.priority || 1})">
                        <i class="bi bi-arrow-up-down"></i>
                    </button>
                    <button type="button" class="btn btn-outline-${provider.enabled ? 'warning' : 'success'}" onclick="toggleProvider('${provider.id}')">
                        <i class="bi bi-${provider.enabled ? 'pause-circle' : 'play-circle'}"></i>
                    </button>
                </div>
            </td>
        `;

        tbody.appendChild(row);
    });
}

// Refresh status manually
function refreshStatus() {
    loadProviderStatus();
    showAlert('info', 'Status refreshed');
}

// Toggle auto refresh
function toggleAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
        document.querySelector('[onclick="toggleAutoRefresh()"]').innerHTML =
            '<i class="bi bi-play-circle"></i> Auto Refresh';
        showAlert('info', 'Auto refresh disabled');
    } else {
        setupAutoRefresh();
        showAlert('info', 'Auto refresh enabled');
    }
}

// Setup auto refresh interval
function setupAutoRefresh() {
    autoRefreshInterval = setInterval(() => {
        loadProviderStatus();
    }, refreshIntervalMs);

    document.querySelector('[onclick="toggleAutoRefresh()"]').innerHTML =
        '<i class="bi bi-pause-circle"></i> Auto Refresh';
}

// Edit provider priority
function editPriority(providerId, providerName, currentPriority) {
    document.getElementById('priorityProviderId').value = providerId;
    document.getElementById('priorityProviderName').value = providerName;
    document.getElementById('priorityValue').value = currentPriority;
    document.getElementById('priorityModal').querySelector('.modal-title').textContent =
        `Update Priority: ${providerName}`;

    new bootstrap.Modal(document.getElementById('priorityModal')).show();
}

// Update provider priority
function updatePriority() {
    const providerId = document.getElementById('priorityProviderId').value;
    const priority = parseInt(document.getElementById('priorityValue').value);

    if (!providerId || !priority) {
        showAlert('danger', 'Provider ID and priority are required');
        return;
    }

    fetch(`${API_BASE}/providers/${providerId}/priority`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ priority: priority })
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            showAlert('danger', 'Error updating priority: ' + data.error);
        } else {
            showAlert('success', 'Priority updated successfully');
            bootstrap.Modal.getInstance(document.getElementById('priorityModal')).hide();
            loadProviderStatus();
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showAlert('danger', 'Error updating priority: ' + error.message);
    });
}

// Toggle provider
function toggleProvider(providerId) {
    if (!confirm('Are you sure you want to toggle this provider?')) {
        return;
    }

    fetch(`${API_BASE}/providers/${providerId}/toggle`, {
        method: 'POST',
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            showAlert('danger', 'Error toggling provider: ' + data.error);
        } else {
            showAlert('success', 'Provider toggled successfully');
            loadProviderStatus();
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showAlert('danger', 'Error toggling provider: ' + error.message);
    });
}

// Show alert
function showAlert(type, message) {
    const alertHtml = `
        <div class="alert alert-${type} alert-dismissible fade show" role="alert">
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        </div>
    `;

    const container = document.getElementById('alert-container');
    container.innerHTML = alertHtml;

    // Auto-dismiss after 5 seconds
    setTimeout(() => {
        const alert = container.querySelector('.alert');
        if (alert) {
            const bsAlert = new bootstrap.Alert(alert);
            bsAlert.close();
        }
    }, 5000);
}

// Cleanup on page unload
window.addEventListener('beforeunload', function() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
    }
});
