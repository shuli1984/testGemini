document.addEventListener('DOMContentLoaded', function () {
    const navList = document.getElementById('navigation-list');
    const addNavItemBtn = document.getElementById('add-nav-item');
    const navItemTemplate = document.getElementById('nav-item-template');
    const navigationJsonInput = document.getElementById('navigation-json');
    const form = document.querySelector('form');

    let initialNavData = [];
    try {
        initialNavData = JSON.parse('{{ .NavigationJSON }}');
    } catch (e) {
        console.error('Could not parse initial navigation data:', e);
    }

    function createNavItem(item) {
        const templateClone = navItemTemplate.content.cloneNode(true);
        const listItem = templateClone.querySelector('li');
        listItem.querySelector('.nav-label').value = item.Label || '';
        listItem.querySelector('.nav-type').value = item.Type || 'page';
        listItem.querySelector('.nav-value').value = item.Value || '';
        return listItem;
    }

    function renderNavItems() {
        navList.innerHTML = '';
        if (initialNavData) {
            initialNavData.forEach(item => {
                navList.appendChild(createNavItem(item));
            });
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
    });

    renderNavItems();
});