// ============ API Client ============
const API = {
  async get(path) {
    const res = await fetch(path);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },
  async post(path, body) {
    const res = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) throw new Error(await res.text());
    const text = await res.text();
    return text ? JSON.parse(text) : {};
  },
  async del(path) {
    const res = await fetch(path, { method: 'DELETE' });
    if (!res.ok) throw new Error(await res.text());
  },
};

// ============ State ============
let devices = [];
let actions = [];
let automations = [];

// ============ Helpers ============
function showBanner(id, msg, type) {
  const el = document.getElementById(id);
  el.textContent = msg;
  el.className = 'banner ' + type;
  setTimeout(() => { el.className = 'banner'; }, 4000);
}

function parseActions(actionsStr) {
  if (!actionsStr || actionsStr === '') return [];
  try { return JSON.parse(actionsStr); } catch { return []; }
}

function actionNamesByIds(ids) {
  return ids.map(id => {
    const a = actions.find(a => a.id === id);
    return a ? a.name : `#${id}`;
  });
}

function getActionsForDevice(device) {
  const ids = parseActions(device.actions);
  return actions.filter(a => ids.includes(a.id));
}

function populateDeviceSelect(selectEl, selectedName) {
  selectEl.innerHTML = '<option value="">Select device...</option>';
  devices.forEach(d => {
    const opt = document.createElement('option');
    opt.value = d.name;
    opt.textContent = d.name;
    if (d.name === selectedName) opt.selected = true;
    selectEl.appendChild(opt);
  });
}

function populateActionSelect(selectEl, device, selectedName) {
  selectEl.innerHTML = '<option value="">Select action...</option>';
  if (!device) return;
  const dev = devices.find(d => d.name === device || d.id === device);
  if (!dev) return;
  const devActions = getActionsForDevice(dev);
  devActions.forEach(a => {
    const opt = document.createElement('option');
    opt.value = a.name;
    opt.textContent = a.name;
    if (a.name === selectedName) opt.selected = true;
    selectEl.appendChild(opt);
  });
}

// ============ Tab Navigation ============
document.querySelectorAll('nav button').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('nav button').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById('tab-' + btn.dataset.tab).classList.add('active');
    loadTab(btn.dataset.tab);
  });
});

async function loadTab(tab) {
  try {
    if (tab === 'devices' || tab === 'automations' || tab === 'execute') {
      actions = await API.get('/actions');
    }
    if (tab === 'devices') {
      devices = await API.get('/devices');
      renderDevices();
      renderDeviceActionCheckboxes();
    } else if (tab === 'actions') {
      actions = await API.get('/actions');
      renderActions();
    } else if (tab === 'automations') {
      devices = await API.get('/devices');
      automations = await API.get('/automations');
      renderAutomations();
    } else if (tab === 'execute') {
      devices = await API.get('/devices');
      populateExecDropdowns();
    }
  } catch (e) {
    showBanner(tab + '-banner', e.message, 'error');
  }
}

// ============ ACTIONS CRUD ============
function renderActions() {
  const tbody = document.getElementById('actions-table');
  tbody.innerHTML = actions.map(a => `
    <tr>
      <td>${esc(a.name)}</td>
      <td>${esc(a.path)}</td>
      <td>${esc(a.params || '')}</td>
      <td>
        <button class="btn btn-secondary btn-sm" onclick="editAction(${a.id})">Edit</button>
        <button class="btn btn-danger btn-sm" onclick="deleteAction(${a.id})">Delete</button>
      </td>
    </tr>
  `).join('');
}

document.getElementById('actions-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const id = document.getElementById('action-id').value;
  const body = {
    name: document.getElementById('action-name').value,
    path: document.getElementById('action-path').value,
    params: document.getElementById('action-params').value || '',
  };
  try {
    if (id) {
      await API.post('/actions/' + id, body);
    } else {
      await API.post('/actions', body);
    }
    resetActionForm();
    await loadTab('actions');
    showBanner('actions-banner', 'Action saved', 'success');
  } catch (e) {
    showBanner('actions-banner', e.message, 'error');
  }
});

function editAction(id) {
  const a = actions.find(a => a.id === id);
  if (!a) return;
  document.getElementById('action-id').value = a.id;
  document.getElementById('action-name').value = a.name;
  document.getElementById('action-path').value = a.path;
  document.getElementById('action-params').value = a.params || '';
  document.getElementById('actions-form-title').textContent = 'Edit Action';
}

async function deleteAction(id) {
  if (!confirm('Delete this action?')) return;
  try {
    await API.del('/actions/' + id);
    await loadTab('actions');
    showBanner('actions-banner', 'Action deleted', 'success');
  } catch (e) {
    showBanner('actions-banner', e.message, 'error');
  }
}

function resetActionForm() {
  document.getElementById('actions-form').reset();
  document.getElementById('action-id').value = '';
  document.getElementById('actions-form-title').textContent = 'Add Action';
}

// ============ DEVICES CRUD ============
function renderDevices() {
  const tbody = document.getElementById('devices-table');
  tbody.innerHTML = devices.map(d => {
    const ids = parseActions(d.actions);
    const names = actionNamesByIds(ids);
    return `
      <tr>
        <td>${esc(d.name)}</td>
        <td>${esc(d.type)}</td>
        <td>${esc(d.chip)}</td>
        <td>${esc(d.board)}</td>
        <td>${esc(d.ip)}</td>
        <td>${names.map(n => esc(n)).join(', ')}</td>
        <td>
          <button class="btn btn-secondary btn-sm" onclick="editDevice(${d.id})">Edit</button>
          <button class="btn btn-danger btn-sm" onclick="deleteDevice(${d.id})">Delete</button>
        </td>
      </tr>
    `;
  }).join('');
}

function renderDeviceActionCheckboxes() {
  const container = document.getElementById('device-actions-checkboxes');
  container.innerHTML = actions.map(a => `
    <label><input type="checkbox" value="${a.id}" class="device-action-cb"> ${esc(a.name)}</label>
  `).join('');
}

document.getElementById('devices-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const id = document.getElementById('device-id').value;
  const checkedIds = [...document.querySelectorAll('.device-action-cb:checked')].map(cb => Number(cb.value));
  const body = {
    name: document.getElementById('device-name').value,
    type: document.getElementById('device-type').value,
    chip: document.getElementById('device-chip').value,
    board: document.getElementById('device-board').value,
    ip: document.getElementById('device-ip').value,
    actions: JSON.stringify(checkedIds),
  };
  try {
    if (id) {
      await API.post('/devices/' + id, body);
    } else {
      await API.post('/devices', body);
    }
    resetDeviceForm();
    await loadTab('devices');
    showBanner('devices-banner', 'Device saved', 'success');
  } catch (e) {
    showBanner('devices-banner', e.message, 'error');
  }
});

function editDevice(id) {
  const d = devices.find(d => d.id === id);
  if (!d) return;
  document.getElementById('device-id').value = d.id;
  document.getElementById('device-name').value = d.name;
  document.getElementById('device-type').value = d.type;
  document.getElementById('device-chip').value = d.chip;
  document.getElementById('device-board').value = d.board;
  document.getElementById('device-ip').value = d.ip;
  document.getElementById('devices-form-title').textContent = 'Edit Device';
  const ids = parseActions(d.actions);
  document.querySelectorAll('.device-action-cb').forEach(cb => {
    cb.checked = ids.includes(Number(cb.value));
  });
}

async function deleteDevice(id) {
  if (!confirm('Delete this device?')) return;
  try {
    await API.del('/devices/' + id);
    await loadTab('devices');
    showBanner('devices-banner', 'Device deleted', 'success');
  } catch (e) {
    showBanner('devices-banner', e.message, 'error');
  }
}

function resetDeviceForm() {
  document.getElementById('devices-form').reset();
  document.getElementById('device-id').value = '';
  document.getElementById('devices-form-title').textContent = 'Add Device';
  document.querySelectorAll('.device-action-cb').forEach(cb => cb.checked = false);
}

// ============ EXECUTE ============
function populateExecDropdowns() {
  const devSel = document.getElementById('exec-device');
  devSel.innerHTML = '<option value="">Select device...</option>';
  devices.forEach(d => {
    const opt = document.createElement('option');
    opt.value = d.id;
    opt.textContent = d.name;
    devSel.appendChild(opt);
  });
  document.getElementById('exec-action').innerHTML = '<option value="">Select action...</option>';
}

document.getElementById('exec-device').addEventListener('change', (e) => {
  const devId = Number(e.target.value);
  const dev = devices.find(d => d.id === devId);
  const actSel = document.getElementById('exec-action');
  actSel.innerHTML = '<option value="">Select action...</option>';
  if (!dev) return;
  getActionsForDevice(dev).forEach(a => {
    const opt = document.createElement('option');
    opt.value = a.id;
    opt.textContent = a.name;
    actSel.appendChild(opt);
  });
});

async function handleExecute() {
  const deviceId = Number(document.getElementById('exec-device').value);
  const actionId = Number(document.getElementById('exec-action').value);
  if (!deviceId || !actionId) {
    showBanner('execute-banner', 'Select both device and action', 'error');
    return;
  }
  try {
    const res = await API.post('/execute', { deviceId, actionId });
    document.getElementById('execute-response').textContent = JSON.stringify(res, null, 2);
  } catch (e) {
    document.getElementById('execute-response').textContent = 'Error: ' + e.message;
  }
}

// ============ AUTOMATIONS CRUD ============
function renderAutomations() {
  const tbody = document.getElementById('automations-table');
  tbody.innerHTML = automations.map(a => {
    const badge = a.enabled ? '<span class="badge badge-on">ON</span>' : '<span class="badge badge-off">OFF</span>';
    let interval = '';
    try {
      const lines = a.definition.split('\n');
      const line = lines.find(l => l.startsWith('interval:'));
      if (line) interval = line.split(':')[1].trim().replace(/"/g, '');
    } catch {}
    return `
      <tr>
        <td>${esc(a.name)}</td>
        <td>${badge}</td>
        <td>${esc(interval)}</td>
        <td>${formatTime(a.last_check)}</td>
        <td>${formatTime(a.last_action_run)}</td>
        <td>
          <button class="btn btn-secondary btn-sm" onclick="editAutomation(${a.id})">Edit</button>
          <button class="btn btn-danger btn-sm" onclick="deleteAutomation(${a.id})">Delete</button>
        </td>
      </tr>
    `;
  }).join('');
}

function formatTime(t) {
  if (!t) return '-';
  try {
    const d = new Date(t);
    return d.toLocaleTimeString();
  } catch { return t; }
}

async function deleteAutomation(id) {
  if (!confirm('Delete this automation?')) return;
  try {
    await API.del('/automations/' + id);
    await loadTab('automations');
    showBanner('automations-banner', 'Automation deleted', 'success');
  } catch (e) {
    showBanner('automations-banner', e.message, 'error');
  }
}

function editAutomation(id) {
  const a = automations.find(a => a.id === id);
  if (!a) return;
  document.getElementById('automation-id').value = a.id;
  document.getElementById('auto-name').value = a.name;
  document.getElementById('auto-enabled').checked = a.enabled;
  document.getElementById('automations-form-title').textContent = 'Edit Automation';
  populateBuilderFromYAML(a.definition);
}

function resetAutomationForm() {
  document.getElementById('automations-form').reset();
  document.getElementById('automation-id').value = '';
  document.getElementById('auto-enabled').checked = true;
  document.getElementById('automations-form-title').textContent = 'Add Automation';
  document.getElementById('triggers-container').innerHTML = '';
  document.getElementById('auto-actions-container').innerHTML = '';
  updateYAMLPreview();
}

function validateAutomationForm() {
  // Clear previous invalid markers
  document.querySelectorAll('#automations-form .invalid').forEach(el => el.classList.remove('invalid'));

  // Validate interval
  const intervalInput = document.getElementById('auto-interval');
  const interval = intervalInput.value.trim();
  if (!interval) {
    intervalInput.classList.add('invalid');
    return 'Interval is required';
  }
  const match = interval.match(/^(\d+)(ms|[smh])$/);
  if (!match) {
    intervalInput.classList.add('invalid');
    return "Interval must be a valid duration (e.g. '5m', '1h', '30s')";
  }
  const num = parseInt(match[1], 10);
  const unit = match[2];
  if ((unit === 's' && num < 1) || (unit === 'ms' && num < 1000)) {
    intervalInput.classList.add('invalid');
    return 'Interval must be at least 1s';
  }

  // Validate triggers (optional, but if present must be complete)
  const triggerSections = document.querySelectorAll('#triggers-container .dynamic-section');
  for (const sec of triggerSections) {
    const deviceSel = sec.querySelector('.trigger-device');
    if (!deviceSel.value) {
      deviceSel.classList.add('invalid');
      return 'Each trigger must have a device selected';
    }
    const actionSel = sec.querySelector('.trigger-action');
    if (!actionSel.value) {
      actionSel.classList.add('invalid');
      return 'Each trigger must have an action selected';
    }
    const condRows = sec.querySelectorAll('.condition-row');
    if (condRows.length === 0) {
      return 'Each trigger must have at least one condition';
    }
    for (const row of condRows) {
      const fieldInput = row.querySelector('.cond-field');
      if (!fieldInput.value.trim()) {
        fieldInput.classList.add('invalid');
        return 'Condition field cannot be empty';
      }
      const threshInput = row.querySelector('.cond-threshold');
      if (threshInput.value === '' || isNaN(parseFloat(threshInput.value))) {
        threshInput.classList.add('invalid');
        return 'Condition threshold is required';
      }
    }
  }

  // Validate actions (required)
  const actionSections = document.querySelectorAll('#auto-actions-container .dynamic-section');
  if (actionSections.length === 0) {
    return 'At least one action is required';
  }
  for (const sec of actionSections) {
    const deviceSel = sec.querySelector('.auto-action-device');
    if (!deviceSel.value) {
      deviceSel.classList.add('invalid');
      return 'Each action must have a device selected';
    }
    const actionSel = sec.querySelector('.auto-action-action');
    if (!actionSel.value) {
      actionSel.classList.add('invalid');
      return 'Each action must have an action selected';
    }
  }

  return null;
}

document.getElementById('automations-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const validationError = validateAutomationForm();
  if (validationError) {
    showBanner('automations-banner', validationError, 'error');
    return;
  }
  const id = document.getElementById('automation-id').value;
  const def = buildDefinitionFromForm();
  const yaml = toYAML(def);
  const body = {
    name: document.getElementById('auto-name').value,
    enabled: document.getElementById('auto-enabled').checked,
    definition: yaml,
  };
  try {
    if (id) {
      await API.post('/automations/' + id, body);
    } else {
      await API.post('/automations', body);
    }
    resetAutomationForm();
    await loadTab('automations');
    showBanner('automations-banner', 'Automation saved', 'success');
  } catch (e) {
    showBanner('automations-banner', e.message, 'error');
  }
});

// ============ AUTOMATION BUILDER ============
let triggerCount = 0;
let autoActionCount = 0;

function addTrigger(data) {
  const idx = triggerCount++;
  const div = document.createElement('div');
  div.className = 'dynamic-section';
  div.dataset.triggerIdx = idx;
  div.innerHTML = `
    <div class="section-header">
      <span>Trigger #${idx + 1}</span>
      <button type="button" class="btn btn-danger btn-sm" onclick="removeTrigger(this)">Remove</button>
    </div>
    <div class="form-row">
      <div class="form-group">
        <label>Device</label>
        <select class="trigger-device" onchange="onTriggerDeviceChange(this)">
          <option value="">Select...</option>
        </select>
      </div>
      <div class="form-group">
        <label>Action</label>
        <select class="trigger-action"><option value="">Select...</option></select>
      </div>
    </div>
    <div class="conditions-list"></div>
    <button type="button" class="btn btn-secondary btn-sm" onclick="addCondition(this)" style="margin-top:0.3rem">+ Condition</button>
  `;
  document.getElementById('triggers-container').appendChild(div);
  populateDeviceSelect(div.querySelector('.trigger-device'), data?.device || '');
  if (data?.device) {
    populateActionSelect(div.querySelector('.trigger-action'), data.device, data?.action || '');
  }
  if (data?.conditions) {
    data.conditions.forEach(c => addCondition(div.querySelector('.btn-secondary'), c));
  } else {
    addCondition(div.querySelector('.btn-secondary'));
  }
  updateYAMLPreview();
}

function removeTrigger(btn) {
  btn.closest('.dynamic-section').remove();
  updateYAMLPreview();
}

function onTriggerDeviceChange(sel) {
  const section = sel.closest('.dynamic-section');
  const actionSel = section.querySelector('.trigger-action');
  populateActionSelect(actionSel, sel.value, '');
  updateYAMLPreview();
}

function addCondition(btn, data) {
  const section = btn.closest('.dynamic-section');
  const list = section.querySelector('.conditions-list');
  const div = document.createElement('div');
  div.className = 'condition-row';
  div.innerHTML = `
    <div class="form-group">
      <label>Field</label>
      <input type="text" class="cond-field" value="${esc(data?.field || '')}" placeholder="temperature">
    </div>
    <div class="form-group" style="min-width:60px;flex:0 0 80px">
      <label>Op</label>
      <select class="cond-op">
        <option value=">"${data?.operator === '>' ? ' selected' : ''}>&gt;</option>
        <option value="<"${data?.operator === '<' ? ' selected' : ''}>&lt;</option>
        <option value=">="${data?.operator === '>=' ? ' selected' : ''}>&gt;=</option>
        <option value="<="${data?.operator === '<=' ? ' selected' : ''}>&lt;=</option>
        <option value="=="${data?.operator === '==' ? ' selected' : ''}>==</option>
        <option value="!="${data?.operator === '!=' ? ' selected' : ''}>!=</option>
      </select>
    </div>
    <div class="form-group" style="min-width:60px;flex:0 0 100px">
      <label>Threshold</label>
      <input type="number" step="any" class="cond-threshold" value="${data?.threshold ?? ''}">
    </div>
    <div class="form-group" style="flex:0 0 auto;min-width:auto;justify-content:end">
      <label>&nbsp;</label>
      <button type="button" class="btn btn-danger btn-sm" onclick="removeCondition(this)">x</button>
    </div>
  `;
  list.appendChild(div);
  updateYAMLPreview();
}

function removeCondition(btn) {
  btn.closest('.condition-row').remove();
  updateYAMLPreview();
}

function addAutoAction(data) {
  const idx = autoActionCount++;
  const div = document.createElement('div');
  div.className = 'dynamic-section';
  div.dataset.actionIdx = idx;
  div.innerHTML = `
    <div class="section-header">
      <span>Action #${idx + 1}</span>
      <button type="button" class="btn btn-danger btn-sm" onclick="removeAutoAction(this)">Remove</button>
    </div>
    <div class="form-row">
      <div class="form-group">
        <label>Device</label>
        <select class="auto-action-device" onchange="onAutoActionDeviceChange(this)">
          <option value="">Select...</option>
        </select>
      </div>
      <div class="form-group">
        <label>Action</label>
        <select class="auto-action-action"><option value="">Select...</option></select>
      </div>
    </div>
  `;
  document.getElementById('auto-actions-container').appendChild(div);
  populateDeviceSelect(div.querySelector('.auto-action-device'), data?.device || '');
  if (data?.device) {
    populateActionSelect(div.querySelector('.auto-action-action'), data.device, data?.action || '');
  }
  updateYAMLPreview();
}

function removeAutoAction(btn) {
  btn.closest('.dynamic-section').remove();
  updateYAMLPreview();
}

function onAutoActionDeviceChange(sel) {
  const section = sel.closest('.dynamic-section');
  const actionSel = section.querySelector('.auto-action-action');
  populateActionSelect(actionSel, sel.value, '');
  updateYAMLPreview();
}

// ============ YAML SERIALIZER ============
function buildDefinitionFromForm() {
  const def = {
    interval: document.getElementById('auto-interval').value || '',
    condition_logic: document.getElementById('auto-logic').value || '',
    triggers: [],
    actions: [],
  };

  document.querySelectorAll('#triggers-container .dynamic-section').forEach(sec => {
    const trigger = {
      device: sec.querySelector('.trigger-device').value,
      action: sec.querySelector('.trigger-action').value,
      conditions: [],
    };
    sec.querySelectorAll('.condition-row').forEach(row => {
      trigger.conditions.push({
        field: row.querySelector('.cond-field').value,
        operator: row.querySelector('.cond-op').value,
        threshold: parseFloat(row.querySelector('.cond-threshold').value) || 0,
      });
    });
    def.triggers.push(trigger);
  });

  document.querySelectorAll('#auto-actions-container .dynamic-section').forEach(sec => {
    def.actions.push({
      device: sec.querySelector('.auto-action-device').value,
      action: sec.querySelector('.auto-action-action').value,
    });
  });

  return def;
}

function toYAML(def) {
  const lines = [];
  lines.push(`interval: "${def.interval}"`);
  if (def.condition_logic) {
    lines.push(`condition_logic: "${def.condition_logic}"`);
  }
  if (def.triggers.length > 0) {
    lines.push('triggers:');
    for (const t of def.triggers) {
      lines.push(`  - device: "${t.device}"`);
      lines.push(`    action: "${t.action}"`);
      if (t.conditions.length > 0) {
        lines.push('    conditions:');
        for (const c of t.conditions) {
          lines.push(`      - field: "${c.field}"`);
          lines.push(`        operator: "${c.operator}"`);
          lines.push(`        threshold: ${c.threshold}`);
        }
      }
    }
  }
  if (def.actions.length > 0) {
    lines.push('actions:');
    for (const a of def.actions) {
      lines.push(`  - device: "${a.device}"`);
      lines.push(`    action: "${a.action}"`);
    }
  }
  return lines.join('\n');
}

function updateYAMLPreview() {
  const def = buildDefinitionFromForm();
  document.getElementById('yaml-preview').textContent = toYAML(def);
}

// Event delegation for live YAML preview
document.getElementById('automations-form').addEventListener('input', updateYAMLPreview);
document.getElementById('automations-form').addEventListener('change', updateYAMLPreview);

// ============ YAML PARSER (for edit mode) ============
function populateBuilderFromYAML(yamlStr) {
  document.getElementById('triggers-container').innerHTML = '';
  document.getElementById('auto-actions-container').innerHTML = '';
  triggerCount = 0;
  autoActionCount = 0;

  const def = parseSimpleYAML(yamlStr);
  document.getElementById('auto-interval').value = def.interval || '';
  document.getElementById('auto-logic').value = def.condition_logic || '';

  (def.triggers || []).forEach(t => addTrigger(t));
  (def.actions || []).forEach(a => addAutoAction(a));
  updateYAMLPreview();
}

function parseSimpleYAML(str) {
  const def = { interval: '', condition_logic: '', triggers: [], actions: [] };
  const lines = str.split('\n');
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];
    const trimmed = line.trim();

    if (trimmed.startsWith('interval:')) {
      def.interval = extractValue(trimmed);
      i++;
    } else if (trimmed.startsWith('condition_logic:')) {
      def.condition_logic = extractValue(trimmed);
      i++;
    } else if (trimmed === 'triggers:') {
      i++;
      while (i < lines.length && lines[i].match(/^  /)) {
        if (lines[i].trim().startsWith('- device:')) {
          const trigger = { device: '', action: '', conditions: [] };
          trigger.device = extractValue(lines[i].trim().replace('- ', ''));
          i++;
          while (i < lines.length && lines[i].match(/^    /) && !lines[i].trim().startsWith('- device:')) {
            const tl = lines[i].trim();
            if (tl.startsWith('action:')) {
              trigger.action = extractValue(tl);
              i++;
            } else if (tl === 'conditions:') {
              i++;
              while (i < lines.length && lines[i].match(/^      /)) {
                if (lines[i].trim().startsWith('- field:')) {
                  const cond = { field: '', operator: '', threshold: 0 };
                  cond.field = extractValue(lines[i].trim().replace('- ', ''));
                  i++;
                  while (i < lines.length && lines[i].match(/^        /) && !lines[i].trim().startsWith('- ')) {
                    const cl = lines[i].trim();
                    if (cl.startsWith('operator:')) cond.operator = extractValue(cl);
                    else if (cl.startsWith('threshold:')) cond.threshold = parseFloat(extractValue(cl)) || 0;
                    i++;
                  }
                  trigger.conditions.push(cond);
                } else {
                  i++;
                }
              }
            } else {
              i++;
            }
          }
          def.triggers.push(trigger);
        } else {
          i++;
        }
      }
    } else if (trimmed === 'actions:') {
      i++;
      while (i < lines.length && lines[i].match(/^  /)) {
        if (lines[i].trim().startsWith('- device:')) {
          const action = { device: '', action: '' };
          action.device = extractValue(lines[i].trim().replace('- ', ''));
          i++;
          while (i < lines.length && lines[i].match(/^    /) && !lines[i].trim().startsWith('- ')) {
            const al = lines[i].trim();
            if (al.startsWith('action:')) action.action = extractValue(al);
            i++;
          }
          def.actions.push(action);
        } else {
          i++;
        }
      }
    } else {
      i++;
    }
  }
  return def;
}

function extractValue(s) {
  const parts = s.split(':');
  parts.shift();
  return parts.join(':').trim().replace(/^["']|["']$/g, '');
}

// ============ Escaping ============
function esc(s) {
  if (s == null) return '';
  const d = document.createElement('div');
  d.textContent = String(s);
  return d.innerHTML;
}

// ============ Theme Toggle ============
function applyTheme(dark) {
  document.documentElement.classList.toggle('dark', dark);
  document.getElementById('theme-icon').innerHTML = dark ? '&#9790;' : '&#9788;';
  document.getElementById('theme-switch').checked = dark;
  localStorage.setItem('theme', dark ? 'dark' : 'light');
}

document.getElementById('theme-switch').addEventListener('change', (e) => {
  applyTheme(e.target.checked);
});

// ============ Init ============
document.addEventListener('DOMContentLoaded', () => {
  const saved = localStorage.getItem('theme');
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  applyTheme(saved ? saved === 'dark' : prefersDark);
  loadTab('devices');
});
