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
        // Serialize navigation data
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

        // Serialize carousel data
        const carouselItems = [];
        carouselPagesList.querySelectorAll('.list-group-item').forEach((item, index) => {
            carouselItems.push({
                page_id: parseInt(item.dataset.pageId, 10),
                order: index + 1
            });
        });
        carouselJsonInput.value = JSON.stringify(carouselItems);
    });

    renderNavItems();

    // --- Carousel Settings Logic ---
    const availablePagesList = document.getElementById('available-pages-list');
    const carouselPagesList = document.getElementById('carousel-pages-list');
    const carouselJsonInput = document.getElementById('homepage-carousel-pages-json');
    let initialCarouselData = window.initialCarouselData || [];
    let allPagesMap = window.allPagesForCarousel || {};

    // Initialize the lists
    const initialCarouselPageIds = initialCarouselData.map(item => item.page_id);
    
    // Populate selected list
    initialCarouselData.forEach(item => {
        const listItem = document.createElement('li');
        listItem.className = 'list-group-item';
        listItem.dataset.pageId = item.page_id;
        listItem.textContent = allPagesMap[String(item.page_id)] || 'Unknown Page';
        carouselPagesList.appendChild(listItem);
    });

    // Hide items in available list that are already selected
    Array.from(availablePagesList.children).forEach(item => {
        const pageId = parseInt(item.dataset.pageId, 10);
        if (initialCarouselPageIds.includes(pageId)) {
            item.style.display = 'none';
        }
    });

    function updateAvailableList() {
        const selectedIds = Array.from(carouselPagesList.children).map(item => parseInt(item.dataset.pageId, 10));
        Array.from(availablePagesList.children).forEach(item => {
            const pageId = parseInt(item.dataset.pageId, 10);
            if (selectedIds.includes(pageId)) {
                item.style.display = 'none';
            } else {
                item.style.display = 'block';
            }
        });
    }

    new Sortable(availablePagesList, {
        group: 'carousel-pages',
        animation: 150,
        ghostClass: 'sortable-ghost',
        onEnd: updateAvailableList
    });

    new Sortable(carouselPagesList, {
        group: 'carousel-pages',
        animation: 150,
        ghostClass: 'sortable-ghost',
        onEnd: updateAvailableList
    });

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