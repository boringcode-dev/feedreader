(() => {
  const root = document.documentElement;
  const filterButtons = Array.from(document.querySelectorAll('[data-filter]'));
  const cards = Array.from(document.querySelectorAll('.item-card'));
  const grid = document.querySelector('[data-card-grid]');
  const viewMoreButton = document.querySelector('[data-view-more]');
  const refreshButton = document.querySelector('[data-refresh-button]');
  const themeToggle = document.querySelector('[data-theme-toggle]');
  const toast = document.querySelector('[data-toast]');
  const initialVisible = Number(grid?.dataset.initialVisible || 12);
  const filterStorageKey = 'feedreader.filter';
  const themeStorageKey = 'feedreader.theme';
  const refreshToastStorageKey = 'feedreader.toast';
  const metaThemeColor = document.querySelector('meta[name="theme-color"]');

  let activeFilter = localStorage.getItem(filterStorageKey) || 'all';
  let visibleCount = initialVisible;

  const filteredCards = () =>
    cards.filter((card) => activeFilter === 'all' || card.dataset.source === activeFilter);

  const renderCards = () => {
    const visibleCards = filteredCards();
    cards.forEach((card) => card.classList.add('is-hidden'));
    visibleCards.slice(0, visibleCount).forEach((card, index) => {
      card.classList.remove('is-hidden');
      const indexNode = card.querySelector('.item-index');
      if (indexNode) {
        indexNode.textContent = `${index + 1}.`;
      }
    });

    if (viewMoreButton) {
      const hasMore = visibleCards.length > visibleCount;
      viewMoreButton.hidden = !hasMore;
      viewMoreButton.disabled = !hasMore;
    }
  };

  const renderFilters = () => {
    filterButtons.forEach((button) => {
      const isActive = button.dataset.filter === activeFilter;
      button.classList.toggle('is-active', isActive);
      button.setAttribute('aria-pressed', String(isActive));
    });
  };

  let toastTimer = null;
  const showToast = (message, kind = 'success') => {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.toggle('is-error', kind === 'error');
    toast.classList.add('is-visible');
    if (toastTimer) {
      window.clearTimeout(toastTimer);
    }
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
    }, 2200);
  };

  const applyFilter = (nextFilter) => {
    activeFilter = nextFilter;
    visibleCount = initialVisible;
    localStorage.setItem(filterStorageKey, activeFilter);
    renderFilters();
    renderCards();
  };

  const applyTheme = (theme) => {
    root.dataset.theme = theme;
    localStorage.setItem(themeStorageKey, theme);
    if (metaThemeColor) {
      metaThemeColor.setAttribute('content', theme === 'dark' ? '#111e2c' : '#e1ebf7');
      document.querySelectorAll('meta[name="theme-color"]').forEach((node) => {
        const media = node.getAttribute('media');
        if (!media) {
          node.setAttribute('content', theme === 'dark' ? '#111e2c' : '#e1ebf7');
        }
      });
    }
    if (themeToggle) {
      themeToggle.setAttribute('aria-label', `Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`);
      themeToggle.setAttribute('title', theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode');
    }
  };

  filterButtons.forEach((button) => {
    button.addEventListener('click', () => applyFilter(button.dataset.filter || 'all'));
  });

  if (viewMoreButton) {
    viewMoreButton.addEventListener('click', () => {
      visibleCount += initialVisible;
      renderCards();
    });
  }

  if (refreshButton) {
    refreshButton.addEventListener('click', async () => {
      refreshButton.disabled = true;
      try {
        const response = await fetch('/api/refresh', { method: 'POST' });
        const payload = await response.json().catch(() => ({}));
        if (!response.ok || !payload.ok) {
          showToast('Refresh completed with errors', 'error');
          return;
        }
        sessionStorage.setItem(refreshToastStorageKey, 'Feed refreshed');
        window.location.reload();
      } catch (error) {
        showToast('Refresh failed', 'error');
      } finally {
        refreshButton.disabled = false;
      }
    });
  }

  if (themeToggle) {
    themeToggle.addEventListener('click', () => {
      applyTheme(root.dataset.theme === 'dark' ? 'light' : 'dark');
    });
  }

  if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
      navigator.serviceWorker.register('/service-worker.js').catch(() => {});
    });
  }

  const savedTheme = localStorage.getItem(themeStorageKey);
  applyTheme(savedTheme === 'light' ? 'light' : 'dark');

  const pendingToast = sessionStorage.getItem(refreshToastStorageKey);
  if (pendingToast) {
    sessionStorage.removeItem(refreshToastStorageKey);
    showToast(pendingToast, 'success');
  }

  if (!filterButtons.some((button) => button.dataset.filter === activeFilter)) {
    activeFilter = 'all';
  }
  renderFilters();
  renderCards();
})();
