// Provider Management JavaScript

// Global configuration
const API_BASE = '/admin/api/v1';

// Initialize page
$(document).ready(function() {
    console.log('Provider management page initialized');
});

// Create Provider
function createProvider() {
    const form = document.getElementById('createProviderForm');
    const formData = new FormData(form);
    const data = Object.fromEntries(formData.entries());

    // Convert checkbox
    data.enabled = formData.get('enabled') === 'on';

    // Send request
    fetch(`${API_BASE}/providers`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data)
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            showAlert('danger', 'Error creating provider: ' + data.error);
        } else {
            showAlert('success', 'Provider created successfully');
            $('#createProviderModal').modal('hide');
            form.reset();
            // Reload page after 1 second
            setTimeout(() => location.reload(), 1000);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showAlert('danger', 'Error creating provider: ' + error.message);
    });
}

// Edit Provider
function editProvider(providerId) {
    // Fetch provider details
    fetch(`${API_BASE}/providers`)
        .then(response => response.json())
        .then(data => {
            const provider = data.providers.find(p => p.ID === providerId);
            if (provider) {
                showEditModal(provider);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showAlert('danger', 'Error loading provider details');
        });
}

// Show edit modal
function showEditModal(provider) {
    const modalHtml = `
        <div class="modal fade" id="editProviderModal" tabindex="-1">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">Edit Provider</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                    </div>
                    <div class="modal-body">
                        <form id="editProviderForm">
                            <input type="hidden" name="id" value="${provider.ID}">
                            <div class="mb-3">
                                <label class="form-label">ID</label>
                                <input type="text" class="form-control" value="${provider.ID}" readonly>
                            </div>
                            <div class="mb-3">
                                <label class="form-label">Name</label>
                                <input type="text" class="form-control" name="name" value="${provider.Name}" required>
                            </div>
                            <div class="mb-3">
                                <label class="form-label">API URL</label>
                                <input type="url" class="form-control" name="api_url" value="${provider.APIURL}" required>
                            </div>
                            <div class="mb-3">
                                <label class="form-label">Priority</label>
                                <input type="number" class="form-control" name="priority" value="${provider.Priority}" min="1" max="100">
                            </div>
                            <div class="mb-3 form-check">
                                <input type="checkbox" class="form-check-input" name="enabled" ${provider.Enabled ? 'checked' : ''}>
                                <label class="form-check-label">Enabled</label>
                            </div>
                        </form>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                        <button type="button" class="btn btn-primary" onclick="updateProvider()">Update</button>
                    </div>
                </div>
            </div>
        </div>
    `;

    // Remove existing modal if present
    $('#editProviderModal').remove();
    $('body').append(modalHtml);
    $('#editProviderModal').modal('show');
}

// Update Provider
function updateProvider() {
    const form = document.getElementById('editProviderForm');
    const formData = new FormData(form);
    const data = Object.fromEntries(formData.entries());
    const providerId = data.id;

    // Convert checkbox
    data.enabled = formData.get('enabled') === 'on';

    // Remove id from updates
    delete data.id;

    // Send request
    fetch(`${API_BASE}/providers/${providerId}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data)
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            showAlert('danger', 'Error updating provider: ' + data.error);
        } else {
            showAlert('success', 'Provider updated successfully');
            $('#editProviderModal').modal('hide');
            // Reload page after 1 second
            setTimeout(() => location.reload(), 1000);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showAlert('danger', 'Error updating provider: ' + error.message);
    });
}

// Toggle Provider
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
            // Reload page after 1 second
            setTimeout(() => location.reload(), 1000);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showAlert('danger', 'Error toggling provider: ' + error.message);
    });
}

// Delete Provider
function deleteProvider(providerId, providerName) {
    if (!confirm(`Are you sure you want to delete provider "${providerName}"? This action cannot be undone.`)) {
        return;
    }

    fetch(`${API_BASE}/providers/${providerId}`, {
        method: 'DELETE',
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            showAlert('danger', 'Error deleting provider: ' + data.error);
        } else {
            showAlert('success', 'Provider deleted successfully');
            // Reload page after 1 second
            setTimeout(() => location.reload(), 1000);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showAlert('danger', 'Error deleting provider: ' + error.message);
    });
}

// Show Alert
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

// Initialize DataTable (if needed)
$(function() {
    $('#providersTable').DataTable({
        "order": [[ 0, "asc" ]],
        "pageLength": 25,
        "language": {
            "search": "Search providers:",
            "emptyTable": "No providers found"
        }
    });
});
