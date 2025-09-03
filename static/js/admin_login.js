document.getElementById('loginForm').addEventListener('submit', async function(event) {
    event.preventDefault();

    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const messageDiv = document.getElementById('message');

    const csrfToken = document.querySelector('input[name="gorilla.csrf.Token"]').value;

    const requestBody = { username, password };

    console.log('Sending login request with the following data:');
    console.log('Username:', username);
    console.log('Password:', password);
    console.log('CSRF Token:', csrfToken);
    console.log('Request Body:', JSON.stringify(requestBody));

    try {
        const response = await fetch('/admin/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken
            },
            body: JSON.stringify(requestBody)
        });

        const result = await response.json();

        if (response.ok) {
            messageDiv.style.color = 'green';
            messageDiv.textContent = result.message || 'Login successful!';
            window.location.href = '/admin/dashboard'; // Redirect to dashboard
        } else {
            messageDiv.style.color = 'red';
            messageDiv.textContent = result.message || 'Login failed!';
        }
    } catch (error) {
        messageDiv.style.color = 'red';
        messageDiv.textContent = 'Network error: ' + error.message;
    }
});