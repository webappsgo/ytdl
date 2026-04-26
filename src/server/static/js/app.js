// ytdl - Main application JavaScript
// Vanilla JS only, no frameworks per PART 16
// See AI.md for frontend specifications

(function() {
  'use strict';

  // State
  const state = {
    downloads: [],
    ws: null,
    theme: localStorage.getItem('ytdl-theme') || 'auto',
    currentTab: 'download'
  };

  // Initialize
  document.addEventListener('DOMContentLoaded', init);

  function init() {
    applyTheme(state.theme);
    setupEventListeners();
    connectWebSocket();
    loadDownloads();
  }

  // Theme management
  function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    state.theme = theme;
    localStorage.setItem('ytdl-theme', theme);
  }

  // WebSocket connection for real-time progress
  function connectWebSocket() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    state.ws = new WebSocket(protocol + '//' + location.host + '/ws');

    state.ws.onmessage = function(event) {
      const msg = JSON.parse(event.data);
      handleWSMessage(msg);
    };

    state.ws.onclose = function() {
      // Reconnect after 3 seconds
      setTimeout(connectWebSocket, 3000);
    };

    state.ws.onerror = function() {
      state.ws.close();
    };
  }

  function handleWSMessage(msg) {
    if (msg.type === 'progress') {
      updateDownloadProgress(msg.data);
    }
  }

  function updateDownloadProgress(data) {
    const el = document.querySelector('[data-download-id="' + data.download_id + '"]');
    if (!el) return;

    const progressBar = el.querySelector('.queue__progress-bar');
    if (progressBar) {
      progressBar.style.width = data.percent + '%';
    }

    const meta = el.querySelector('.queue__meta');
    if (meta && data.speed) {
      meta.textContent = Math.round(data.percent) + '% - ' + data.speed + ' - ETA: ' + data.eta;
    }
  }

  // Event listeners
  function setupEventListeners() {
    // Download form
    const form = document.getElementById('downloadForm');
    if (form) {
      form.addEventListener('submit', handleDownloadSubmit);
    }

    // Search form
    const searchForm = document.getElementById('searchForm');
    if (searchForm) {
      searchForm.addEventListener('submit', handleSearch);
    }

    // Tab switching
    document.querySelectorAll('.tabs__tab').forEach(function(tab) {
      tab.addEventListener('click', function() {
        switchTab(this.dataset.tab);
      });
    });

    // Theme toggle
    const themeBtn = document.getElementById('themeToggle');
    if (themeBtn) {
      themeBtn.addEventListener('click', function() {
        const themes = ['dark', 'light', 'auto'];
        const idx = (themes.indexOf(state.theme) + 1) % themes.length;
        applyTheme(themes[idx]);
      });
    }
  }

  // Submit download
  async function handleDownloadSubmit(e) {
    e.preventDefault();

    const urlInput = document.getElementById('downloadUrl');
    const formatSelect = document.getElementById('downloadFormat');
    const qualitySelect = document.getElementById('downloadQuality');

    const url = urlInput.value.trim();
    if (!url) return;

    // Check for multiple URLs (batch mode)
    const urls = url.split('\n').map(function(u) { return u.trim(); }).filter(Boolean);

    try {
      let response;
      if (urls.length > 1) {
        response = await fetch('/api/v1/downloads/batch', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            urls: urls,
            format: formatSelect ? formatSelect.value : 'mp4',
            quality: qualitySelect ? qualitySelect.value : '1080'
          })
        });
      } else {
        response = await fetch('/api/v1/downloads', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            url: urls[0],
            format: formatSelect ? formatSelect.value : 'mp4',
            quality: qualitySelect ? qualitySelect.value : '1080'
          })
        });
      }

      const data = await response.json();
      if (response.ok) {
        urlInput.value = '';
        showToast('Download queued', 'success');
        loadDownloads();
      } else {
        showToast(data.error || 'Failed to submit', 'error');
      }
    } catch (err) {
      showToast('Connection error', 'error');
    }
  }

  // Search
  async function handleSearch(e) {
    e.preventDefault();
    const query = document.getElementById('searchInput').value.trim();
    if (!query) return;

    const resultsEl = document.getElementById('searchResults');
    if (resultsEl) resultsEl.innerHTML = '<p class="text-muted text-center">Searching...</p>';

    try {
      const response = await fetch('/api/v1/search?q=' + encodeURIComponent(query) + '&limit=12');
      const data = await response.json();

      if (data.data && data.data.length > 0) {
        renderSearchResults(data.data);
      } else {
        resultsEl.innerHTML = '<p class="text-muted text-center">No results found</p>';
      }
    } catch (err) {
      resultsEl.innerHTML = '<p class="text-muted text-center">Search failed</p>';
    }
  }

  function renderSearchResults(results) {
    const el = document.getElementById('searchResults');
    if (!el) return;

    el.innerHTML = '';
    const grid = document.createElement('div');
    grid.className = 'media-grid';

    results.forEach(function(item) {
      const card = document.createElement('div');
      card.className = 'media-card';
      card.onclick = function() { downloadFromSearch(item); };

      card.innerHTML =
        '<img class="media-card__thumb" src="' + (item.thumbnail || '') + '" alt="" loading="lazy">' +
        '<div class="media-card__body">' +
        '<div class="media-card__title">' + escapeHtml(item.title || 'Untitled') + '</div>' +
        '<div class="media-card__channel">' + escapeHtml(item.uploader || '') + '</div>' +
        '<div class="media-card__meta">' + formatDuration(item.duration) + '</div>' +
        '</div>';

      grid.appendChild(card);
    });

    el.appendChild(grid);
  }

  function downloadFromSearch(item) {
    const urlInput = document.getElementById('downloadUrl');
    if (urlInput) {
      urlInput.value = item.webpage_url || item.url || '';
      switchTab('download');
      urlInput.focus();
    }
  }

  // Load downloads
  async function loadDownloads() {
    try {
      const response = await fetch('/api/v1/downloads?per_page=50');
      const data = await response.json();

      if (data.data) {
        state.downloads = data.data;
        renderQueue(data.data);
      }
    } catch (err) {
      // Silent fail
    }
  }

  function renderQueue(downloads) {
    const el = document.getElementById('downloadQueue');
    if (!el) return;

    if (!downloads || downloads.length === 0) {
      el.innerHTML = '<p class="text-muted text-center mt-2">No downloads yet. Paste a URL above to get started.</p>';
      return;
    }

    el.innerHTML = '';
    downloads.forEach(function(d) {
      const item = document.createElement('div');
      item.className = 'queue__item';
      item.setAttribute('data-download-id', d.id);

      const thumbSrc = d.thumbnail_url || '';
      const statusClass = 'badge--' + d.status;

      item.innerHTML =
        '<img class="queue__thumb" src="' + thumbSrc + '" alt="" loading="lazy">' +
        '<div class="queue__info">' +
        '<div class="queue__title">' + escapeHtml(d.title || d.url) + '</div>' +
        '<div class="queue__meta">' +
        '<span class="badge ' + statusClass + '">' + d.status + '</span> ' +
        (d.source_site ? escapeHtml(d.source_site) + ' - ' : '') +
        d.format.toUpperCase() + ' ' + d.quality + 'p' +
        (d.file_size > 0 ? ' - ' + formatSize(d.file_size) : '') +
        '</div>' +
        (d.status === 'downloading' ?
          '<div class="queue__progress"><div class="queue__progress-bar" style="width:' + d.progress_percent + '%"></div></div>' : '') +
        '</div>' +
        '<div class="queue__actions">' +
        (d.status === 'completed' ? '<a class="btn btn--sm btn--primary" href="/api/v1/downloads/' + d.id + '/file">Download</a>' : '') +
        (d.status === 'queued' ? '<button class="btn btn--sm btn--secondary" onclick="cancelDownload(' + d.id + ')">Cancel</button>' : '') +
        (d.status === 'failed' ? '<button class="btn btn--sm btn--secondary" onclick="retryDownload(' + d.id + ')">Retry</button>' : '') +
        (d.status === 'completed' ? '<button class="btn btn--sm btn--secondary" onclick="deleteDownload(' + d.id + ')">Delete</button>' : '') +
        '</div>';

      el.appendChild(item);
    });
  }

  // Tab switching
  function switchTab(tab) {
    state.currentTab = tab;

    document.querySelectorAll('.tabs__tab').forEach(function(t) {
      t.classList.toggle('tabs__tab--active', t.dataset.tab === tab);
    });

    document.querySelectorAll('.tab-content').forEach(function(c) {
      c.classList.toggle('hidden', c.dataset.tab !== tab);
    });

    if (tab === 'library') {
      loadLibrary();
    }
  }

  // Load library
  async function loadLibrary() {
    const el = document.getElementById('libraryGrid');
    if (!el) return;

    try {
      const response = await fetch('/api/v1/library?per_page=50');
      const data = await response.json();

      if (data.data && data.data.length > 0) {
        el.innerHTML = '';
        const grid = document.createElement('div');
        grid.className = 'media-grid';

        data.data.forEach(function(item) {
          const card = document.createElement('div');
          card.className = 'media-card';

          card.innerHTML =
            '<img class="media-card__thumb" src="' + (item.thumbnail_url || '') + '" alt="" loading="lazy">' +
            '<div class="media-card__body">' +
            '<div class="media-card__title">' + escapeHtml(item.title || 'Untitled') + '</div>' +
            '<div class="media-card__channel">' + escapeHtml(item.channel_name || '') + '</div>' +
            '<div class="media-card__meta">' + formatDuration(item.duration) + ' - ' + formatSize(item.file_size) + '</div>' +
            '</div>';

          card.onclick = function() {
            window.location.href = '/api/v1/downloads/' + item.id + '/file';
          };

          grid.appendChild(card);
        });

        el.appendChild(grid);
      } else {
        el.innerHTML = '<p class="text-muted text-center mt-2">No completed downloads yet.</p>';
      }
    } catch (err) {
      el.innerHTML = '<p class="text-muted text-center mt-2">Failed to load library.</p>';
    }
  }

  // Actions (exposed globally for onclick handlers)
  window.cancelDownload = async function(id) {
    await fetch('/api/v1/downloads/' + id + '/cancel', { method: 'POST' });
    loadDownloads();
  };

  window.retryDownload = async function(id) {
    await fetch('/api/v1/downloads/' + id + '/retry', { method: 'POST' });
    loadDownloads();
  };

  window.deleteDownload = async function(id) {
    await fetch('/api/v1/downloads/' + id, { method: 'DELETE' });
    loadDownloads();
  };

  // Toast notifications
  function showToast(message, type) {
    var container = document.querySelector('.toast-container');
    if (!container) {
      container = document.createElement('div');
      container.className = 'toast-container';
      document.body.appendChild(container);
    }

    var toast = document.createElement('div');
    toast.className = 'toast';
    toast.style.borderLeftColor = type === 'error' ? 'var(--error)' :
                                  type === 'success' ? 'var(--success)' : 'var(--accent)';
    toast.style.borderLeftWidth = '3px';
    toast.style.borderLeftStyle = 'solid';
    toast.textContent = message;

    container.appendChild(toast);

    setTimeout(function() {
      toast.remove();
    }, 4000);
  }

  // Helpers
  function escapeHtml(text) {
    var div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  function formatDuration(seconds) {
    if (!seconds) return '';
    var h = Math.floor(seconds / 3600);
    var m = Math.floor((seconds % 3600) / 60);
    var s = seconds % 60;
    if (h > 0) return h + ':' + pad(m) + ':' + pad(s);
    return m + ':' + pad(s);
  }

  function pad(n) {
    return n < 10 ? '0' + n : '' + n;
  }

  function formatSize(bytes) {
    if (!bytes) return '';
    var units = ['B', 'KB', 'MB', 'GB', 'TB'];
    var i = 0;
    var size = bytes;
    while (size >= 1024 && i < units.length - 1) {
      size /= 1024;
      i++;
    }
    return size.toFixed(i > 0 ? 1 : 0) + ' ' + units[i];
  }

  // Auto-refresh downloads every 5 seconds
  setInterval(loadDownloads, 5000);
})();
