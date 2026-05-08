const state = {
  tempChart: null,
  humidityChart: null,
};

const dom = {
  statusText: document.getElementById('status-text'),
  geoHint: document.getElementById('geo-hint'),
  cityName: document.getElementById('city-name'),
  cityMeta: document.getElementById('city-meta'),
  weatherText: document.getElementById('weather-text'),
  weatherTemp: document.getElementById('weather-temp'),
  weatherFeelsLike: document.getElementById('weather-feels-like'),
  weatherHumidity: document.getElementById('weather-humidity'),
  weatherWindDir: document.getElementById('weather-wind-dir'),
  weatherWindSpeed: document.getElementById('weather-wind-speed'),
  weatherWindScale: document.getElementById('weather-wind-scale'),
  airCard: document.getElementById('air-card'),
  airCategory: document.getElementById('air-category'),
  airAQI: document.getElementById('air-aqi'),
  airPollutant: document.getElementById('air-pollutant'),
  airAdvice: document.getElementById('air-advice'),
  airStation: document.getElementById('air-station'),
  updatedAt: document.getElementById('updated-at'),
  locateBtn: document.getElementById('locate-btn'),
  manualForm: document.getElementById('manual-form'),
  actionSelect: document.getElementById('action-select'),
  cityInput: document.getElementById('city-input'),
  citySubmit: document.getElementById('city-submit'),
  tempChartEl: document.getElementById('temp-chart'),
  humidityChartEl: document.getElementById('humidity-chart'),
};

const API_ERROR_MESSAGE = '请求失败';

function getClientID() {
  let clientID = window.localStorage.getItem('smog_client_id');
  if (!clientID) {
    clientID = window.crypto?.randomUUID?.() || `client-${Date.now()}`;
    window.localStorage.setItem('smog_client_id', clientID);
  }
  return clientID;
}

function isLocalDebugHost() {
  const host = window.location.hostname;
  return host === 'localhost' || host === '127.0.0.1' || host === '::1';
}

function isPrivateNetworkHost() {
  const host = window.location.hostname.replace(/^\[|\]$/g, '');
  const parts = host.split('.').map((part) => Number(part));

  if (parts.length === 4 && parts.every((part) => Number.isInteger(part) && part >= 0 && part <= 255)) {
    return parts[0] === 10
      || (parts[0] === 172 && parts[1] >= 16 && parts[1] <= 31)
      || (parts[0] === 192 && parts[1] === 168)
      || (parts[0] === 169 && parts[1] === 254)
      || (parts[0] === 100 && parts[1] >= 64 && parts[1] <= 127);
  }

  return host.endsWith('.local');
}

function supportsSecureGeolocation() {
  return window.isSecureContext || isLocalDebugHost();
}

function isTouchDevice() {
  return window.matchMedia('(hover: none), (pointer: coarse)').matches;
}

function showGeoHint(message) {
  dom.geoHint.classList.remove('is-error');
  if (!message) {
    dom.geoHint.textContent = '';
    dom.geoHint.classList.add('hidden');
    return;
  }
  dom.geoHint.textContent = message;
  dom.geoHint.classList.remove('hidden');
}

function setStatus(message, tone = 'normal') {
  dom.statusText.textContent = message;
  dom.statusText.classList.toggle('is-error', tone === 'error');
}

function showHint(message, tone = 'normal') {
  showGeoHint(message);
  if (message) {
    dom.geoHint.classList.toggle('is-error', tone === 'error');
  }
}

function showAPIError() {
  setStatus(API_ERROR_MESSAGE, 'error');
}

async function requestJSON(url, options = {}) {
  let response;
  try {
    response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...(options.headers || {}),
      },
      ...options,
    });
  } catch (error) {
    throw new Error(API_ERROR_MESSAGE);
  }

  if (!response.ok) {
    throw new Error(API_ERROR_MESSAGE);
  }
  return response.json().catch(() => ({}));
}

function setBusy(busy, message) {
  dom.locateBtn.disabled = busy;
  dom.actionSelect.disabled = busy;
  dom.citySubmit.disabled = busy;
  dom.cityInput.disabled = busy || dom.actionSelect.value !== 'manual';
  if (message) {
    setStatus(message);
  }
}

function syncManualFormState() {
  const mode = dom.actionSelect.value;
  const needsCityInput = mode === 'manual';
  dom.cityInput.disabled = !needsCityInput;
  dom.cityInput.hidden = !needsCityInput;
  dom.cityInput.required = needsCityInput;
  dom.citySubmit.textContent = mode === 'manual' ? '查询城市' : '执行';
}

function renderDashboard(data) {
  dom.cityName.textContent = data.city.name || '--';
  dom.cityMeta.textContent = [data.city.adm1, data.city.adm2].filter(Boolean).join(' / ') || '已定位';
  dom.weatherText.textContent = data.weather_now.text || '--';
  dom.weatherTemp.textContent = `${data.weather_now.temp || '--'}°`;
  dom.weatherFeelsLike.textContent = `${data.weather_now.feels_like || '--'}°`;
  dom.weatherHumidity.textContent = `${data.weather_now.humidity || '--'}%`;
  dom.weatherWindDir.textContent = data.weather_now.wind_dir || '--';
  dom.weatherWindSpeed.textContent = data.weather_now.wind_speed ? `${data.weather_now.wind_speed} km/h` : '--';
  dom.weatherWindScale.textContent = data.weather_now.wind_scale || '--';
  dom.airCategory.textContent = data.air_now.category || '--';
  dom.airAQI.textContent = data.air_now.aqi || '--';
  dom.airPollutant.textContent = data.air_now.primary_pollutant || '--';
  dom.airAdvice.textContent = data.air_now.health_advice || '暂无健康建议';
  dom.airStation.textContent = `监测站：${data.air_now.monitoring_station || '未返回'}`;
  dom.updatedAt.textContent = `更新时间：${formatTime(data.updated_at)}`;
  setStatus(data.from_cache ? '已显示缓存数据' : '数据已更新');
  showHint('');
  updateAirLevel(data.air_now.aqi, data.air_now.category);
  renderCharts(data.hourly || []);
}

function updateAirLevel(aqiText, category) {
  const value = Number(aqiText);
  dom.airCard.classList.remove('air-level-default', 'air-level-good', 'air-level-fair', 'air-level-bad');
  if ((category || '').includes('优') || (category || '').toLowerCase().includes('good') || (!Number.isNaN(value) && value <= 50)) {
    dom.airCard.classList.add('air-level-good');
    return;
  }
  if ((category || '').includes('良') || (category || '').toLowerCase().includes('moderate') || (!Number.isNaN(value) && value <= 100)) {
    dom.airCard.classList.add('air-level-fair');
    return;
  }
  if (!Number.isNaN(value)) {
    dom.airCard.classList.add('air-level-bad');
    return;
  }
  dom.airCard.classList.add('air-level-default');
}

function renderCharts(hourly) {
  if (!state.tempChart) {
    state.tempChart = echarts.init(dom.tempChartEl);
  }
  if (!state.humidityChart) {
    state.humidityChart = echarts.init(dom.humidityChartEl);
  }

  const times = hourly.map((item) => item.time);
  const temps = hourly.map((item) => Number(item.temp));
  const humidities = hourly.map((item) => Number(item.humidity));

  state.tempChart.setOption(buildLineOption(times, temps, '#79b8ff', '温度', '°C'));
  state.humidityChart.setOption(buildLineOption(times, humidities, '#8ef0c8', '湿度', '%'));
}

function buildLineOption(times, values, color, name, unit) {
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(255,255,255,0.9)',
      borderColor: 'rgba(195,211,231,0.85)',
      borderWidth: 1,
      textStyle: { color: '#1d3c5d' },
    },
    grid: { left: 16, right: 12, top: 16, bottom: 30, containLabel: true },
    xAxis: {
      type: 'category',
      data: times,
      axisLabel: { color: '#7d92ab', interval: 3 },
      axisLine: { lineStyle: { color: 'rgba(145,171,201,0.28)' } },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#7d92ab', formatter: `{value}${unit}` },
      splitLine: { lineStyle: { color: 'rgba(160,185,214,0.18)' } },
    },
    series: [
      {
        name,
        type: 'line',
        smooth: true,
        symbol: 'none',
        data: values,
        lineStyle: { color, width: 3 },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: `${color}aa` },
              { offset: 1, color: `${color}00` },
            ],
          },
        },
      },
    ],
  };
}

async function loadDashboard(autoLocate = true) {
  try {
    setBusy(true, '正在读取服务器数据...');
    const data = await requestJSON(`/api/dashboard?client_id=${encodeURIComponent(getClientID())}`);
    renderDashboard(data);
  } catch (error) {
    if (autoLocate) {
      setStatus('正在尝试网络定位...');
      showHint('先使用网络定位估算城市；如城市不准，可点击重新定位或手动选城。');
      const fallbackResult = await syncDashboardByIP({ fallback: true, manageBusy: false });
      if (!fallbackResult.ok && fallbackResult.kind !== 'api-error') {
        showHint('可尝试重新定位，或切换为手动输入城市。', 'error');
      }
      return;
    }
    showAPIError();
  } finally {
    setBusy(false);
  }
}

function buildGeoErrorMessage(error) {
  const rawMessage = error?.message || '未知定位错误';
  if (!supportsSecureGeolocation()) {
    return '定位失败：当前页面不是可信安全上下文。请确认手机访问的是有效 HTTPS，且证书已被系统信任。';
  }
  if (rawMessage.includes('Origin does not have permission to use Geolocation service')) {
    return '定位失败：当前来源没有定位权限。请检查浏览器站点定位权限。';
  }
  if (rawMessage.includes('Only secure origins') || rawMessage.includes('secure origin')) {
    return '定位失败：浏览器认为当前页面不是可信 HTTPS。请检查 Caddy 证书是否被手机系统信任。';
  }
  if (error?.code === 1) {
    return '定位失败：浏览器或系统拒绝了定位权限。请在站点设置中允许位置权限后重试。';
  }
  if (error?.code === 2) {
    return '定位失败：设备暂时无法提供当前位置，可能是系统定位服务关闭、室内信号弱或浏览器拿不到定位源。';
  }
  if (error?.code === 3) {
    return '定位失败：定位超时，没有在限定时间内得到位置结果。';
  }
  return `定位失败：${rawMessage}`;
}

async function locateAndSync() {
  setBusy(true, '正在请求定位权限...');

  if (!navigator.geolocation) {
    setStatus('当前浏览器不支持 GPS，正在尝试网络定位...');
    showHint('');
    const fallbackResult = await syncDashboardByIP({ fallback: true, manageBusy: false });
    if (!fallbackResult.ok) {
      if (fallbackResult.kind === 'api-error') {
        focusCityInput();
        setBusy(false);
        return;
      }
      setStatus('当前浏览器不支持 GPS', 'error');
      showHint('网络定位不可用时，可以直接输入城市。', 'error');
    }
    focusCityInput();
    setBusy(false);
    return;
  }
  if (!supportsSecureGeolocation()) {
    setStatus('GPS 不可用，正在尝试网络定位...');
    showHint('');
    const fallbackResult = await syncDashboardByIP({ fallback: true, manageBusy: false });
    if (!fallbackResult.ok) {
      if (fallbackResult.kind === 'api-error') {
        focusCityInput();
        setBusy(false);
        return;
      }
      setStatus('当前访问地址无法调用手机 GPS', 'error');
      showHint('手机 GPS 需要可信 HTTPS，网络定位不可用时可以直接输入城市。', 'error');
    }
    focusCityInput();
    setBusy(false);
    return;
  }

  showHint('');
  try {
    const position = await getPositionWithFallback();
    setBusy(true, '定位成功，正在同步天气数据...');
    const data = await requestJSON('/api/location', {
      method: 'POST',
      body: JSON.stringify({
        client_id: getClientID(),
        lat: position.coords.latitude,
        lon: position.coords.longitude,
      }),
    });
    renderDashboard(data);
  } catch (error) {
    const gpsMessage = buildGeoErrorMessage(error);
    setStatus('GPS 定位失败，正在尝试网络定位...');
    showHint('');
    const fallbackResult = await syncDashboardByIP({ fallback: true, manageBusy: false });
    if (!fallbackResult.ok) {
      if (fallbackResult.kind === 'api-error') {
        focusCityInput();
        return;
      }
      setStatus(gpsMessage, 'error');
      showHint('网络定位失败时，请切换为手动输入城市。', 'error');
    }
    focusCityInput();
  } finally {
    setBusy(false);
  }
}

async function locateByIP() {
  await syncDashboardByIP({ fallback: false });
}

async function syncDashboardByIP({ fallback = false, manageBusy = true } = {}) {
  if (isPrivateNetworkHost()) {
    setStatus('当前网络环境不支持网络定位', 'error');
    showHint('局域网地址无法估算真实城市，请切换为手动输入。', 'error');
    focusCityInput();
    return { ok: false, kind: 'unsupported' };
  }

  try {
    if (manageBusy) {
      setBusy(true, fallback ? 'GPS 不可用，正在尝试网络定位...' : '正在根据公网 IP 估算城市...');
    }
    const data = await requestJSON('/api/location/ip', {
      method: 'POST',
      body: JSON.stringify({ client_id: getClientID() }),
    });
    renderDashboard(data);
    setStatus(fallback ? '已使用网络定位显示近似城市，可手动切换' : '网络定位成功，为近似城市，可手动切换');
    showHint('');
    return { ok: true, kind: 'ok' };
  } catch (error) {
    showAPIError();
    showHint('');
    if (!fallback) {
      focusCityInput();
    }
    return { ok: false, kind: 'api-error' };
  } finally {
    if (manageBusy) {
      setBusy(false);
    }
  }
}

function getCurrentPosition(options) {
  return new Promise((resolve, reject) => {
    navigator.geolocation.getCurrentPosition(resolve, reject, options);
  });
}

function watchPositionOnce(options, timeoutMs) {
  return new Promise((resolve, reject) => {
    let settled = false;
    const timer = window.setTimeout(() => {
      if (settled) {
        return;
      }
      settled = true;
      if (watchID !== null) {
        navigator.geolocation.clearWatch(watchID);
      }
      reject({ code: 3, message: 'watch-position timeout' });
    }, timeoutMs);

    let watchID = null;
    watchID = navigator.geolocation.watchPosition((position) => {
      if (settled) {
        return;
      }
      settled = true;
      window.clearTimeout(timer);
      navigator.geolocation.clearWatch(watchID);
      resolve(position);
    }, (error) => {
      if (settled) {
        return;
      }
      settled = true;
      window.clearTimeout(timer);
      navigator.geolocation.clearWatch(watchID);
      reject(error);
    }, options);
  });
}

async function getPositionWithFallback() {
  try {
    setStatus('请在浏览器弹窗中允许位置权限...');
    return await getCurrentPosition({
      enableHighAccuracy: false,
      timeout: 18000,
      maximumAge: 300000,
    });
  } catch (error) {
    if (error?.code === 1) {
      throw error;
    }
  }

  try {
    setStatus('正在监听手机定位结果...');
    return await watchPositionOnce({
      enableHighAccuracy: false,
      maximumAge: 300000,
    }, 20000);
  } catch (error) {
    if (error?.code === 1) {
      throw error;
    }
  }

  try {
    setStatus('大致定位失败，正在尝试手机 GPS 精确定位...');
    return await getCurrentPosition({
      enableHighAccuracy: true,
      timeout: 30000,
      maximumAge: 0,
    });
  } catch (error) {
    if (error?.code === 1) {
      throw error;
    }
  }

  try {
    setStatus('正在读取手机最近一次可用位置...');
    return await getCurrentPosition({
      enableHighAccuracy: false,
      timeout: 6000,
      maximumAge: Infinity,
    });
  } catch (error) {
    if (error?.code === 1) {
      throw error;
    }
  }

  setStatus('正在做最后一次高精度定位尝试...');
  return watchPositionOnce({
    enableHighAccuracy: true,
    maximumAge: 0,
  }, 30000);
}

async function selectCity(cityName) {
  const trimmed = (cityName || '').trim();
  if (!trimmed) {
    setStatus('请输入城市名称', 'error');
    focusCityInput();
    return;
  }

  try {
    setBusy(true, '正在切换城市...');
    const data = await requestJSON('/api/city/select', {
      method: 'POST',
      body: JSON.stringify({ client_id: getClientID(), city_name: trimmed }),
    });
    dom.cityInput.value = trimmed;
    renderDashboard(data);
  } catch (error) {
    showAPIError();
    showHint('');
    focusCityInput();
  } finally {
    setBusy(false);
  }
}

function focusCityInput() {
  if (dom.actionSelect.value !== 'manual') {
    dom.actionSelect.value = 'manual';
    syncManualFormState();
  }
  if (isTouchDevice()) {
    return;
  }
  try {
    dom.cityInput.focus({ preventScroll: true });
  } catch (error) {
    dom.cityInput.focus();
  }
  dom.cityInput.select();
}

function focusCityInputFromUser() {
  if (dom.actionSelect.value !== 'manual') {
    dom.actionSelect.value = 'manual';
    syncManualFormState();
  }
  dom.cityInput.scrollIntoView({ behavior: 'smooth', block: 'center' });
  window.setTimeout(() => {
    dom.cityInput.focus();
    dom.cityInput.select();
  }, 220);
}

function formatTime(value) {
  if (!value) {
    return '--';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('zh-CN', { hour12: false });
}

dom.locateBtn.addEventListener('click', locateAndSync);
dom.actionSelect.addEventListener('change', () => {
  syncManualFormState();
  if (dom.actionSelect.value === 'manual') {
    focusCityInputFromUser();
  }
});
dom.manualForm.addEventListener('submit', (event) => {
  event.preventDefault();
  if (dom.actionSelect.value === 'manual') {
    selectCity(dom.cityInput.value);
    return;
  }
  if (dom.actionSelect.value === 'ip') {
    locateByIP();
    return;
  }
  locateAndSync();
});

window.addEventListener('resize', () => {
  state.tempChart?.resize();
  state.humidityChart?.resize();
});

if (!supportsSecureGeolocation()) {
  showHint('当前页面不是可信 HTTPS 或 localhost，GPS 可能不可用。');
}

syncManualFormState();
loadDashboard(true);
