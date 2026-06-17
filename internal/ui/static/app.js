const state = {
  providers: [],
  recentRequests: [],
  lastResponse: '{}',
};

const presets = [
  {
    id: 'login',
    label: 'Umember login',
    method: 'POST',
    path: '/open/login',
    body: () => ({ email: 'test@example.com', password: 'test' }),
  },
  {
    id: 'detail',
    label: 'Meituan coupon detail',
    method: 'GET',
    path: ({ couponCode }) => `/open/user/meituan/coupon/detail?store_id=8674228&coupon_code=${encodeURIComponent(couponCode)}`,
  },
  {
    id: 'verify',
    label: 'Meituan coupon verify',
    method: 'POST',
    path: '/open/user/meituan/coupon/verify',
    body: ({ couponCode }) => ({ store_id: '8674228', coupon_code: couponCode }),
  },
  {
    id: 'store_list',
    label: 'Meituan store list',
    method: 'GET',
    path: '/open/user/meituan/store/list',
  },
  {
    id: 'account',
    label: 'Umember account',
    method: 'GET',
    path: '/open/user/detail',
  },
  {
    id: 'logs',
    label: 'Request logs API',
    method: 'POST',
    path: '/open/user/request/logs',
    body: () => ({ page: 1, rows: 10 }),
  },
];

const els = {
  healthBadge: document.querySelector('#healthBadge'),
  refreshButton: document.querySelector('#refreshButton'),
  resetButton: document.querySelector('#resetButton'),
  providersList: document.querySelector('#providersList'),
  requestRows: document.querySelector('#requestRows'),
  probeForm: document.querySelector('#probeForm'),
  probePreset: document.querySelector('#probePreset'),
  couponCode: document.querySelector('#couponCode'),
  scenarioHeader: document.querySelector('#scenarioHeader'),
  responseStatus: document.querySelector('#responseStatus'),
  responseBody: document.querySelector('#responseBody'),
  copyResponseButton: document.querySelector('#copyResponseButton'),
  toast: document.querySelector('#toast'),
};

function init() {
  renderPresetOptions();
  bindEvents();
  refreshAll();
}

function bindEvents() {
  els.refreshButton.addEventListener('click', refreshAll);
  els.resetButton.addEventListener('click', resetState);
  els.probeForm.addEventListener('submit', sendProbe);
  els.copyResponseButton.addEventListener('click', copyResponse);
}

function renderPresetOptions() {
  els.probePreset.innerHTML = presets
    .map((preset) => `<option value="${preset.id}">${escapeHTML(preset.label)}</option>`)
    .join('');
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
    renderProviders();
    renderScenarioHeaderOptions();
    renderRequests();
  } catch (error) {
    showToast(error.message);
  }
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
}

function renderProvider(provider) {
  const scenarioOptions = [''].concat(provider.scenarios || []).map((scenario) => {
    const label = scenario || 'default route scenarios';
    const selected = (provider.globalScenario || '') === scenario ? 'selected' : '';
    return `<option value="${escapeAttr(scenario)}" ${selected}>${escapeHTML(label)}</option>`;
  }).join('');
  const routeSummary = `${(provider.routes || []).length} routes, ${(provider.scenarios || []).length} scenarios`;
  return `
    <article class="provider-row">
      <div>
        <div class="provider-title">
          <span>${escapeHTML(provider.name)}</span>
          <span class="status-pill ${provider.enabled ? 'success' : 'disabled'}">${provider.enabled ? 'enabled' : 'disabled'}</span>
        </div>
        <div class="provider-meta">${escapeHTML(routeSummary)}</div>
      </div>
      <label class="scenario-control">
        Global scenario
        <select data-provider="${escapeAttr(provider.name)}" ${provider.enabled ? '' : 'disabled'}>
          ${scenarioOptions}
        </select>
      </label>
    </article>
  `;
}

function renderScenarioHeaderOptions() {
  const scenarios = new Set();
  state.providers.forEach((provider) => {
    if (provider.name === 'umember') {
      (provider.scenarios || []).forEach((scenario) => scenarios.add(scenario));
    }
  });
  const sorted = Array.from(scenarios).sort();
  els.scenarioHeader.innerHTML = '<option value="">none</option>' + sorted
    .map((scenario) => `<option value="${escapeAttr(scenario)}">${escapeHTML(scenario)}</option>`)
    .join('');
}

function renderRequests() {
  if (!state.recentRequests.length) {
    els.requestRows.innerHTML = '<tr><td colspan="6" class="empty-cell">No mock traffic yet.</td></tr>';
    return;
  }
  els.requestRows.innerHTML = state.recentRequests.slice().reverse().map((entry) => `
    <tr>
      <td class="code">${escapeHTML(formatTime(entry.time))}</td>
      <td>${escapeHTML(entry.provider || '')}</td>
      <td><span class="code">${escapeHTML(entry.routeId || '')}</span></td>
      <td><span class="status-pill neutral">${escapeHTML(entry.scenario || '')}</span></td>
      <td><span class="status-pill ${Number(entry.status) >= 400 ? 'warning' : 'success'}">${escapeHTML(String(entry.status || ''))}</span></td>
      <td class="code">${escapeHTML(entry.code || '')}</td>
    </tr>
  `).join('');
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
  const headers = {};
  const options = { method: preset.method, headers };
  if (scenarioHeader) headers['X-Chaos-Scenario'] = scenarioHeader;
  if (preset.body) {
    headers['Content-Type'] = 'application/json';
    options.body = JSON.stringify(preset.body({ couponCode }));
  }
  els.responseStatus.textContent = 'Sending request...';
  try {
    const started = performance.now();
    const res = await fetch(path, options);
    const text = await res.text();
    const duration = Math.round(performance.now() - started);
    state.lastResponse = formatBody(text);
    els.responseStatus.textContent = `${preset.method} ${path} -> HTTP ${res.status}, ${duration} ms`;
    els.responseBody.textContent = state.lastResponse;
    await loadState();
  } catch (error) {
    state.lastResponse = error.message;
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
