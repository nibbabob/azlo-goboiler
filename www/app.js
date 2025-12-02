// State
const state = {
    user: null
};

// --- API Helper ---
const api = {
    request: async (endpoint, method = 'GET', body = null) => {
        const options = {
            method,
            credentials: 'include',
            headers: { 
                'Content-Type': 'application/json', 
                'X-Requested-With': 'XMLHttpRequest' 
            },
        };
        if (body) options.body = JSON.stringify(body);

        try {
            const res = await fetch(endpoint, options);
            if (res.status === 401) { window.location.href = '/login.html'; return; }
            if (res.status === 204) return null;

            const contentType = res.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                const json = await res.json();
                if (!res.ok) throw new Error(json.error || json.message || 'Request failed');
                return json.success ? json.data : json;
            }
            return null;
        } catch (err) {
            console.error(err);
            throw err;
        }
    },
    get: (ep) => api.request(ep, 'GET'),
    post: (ep, body) => api.request(ep, 'POST', body),
};

// --- Initialization ---
document.addEventListener('DOMContentLoaded', async () => {
    try {
        // Just fetch the profile to ensure we are logged in
        const user = await api.get('/api/v1/profile');
        state.user = user;
        
        const welcomeEl = document.getElementById('welcomeMsg');
        if(welcomeEl && user.username) {
            welcomeEl.textContent = `Welcome, ${user.username}`;
        }
    } catch (e) { 
        console.error("Init failed", e); 
    }

    // Mobile Toggle
    const mobileBtn = document.getElementById('mobileMenuBtn');
    if(mobileBtn) mobileBtn.addEventListener('click', () => document.getElementById('sidebar').classList.toggle('open'));

    // Logout
    const logoutBtn = document.getElementById('logoutBtn');
    if(logoutBtn) logoutBtn.addEventListener('click', async () => {
        try { await api.post('/auth/logout'); } finally { window.location.href = '/login.html'; }
    });
});