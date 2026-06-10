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

function setInlineError(elementId, message) {
    const el = document.getElementById(elementId);
    if (!message) {
        el.textContent = "";
        el.classList.add("hidden");
        return;
    }
    el.textContent = message;
    el.classList.remove("hidden");
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
    return res;
}

async function login() {
    const loginUser = document.getElementById("loginUser").value;
    const loginPass = document.getElementById("loginPass").value;
    setInlineError("loginError");

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
        const message = showError(err, "Ошибка авторизации");
        setInlineError("loginError", message);
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
            document.getElementById("liveImg").src = "";
        }
    })
    .catch(err => showError(err, "Ошибка удаления камеры"));
}

function startCameraStream(id) {
    apiFetch(`${API_HOST}/api/cameras/${id}/stream/start`)
    .then(() => {
        showSuccess(`Трансляция камеры ${id} запущена`);
        if (document.getElementById("camSelect").value === String(id)) loadLiveStream();
    })
    .catch(err => showError(err, "Ошибка запуска трансляции"));
}

function stopCameraStream(id) {
    apiFetch(`${API_HOST}/api/cameras/${id}/stream/stop`)
    .then(() => {
        showSuccess(`Трансляция камеры ${id} остановлена`);
        if (document.getElementById("camSelect").value === String(id)) {
            liveStreamOnline = false;
            document.getElementById("liveImg").src = "";
            setInlineError("liveError");
        }
    })
    .catch(err => showError(err, "Ошибка остановки трансляции"));
}

function startRecording(id) {
    apiFetch(`${API_HOST}/api/cameras/${id}/record/start`)
    .then(() => showSuccess(`Запись камеры ${id} запущена`))
    .catch(err => showError(err, "Ошибка запуска записи"));
}

function stopRecording(id) {
    apiFetch(`${API_HOST}/api/cameras/${id}/record/stop`)
    .then(() => showSuccess(`Запись камеры ${id} остановлена`))
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
    loadLiveStream(false);
    document.getElementById("archiveButtons").innerHTML = "";
    document.getElementById("archivePlayerBlock").classList.add("hidden");
    setInlineError("archiveVideoError");
}

function startLiveStreamPolling() {
    stopLiveStreamPolling();

    liveStreamPollId = setInterval(async () => {
        const camId = document.getElementById("camSelect").value;
        if (!camId) return;

        if (document.getElementById("tabView").classList.contains("hidden")) return;

        const url = `${API_HOST}/api/stream/${camId}?token=${encodeURIComponent(token)}`;

        try {
            const apiError = await checkMediaUrl(url);

            if (apiError) {
                if (liveStreamOnline) {
                    document.getElementById("liveImg").removeAttribute("src");
                }
                liveStreamOnline = false;
                return;
            }

            const img = document.getElementById("liveImg");
            if (!liveStreamOnline || !img.getAttribute("src")) {
                await loadLiveStream(true);
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

    if (!silent) {
        setInlineError("liveError");
    }

    if (!camId) {
        liveStreamOnline = false;
        img.removeAttribute("src");
        return;
    }

    const url = `${API_HOST}/api/stream/${camId}?token=${encodeURIComponent(token)}`;
    try {
        const apiError = await checkMediaUrl(url);
        if (apiError) {
            liveStreamOnline = false;
            const message = apiError.message;
            if (!silent) {
                setInlineError("liveError", message);
                showError({ message, time: apiError.time }, "Ошибка live-потока");
            }
            img.removeAttribute("src");
            return;
        }
        liveStreamOnline = true;
        if (!silent) {
            setInlineError("liveError");
        }
        img.src = `${url}&t=${Date.now()}`;
    } catch (err) {
        liveStreamOnline = false;
        const message = err.message || "Не удалось подключиться к потоку";
        if (!silent) {
            showError(message, "Ошибка live-потока");
            setInlineError("liveError", message);
        }
        img.removeAttribute("src");
    }
}

function onLiveStreamError() {
    liveStreamOnline = false;
    const message = "Не удалось отобразить видеопоток. Убедитесь, что трансляция камеры запущена.";
    setInlineError("liveError", message);
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
        if (!data.length) {
            container.innerHTML = '<p class="empty-msg">Записи за выбранный день отсутствуют.</p>';
            return;
        }
        data.forEach(f => {
            const btn = document.createElement("button");
            btn.textContent = `Фрагмент ${f.time}`;
            btn.onclick = () => playVideo(f.path, f.time);
            container.appendChild(btn);
        });
    })
    .catch(err => {
        const message = showError(err, "Ошибка поиска в архиве");
        container.innerHTML = `<p class="inline-error" style="margin:0;">${escapeHtml(message)}</p>`;
    });
}

async function playVideo(relativePath, startTime) {
    document.getElementById("archivePlayerBlock").classList.remove("hidden");
    document.getElementById("currentPlayingTime").innerText = `Фрагмент: ${startTime}`;
    setInlineError("archiveVideoError");

    const url = `${API_HOST}/api/recordings/${relativePath}?token=${encodeURIComponent(token)}`;
    try {
        const apiError = await checkMediaUrl(url);
        if (apiError) {
            const message = apiError.message;
            setInlineError("archiveVideoError", message);
            showError({ message, time: apiError.time }, "Ошибка воспроизведения");
            video.removeAttribute("src");
            return;
        }
        video.src = url;
        video.play().catch(() => {});
    } catch (err) {
        const message = showError(err.message || "Не удалось загрузить видео", "Ошибка воспроизведения");
        setInlineError("archiveVideoError", message);
    }
}

video.addEventListener("error", () => {
    const message = "Не удалось воспроизвести видеофайл.";
    setInlineError("archiveVideoError", message);
    showError(message, "Ошибка воспроизведения");
});

function rewind(seconds) {
    video.currentTime = Math.max(0, Math.min(video.duration || 0, video.currentTime + seconds));
}

video.addEventListener("timeupdate", () => {
    const current = Math.floor(video.currentTime);
    const hrs = String(Math.floor(current / 3600)).padStart(2, "0");
    const mins = String(Math.floor((current % 3600) / 60)).padStart(2, "0");
    const secs = String(current % 60).padStart(2, "0");
    document.getElementById("currentPlayingTime").innerText = `Смещение в плеере: ${hrs}:${mins}:${secs}`;
});
