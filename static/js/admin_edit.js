document.getElementById('editPageForm').addEventListener('submit', async function(event) {
    event.preventDefault();

    const pageName = document.getElementById('pageName').value;
    const title = document.getElementById('title').value;
    const description = document.getElementById('description').value;
    const message = document.getElementById('message').value;

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
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(pageData)
        });

        const result = await response.json();

        if (response.ok) {
            responseDiv.textContent = 'Success: ' + JSON.stringify(result, null, 2);
            responseDiv.style.color = 'green';
        } else {
            responseDiv.textContent = 'Error: ' + (result.error || JSON.stringify(result, null, 2));
            responseDiv.style.color = 'red';
        }
    } catch (error) {
        responseDiv.textContent = 'Network Error: ' + error.message;
        responseDiv.style.color = 'red';
    }
});

// Optional: Fetch current page data on load (not implemented in this prototype)
// For now, user needs to manually enter page name and content.