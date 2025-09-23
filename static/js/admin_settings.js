document.addEventListener('DOMContentLoaded', function () {
    const navList = document.getElementById('navigation-list');
    const addNavItemBtn = document.getElementById('add-nav-item');
    const navItemTemplate = document.getElementById('nav-item-template');
    const navigationJsonInput = document.getElementById('navigation-json');
    const form = document.getElementById('settingsForm'); // Use ID for specificity

    let initialNavData = window.initialNavData || [];

    function createNavItem(item) {
        const templateClone = navItemTemplate.content.cloneNode(true);
        const listItem = templateClone.querySelector('li');
        listItem.querySelector('.nav-label').value = item.label || '';
        listItem.querySelector('.nav-type').value = item.type || 'page';
        listItem.querySelector('.nav-value').value = item.value || '';
        return listItem;
    }

    function renderNavItems() {
        navList.innerHTML = '';
        if (initialNavData) {
            initialNavData.forEach(item => {
                navList.appendChild(createNavItem(item));
            });
            // Add this line: Populate hidden input with initial data
            navigationJsonInput.value = JSON.stringify(initialNavData);
        }
    }

    addNavItemBtn.addEventListener('click', () => {
        navList.appendChild(createNavItem({}));
    });

    navList.addEventListener('click', function (e) {
        if (e.target.classList.contains('remove-nav-item')) {
            e.target.closest('.list-group-item').remove();
        }
    });

    new Sortable(navList, {
        animation: 150,
        ghostClass: 'sortable-ghost'
    });

    form.addEventListener('submit', function (e) {
        const navItems = [];
        navList.querySelectorAll('.list-group-item').forEach(item => {
            navItems.push({
                label: item.querySelector('.nav-label').value,
                type: item.querySelector('.nav-type').value,
                value: item.querySelector('.nav-value').value,
                new_tab: false // This can be implemented later
            });
        });
        navigationJsonInput.value = JSON.stringify(navItems);

        // Add validation for navigation data
        if (initialNavData && initialNavData.length > 0 && navItems.length === 0) {
            if (!confirm("You had existing navigation items, but none are present now. Saving will clear your navigation menu. Are you sure you want to proceed?")) {
                e.preventDefault(); // Prevent form submission
                return;
            }
        }
    });

    renderNavItems();

    // --- Translation Logic ---
    const editLang = form.dataset.editLang;
    const csrfToken = form.dataset.csrfToken;
    const i18n = {
        confirmOverwrite: form.dataset.i18nConfirmOverwriteTranslation,
        translationError: form.dataset.i18nTranslationError,
        unknownError: form.dataset.i18nUnknownError,
        networkError: form.dataset.i18nNetworkError
    };

    async function translateField(field, sourceLang) {
        try {
            const response = await fetch('/admin/api/translate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': csrfToken
                },
                body: JSON.stringify({
                    source_language: sourceLang,
                    target_language: editLang,
                    field: field
                })
            });

            const result = await response.json();

            if (response.ok) {
                return result.translated_text;
            } else {
                alert(`${i18n.translationError}: ${result.message || i18n.unknownError}`);
                return null;
            }
        } catch (error) {
            alert(`${i18n.networkError}: ${error.message}`);
            return null;
        }
    }

    document.querySelectorAll('.translate-btn').forEach(button => {
        button.addEventListener('click', async function() {
            const field = this.dataset.field;
            const inputId = field.replace(/_([a-z])/g, (g) => g[1].toUpperCase()); // site_title -> siteTitle
            const inputElement = document.getElementById(inputId);

            if (!confirm(i18n.confirmOverwrite)) {
                return;
            }

            const translatedText = await translateField(field, 'en');
            if (translatedText !== null && inputElement) {
                inputElement.value = translatedText;
            }
        });
    });

    const translateAllBtn = document.getElementById('translateAllBtn');
    if (translateAllBtn) {
        translateAllBtn.addEventListener('click', async () => {
            if (!confirm(i18n.confirmOverwrite)) {
                return;
            }

            const fieldsToTranslate = ['site_title', 'site_tagline', 'meta_description', 'meta_keywords', 'maintenance_message'];
            for (const field of fieldsToTranslate) {
                const inputId = field.replace(/_([a-z])/g, (g) => g[1].toUpperCase());
                const inputElement = document.getElementById(inputId);
                const translatedText = await translateField(field, 'en');
                if (translatedText !== null && inputElement) {
                    inputElement.value = translatedText;
                }
            }
        });
    }

    // Auto-hide success message
    const settingsMessageAlert = document.getElementById('settings-message-alert');
    if (settingsMessageAlert) {
        setTimeout(() => {
            settingsMessageAlert.style.display = 'none';
        }, 5000); // Hide after 5 seconds
    }
});