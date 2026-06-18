(() => {
  const root = document.documentElement;
  const filterButtons = Array.from(document.querySelectorAll('[data-filter]'));
  const controlsRow = document.querySelector('[data-controls-row]');
  const cardsGrid = document.querySelector('[data-card-grid]');
  const viewMoreButton = document.querySelector('[data-view-more]');
  const searchToggle = document.querySelector('[data-search-toggle]');
  const searchForm = document.querySelector('[data-search-form]');
  const searchInput = document.querySelector('[data-search-input]');
  const searchSourceInput = document.querySelector('[data-search-source]');
  const refreshButton = document.querySelector('[data-refresh-button]');
  const themeToggle = document.querySelector('[data-theme-toggle]');
  const toast = document.querySelector('[data-toast]');
  const pageSize = Number(cardsGrid?.dataset.pageSize || 12);
  const searchDebounceMs = 1100;
  const themeStorageKey = 'feedreader.theme';
  const refreshToastStorageKey = 'feedreader.toast';
  const metaThemeColor = document.querySelector('meta[name="theme-color"]');

  let activeFilter = cardsGrid?.dataset.currentSource || 'all';
  let activeQuery = (searchInput?.value || '').trim();
  let searchOpen = Boolean(activeQuery);
  let loadedCount = cardsGrid ? cardsGrid.querySelectorAll('.item-card').length : 0;
  let hasNext = cardsGrid?.dataset.hasNext === 'true';
  let searchTimer = null;
  let requestSequence = 0;

  const cardTemplate = (item) => `
    <article class="item-card" data-source="${escapeHtml(item.source || '')}">
      <h2 class="item-title">
        <span class="item-index">${escapeHtml(item.index ?? '')}.</span>
        <a href="${escapeAttr(item.url || '#')}" target="_blank" rel="noreferrer">${escapeHtml(item.title || '')}</a>
      </h2>
      ${item.brief ? `<p class="item-brief">${escapeHtml(item.brief)}</p>` : ''}
      <p class="item-host">${escapeHtml(item.host || hostLabel(item.url || ''))}</p>
    </article>
  `;

  const renderFilters = () => {
    filterButtons.forEach((button) => {
      const isActive = button.dataset.filter === activeFilter;
      button.classList.toggle('is-active', isActive);
      button.setAttribute('aria-pressed', String(isActive));
    });
  };

  const renderViewMore = () => {
    if (!viewMoreButton) return;
    viewMoreButton.hidden = !hasNext;
    viewMoreButton.disabled = !hasNext;
  };

  const renderSearch = () => {
    const isVisible = searchOpen || Boolean(activeQuery);
    if (searchForm) {
      searchForm.classList.toggle('is-open', isVisible);
      searchForm.setAttribute('aria-hidden', String(!isVisible));
    }
    if (searchToggle) {
      searchToggle.classList.toggle('is-active', isVisible);
      searchToggle.setAttribute('aria-expanded', String(isVisible));
      searchToggle.setAttribute('aria-label', isVisible ? 'Close search' : 'Search feed');
      searchToggle.setAttribute('title', isVisible ? 'Close search' : 'Search feed');
    }
    if (searchSourceInput) {
      searchSourceInput.value = activeFilter === 'all' ? '' : activeFilter;
    }
    if (controlsRow) {
      controlsRow.classList.toggle('is-search-active', isVisible);
    }
  };

  const updateURL = () => {
    const url = new URL(window.location.href);
    if (activeFilter === 'all') {
      url.searchParams.delete('source');
    } else {
      url.searchParams.set('source', activeFilter);
    }
    if (activeQuery) {
      url.searchParams.set('q', activeQuery);
    } else {
      url.searchParams.delete('q');
    }
    history.replaceState({}, '', `${url.pathname}${url.search}`);
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

  const fetchItems = async ({ source, query, offset, append }) => {
    const requestId = ++requestSequence;
    const url = new URL('/api/items', window.location.origin);
    url.searchParams.set('limit', String(pageSize));
    url.searchParams.set('offset', String(offset));
    if (source && source !== 'all') {
      url.searchParams.set('source', source);
    }
    if (query) {
      url.searchParams.set('q', query);
    }
    const response = await fetch(url.toString());
    if (!response.ok) {
      throw new Error(`fetch failed: ${response.status}`);
    }
    const payload = await response.json();
    if (requestId !== requestSequence) {
      return;
    }
    const items = payload.items || [];
    hasNext = Boolean(payload.has_next);
    cardsGrid.dataset.hasNext = hasNext ? 'true' : 'false';
    cardsGrid.dataset.currentSource = source;

    if (!append) {
      cardsGrid.innerHTML = '';
      loadedCount = 0;
    }

    const html = items.map((item) => cardTemplate(item)).join('');
    cardsGrid.insertAdjacentHTML('beforeend', html);
    loadedCount += items.length;
    renderViewMore();
  };

  const cancelPendingSearch = () => {
    if (searchTimer) {
      window.clearTimeout(searchTimer);
      searchTimer = null;
    }
  };

  const currentSearchInputValue = () => (searchInput?.value || '').trim();

  const applySearch = async (nextQuery, { collapseWhenEmpty = false } = {}) => {
    cancelPendingSearch();
    activeQuery = nextQuery;
    searchOpen = Boolean(nextQuery) || (searchOpen && !collapseWhenEmpty);
    loadedCount = 0;
    renderSearch();
    updateURL();
    await fetchItems({ source: activeFilter, query: activeQuery, offset: 0, append: false });
  };

  const scheduleSearch = () => {
    searchOpen = true;
    renderSearch();
    cancelPendingSearch();
    searchTimer = window.setTimeout(async () => {
      try {
        await applySearch(currentSearchInputValue());
      } catch (error) {
        showToast('Failed to search feed', 'error');
      }
    }, searchDebounceMs);
  };

  filterButtons.forEach((button) => {
    button.addEventListener('click', async () => {
      const nextFilter = button.dataset.filter || 'all';
      if (nextFilter === activeFilter) return;
      cancelPendingSearch();
      activeFilter = nextFilter;
      activeQuery = currentSearchInputValue();
      searchOpen = searchOpen || Boolean(activeQuery);
      loadedCount = 0;
      renderFilters();
      renderSearch();
      updateURL();
      try {
        await fetchItems({ source: activeFilter, query: activeQuery, offset: 0, append: false });
      } catch (error) {
        showToast('Failed to load feed', 'error');
      }
    });
  });

  if (searchToggle) {
    searchToggle.addEventListener('click', async () => {
      const hasDraftOrQuery = Boolean(currentSearchInputValue() || activeQuery);
      if (!searchOpen && !hasDraftOrQuery) {
        searchOpen = true;
        renderSearch();
        if (searchInput) {
          window.requestAnimationFrame(() => {
            searchInput.focus({ preventScroll: true });
            const valueLength = searchInput.value.length;
            searchInput.setSelectionRange(valueLength, valueLength);
          });
        }
        return;
      }

      cancelPendingSearch();
      if (searchInput) {
        searchInput.value = '';
      }
      try {
        await applySearch('', { collapseWhenEmpty: true });
      } catch (error) {
        showToast('Failed to clear search', 'error');
      }
    });
  }

  if (searchForm) {
    searchForm.addEventListener('submit', async (event) => {
      event.preventDefault();
      try {
        await applySearch(currentSearchInputValue());
      } catch (error) {
        showToast('Failed to search feed', 'error');
      }
    });
  }

  if (searchInput) {
    searchInput.addEventListener('input', () => {
      scheduleSearch();
    });

    searchInput.addEventListener('keydown', (event) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        cancelPendingSearch();
        if (searchInput) {
          searchInput.value = '';
        }
        applySearch('', { collapseWhenEmpty: true }).catch(() => {
          showToast('Failed to clear search', 'error');
        });
        searchToggle?.focus();
      }
    });
  }

  if (viewMoreButton) {
    viewMoreButton.addEventListener('click', async () => {
      viewMoreButton.disabled = true;
      try {
        await fetchItems({ source: activeFilter, query: activeQuery, offset: loadedCount, append: true });
      } catch (error) {
        showToast('Failed to load more items', 'error');
      } finally {
        viewMoreButton.disabled = !hasNext;
      }
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

  renderFilters();
  renderSearch();
  renderViewMore();

  function hostLabel(rawURL) {
    try {
      const url = new URL(rawURL);
      return url.hostname.replace(/^www\./, '');
    } catch {
      return rawURL;
    }
  }

  function escapeHtml(value) {
    return String(value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function escapeAttr(value) {
    return escapeHtml(value);
  }
})();
