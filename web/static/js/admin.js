// Admin Panel Common JavaScript

$(document).ready(function() {
    console.log('Admin panel initialized');

    // Auto-dismiss alerts after 5 seconds
    setTimeout(function() {
        $('.alert').each(function() {
            if ($(this).find('.btn-close').length) {
                $(this).find('.btn-close').click();
            }
        });
    }, 5000);

    // Initialize tooltips
    var tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    var tooltipList = tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });

    // Initialize popovers
    var popoverTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="popover"]'));
    var popoverList = popoverTriggerList.map(function (popoverTriggerEl) {
        return new bootstrap.Popover(popoverTriggerEl);
    });

    // Form validation
    $('.needs-validation').on('submit', function(e) {
        if (!this.checkValidity()) {
            e.preventDefault();
            e.stopPropagation();
        }
        $(this).addClass('was-validated');
    });

    // CSRF Token (if using CSRF protection)
    const csrfToken = document.querySelector('meta[name="csrf-token"]');
    if (csrfToken) {
        window.csrfToken = csrfToken.getAttribute('content');
    }
});

// Utility Functions
const AdminUtils = {
    // Show success alert
    showSuccess: function(message) {
        this.showAlert('success', message);
    },

    // Show error alert
    showError: function(message) {
        this.showAlert('danger', message);
    },

    // Show warning alert
    showWarning: function(message) {
        this.showAlert('warning', message);
    },

    // Show info alert
    showInfo: function(message) {
        this.showAlert('info', message);
    },

    // Generic alert function
    showAlert: function(type, message) {
        const alertHtml = `
            <div class="alert alert-${type} alert-dismissible fade show" role="alert">
                <i class="bi bi-${this.getAlertIcon(type)}"></i>
                ${message}
                <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
            </div>
        `;

        const container = document.getElementById('alert-container') || document.querySelector('.container-fluid');
        container.insertAdjacentHTML('afterbegin', alertHtml);

        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            const alert = container.querySelector('.alert');
            if (alert) {
                const bsAlert = new bootstrap.Alert(alert);
                bsAlert.close();
            }
        }, 5000);
    },

    // Get icon for alert type
    getAlertIcon: function(type) {
        const icons = {
            success: 'check-circle',
            danger: 'exclamation-triangle',
            warning: 'exclamation-circle',
            info: 'info-circle'
        };
        return icons[type] || 'info-circle';
    },

    // Confirm action
    confirm: function(message, callback) {
        if (confirm(message)) {
            callback();
        }
    },

    // Format date
    formatDate: function(date) {
        if (!date) return '';
        const d = new Date(date);
        return d.toLocaleDateString() + ' ' + d.toLocaleTimeString();
    },

    // Truncate text
    truncate: function(text, length) {
        if (!text) return '';
        if (text.length <= length) return text;
        return text.substring(0, length) + '...';
    }
};

// Export for use in other scripts
window.AdminUtils = AdminUtils;
