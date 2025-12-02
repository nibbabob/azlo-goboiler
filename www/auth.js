
// API helper
const api = {
    request: async (endpoint, method = 'GET', body = null) => {
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
        };

        if (body) {
            options.body = JSON.stringify(body);
        }

        try {
            const response = await fetch(endpoint, options);
            const contentType = response.headers.get('content-type');
            let responseData;

            // Try to parse JSON if content-type is correct
            if (contentType && contentType.includes('application/json')) {
                responseData = await response.json();
            } else {
                responseData = await response.text();
            }

            if (!response.ok) {
                let errorBody;
                if (responseData && (responseData.error || responseData.message)) {
                     errorBody = responseData.error || responseData.message;
                } else if (typeof responseData === 'string' && responseData.length > 0) {
                    errorBody = responseData;
                } else {
                    errorBody = `HTTP ${response.status} ${response.statusText}`;
                }
                throw new Error(errorBody || 'Request failed');
            }

            return responseData; // Return the parsed data

        } catch (error) {
            console.error(`API Error (${method} ${endpoint}):`, error);
            throw error;
        }
    },
    post: (endpoint, body) => api.request(endpoint, 'POST', body),
};

// Error display helper
function showError(message) {
    const errorDiv = document.getElementById('errorMessage');
    if (errorDiv) {
        errorDiv.textContent = message;
        errorDiv.classList.add('show');
        
        // Auto-hide after 5 seconds
        setTimeout(() => {
            errorDiv.classList.remove('show');
        }, 5000);
    }
}

function hideError() {
    const errorDiv = document.getElementById('errorMessage');
    if (errorDiv) {
        errorDiv.classList.remove('show');
    }
}

// Login form handler
const loginForm = document.getElementById('loginForm');
if (loginForm) {
    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        hideError();

        const formData = {
            // CHANGED: Send the field value as 'username' to match backend model
            username: document.getElementById('email').value,
            password: document.getElementById('password').value,
            // 'remember' isn't used by the backend, but we can leave it
        };

        const submitBtn = loginForm.querySelector('button[type="submit"]');
        const originalText = submitBtn.textContent;
        submitBtn.disabled = true;
        submitBtn.textContent = 'Signing in...';

        try {
            // CHANGED: Corrected endpoint from /api/auth/login to /auth/login
            const response = await api.post('/auth/login', formData);
            
            // CHANGED: Store the token on successful login
            if (response && response.success) {
                // Successful login - redirect to dashboard
                window.location.href = '/index.html';
            } else {
                 throw new Error('Login was not successful.');
            }
            
        } catch (error) {
            showError(error.message || 'Login failed. Please check your credentials.');
            submitBtn.disabled = false;
            submitBtn.textContent = originalText;
        }
    });
}

// Register form handler
const registerForm = document.getElementById('registerForm');
if (registerForm) {
    registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        hideError();

        const password = document.getElementById('password').value;
        const confirmPassword = document.getElementById('confirmPassword').value;

        if (password !== confirmPassword) {
            showError('Passwords do not match');
            return;
        }
        
        // Note: More complex password validation is handled by the backend
        if (password.length < 8) {
            showError('Password must be at least 8 characters long');
            return;
        }

        const formData = {
            // CHANGED: Send 'username' instead of 'name'
            username: document.getElementById('username').value,
            email: document.getElementById('email').value,
            password: password,
        };

        const submitBtn = registerForm.querySelector('button[type="submit"]');
        const originalText = submitBtn.textContent;
        submitBtn.disabled = true;
        submitBtn.textContent = 'Creating account...';

        try {
            // CHANGED: Corrected endpoint from /api/auth/register to /auth/register
            const response = await api.post('/auth/register', formData);
            
            // Successful registration - show success and redirect to login
            alert('Registration successful! Please sign in.');
            window.location.href = '/login.html';
            
        } catch (error) {
            // Error.message will contain the validation error from the backend
            showError(error.message || 'Registration failed. Please try again.');
            submitBtn.disabled = false;
            submitBtn.textContent = originalText;
        }
    });
}