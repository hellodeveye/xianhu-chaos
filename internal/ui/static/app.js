const state = {
  providers: [],
  recentRequests: [],
  lastResponse: '{}',
  lastRequest: null,
  collapsedGroups: new Set(),
  expandedRequests: new Set(),
  providersSig: '',
  autoRefresh: true,
  pollTimer: null,
};

const THEME_KEY = 'xianhu-chaos-theme';
const POLL_MS = 3000;

// Double-chevron glyphs for the per-provider expand / collapse-all route
// controls: chevrons-down = expand (reveal), chevrons-up = collapse. This
// mirrors the rotating chevron on each route row.
const ICON_EXPAND = '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M3 4.5 8 9l5-4.5"/><path d="M3 9.5 8 14l5-4.5"/></svg>';
const ICON_COLLAPSE = '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M3 11.5 8 7l5 4.5"/><path d="M3 6.5 8 2l5 4.5"/></svg>';

const presets = [
  {
    id: 'login',
    provider: 'umember',
    routeId: 'login',
    label: 'Umember login',
    method: 'POST',
    path: '/open/login',
    body: () => ({ email: 'test@example.com', password: 'test' }),
  },
  {
    id: 'detail',
    provider: 'umember',
    routeId: 'meituan_coupon_detail',
    label: 'Meituan coupon detail',
    method: 'GET',
    path: ({ couponCode }) => `/open/user/meituan/coupon/detail?store_id=8674228&coupon_code=${encodeURIComponent(couponCode)}`,
  },
  {
    id: 'verify',
    provider: 'umember',
    routeId: 'meituan_coupon_verify',
    label: 'Meituan coupon verify',
    method: 'POST',
    path: '/open/user/meituan/coupon/verify',
    body: ({ couponCode }) => ({ store_id: '8674228', coupon_code: couponCode }),
  },
  {
    id: 'store_list',
    provider: 'umember',
    routeId: 'meituan_store_list',
    label: 'Meituan store list',
    method: 'GET',
    path: '/open/user/meituan/store/list',
  },
  {
    id: 'account',
    provider: 'umember',
    routeId: 'user_detail',
    label: 'Umember account',
    method: 'GET',
    path: '/open/user/detail',
  },
  {
    id: 'logs',
    provider: 'umember',
    routeId: 'request_logs',
    label: 'Request logs API',
    method: 'POST',
    path: '/open/user/request/logs',
    body: () => ({ page: 1, rows: 10 }),
  },
  {
    id: 'dy_token',
    provider: 'douyin',
    routeId: 'client_token',
    label: 'Douyin client token',
    method: 'POST',
    path: '/oauth/client_token/',
    body: () => ({
      client_key: 'chaos-client-key',
      client_secret: 'chaos-client-secret',
      grant_type: 'client_credential',
    }),
  },
  {
    id: 'dy_prepare',
    provider: 'douyin',
    routeId: 'certificate_prepare',
    label: 'Douyin certificate prepare',
    method: 'GET',
    path: ({ couponCode }) => `/goodlife/v1/fulfilment/certificate/prepare/?poi_id=7630290236999731263&code=${encodeURIComponent(couponCode)}`,
  },
  {
    id: 'dy_verify',
    provider: 'douyin',
    routeId: 'certificate_verify',
    label: 'Douyin certificate verify',
    method: 'POST',
    path: '/goodlife/v1/fulfilment/certificate/verify/',
    body: ({ couponCode }) => ({
      verify_token: couponCode || 'chaos-verify-token',
      poi_id: '7630290236999731263',
      encrypted_codes: ['chaos-encrypted-code'],
    }),
  },
];

const els = {
  healthBadge: document.querySelector('#healthBadge'),
  themeToggle: document.querySelector('#themeToggle'),
  refreshButton: document.querySelector('#refreshButton'),
  resetButton: document.querySelector('#resetButton'),
  statProviders: document.querySelector('#statProviders'),
  statRoutes: document.querySelector('#statRoutes'),
  statScenarios: document.querySelector('#statScenarios'),
  statRequests: document.querySelector('#statRequests'),
  providersList: document.querySelector('#providersList'),
  providerFilter: document.querySelector('#providerFilter'),
  filterHint: document.querySelector('#filterHint'),
  requestRows: document.querySelector('#requestRows'),
  logCount: document.querySelector('#logCount'),
  liveDot: document.querySelector('#liveDot'),
  autoRefreshToggle: document.querySelector('#autoRefreshToggle'),
  probeForm: document.querySelector('#probeForm'),
  probeProvider: document.querySelector('#probeProvider'),
  probePreset: document.querySelector('#probePreset'),
  couponCode: document.querySelector('#couponCode'),
  scenarioHeader: document.querySelector('#scenarioHeader'),
  probePreviewMethod: document.querySelector('#probePreviewMethod'),
  probePreviewPath: document.querySelector('#probePreviewPath'),
  responseCode: document.querySelector('#responseCode'),
  responseStatus: document.querySelector('#responseStatus'),
  responseBody: document.querySelector('#responseBody'),
  copyResponseButton: document.querySelector('#copyResponseButton'),
  copyRequestButton: document.querySelector('#copyRequestButton'),
  toast: document.querySelector('#toast'),
  modal: document.querySelector('#scenarioModal'),
  modalTitle: document.querySelector('#modalTitle'),
  modalSubtitle: document.querySelector('#modalSubtitle'),
  modalClose: document.querySelector('#modalClose'),
  modalCancel: document.querySelector('#modalCancel'),
  modalSave: document.querySelector('#modalSave'),
  modalReset: document.querySelector('#modalReset'),
  modalStatus: document.querySelector('#modalStatus'),
  modalContentType: document.querySelector('#modalContentType'),
  modalBody: document.querySelector('#modalBody'),
  modalOverrideBadge: document.querySelector('#modalOverrideBadge'),
};

const modalState = {
  provider: '',
  scenario: '',
  hasOverride: false,
};

function init() {
  initTheme();
  renderProbeProviderOptions();
  renderPresetOptions();
  updateProbePreview();
  bindEvents();
  refreshAll();
  startAutoRefresh();
}

function bindEvents() {
  els.themeToggle.addEventListener('click', toggleTheme);
  els.refreshButton.addEventListener('click', refreshAll);
  els.resetButton.addEventListener('click', resetState);
  els.providerFilter.addEventListener('input', applyProviderFilter);
  els.autoRefreshToggle.addEventListener('change', onAutoRefreshToggle);
  els.probeProvider.addEventListener('change', onProbeProviderChange);
  els.probePreset.addEventListener('change', onProbePresetChange);
  els.couponCode.addEventListener('input', updateProbePreview);
  els.probeForm.addEventListener('submit', sendProbe);
  els.copyResponseButton.addEventListener('click', copyResponse);
  els.copyRequestButton.addEventListener('click', copyRequest);
  els.modalClose.addEventListener('click', closeScenarioModal);
  els.modalCancel.addEventListener('click', closeScenarioModal);
  els.modalSave.addEventListener('click', saveScenarioOverride);
  els.modalReset.addEventListener('click', clearScenarioOverride);
  els.modal.addEventListener('click', (event) => {
    if (event.target === els.modal) closeScenarioModal();
  });
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && !els.modal.hidden) closeScenarioModal();
  });
}

function renderProbeProviderOptions() {
  const providers = [];
  presets.forEach((preset) => {
    if (!providers.includes(preset.provider)) providers.push(preset.provider);
  });
  els.probeProvider.innerHTML = providers
    .map((name) => `<option value="${escapeAttr(name)}">${escapeHTML(name)}</option>`)
    .join('');
}

function currentProbeProvider() {
  return els.probeProvider.value || (presets[0] && presets[0].provider) || '';
}

function currentProbePreset() {
  const selected = presets.find((preset) => preset.id === els.probePreset.value);
  if (selected) return selected;
  return presets.find((preset) => preset.provider === currentProbeProvider()) || null;
}

function renderPresetOptions() {
  const provider = currentProbeProvider();
  els.probePreset.innerHTML = presets
    .filter((preset) => preset.provider === provider)
    .map((preset) => `<option value="${preset.id}">${escapeHTML(preset.label)}</option>`)
    .join('');
}

function onProbeProviderChange() {
  renderPresetOptions();
  renderScenarioHeaderOptions();
  updateProbePreview();
}

function onProbePresetChange() {
  renderScenarioHeaderOptions();
  updateProbePreview();
}

function updateProbePreview() {
  const preset = currentProbePreset();
  if (!preset) {
    els.probePreviewMethod.textContent = '—';
    els.probePreviewMethod.className = 'method-badge other';
    els.probePreviewPath.textContent = '—';
    return;
  }
  const couponCode = (els.couponCode.value || '').trim() || '0109760017002';
  const path = typeof preset.path === 'function' ? preset.path({ couponCode }) : preset.path;
  els.probePreviewMethod.textContent = preset.method;
  els.probePreviewMethod.className = `method-badge ${methodClass(preset.method)}`;
  els.probePreviewPath.textContent = path;
}

async function refreshAll() {
  await Promise.all([checkHealth(), loadState()]);
}

async function checkHealth() {
  try {
    const res = await fetch('/health');
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    els.healthBadge.textContent = 'healthy';
    els.healthBadge.className = 'status-pill success';
  } catch (error) {
    els.healthBadge.textContent = 'offline';
    els.healthBadge.className = 'status-pill warning';
  }
}

async function loadState() {
  try {
    const data = await getJSON('/__admin/state');
    state.providers = data.providers || [];
    state.recentRequests = data.recentRequests || [];
    state.providersSig = JSON.stringify(state.providers);
    renderStats();
    renderProviders();
    renderScenarioHeaderOptions();
    renderRequests();
    updateProbePreview();
  } catch (error) {
    showToast(error.message);
  }
}

// pollState backs the auto-refresh loop. It always refreshes the lightweight
// stats and traffic table, but only re-renders the providers panel when the
// data actually changed and the user is not mid-interaction (editing an
// override in the modal or focused on a control inside the panel).
async function pollState() {
  if (!state.autoRefresh) return;
  checkHealth();
  try {
    const data = await getJSON('/__admin/state');
    state.providers = data.providers || [];
    state.recentRequests = data.recentRequests || [];
    renderStats();
    renderRequests();
    const sig = JSON.stringify(state.providers);
    const safeToRender = els.modal.hidden && !els.providersList.contains(document.activeElement);
    if (sig !== state.providersSig && safeToRender) {
      renderProviders();
      renderScenarioHeaderOptions();
      state.providersSig = sig;
    }
  } catch (error) {
    /* keep last good state during transient poll errors */
  }
}

function renderStats() {
  const providers = state.providers || [];
  const enabled = providers.filter((p) => p.enabled).length;
  let routes = 0;
  let scenarios = 0;
  providers.forEach((p) => {
    routes += (p.routes || []).length;
    scenarios += (p.scenarios || []).length;
  });
  els.statProviders.textContent = providers.length ? `${enabled}/${providers.length}` : '0';
  els.statRoutes.textContent = String(routes);
  els.statScenarios.textContent = String(scenarios);
  els.statRequests.textContent = String((state.recentRequests || []).length);
}

function initTheme() {
  let theme = document.documentElement.getAttribute('data-theme');
  if (theme !== 'dark' && theme !== 'light') {
    try {
      theme = localStorage.getItem(THEME_KEY);
    } catch (error) {
      theme = null;
    }
    if (theme !== 'dark' && theme !== 'light') {
      theme = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
  }
  applyTheme(theme);
}

function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  els.themeToggle.textContent = theme === 'dark' ? '☀️' : '🌙';
  els.themeToggle.setAttribute('aria-pressed', theme === 'dark' ? 'true' : 'false');
}

function toggleTheme() {
  const next = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
  applyTheme(next);
  try {
    localStorage.setItem(THEME_KEY, next);
  } catch (error) {
    /* ignore storage failures (private mode, etc.) */
  }
}

function startAutoRefresh() {
  stopAutoRefresh();
  state.autoRefresh = true;
  els.liveDot.classList.add('on');
  state.pollTimer = window.setInterval(pollState, POLL_MS);
}

function stopAutoRefresh() {
  state.autoRefresh = false;
  els.liveDot.classList.remove('on');
  if (state.pollTimer) {
    window.clearInterval(state.pollTimer);
    state.pollTimer = null;
  }
}

function onAutoRefreshToggle() {
  if (els.autoRefreshToggle.checked) startAutoRefresh();
  else stopAutoRefresh();
}

// renderGroupToggle builds the single per-provider toggle. Its icon reflects
// the action it will perform: chevrons-up to collapse when anything is open,
// chevrons-down to expand when everything is already collapsed.
function renderGroupToggle(provider) {
  const keys = (provider.routeGroups || []).map((group) => `${provider.name}::${group.routeId}`);
  if ((provider.sharedScenarios || []).length) keys.push(`${provider.name}::__shared__`);
  const anyOpen = keys.length === 0 || keys.some((key) => !state.collapsedGroups.has(key));
  const label = anyOpen ? 'Collapse all routes' : 'Expand all routes';
  const icon = anyOpen ? ICON_COLLAPSE : ICON_EXPAND;
  return `<button class="icon-button sm group-toggle" type="button" title="${label}" aria-label="${label}">${icon}</button>`;
}

// toggleProviderGroups flips every route group within the provider card that
// owns the clicked toggle: if any group is open it collapses them all,
// otherwise it expands them all. Other providers are left alone.
function toggleProviderGroups(event) {
  const card = event.currentTarget.closest('.provider-card');
  if (!card) return;
  const groups = card.querySelectorAll('details.route-group');
  const open = !cardHasOpenGroup(card);
  groups.forEach((details) => {
    if (details.classList.contains('group-hidden')) return;
    if (details.open !== open) {
      details._programmatic = true;
      details.open = open;
    }
    const key = details.dataset.key;
    if (!key) return;
    if (open) state.collapsedGroups.delete(key);
    else state.collapsedGroups.add(key);
  });
  syncGroupToggle(card);
}

function cardHasOpenGroup(card) {
  return Array.from(card.querySelectorAll('details.route-group'))
    .some((details) => !details.classList.contains('group-hidden') && details.open);
}

// syncGroupToggle keeps a provider's toggle icon/label in step with the actual
// open state, so it stays correct after manual single-row toggles or filtering.
function syncGroupToggle(card) {
  const button = card.querySelector('.group-toggle');
  if (!button) return;
  const anyOpen = cardHasOpenGroup(card);
  const label = anyOpen ? 'Collapse all routes' : 'Expand all routes';
  button.title = label;
  button.setAttribute('aria-label', label);
  button.innerHTML = anyOpen ? ICON_COLLAPSE : ICON_EXPAND;
}

function renderProviders() {
  els.providersList.classList.remove('skeleton-list');
  if (!state.providers.length) {
    els.providersList.innerHTML = '<div class="empty">No providers loaded.</div>';
    return;
  }
  els.providersList.innerHTML = state.providers.map(renderProvider).join('');
  els.providersList.querySelectorAll('select[data-provider]').forEach((select) => {
    select.addEventListener('change', setProviderScenario);
  });
  els.providersList.querySelectorAll('button.group-toggle').forEach((button) => {
    button.addEventListener('click', toggleProviderGroups);
  });
  els.providersList.querySelectorAll('button.scenario-chip').forEach((button) => {
    if (button.dataset.routeId) {
      button.addEventListener('click', setRouteScenarioFromChip);
    } else {
      button.addEventListener('click', openScenarioModal);
    }
  });
  els.providersList.querySelectorAll('details.route-group').forEach((details) => {
    details.addEventListener('toggle', () => {
      if (details._programmatic) {
        details._programmatic = false;
        return;
      }
      const key = details.dataset.key;
      if (!key) return;
      if (details.open) state.collapsedGroups.delete(key);
      else state.collapsedGroups.add(key);
      const card = details.closest('.provider-card');
      if (card) syncGroupToggle(card);
    });
  });
  applyProviderFilter();
}

function renderProvider(provider) {
  const globalScenarios = (provider.sharedScenarios || []).slice();
  const scenarioOptions = [''].concat(globalScenarios).map((scenario) => {
    const label = scenario || 'Default';
    const selected = (provider.globalScenario || '') === scenario ? 'selected' : '';
    return `<option value="${escapeAttr(scenario)}" ${selected}>${escapeHTML(label)}</option>`;
  }).join('');
  const summary = `${(provider.routes || []).length} routes &middot; ${(provider.scenarios || []).length} scenarios`;
  const groups = (provider.routeGroups || []).map((group) => renderRouteGroup(provider, group)).join('');
  const shared = (provider.sharedScenarios || []).length
    ? renderSharedGroup(provider, provider.sharedScenarios)
    : '';
  return `
    <article class="provider-card" data-provider="${escapeAttr(provider.name)}">
      <header class="provider-card-header">
        <div class="provider-heading">
          <span class="provider-name">${escapeHTML(provider.name)}</span>
          <span class="status-pill ${provider.enabled ? 'success' : 'disabled'}">${provider.enabled ? 'enabled' : 'disabled'}</span>
          <span class="provider-meta">${summary}</span>
          ${renderGroupToggle(provider)}
        </div>
        <label class="scenario-control">
          <span>Global scenario <span class="hint-mark" title="Only shared cross-cutting scenarios can be pinned globally. Route-specific scenarios stay under their owning route. Header and coupon-code rules still take precedence.">?</span></span>
          <select data-provider="${escapeAttr(provider.name)}" ${provider.enabled ? '' : 'disabled'}>
            ${scenarioOptions}
          </select>
        </label>
      </header>
      <div class="route-groups">
        ${groups}
        ${shared}
      </div>
    </article>
  `;
}

function renderRouteGroup(provider, group) {
  const key = `${provider.name}::${group.routeId}`;
  const open = state.collapsedGroups.has(key) ? '' : 'open';
  const method = (group.method || '').toUpperCase();
  const count = (group.scenarios || []).length;
  const activeScenario = group.activeScenario || group.defaultScenario;
  const chips = (group.scenarios || []).map((scenario) =>
    renderChip(provider.name, scenario, {
      isDefault: scenario === group.defaultScenario,
      isActive: scenario === activeScenario,
      routeId: group.routeId,
      defaultScenario: group.defaultScenario,
      activeScenario: group.activeScenario || '',
    })
  ).join('');
  const header = `${method} ${group.path}`.toLowerCase();
  return `
    <details class="route-group" data-key="${escapeAttr(key)}" data-header="${escapeAttr(header)}" ${open}>
      <summary class="route-summary">
        <span class="method-badge ${methodClass(method)}">${escapeHTML(method)}</span>
        <span class="route-path">${escapeHTML(group.path)}</span>
        <span class="route-count">${count}</span>
      </summary>
      <div class="scenario-chips">${chips}</div>
    </details>
  `;
}

function renderSharedGroup(provider, scenarios) {
  const key = `${provider.name}::__shared__`;
  const open = state.collapsedGroups.has(key) ? '' : 'open';
  const chips = scenarios.map((scenario) => renderChip(provider.name, scenario, {})).join('');
  return `
    <details class="route-group shared" data-key="${escapeAttr(key)}" data-header="shared cross-cutting" ${open}>
      <summary class="route-summary">
        <span class="group-label">Shared</span>
        <span class="route-hint">cross-cutting scenarios</span>
        <span class="route-count">${scenarios.length}</span>
      </summary>
      <div class="scenario-chips">${chips}</div>
    </details>
  `;
}

function renderChip(providerName, scenario, options) {
  const opts = options || {};
  const defaultClass = opts.isDefault ? ' is-default' : '';
  const activeClass = opts.isActive ? ' is-active' : '';
  const title = opts.routeId
    ? ' title="Select route scenario"'
    : (opts.isDefault ? ' title="Route default scenario"' : '');
  const routeAttrs = opts.routeId
    ? ` data-route-id="${escapeAttr(opts.routeId)}" data-default-scenario="${escapeAttr(opts.defaultScenario || '')}" data-active-scenario="${escapeAttr(opts.activeScenario || '')}"`
    : '';
  return `<button class="scenario-chip${defaultClass}${activeClass}" type="button" data-provider="${escapeAttr(providerName)}" data-scenario="${escapeAttr(scenario)}" data-name="${escapeAttr(scenario.toLowerCase())}"${routeAttrs}${title}>${escapeHTML(scenario)}</button>`;
}

function methodClass(method) {
  switch (method) {
    case 'GET': return 'get';
    case 'POST': return 'post';
    case 'PUT': return 'put';
    case 'DELETE': return 'delete';
    case 'PATCH': return 'patch';
    default: return 'other';
  }
}

function applyProviderFilter() {
  if (!els.providerFilter) return;
  const q = (els.providerFilter.value || '').trim().toLowerCase();
  let visibleGroups = 0;
  let visibleProviders = 0;
  els.providersList.querySelectorAll('.provider-card').forEach((card) => {
    const providerName = (card.dataset.provider || '').toLowerCase();
    const providerHit = !!q && providerName.includes(q);
    let cardVisible = false;
    card.querySelectorAll('details.route-group').forEach((details) => {
      const header = details.dataset.header || '';
      const headerHit = providerHit || (!!q && header.includes(q));
      let anyChip = false;
      details.querySelectorAll('.scenario-chip').forEach((chip) => {
        const hit = !q || headerHit || (chip.dataset.name || '').includes(q);
        chip.classList.toggle('chip-hidden', !hit);
        if (hit) anyChip = true;
      });
      const groupVisible = !q || headerHit || anyChip;
      details.classList.toggle('group-hidden', !groupVisible);
      if (groupVisible) {
        visibleGroups += 1;
        cardVisible = true;
        if (q && !details.open) {
          details._programmatic = true;
          details.open = true;
        }
      }
    });
    card.classList.toggle('card-hidden', !cardVisible);
    if (cardVisible) visibleProviders += 1;
    syncGroupToggle(card);
  });
  els.filterHint.textContent = q
    ? `${visibleGroups} route${visibleGroups === 1 ? '' : 's'} in ${visibleProviders} provider${visibleProviders === 1 ? '' : 's'}`
    : '';
}

function renderScenarioHeaderOptions() {
  const preset = currentProbePreset();
  const provider = preset ? preset.provider : currentProbeProvider();
  const match = state.providers.find((item) => item.name === provider);
  const routeGroup = match && preset
    ? (match.routeGroups || []).find((group) => group.routeId === preset.routeId)
    : null;
  const routeScenarios = routeGroup ? routeGroup.scenarios || [] : [];
  const sharedScenarios = match ? match.sharedScenarios || [] : [];
  const scenarios = Array.from(new Set(routeScenarios.concat(sharedScenarios)));
  const selectedValue = scenarios.includes(els.scenarioHeader.value) ? els.scenarioHeader.value : '';
  els.scenarioHeader.innerHTML = `<option value="" ${selectedValue === '' ? 'selected' : ''}>none</option>` + scenarios
    .map((scenario) => {
      const selected = scenario === selectedValue ? 'selected' : '';
      return `<option value="${escapeAttr(scenario)}" ${selected}>${escapeHTML(scenario)}</option>`;
    })
    .join('');
}

function renderRequests() {
  const count = state.recentRequests.length;
  els.logCount.textContent = String(count);
  if (!count) {
    els.requestRows.innerHTML = '<tr><td colspan="7" class="empty-cell">No mock traffic yet. Send a probe above or hit a mock route.</td></tr>';
    return;
  }
  els.requestRows.innerHTML = state.recentRequests.slice().reverse().map((entry) => {
    const sig = requestSig(entry);
    const open = state.expandedRequests.has(sig);
    const method = (entry.method || '').toUpperCase();
    return `
    <tr class="request-summary-row${open ? ' is-open' : ''}" data-sig="${escapeAttr(sig)}">
      <td class="col-toggle"><span class="row-toggle"></span></td>
      <td class="code">${escapeHTML(formatTime(entry.time))}</td>
      <td>${escapeHTML(entry.provider || '')}</td>
      <td><span class="method-badge ${methodClass(method)} req-method">${escapeHTML(method)}</span><span class="code">${escapeHTML(entry.routeId || '')}</span></td>
      <td><span class="status-pill neutral">${escapeHTML(entry.scenario || '')}</span></td>
      <td><span class="status-pill ${Number(entry.status) >= 400 ? 'warning' : 'success'}">${escapeHTML(String(entry.status || ''))}</span></td>
      <td class="code">${escapeHTML(entry.code || '')}</td>
    </tr>
    <tr class="request-detail-row${open ? '' : ' is-collapsed'}">
      <td colspan="7">
        <div class="request-detail-grid">
          <section>
            <div class="detail-title">Request parameters</div>
            <pre>${escapeHTML(formatRequestDetail(entry))}</pre>
          </section>
          <section>
            <div class="detail-title">Response result</div>
            <pre>${escapeHTML(formatResponseDetail(entry))}</pre>
          </section>
        </div>
      </td>
    </tr>`;
  }).join('');
  els.requestRows.querySelectorAll('.request-summary-row').forEach((row) => {
    row.addEventListener('click', () => toggleRequestRow(row));
  });
}

function requestSig(entry) {
  return [entry.time, entry.provider, entry.routeId, entry.status, entry.code, entry.scenario].join('|');
}

function toggleRequestRow(row) {
  const sig = row.dataset.sig;
  const detail = row.nextElementSibling;
  if (!detail) return;
  const willOpen = detail.classList.contains('is-collapsed');
  detail.classList.toggle('is-collapsed', !willOpen);
  row.classList.toggle('is-open', willOpen);
  if (willOpen) state.expandedRequests.add(sig);
  else state.expandedRequests.delete(sig);
}

function formatRequestDetail(entry) {
  const query = entry.query || '';
  const body = entry.requestBody || '';
  const path = `${entry.path || ''}${query ? `?${query}` : ''}`;
  const parts = [`${entry.method || ''} ${path}`.trim()];
  if (entry.code) parts.push(`code: ${entry.code}`);
  parts.push('', 'query:', query ? formatQuery(query) : '(empty)');
  parts.push('', 'body:', body ? formatBody(body) : '(empty)');
  return parts.join('\n');
}

function formatResponseDetail(entry) {
  const parts = [
    `HTTP ${entry.status || ''}`.trim(),
    `content-type: ${entry.contentType || ''}`,
    '',
    'body:',
    entry.responseBody ? formatBody(entry.responseBody) : '(empty)',
  ];
  return parts.join('\n');
}

function formatQuery(query) {
  try {
    const params = new URLSearchParams(query);
    const out = {};
    params.forEach((value, key) => {
      if (Object.prototype.hasOwnProperty.call(out, key)) {
        if (!Array.isArray(out[key])) out[key] = [out[key]];
        out[key].push(value);
      } else {
        out[key] = value;
      }
    });
    return JSON.stringify(out, null, 2);
  } catch (error) {
    return query;
  }
}

async function setProviderScenario(event) {
  const select = event.currentTarget;
  const provider = select.dataset.provider;
  select.disabled = true;
  try {
    await putJSON(`/__admin/providers/${encodeURIComponent(provider)}/scenario`, {
      scenario: select.value,
    });
    showToast(`${provider} scenario updated`);
    await loadState();
  } catch (error) {
    showToast(error.message);
  } finally {
    select.disabled = false;
  }
}

async function setRouteScenarioFromChip(event) {
  const button = event.currentTarget;
  const provider = button.dataset.provider;
  const routeId = button.dataset.routeId;
  const scenario = button.dataset.scenario;
  const defaultScenario = button.dataset.defaultScenario;
  const activeScenario = button.dataset.activeScenario;
  const nextScenario = scenario === defaultScenario || scenario === activeScenario ? '' : scenario;
  button.disabled = true;
  try {
    await putJSON(
      `/__admin/providers/${encodeURIComponent(provider)}/routes/${encodeURIComponent(routeId)}/scenario`,
      { scenario: nextScenario }
    );
    showToast(nextScenario ? `${routeId} pinned to ${nextScenario}` : `${routeId} restored to default`);
    await loadState();
  } catch (error) {
    showToast(error.message);
  } finally {
    button.disabled = false;
  }
}

async function openScenarioModal(event) {
  const button = event.currentTarget;
  const provider = button.dataset.provider;
  const scenario = button.dataset.scenario;
  modalState.provider = provider;
  modalState.scenario = scenario;
  els.modalTitle.textContent = scenario;
  els.modalSubtitle.textContent = `${provider} scenario`;
  els.modalStatus.value = '';
  els.modalContentType.value = '';
  els.modalBody.value = 'Loading...';
  els.modalSave.disabled = true;
  els.modalReset.hidden = true;
  els.modalOverrideBadge.hidden = true;
  els.modal.hidden = false;
  try {
    const detail = await getJSON(`/__admin/providers/${encodeURIComponent(provider)}/scenarios/${encodeURIComponent(scenario)}`);
    modalState.hasOverride = !!detail.hasOverride;
    const source = detail.hasOverride && detail.overridden ? detail.overridden : detail;
    els.modalStatus.value = source.status;
    els.modalContentType.value = source.contentType;
    els.modalBody.value = formatBody(source.body);
    els.modalReset.hidden = !detail.hasOverride;
    els.modalOverrideBadge.hidden = !detail.hasOverride;
    els.modalSave.disabled = false;
  } catch (error) {
    els.modalBody.value = error.message;
    showToast(error.message);
  }
}

function closeScenarioModal() {
  els.modal.hidden = true;
}

async function saveScenarioOverride() {
  if (!modalState.provider || !modalState.scenario) return;
  const payload = {};
  const status = parseInt(els.modalStatus.value, 10);
  if (Number.isFinite(status) && status > 0) payload.status = status;
  const contentType = els.modalContentType.value.trim();
  if (contentType) payload.contentType = contentType;
  payload.body = els.modalBody.value;
  els.modalSave.disabled = true;
  try {
    await putJSON(
      `/__admin/providers/${encodeURIComponent(modalState.provider)}/scenarios/${encodeURIComponent(modalState.scenario)}`,
      payload,
    );
    showToast(`${modalState.scenario} overridden`);
    closeScenarioModal();
    await loadState();
  } catch (error) {
    showToast(error.message);
    els.modalSave.disabled = false;
  }
}

async function clearScenarioOverride() {
  if (!modalState.provider || !modalState.scenario) return;
  els.modalReset.disabled = true;
  try {
    await deleteJSON(`/__admin/providers/${encodeURIComponent(modalState.provider)}/scenarios/${encodeURIComponent(modalState.scenario)}`);
    showToast(`${modalState.scenario} restored to fixture`);
    closeScenarioModal();
    await loadState();
  } catch (error) {
    showToast(error.message);
  } finally {
    els.modalReset.disabled = false;
  }
}

async function resetState() {
  els.resetButton.disabled = true;
  try {
    await fetch('/__admin/reset', { method: 'POST' });
    showToast('State reset');
    await refreshAll();
  } catch (error) {
    showToast(error.message);
  } finally {
    els.resetButton.disabled = false;
  }
}

async function sendProbe(event) {
  event.preventDefault();
  const preset = presets.find((item) => item.id === els.probePreset.value);
  if (!preset) return;
  const couponCode = els.couponCode.value.trim() || '0109760017002';
  const scenarioHeader = els.scenarioHeader.value;
  const path = typeof preset.path === 'function' ? preset.path({ couponCode }) : preset.path;
  els.responseStatus.textContent = 'Sending request...';
  els.responseCode.textContent = '…';
  els.responseCode.className = 'status-pill neutral';
  const headers = {};
  const options = { method: preset.method, headers };
  if (scenarioHeader) headers['X-Chaos-Scenario'] = scenarioHeader;
  let bodyText = '';
  if (preset.body) {
    headers['Content-Type'] = 'application/json';
    bodyText = JSON.stringify(preset.body({ couponCode }));
    options.body = bodyText;
  }
  state.lastRequest = { method: preset.method, path, headers, body: bodyText };
  try {
    const started = performance.now();
    const res = await fetch(path, options);
    const text = await res.text();
    const duration = Math.round(performance.now() - started);
    state.lastResponse = formatBody(text);
    els.responseCode.textContent = `HTTP ${res.status}`;
    els.responseCode.className = `status-pill ${res.status >= 400 ? 'warning' : 'success'}`;
    els.responseStatus.textContent = `${preset.method} ${path} · ${duration} ms`;
    els.responseBody.textContent = state.lastResponse;
    await loadState();
  } catch (error) {
    state.lastResponse = error.message;
    els.responseCode.textContent = 'error';
    els.responseCode.className = 'status-pill warning';
    els.responseStatus.textContent = 'Request failed';
    els.responseBody.textContent = error.message;
  }
}

async function copyResponse() {
  try {
    await navigator.clipboard.writeText(state.lastResponse);
    showToast('Response copied');
  } catch (error) {
    showToast('Copy failed');
  }
}

async function copyRequest() {
  if (!state.lastRequest) {
    showToast('No request yet');
    return;
  }
  try {
    await navigator.clipboard.writeText(toCurl(state.lastRequest));
    showToast('cURL copied');
  } catch (error) {
    showToast('Copy failed');
  }
}

function toCurl(req) {
  const parts = [`curl -sS -X ${req.method} 'http://127.0.0.1:18080${req.path}'`];
  Object.entries(req.headers || {}).forEach(([key, value]) => {
    parts.push(`  -H '${key}: ${value}'`);
  });
  if (req.body) {
    parts.push(`  -d '${req.body.replace(/'/g, "'\\''")}'`);
  }
  return parts.join(' \\\n');
}

async function getJSON(path) {
  const res = await fetch(path);
  if (!res.ok) throw new Error(`${path} returned HTTP ${res.status}`);
  return res.json();
}

async function putJSON(path, body) {
  const res = await fetch(path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `${path} returned HTTP ${res.status}`);
  }
  return res.json();
}

async function deleteJSON(path) {
  const res = await fetch(path, { method: 'DELETE' });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `${path} returned HTTP ${res.status}`);
  }
  return res.json();
}

function formatBody(text) {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch (error) {
    return text;
  }
}

function formatTime(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleTimeString();
}

function showToast(message) {
  els.toast.textContent = message;
  els.toast.classList.add('show');
  window.clearTimeout(showToast.timer);
  showToast.timer = window.setTimeout(() => {
    els.toast.classList.remove('show');
  }, 2200);
}

function escapeHTML(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function escapeAttr(value) {
  return escapeHTML(value);
}

init();
