document.getElementById('editPageForm').addEventListener('submit', async function(event) {
    event.preventDefault();

    const csrfToken = document.querySelector('input[name="csrf_token"]').value; // Get CSRF token

    const pageName = document.getElementById('pageName').value;
    const title = document.getElementById('title').value;
    const description = document.getElementById('description').value;
    const message = quill.root.innerHTML; // Get content from Quill

    const pageData = {
        Name: pageName,
        Title: title,
        Description: description,
        Message: message
    };

    const responseDiv = document.getElementById('response');
    responseDiv.textContent = 'Saving...';

    try {
        const response = await fetch(`/admin/pages/${pageName}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken // Use the retrieved token
            },
            body: JSON.stringify(pageData)
        });

        const result = await response.json();

        if (response.ok) {
            responseDiv.textContent = 'Page updated successfully!';
            responseDiv.style.color = 'green';
        } else {
            responseDiv.textContent = 'Error: ' + (result.error || 'An unknown error occurred.');
            responseDiv.style.color = 'red';
        }
    } catch (error) {
        responseDiv.textContent = 'Network Error: ' + error.message;
        responseDiv.style.color = 'red';
    }
});

// Optional: Fetch current page data on load (not implemented in this prototype)
// For now, user needs to manually enter page name and content.