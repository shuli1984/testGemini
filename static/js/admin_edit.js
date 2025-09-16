document.getElementById('pageForm').addEventListener('submit', async (e) => {
        e.preventDefault();

        const pageName = document.getElementById('pageName').value;
        const title = document.getElementById('title').value;
        const description = document.getElementById('description').value;
        const isCoreSolution = document.getElementById('isCoreSolution').checked;
        const icon = document.getElementById('icon').value;
        const message = quill.root.innerHTML;
        const csrfToken = document.querySelector('input[name="csrf_token"]').value;
        const editLang = document.getElementById('editLang').value; // <-- 确保这一行正确获取了值

        const url = isNew ? '/admin/pages/new' : `/admin/pages/${pageName}`;
        const method = isNew ? 'POST' : 'PUT';

        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken
            },
            body: JSON.stringify({
                Name: pageName,
                Title: title,
                Description: description,
                Message: message,
                IsCoreSolution: isCoreSolution,
                Icon: icon,
                LanguageCode: editLang // <-- 确保这里使用了正确获取到的 editLang
            })
        });

        const responseDiv = document.getElementById('response');

        if (response.ok) {
            isFormDirty = false; // Reset dirty flag on successful save
            if (isNew) {
                const result = await response.json();
                window.location.href = `/admin/pages/edit/${result.Name}`;
            } else {
                const saveSuccessMsg = 'Save successful!';
                responseDiv.innerHTML = `<div class="alert alert-success">` + saveSuccessMsg + `</div>`; // Changed message
            }
        } else {
            const contentType = response.headers.get("content-type");
            let errorMsg = 'An unknown error occurred.';
            if (contentType && contentType.indexOf("application/json") !== -1) {
                const result = await response.json();
                errorMsg = result.message;
            } else {
                errorMsg = await response.text();
            }
            responseDiv.innerHTML = `<div class="alert alert-danger">${errorMsg}</div>`;
        }
    });