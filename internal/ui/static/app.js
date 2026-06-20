const state = {
  providers: [],
  recentRequests: [],
  lastResponse: '{}',
  lastRequest: null,
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
  {
    id: 'dy_token',
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
    label: 'Douyin certificate prepare',
    method: 'GET',
    path: ({ couponCode }) => `/goodlife/v1/fulfilment/certificate/prepare/?poi_id=7630290236999731263&code=${encodeURIComponent(couponCode)}`,
  },
  {
    id: 'dy_verify',
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
  renderPresetOptions();
  bindEvents();
  refreshAll();
}

function bindEvents() {
  els.refreshButton.addEventListener('click', refreshAll);
  els.resetButton.addEventListener('click', resetState);
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
  els.providersList.querySelectorAll('button.scenario-chip').forEach((button) => {
    button.addEventListener('click', openScenarioModal);
  });
}

function renderProvider(provider) {
  const scenarioOptions = [''].concat(provider.scenarios || []).map((scenario) => {
    const label = scenario || 'default route scenarios';
    const selected = (provider.globalScenario || '') === scenario ? 'selected' : '';
    return `<option value="${escapeAttr(scenario)}" ${selected}>${escapeHTML(label)}</option>`;
  }).join('');
  const routeSummary = `${(provider.routes || []).length} routes, ${(provider.scenarios || []).length} scenarios`;
  const scenarioChips = (provider.scenarios || []).map((scenario) =>
    `<button class="scenario-chip" type="button" data-provider="${escapeAttr(provider.name)}" data-scenario="${escapeAttr(scenario)}">${escapeHTML(scenario)}</button>`
  ).join('');
  return `
    <article class="provider-row">
      <div class="provider-info">
        <div class="provider-title">
          <span>${escapeHTML(provider.name)}</span>
          <span class="status-pill ${provider.enabled ? 'success' : 'disabled'}">${provider.enabled ? 'enabled' : 'disabled'}</span>
        </div>
        <div class="provider-meta">${escapeHTML(routeSummary)}</div>
        <div class="scenario-chips">${scenarioChips}</div>
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
    (provider.scenarios || []).forEach((scenario) => scenarios.add(scenario));
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
