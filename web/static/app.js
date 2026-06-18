(() => {
  const root = document.documentElement;
  const availableSources = ['hackernews', 'github', 'huggingface', 'alphaxiv'];
  const legacyDefaultSources = ['hackernews', 'github', 'huggingface'];
  const sourceLabels = {
    all: 'All enabled sources',
    hackernews: 'Hacker News',
    github: 'GitHub Trending',
    huggingface: 'Hugging Face Papers Trending',
    alphaxiv: 'alphaXiv',
  };
  const sourceIconPaths = {
    hackernews: '/static/source-icons/hackernews.svg',
    github: '/static/source-icons/github.svg',
    huggingface: '/static/source-icons/huggingface.svg',
    alphaxiv: '/static/source-icons/alphaxiv.png',
  };
  const filterNav = document.querySelector('[data-filter-nav]');
  const controlsRow = document.querySelector('[data-controls-row]');
  const configOpenButton = document.querySelector('[data-source-config-open]');
  const configDialog = document.querySelector('[data-source-config-dialog]');
  const configCloseButtons = Array.from(document.querySelectorAll('[data-source-config-close], [data-source-config-cancel]'));
  const configSaveButton = document.querySelector('[data-source-config-save]');
  const configOptions = Array.from(document.querySelectorAll('[data-source-option]'));
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
  const sourceConfigStorageKey = 'feedreader.sources';
  const themeStorageKey = 'feedreader.theme';
  const refreshToastStorageKey = 'feedreader.toast';
  const metaThemeColor = document.querySelector('meta[name="theme-color"]');

  let activeFilter = cardsGrid?.dataset.currentSource || 'all';
  let selectedSources = loadSelectedSources();
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
      ${item.brief ? `<p class="item-brief"><span class="item-brief-text">${escapeHtml(item.brief)}</span></p>` : ''}
      <p class="item-host"><img class="source-icon-image source-icon-image--host source-icon-image--${escapeAttr(item.source || '')}" src="${escapeAttr(sourceIconPaths[item.source] || '')}" alt="" aria-hidden="true" /><span class="item-host-text">${escapeHtml(item.host || hostLabel(item.url || ''))}</span></p>
    </article>
  `;

  function normalizeSelectedSources(rawValue) {
    const values = Array.isArray(rawValue) ? rawValue : [];
    const seen = new Set();
    return values.filter((value) => {
      if (!availableSources.includes(value) || seen.has(value)) return false;
      seen.add(value);
      return true;
    });
  }

  function loadSelectedSources() {
    try {
      const parsed = JSON.parse(localStorage.getItem(sourceConfigStorageKey) || 'null');
      const normalized = normalizeSelectedSources(parsed);
      if (normalized.length === legacyDefaultSources.length && legacyDefaultSources.every((source, index) => normalized[index] === source)) {
        return [...availableSources];
      }
      return normalized.length > 0 ? normalized : [...availableSources];
    } catch {
      return [...availableSources];
    }
  }

  function persistSelectedSources() {
    localStorage.setItem(sourceConfigStorageKey, JSON.stringify(selectedSources));
  }

  function shouldRestrictAllSources() {
    return selectedSources.length > 0 && selectedSources.length < availableSources.length;
  }

  function visibleFilterKeys() {
    if (selectedSources.length > 1) {
      return ['all', ...selectedSources];
    }
    return [...selectedSources];
  }

  function renderFilters() {
    if (!filterNav) return;
    const keys = visibleFilterKeys();
    filterNav.innerHTML = keys.map((key) => {
      const isActive = key === activeFilter;
      if (key === 'all') {
        return `<button class="filter-button${isActive ? ' is-active' : ''}" type="button" data-filter="${key}" aria-pressed="${String(isActive)}" aria-label="${escapeAttr(sourceLabels[key] || key)}" title="${escapeAttr(sourceLabels[key] || key)}">All</button>`;
      }
      return `<button class="filter-button filter-button--icon${isActive ? ' is-active' : ''}" type="button" data-filter="${key}" aria-pressed="${String(isActive)}" aria-label="${escapeAttr(sourceLabels[key] || key)}" title="${escapeAttr(sourceLabels[key] || key)}"><img class="source-icon-image source-icon-image--filter source-icon-image--${escapeAttr(key)}" src="${escapeAttr(sourceIconPaths[key] || '')}" alt="" aria-hidden="true" /></button>`;
    }).join(''); // source keys/icon paths are fixed local constants, not user content
  }

  function syncConfigOptions() {
    configOptions.forEach((option) => {
      option.checked = selectedSources.includes(option.value);
    });
  }

  function currentSourceSelection() {
    return configOptions.filter((option) => option.checked).map((option) => option.value);
  }

  function ensureActiveFilterIsVisible() {
    const visibleKeys = visibleFilterKeys();
    if (!visibleKeys.includes(activeFilter)) {
      activeFilter = visibleKeys[0] || 'all';
    }
  }

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
    } else if (shouldRestrictAllSources()) {
      url.searchParams.set('sources', selectedSources.join(','));
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

  async function refetchCurrentView() {
    loadedCount = 0;
    ensureActiveFilterIsVisible();
    renderFilters();
    renderSearch();
    updateURL();
    await fetchItems({ source: activeFilter, query: activeQuery, offset: 0, append: false });
  }

  function openConfigDialog() {
    syncConfigOptions();
    if (typeof configDialog?.showModal === 'function' && !configDialog.open) {
      configDialog.showModal();
      return;
    }
    if (configDialog) {
      configDialog.setAttribute('open', 'open');
    }
  }

  function closeConfigDialog() {
    if (configDialog?.open && typeof configDialog.close === 'function') {
      configDialog.close();
      return;
    }
    configDialog?.removeAttribute('open');
  }

  async function applySelectedSources(nextSources) {
    const normalized = normalizeSelectedSources(nextSources);
    if (normalized.length === 0) {
      showToast('Select at least one source', 'error');
      return;
    }
    selectedSources = normalized;
    persistSelectedSources();
    syncConfigOptions();
    closeConfigDialog();
    await refetchCurrentView();
  }

  if (filterNav) {
    filterNav.addEventListener('click', async (event) => {
      const button = event.target.closest('[data-filter]');
      if (!button) return;
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
  }

  if (configOpenButton) {
    configOpenButton.addEventListener('click', () => {
      openConfigDialog();
    });
  }

  configCloseButtons.forEach((button) => {
    button.addEventListener('click', () => {
      closeConfigDialog();
    });
  });

  if (configSaveButton) {
    configSaveButton.addEventListener('click', async () => {
      try {
        await applySelectedSources(currentSourceSelection());
      } catch (error) {
        showToast('Failed to apply source settings', 'error');
      }
    });
  }

  if (configDialog) {
    configDialog.addEventListener('cancel', (event) => {
      event.preventDefault();
      closeConfigDialog();
    });
  }

  if (filterNav) {
    filterNav.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' && document.activeElement?.dataset?.filter) {
        document.activeElement.click();
      }
    });
  }

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

  syncConfigOptions();
  const shouldBootstrapRefetch = activeFilter === 'all'
    ? selectedSources.length !== availableSources.length
    : !selectedSources.includes(activeFilter);
  ensureActiveFilterIsVisible();
  renderFilters();
  renderSearch();
  renderViewMore();

  if (shouldBootstrapRefetch) {
    refetchCurrentView().catch(() => {
      showToast('Failed to load configured sources', 'error');
    });
  }

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
