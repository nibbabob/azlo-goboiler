/**
 * Security Helper
 */
function escapeHTML(str) {
    if (str === null || str === undefined) return '';
    return str.toString().replace(/[&<>"']/g, (m) => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[m]));
}

/**
 * Settings Logic
 */
const Settings = {
    api: {
        request: async (endpoint, method = 'GET', body = null) => {
            const options = {
                method,
                credentials: 'include',
                headers: { 'Content-Type': 'application/json' },
            };
            if (body) options.body = JSON.stringify(body);

            try {
                const res = await fetch(endpoint, options);
                if (res.status === 401) { window.location.href = '/login.html'; return; }
                
                const json = await res.json();
                if (!res.ok) throw new Error(json.error || json.message || 'Request failed');
                return json.success ? json.data : json;
            } catch (err) {
                Settings.ui.showToast('Error', err.message, 'error');
                throw err;
            }
        },
        get: (ep) => Settings.api.request(ep, 'GET'),
        put: (ep, body) => Settings.api.request(ep, 'PUT', body),
        post: (ep, body) => Settings.api.request(ep, 'POST', body),
    },

    ui: {
        showToast: (title, message, type = 'info') => {
            const container = document.getElementById('toastContainer');
            const toast = document.createElement('div');
            const color = type === 'error' ? '#ef4444' : '#06b6d4';
            
            toast.className = 'toast';
            toast.style.borderLeftColor = color;
            toast.innerHTML = `
                <div style="font-weight:600; margin-bottom:4px">${escapeHTML(title)}</div>
                <div style="font-size:0.9rem; color:#cbd5e1">${escapeHTML(message)}</div>
                <div class="toast-progress" style="background:${color}; transition-duration: 3000ms; width: 100%;"></div>
            `;
            container.appendChild(toast);
            
            requestAnimationFrame(() => {
                toast.classList.add('show');
                toast.querySelector('.toast-progress').getBoundingClientRect();
                toast.querySelector('.toast-progress').style.width = '0%';
            });

            setTimeout(() => {
                toast.classList.remove('show');
                setTimeout(() => toast.remove(), 300);
            }, 3000);
        },

        switchTab: (targetId) => {
            // Update Nav Buttons
            document.querySelectorAll('#settingsTabs .nav-item').forEach(btn => {
                if (btn.dataset.tab === targetId) btn.classList.add('active');
                else btn.classList.remove('active');
            });

            // Update Panels
            document.querySelectorAll('.settings-panel').forEach(panel => {
                if (panel.id === targetId) {
                    panel.style.display = 'block';
                    // Small fade in effect
                    panel.style.opacity = '0';
                    requestAnimationFrame(() => {
                        panel.style.transition = 'opacity 0.2s';
                        panel.style.opacity = '1';
                    });
                } else {
                    panel.style.display = 'none';
                }
            });
        }
    },

    actions: {
        init: async () => {
            // Load Profile
            try {
                const user = await Settings.api.get('/api/v1/profile');
                document.getElementById('settingsUsername').value = user.username;
                document.getElementById('settingsEmail').value = user.email;
            } catch (e) {}

            // Load Preferences
            try {
                const prefs = await Settings.api.get('/api/v1/preferences');
                document.getElementById('prefEmailEnabled').checked = prefs.email_enabled;
                document.getElementById('prefFrequency').value = prefs.frequency || 'immediate';
            } catch (e) {}
        },

        updateProfile: async (e) => {
            e.preventDefault();
            const username = document.getElementById('settingsUsername').value;
            const email = document.getElementById('settingsEmail').value;
            try {
                await Settings.api.put('/api/v1/profile', { username, email });
                Settings.ui.showToast('Success', 'Profile updated successfully');
            } catch (e) {}
        },

        updatePassword: async (e) => {
            e.preventDefault();
            const current = document.getElementById('currentPassword').value;
            const newer = document.getElementById('newPassword').value;
            const confirm = document.getElementById('confirmNewPassword').value;

            if (newer !== confirm) {
                Settings.ui.showToast('Error', 'New passwords do not match', 'error');
                return;
            }

            try {
                await Settings.api.put('/api/v1/password', { 
                    current_password: current, 
                    new_password: newer 
                });
                Settings.ui.showToast('Success', 'Password changed successfully');
                document.getElementById('passwordForm').reset();
            } catch (e) {}
        },

        updatePreferences: async (e) => {
            e.preventDefault();
            const body = {
                email_enabled: document.getElementById('prefEmailEnabled').checked,
                frequency: document.getElementById('prefFrequency').value
            };

            try {
                await Settings.api.put('/api/v1/preferences', body);
                Settings.ui.showToast('Success', 'Preferences saved');
            } catch (e) {}
        }
    }
};

// Initialization
document.addEventListener('DOMContentLoaded', () => {
    Settings.actions.init();

    // Tab Switching Logic
    document.querySelectorAll('#settingsTabs .nav-item').forEach(btn => {
        btn.addEventListener('click', () => Settings.ui.switchTab(btn.dataset.tab));
    });

    // Form Handlers
    document.getElementById('profileForm').addEventListener('submit', Settings.actions.updateProfile);
    document.getElementById('passwordForm').addEventListener('submit', Settings.actions.updatePassword);
    document.getElementById('preferencesForm').addEventListener('submit', Settings.actions.updatePreferences);

    // Sidebar Toggle (Mobile)
    document.getElementById('mobileMenuBtn').addEventListener('click', () => {
        document.getElementById('sidebar').classList.toggle('open');
    });

    // Logout
    document.getElementById('logoutBtn').addEventListener('click', async () => {
        try { await Settings.api.post('/auth/logout'); } 
        finally { window.location.href = '/login.html'; }
    });
});