document.addEventListener('DOMContentLoaded', () => {
    const pageForm = document.getElementById('pageForm');
    if (!pageForm) {
        return;
    }

    // --- Read data from HTML attributes ---
    const siteDefaultLanguage = pageForm.dataset.siteDefaultLanguage;
    const targetLanguage = pageForm.dataset.editLang;
    const isNew = pageForm.dataset.isNew === 'true';
    const csrfToken = pageForm.dataset.csrfToken;
    const i18n = {
        confirmDiscardChanges: pageForm.dataset.i18nConfirmDiscardChanges,
        uploadFailed: pageForm.dataset.i18nUploadFailed,
        noTranslationNeeded: pageForm.dataset.i18nNoTranslationNeeded,
        translationError: pageForm.dataset.i18nTranslationError,
        unknownError: pageForm.dataset.i18nUnknownError,
        networkError: pageForm.dataset.i18nNetworkError,
        confirmOverwriteTranslation: pageForm.dataset.i18nConfirmOverwriteTranslation,
        translatingAllFields: pageForm.dataset.i18nTranslatingAllFields,
        allFieldsTranslatedSuccess: pageForm.dataset.i18nAllFieldsTranslatedSuccess,
        someFieldsNotTranslated: pageForm.dataset.i18nSomeFieldsNotTranslated,
        saveSuccessful: pageForm.dataset.i18nSaveSuccessful
    };


    function normalizeNewlines(str) {
        if (typeof str !== 'string') {
            return str;
        }
        return str.replace(/\r\n|\r/g, '\n');
    }

    let isFormDirty = false;
    const markFormDirty = () => { isFormDirty = true; };

    document.getElementById('languageSelector').addEventListener('change', function() {
        if (isFormDirty) {
            if (!confirm(i18n.confirmDiscardChanges)) {
                this.value = targetLanguage;
                return;
            }
        }
        const selectedLang = this.value;
        const currentUrl = new URL(window.location.href);
        currentUrl.searchParams.set('lang', selectedLang);
        window.location.href = currentUrl.toString();
    });

    function imageHandler() {
        const input = document.createElement('input');
        input.setAttribute('type', 'file');
        input.setAttribute('accept', 'image/*');
        input.click();

        input.onchange = async () => {
            const file = input.files[0];
            const formData = new FormData();
            formData.append('image', file);

            const response = await fetch('/admin/upload', {
                method: 'POST',
                headers: {
                    'X-CSRF-Token': csrfToken
                },
                body: formData
            });

            const result = await response.json();
            if (response.ok) {
                const imageUrl = result.url;
                const range = quill.getSelection();
                quill.insertEmbed(range.index, 'image', imageUrl);
            } else {
                console.error(i18n.uploadFailed + ':', result);
            }
        };
    }

    const quill = new Quill('#message', {
        theme: 'snow',
        modules: {
            toolbar: {
                container: [
                    [{ 'font': [] }],
                    [{ 'header': [1, 2, 3, 4, 5, 6, false] }],
                    [{ 'color': [] }, { 'background': [] }],
                    [{ 'align': [] }],
                    ['bold', 'italic', 'underline', 'strike'],
                    ['blockquote', 'code-block'],
                    [{ 'list': 'ordered'}, { 'list': 'bullet' }],
                    [{ 'script': 'sub'}, { 'script': 'super' }],
                    [{ 'indent': '-1'}, { 'indent': '+1' }],
                    [{ 'direction': 'rtl' }],
                    ['link', 'image', 'video'],
                    ['clean']
                ],
                handlers: {
                    'image': imageHandler
                }
            }
        }
    });

    // Attach listeners to mark form as dirty
    document.getElementById('title').addEventListener('input', markFormDirty);
    document.getElementById('description').addEventListener('input', markFormDirty);
    document.getElementById('isCoreSolution').addEventListener('change', markFormDirty);
    document.getElementById('icon').addEventListener('input', markFormDirty);
    quill.on('text-change', markFormDirty);

    // --- Translation Logic ---
    async function translateField(field) {
        const responseDiv = document.getElementById('response');
        responseDiv.innerHTML = '';

        if (targetLanguage === siteDefaultLanguage) {
            responseDiv.innerHTML = `<div class="alert alert-info">${i18n.noTranslationNeeded}</div>`;
            return null;
        }

        const sourceLangForTranslation = siteDefaultLanguage;

        try {
            const response = await fetch('/admin/api/translate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': csrfToken
                },
                body: JSON.stringify({
                    page_name: document.getElementById('pageName').value,
                    source_language: sourceLangForTranslation,
                    target_language: targetLanguage,
                    content: "",
                    field: field
                })
            });

            const result = await response.json();

            if (response.ok) {
                return result.translated_text;
            } else {
                responseDiv.innerHTML = `<div class="alert alert-danger">${i18n.translationError}: ${result.message || i18n.unknownError}</div>`;
                return null;
            }
        } catch (error) {
            responseDiv.innerHTML = `<div class="alert alert-danger">${i18n.networkError}: ${error.message}</div>`;
            return null;
        }
    }

    document.querySelectorAll('.translate-btn').forEach(button => {
        button.addEventListener('click', async function() {
            const field = this.dataset.field;
            let hasContent = false;

            if (field === 'title') {
                hasContent = document.getElementById('title').value.trim() !== '';
            } else if (field === 'description') {
                hasContent = document.getElementById('description').value.trim() !== '';
            } else if (field === 'message') {
                hasContent = quill.getText().trim() !== '';
            }

            if (hasContent) {
                if (!confirm(i18n.confirmOverwriteTranslation)) {
                    return;
                }
            }

            const translatedText = await translateField(field);
            if (translatedText !== null) {
                if (field === 'title') {
                    document.getElementById('title').value = translatedText;
                } else if (field === 'description') {
                    document.getElementById('description').value = translatedText;
                } else if (field === 'message') {
                    let normalizedTranslatedText = normalizeNewlines(translatedText);
                    quill.setText(normalizedTranslatedText);
                }
                markFormDirty();
            }
        });
    });

    const translateAllButton = document.getElementById('translateAllButton');
    if (translateAllButton) {
        translateAllButton.addEventListener('click', async () => {
            if (!confirm(i18n.confirmOverwriteTranslation)) {
                return;
            }

            const responseDiv = document.getElementById('response');
            responseDiv.innerHTML = `<div class="alert alert-info">${i18n.translatingAllFields}</div>`;

            const fieldsToTranslate = ['title', 'description', 'message'];
            let allTranslated = true;

            for (const field of fieldsToTranslate) {
                const translatedText = await translateField(field);
                if (translatedText !== null) {
                    if (field === 'title') {
                        document.getElementById('title').value = translatedText;
                    } else if (field === 'description') {
                        document.getElementById('description').value = translatedText;
                    } else if (field === 'message') {
                        let normalizedTranslatedText = normalizeNewlines(translatedText);
                        quill.setText(normalizedTranslatedText);
                    }
                    markFormDirty();
                } else {
                    allTranslated = false;
                }
            }

            if (allTranslated) {
                responseDiv.innerHTML = `<div class="alert alert-success">${i18n.allFieldsTranslatedSuccess}</div>`;
            } else {
                responseDiv.innerHTML = `<div class="alert alert-warning">${i18n.someFieldsNotTranslated}</div>`;
            }
        });
    }

    const previewButton = document.getElementById('previewButton');
    if(previewButton) {
        previewButton.addEventListener('click', () => {
            const title = document.getElementById('title').value;
            const description = document.getElementById('description').value;
            const message = quill.root.innerHTML;
    
            const previewContentDiv = document.getElementById('previewContent');
            previewContentDiv.innerHTML = '';
    
            const titleEl = document.createElement('h1');
            titleEl.textContent = title;
    
            const descriptionEl = document.createElement('p');
            descriptionEl.textContent = description;
    
            const messageDiv = document.createElement('div');
            messageDiv.innerHTML = message;
    
            previewContentDiv.appendChild(titleEl);
            previewContentDiv.appendChild(descriptionEl);
            previewContentDiv.appendChild(messageDiv);
    
            const previewModal = new bootstrap.Modal(document.getElementById('previewModal'));
            previewModal.show();
        });
    }

    const toggleHtmlButton = document.getElementById('toggleHtmlButton');
    if(toggleHtmlButton) {
        let isHtmlMode = false;
        let htmlTextarea;
        toggleHtmlButton.addEventListener('click', () => {
            const messageContainer = document.getElementById('message');
            if (isHtmlMode) {
                quill.root.innerHTML = htmlTextarea.value;
                quill.container.style.display = 'block';
                htmlTextarea.remove();
                isHtmlMode = false;
            } else {
                htmlTextarea = document.createElement('textarea');
                htmlTextarea.style.width = '100%';
                htmlTextarea.style.height = quill.container.offsetHeight + 'px';
                htmlTextarea.value = quill.root.innerHTML;
                quill.container.style.display = 'none';
                messageContainer.parentNode.insertBefore(htmlTextarea, messageContainer.nextSibling);
                isHtmlMode = true;
            }
        });
    }

    pageForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const pageName = document.getElementById('pageName').value;
        const title = document.getElementById('title').value;
        const description = document.getElementById('description').value;
        const isCoreSolution = document.getElementById('isCoreSolution').checked;
        const icon = document.getElementById('icon').value;
        const message = quill.root.innerHTML;
        const editLang = document.getElementById('editLang').value;

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
                LanguageCode: editLang
            })
        });

        const responseDiv = document.getElementById('response');

        if (response.ok) {
            isFormDirty = false;
            if (isNew) {
                const result = await response.json();
                window.location.href = `/admin/pages/edit/${result.Name}`;
            } else {
                responseDiv.innerHTML = `<div class="alert alert-success">${i18n.saveSuccessful}</div>`;
            }
        } else {
            const contentType = response.headers.get("content-type");
            let errorMsg = i18n.unknownError;
            if (contentType && contentType.indexOf("application/json") !== -1) {
                const result = await response.json();
                errorMsg = result.message;
            } else {
                errorMsg = await response.text();
            }
            responseDiv.innerHTML = `<div class="alert alert-danger">${errorMsg}</div>`;
        }
    });
});
