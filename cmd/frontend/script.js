const API_HOST = window.location.origin;
let token = localStorage.getItem("vms_token") || "";
let role = localStorage.getItem("vms_role") || "";

let liveStreamPollId = null;
let liveStreamOnline = false;
const LIVE_STREAM_POLL_MS = 3000;

const video = document.getElementById("archiveVideo");

if (token) showDashboard();

function authHeaders(json) {
    const headers = { "Authorization": token };
    if (json) headers["Content-Type"] = "application/json";
    return headers;
}

function formatApiTime(iso) {
    if (!iso) return "";
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? "" : d.toLocaleString("ru-RU");
}

async function parseApiError(res) {
    const text = await res.text();
    if (!text) return { message: `Ошибка ${res.status}`, time: null, status: res.status };

    try {
        const err = JSON.parse(text);
        return {
            message: err.message || text,
            time: err.time || null,
            status: res.status
        };
    } catch (_) {
        return { message: text.trim(), time: null, status: res.status };
    }
}

function showToast(type, title, message, time) {
    const container = document.getElementById("toastContainer");
    const toast = document.createElement("div");
    toast.className = `toast toast-${type}`;
    toast.innerHTML = `
        <div class="toast-title">${title}</div>
        <div class="toast-message">${escapeHtml(message)}</div>
        ${time ? `<div class="toast-time">${escapeHtml(formatApiTime(time))}</div>` : ""}
    `;
    const remove = () => toast.remove();
    toast.onclick = remove;
    container.appendChild(toast);
    setTimeout(remove, type === "error" ? 8000 : 4000);
}

function escapeHtml(text) {
    return String(text)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

function showError(err, title = "Ошибка") {
    const message = typeof err === "string" ? err : (err.message || "Неизвестная ошибка");
    const time = err && err.time ? err.time : null;
    showToast("error", title, message, time);
    return message;
}

function showSuccess(message) {
    showToast("success", "Успешно", message);
}

async function apiFetch(url, options = {}) {
    const res = await fetch(url, {
        ...options,
        headers: { ...authHeaders(options.body != null), ...(options.headers || {}) }
    });
    if (!res.ok) {
        const apiError = await parseApiError(res);
        const error = new Error(apiError.message);
        error.time = apiError.time;
        error.status = apiError.status;
        throw error;
    }
    if (res.status === 204) return null;
    const ct = res.headers.get("content-type") || "";
    if (ct.includes("application/json")) return res.json();
    return res.text();
}

async function login() {
    const loginUser = document.getElementById("loginUser").value;
    const loginPass = document.getElementById("loginPass").value;

    try {
        const data = await apiFetch(`${API_HOST}/api/login`, {
            method: "POST",
            body: JSON.stringify({ login: loginUser, password: loginPass })
        });
        token = data.token;
        role = data.role;
        localStorage.setItem("vms_token", token);
        localStorage.setItem("vms_role", role);
        showSuccess("Вход выполнен");
        showDashboard();
    } catch (err) {
        showError(err, "Ошибка авторизации");
    }
}

function logout() {
    stopLiveStreamPolling();
    localStorage.clear();
    location.reload();
}

function showDashboard() {
    document.getElementById("loginBlock").classList.add("hidden");
    document.getElementById("vmsBlock").classList.remove("hidden");

    if (role !== "admin") {
        document.getElementById("tabCamBtn").classList.add("hidden");
        switchTab("view");
    } else {
        loadCamerasTable();
    }
    loadCamerasSelect();

    if (!document.getElementById("tabView").classList.contains("hidden")) {
        startLiveStreamPolling();
    }
}

function switchTab(tab) {
    document.querySelectorAll(".tab-btn").forEach(b => b.classList.remove("active"));
    if (tab === "cam") {
        document.getElementById("tabCamBtn").classList.add("active");
        document.getElementById("tabCameras").classList.remove("hidden");
        document.getElementById("tabView").classList.add("hidden");
        loadCamerasTable();
        stopLiveStreamPolling();
    } else {
        document.getElementById("tabViewBtn").classList.add("active");
        document.getElementById("tabCameras").classList.add("hidden");
        document.getElementById("tabView").classList.remove("hidden");
        startLiveStreamPolling();
    }
}

function loadCamerasTable() {
    apiFetch(`${API_HOST}/api/cameras`)
    .then(data => {
        const tbody = document.getElementById("camerasTable");
        tbody.innerHTML = "";
        if (!data.length) {
            tbody.innerHTML = '<tr><td colspan="4" class="empty-msg">Камеры не добавлены</td></tr>';
            return;
        }
        data.forEach(c => {
            const tr = document.createElement("tr");
            tr.innerHTML = `
                <td>${c.id}</td>
                <td>${c.name}</td>
                <td style="word-break: break-all;">${c.rtsplink}</td>
                <td>
                    <div class="actions">
                        <label class="checkbox-inline">
                            <input type="checkbox" id="detect-stream-${c.id}"> Включить обнаружение людей
                        </label>
                        <button class="secondary small" onclick="startCameraStream(${c.id})">Старт трансляции</button>
                        <button class="small" onclick="stopCameraStream(${c.id})">Стоп трансляции</button>
                        <button class="danger small" onclick="deleteCamera(${c.id})">Удалить</button>
                    </div>
                </td>`;
            tbody.appendChild(tr);
        });
    })
    .catch(err => showError(err, "Ошибка загрузки камер"));
}

function loadCamerasSelect() {
    apiFetch(`${API_HOST}/api/cameras`)
    .then(data => {
        const select = document.getElementById("camSelect");
        const current = select.value;
        select.innerHTML = '<option value="">-- Выберите видеоканал --</option>';
        data.forEach(c => {
            const opt = document.createElement("option");
            opt.value = c.id;
            opt.textContent = c.name;
            select.appendChild(opt);
        });
        if (current) select.value = current;
    })
    .catch(err => showError(err, "Ошибка загрузки списка"));
}

function addCamera() {
    const name = document.getElementById("camName").value.trim();
    const rtsplink = document.getElementById("camUrl").value.trim();
    if (!name || !rtsplink) return showError("Заполните название и RTSP-адрес");

    apiFetch(`${API_HOST}/api/cameras`, {
        method: "POST",
        body: JSON.stringify({ name, rtsplink })
    })
    .then(() => {
        showSuccess("Камера добавлена");
        loadCamerasTable();
        loadCamerasSelect();
        document.getElementById("camName").value = "";
        document.getElementById("camUrl").value = "";
    })
    .catch(err => showError(err, "Ошибка добавления камеры"));
}

function deleteCamera(id) {
    if (!confirm("Удалить камеру?")) return;
    apiFetch(`${API_HOST}/api/cameras/${id}`, { method: "DELETE" })
    .then(() => {
        showSuccess("Камера удалена");
        loadCamerasTable();
        loadCamerasSelect();
        if (document.getElementById("camSelect").value === String(id)) {
            liveStreamOnline = false;
            document.getElementById("liveImg").removeAttribute("src");
            setLivePlayerState("idle");
        }
    })
    .catch(err => showError(err, "Ошибка удаления камеры"));
}

function startCameraStream(id) {
    const cb = document.getElementById(`detect-stream-${id}`);
    const detection = cb ? cb.checked : false;

    apiFetch(`${API_HOST}/api/cameras/${id}/stream/start?detection=${detection}`)
    .then(data => {
        showSuccess(data);
        if (document.getElementById("camSelect").value === String(id)) loadLiveStream();
    })
    .catch(err => showError(err, "Ошибка запуска трансляции"));
}

function stopCameraStream(id) {
    apiFetch(`${API_HOST}/api/cameras/${id}/stream/stop`)
    .then(data => {
        showSuccess(data);
        if (document.getElementById("camSelect").value === String(id)) {
            liveStreamOnline = false;
            document.getElementById("liveImg").removeAttribute("src");
            setLivePlayerState("loading");
        }
    })
    .catch(err => showError(err, "Ошибка остановки трансляции"));
}

function startRecording(id) {
    const cb = document.getElementById("detectionRecordCheck");
    const detection = cb ? cb.checked : false;

    apiFetch(`${API_HOST}/api/cameras/${id}/record/start?detection=${detection}`)
    .then(data => showSuccess(data))
    .catch(err => showError(err, "Ошибка запуска записи"));
}

function stopRecording(id) {
    apiFetch(`${API_HOST}/api/cameras/${id}/record/stop`)
    .then(data => showSuccess(data))
    .catch(err => showError(err, "Ошибка остановки записи"));
}

function startSelectedRecording() {
    const camId = document.getElementById("camSelect").value;
    if (!camId) return showError("Выберите камеру");
    startRecording(camId);
}

function stopSelectedRecording() {
    const camId = document.getElementById("camSelect").value;
    if (!camId) return showError("Выберите камеру");
    stopRecording(camId);
}

function onCameraSelectChange() {
    liveStreamOnline = false;
    document.getElementById("liveImg").removeAttribute("src");
    loadLiveStream(false);
    document.getElementById("archiveButtons").innerHTML = "";
    document.getElementById("archivePlayerBlock").classList.add("hidden");
}

function setLivePlayerState(state) {
    const loading = document.getElementById("liveLoading");
    const img = document.getElementById("liveImg");
    const errorBlock = document.getElementById("liveError"); 

    loading.classList.toggle("hidden", state !== "loading");
    img.classList.toggle("hidden", state !== "playing");
    errorBlock.classList.toggle("hidden", state !== "error");
}

function connectLiveStream(url) {
    const img = document.getElementById("liveImg");
    setLivePlayerState("loading");

    img.onload = function() {
        if (!liveStreamOnline) {
            liveStreamOnline = true;
        }
        setLivePlayerState("playing");
    };

    img.onerror = function() {
        if (liveStreamOnline) { 
            liveStreamOnline = false;
            setLivePlayerState("error");
            img.removeAttribute("src");
        }
    };

    img.onabort = function() {
        if (liveStreamOnline) {
            liveStreamOnline = false;
            setLivePlayerState("error");
            img.removeAttribute("src");
        }
    };

    img.src = url;
}

function retryLiveStream() {
    const camId = document.getElementById("camSelect").value;
    if (!camId) return;
    
    liveStreamOnline = false;
    
    const url = `${API_HOST}/api/cameras/${camId}/stream?token=${encodeURIComponent(token)}&t=${new Date().getTime()}`;
    connectLiveStream(url);
}

function startLiveStreamPolling() {
    stopLiveStreamPolling();

    liveStreamPollId = setInterval(async () => {
        const camId = document.getElementById("camSelect").value;
        if (!camId) return;

        if (document.getElementById("tabView").classList.contains("hidden")) return;

        const img = document.getElementById("liveImg");
        if (liveStreamOnline && img.src) return;

        const url = `${API_HOST}/api/stream/${camId}?token=${encodeURIComponent(token)}`;

        try {
            const apiError = await checkMediaUrl(url);
            if (!apiError) {
                await loadLiveStream(true);
            } else {
                setLivePlayerState("loading");
            }
        } catch (_) {
            liveStreamOnline = false;
        }
    }, LIVE_STREAM_POLL_MS);
}

function stopLiveStreamPolling() {
    if (liveStreamPollId !== null) {
        clearInterval(liveStreamPollId);
        liveStreamPollId = null;
    }
}

async function checkMediaUrl(url) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 5000);
    try {
        const res = await fetch(url, { headers: authHeaders(), signal: controller.signal });
        if (!res.ok) {
            clearTimeout(timeout);
            return parseApiError(res);
        }
        controller.abort();
        return null;
    } catch (err) {
        if (err.name === "AbortError") return null;
        throw err;
    } finally {
        clearTimeout(timeout);
    }
}

async function loadLiveStream(silent = false) {
    const camId = document.getElementById("camSelect").value;
    const img = document.getElementById("liveImg");

    if (!camId) {
        liveStreamOnline = false;
        img.removeAttribute("src");
        setLivePlayerState("idle");
        return;
    }

    if (liveStreamOnline && img.src) return;

    const url = `${API_HOST}/api/stream/${camId}?token=${encodeURIComponent(token)}`;
    try {
        const apiError = await checkMediaUrl(url);
        if (apiError) {
            liveStreamOnline = false;
            img.removeAttribute("src");
            setLivePlayerState("loading");
            if (!silent) {
                showError({ message: apiError.message, time: apiError.time }, "Ошибка live-потока");
            }
            return;
        }
        connectLiveStream(url);
    } catch (err) {
        liveStreamOnline = false;
        img.removeAttribute("src");
        setLivePlayerState("loading");
        if (!silent) {
            showError(err.message || "Не удалось подключиться к потоку", "Ошибка live-потока");
        }
    }
}

function onLiveStreamError() {
    liveStreamOnline = false;
    document.getElementById("liveImg").removeAttribute("src");
    const camId = document.getElementById("camSelect").value;
    setLivePlayerState(camId ? "loading" : "idle");
}

function toArchiveDate(isoDate) {
    const [year, month, day] = isoDate.split("-");
    return `${day}-${month}-${year}`;
}

function searchArchive() {
    const camId = document.getElementById("camSelect").value;
    const isoDate = document.getElementById("archiveDate").value;
    if (!camId || !isoDate) return showError("Выберите камеру и дату для поиска");

    const date = toArchiveDate(isoDate);
    const container = document.getElementById("archiveButtons");

    apiFetch(`${API_HOST}/api/archive/${camId}/${date}`)
    .then(data => {
        container.innerHTML = "";

        data.forEach(f => {
            const btn = document.createElement("button");
            btn.textContent = `${f.time}`;
            btn.onclick = () => playVideo(f.path, f.time);
            container.appendChild(btn);
        });
    })
    .catch(err => showError(err, "Ошибка поиска в архиве"));
}

async function playVideo(relativePath, startTime) {
    document.getElementById("archivePlayerBlock").classList.remove("hidden");
    document.getElementById("currentPlayingTime").innerText = `Фрагмент: ${startTime}`;

    const url = `${API_HOST}/api/recordings/${relativePath}?token=${encodeURIComponent(token)}`;
    try {
        const apiError = await checkMediaUrl(url);
        if (apiError) {
            showError({ message: apiError.message, time: apiError.time }, "Ошибка воспроизведения");
            video.removeAttribute("src");
            return;
        }
        video.src = url;
        video.play().catch(() => {});
    } catch (err) {
        showError("Не удалось загрузить видео", "Ошибка воспроизведения");
    }
}

video.addEventListener("error", () => {
    showError("Не удалось воспроизвести видеофайл. Возможно, файл обрабатывается. Попробуйте обновить страницу", "Ошибка воспроизведения");
});