# VMS — Video Management System

**Languages:** [Русский](#русский) · [English](#english)

---

<a id="русский"></a>

## Русский

Система управления видеонаблюдением на Go с веб-интерфейсом. Поддерживает подключение IP-камер по RTSP, live-трансляцию (MJPEG), запись видео в архив, просмотр записей и опциональное обнаружение людей на кадре (YOLOv8 + OpenCV).

### Возможности

- JWT-авторизация пользователей
- CRUD камер (имя + RTSP URL) с хранением в PostgreSQL
- Live-трансляция с камеры в браузере (MJPEG)
- Запись RTSP-потока в MP4 (сегменты по 60 секунд, новая папка каждый день)
- Архив записей с воспроизведением в браузере
- Детекция людей (YOLOv8 ONNX) при трансляции и записи

### Стек

| Компонент | Технология |
|-----------|------------|
| Backend | Go 1.26+, Gorilla Mux |
| База данных | PostgreSQL |
| Видео | OpenCV (gocv), FFmpeg |
| Детекция | YOLOv8 (ONNX) |
| Frontend | HTML / CSS / JavaScript (статика) |

### Структура проекта

```
vms/
├── api/                  # HTTP API, бизнес-логика
│   ├── auth/             # JWT-авторизация
│   ├── cameras/          # Работа с камерами в БД
│   ├── capture/          # Захват RTSP-потока
│   ├── stream/           # MJPEG-трансляция
│   ├── record/           # Запись через FFmpeg
│   ├── detection/        # YOLO-детекция
│   ├── archive/          # Список файлов архива
│   └── server/           # Маршруты HTTP-сервера
├── cmd/
│   ├── main.go           # Точка входа
│   └── frontend/         # Веб-интерфейс
├── docker/
│   └── init.sql          # Схема БД и пользователь по умолчанию
├── models/               # Каталог для YOLO-модели (создаётся вручную)
├── recordings/           # Архив записей (создаётся автоматически)
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── .env                  # Конфигурация (не коммитится)
```

### Требования

#### Для локального запуска

- **Go** 1.26.3 или новее
- **PostgreSQL** 14+
- **OpenCV** 4.x (для сборки `gocv.io/x/gocv`)
- **FFmpeg** (кодирование записей)
- **pkg-config**, компилятор C/C++ (CGO)

#### Модель YOLO

Скачайте ONNX-модель YOLOv8 и положите в каталог `models/`:

```bash
mkdir -p models
pip install ultralytics   # или используйте другой способ экспорта
python3 - <<'PY'
from ultralytics import YOLO
YOLO("yolov8n.pt").export(format="onnx")
PY
mv yolov8n.onnx models/
```

Либо скачайте готовый файл `yolov8n.onnx` из [репозитория Ultralytics](https://github.com/ultralytics/assets/releases) и поместите в `models/yolov8n.onnx`.

### Локальный запуск

#### 1. Клонирование и зависимости Go

```bash
git clone <url-репозитория> vms
cd vms
go mod download
```

#### 2. Установка OpenCV (Ubuntu / Debian)

```bash
sudo apt update
sudo apt install -y \
  build-essential pkg-config \
  libopencv-dev \
  ffmpeg
```

Для других дистрибутивов см. [документацию GoCV](https://gocv.io/getting-started/linux/).

#### 3. PostgreSQL

Создайте базу и примените схему:

```bash
sudo -u postgres psql <<'SQL'
CREATE DATABASE vms;
\c vms
SQL
psql -U postgres -d vms -f docker/init.sql
```

#### 4. Конфигурация `.env`

Создайте файл `.env` в **корне проекта** (рядом с `go.mod`):

```env
DB_URL=postgres://postgres:123@localhost/postgres?sslmode=disable
DB_DRIVER=postgres

PORT=:8091
IP_ADDR=0.0.0.0

JWT_SECRET_KEY=ultramegasecretkey

YOLO_MODEL=./models/yolov8n.onnx
YOLO_SCORE_THRESHOLD=0.45
YOLO_NMS_THRESHOLD=0.5

RECORDINGS_DIR=../recordings

TIMEOUT_CAPTURE=3000000
```

| Переменная | Описание |
|------------|----------|
| `DB_URL` | Строка подключения PostgreSQL |
| `DB_DRIVER` | Драйвер БД (`postgres`) |
| `IP_ADDR` | IP для прослушивания (`0.0.0.0` — все интерфейсы) |
| `PORT` | Порт с двоеточием, например `:8091` |
| `JWT_SECRET_KEY` | Секрет для подписи JWT |
| `YOLO_MODEL` | Путь к ONNX-модели относительно `cmd/` |
| `YOLO_SCORE_THRESHOLD` | Порог уверенности детекции (0–1) |
| `YOLO_NMS_THRESHOLD` | Порог NMS (0–1) |
| `RECORDINGS_DIR` | Каталог записей относительно `cmd/` |
| `TIMEOUT_CAPTURE` | Таймаут RTSP в микросекундах (для OpenCV FFmpeg) |

> **Важно:** приложение использует относительные пути и должно запускаться из каталога `cmd/`. Файл `.env` читается как `../.env`, frontend — из `./frontend`, записи — в `../recordings`.

#### 5. Запуск

```bash
cd cmd
go run .
```

Откройте в браузере: **http://localhost:8091**

#### 6. Вход в систему

| Логин | Пароль |
|-------|--------|
| `admin` | `admin123` |

Пользователь создаётся скриптом `docker/init.sql`. Для продакшена смените пароль.

### Запуск в Docker

Сборка использует **официальные pre-built образы GoCV** с OpenCV и Go toolchain — см. [документацию GoCV для Docker](https://gocv.io/getting-started/docker/) и [gocv/opencv на Docker Hub](https://hub.docker.com/r/gocv/opencv).

Базовый образ: `gocv/opencv:4.13.0` (совместим с `gocv v0.43.0` из `go.mod`). OpenCV **не собирается** внутри Dockerfile — используется готовый образ, как рекомендует GoCV. Сборка и runtime выполняются на одном и том же базовом образе (без копирования библиотек в `golang:*-bookworm`).

#### Быстрый старт

```bash
# 1. Модель YOLO (см. раздел выше)
mkdir -p models
# положите yolov8n.onnx в models/

# 2. Сборка и запуск
docker compose up --build -d

# 3. Логи
docker compose logs -f vms
```

Веб-интерфейс: **http://localhost:8091**

Остановка:

```bash
docker compose down
```

Данные PostgreSQL и записи сохраняются в Docker volumes (`postgres_data`, `recordings`).

#### Сервисы docker-compose

| Сервис | Описание | Порт |
|--------|----------|------|
| `db` | PostgreSQL 16 | 5432 (внутри сети) |
| `vms` | Приложение VMS | 8091 |

#### Переменные окружения в Docker

Переопределите через файл `.env` в корне проекта или экспорт в shell:

```bash
export JWT_SECRET_KEY=my-secret
docker compose up --build -d
```

Поддерживаются переменные из `docker-compose.yml`: `JWT_SECRET_KEY`, `YOLO_SCORE_THRESHOLD`, `YOLO_NMS_THRESHOLD`, `TIMEOUT_CAPTURE`.

Версию OpenCV можно переопределить при сборке (должна совпадать с требованиями GoCV):

```bash
docker compose build --build-arg OPENCV_VERSION=4.13.0
```

#### RTSP-камеры в локальной сети

Если камеры доступны только с хост-машины, а контейнер их не видит, включите режим сети хоста в `docker-compose.yml`:

```yaml
vms:
  network_mode: host
  # и уберите секцию ports
```

При `network_mode: host` приложение будет доступно на `http://<IP-хоста>:8091`.

#### Сборка образа вручную

```bash
docker build -t vms:latest .
docker run --rm -p 8091:8091 \
  -e DB_URL=postgres://postgres:postgres@db:5432/vms?sslmode=disable \
  -v "$(pwd)/models:/app/cmd/models:ro" \
  -v vms_recordings:/app/recordings \
  vms:latest
```

Для `docker run` нужен отдельный контейнер PostgreSQL с той же схемой БД.

### HTTP API

Базовый URL: `http://<host>:8091`

#### Авторизация

```http
POST /api/login
Content-Type: application/json

{"login": "admin", "password": "admin123"}
```

Ответ:

```json
{
    "role": "admin",
    "token": "<JWT>"
}
```

Все остальные `/api/*` маршруты требуют заголовок:

```http
Authorization: Bearer <JWT>
```

Для `<img>` / `<video>` допускается query-параметр `?token=<JWT>`.

#### Камеры

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/cameras` | Список камер |
| `POST` | `/api/cameras` | Добавить камеру |
| `DELETE` | `/api/cameras/{id}` | Удалить камеру |

Тело создания камеры:

```json
{
    "name": "Склад 1",
    "rtsplink": "rtsp://user:pass@192.168.1.100:554/stream"
}
```

#### Трансляция

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/cameras/{id}/stream/start?detection=false` | Запуск MJPEG |
| `GET` | `/api/cameras/{id}/stream/stop` | Остановка |
| `GET` | `/api/stream/{id}` | MJPEG-поток (для `<img src>`) |

#### Запись

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/cameras/{id}/record/start?detection=false` | Начать запись |
| `GET` | `/api/cameras/{id}/record/stop` | Остановить запись |

Записи сохраняются в:

```
recordings/{camera_id}/{DD-MM-YYYY}/{HH-MM-SS}.mp4
```

#### Архив

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/archive/{id}/{date}` | Список файлов за дату (`DD-MM-YYYY`) |
| `GET` | `/api/recordings/{camera_id}/{date}/{file}.mp4` | Скачивание / воспроизведение |

### Веб-интерфейс

После авторизации доступны вкладки:

1. **Мониторинг** — выбор камеры, live-поток, запуск/остановка записи, поиск в архиве
2. **Конфигурация камер** — добавление и удаление RTSP-источников

### Схема базы данных

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL
);

CREATE TABLE cameras (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rtsp_link TEXT NOT NULL
);
```

Полный скрипт инициализации: [`docker/init.sql`](docker/init.sql).

### Устранение неполадок

| Проблема | Решение |
|----------|---------|
| `panic: open ../.env: no such file` | Запускайте из `cmd/`, `.env` должен быть в корне проекта |
| Ошибка подключения к БД | Проверьте `DB_URL`, что PostgreSQL запущен и схема создана |
| Не загружается YOLO | Убедитесь, что файл существует по пути из `YOLO_MODEL` |
| Нет кадров с камеры | Проверьте RTSP URL (VLC), увеличьте `TIMEOUT_CAPTURE`, для Docker — `network_mode: host` |
| FFmpeg не найден | Установите `ffmpeg` (`apt install ffmpeg`) или используйте Docker-образ |
| Ошибка сборки gocv | Установите OpenCV 4.x и `pkg-config`, проверьте `CGO_ENABLED=1` |

---

<a id="english"></a>

## English

A Go-based video surveillance management system with a web interface. Supports IP cameras over RTSP, live streaming (MJPEG), video recording to an archive, playback of recordings, and optional person detection on frames (YOLOv8 + OpenCV).

### Features

- JWT user authentication
- Camera CRUD (name + RTSP URL) stored in PostgreSQL
- Live camera streaming in the browser (MJPEG)
- RTSP stream recording to MP4 (60-second segments, new folder each day)
- Recording archive with in-browser playback
- Person detection (YOLOv8 ONNX) during streaming and recording

### Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.26+, Gorilla Mux |
| Database | PostgreSQL |
| Video | OpenCV (gocv), FFmpeg |
| Detection | YOLOv8 (ONNX) |
| Frontend | HTML / CSS / JavaScript (static) |

### Project structure

```
vms/
├── api/                  # HTTP API, business logic
│   ├── auth/             # JWT authentication
│   ├── cameras/          # Camera DB operations
│   ├── capture/          # RTSP stream capture
│   ├── stream/           # MJPEG streaming
│   ├── record/           # Recording via FFmpeg
│   ├── detection/        # YOLO detection
│   ├── archive/          # Archive file listing
│   └── server/           # HTTP server routes
├── cmd/
│   ├── main.go           # Entry point
│   └── frontend/         # Web interface
├── docker/
│   └── init.sql          # DB schema and default user
├── models/               # YOLO model directory (created manually)
├── recordings/           # Recording archive (created automatically)
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── .env                  # Configuration (not committed)
```

### Requirements

#### Local run

- **Go** 1.26.3 or newer
- **PostgreSQL** 14+
- **OpenCV** 4.x (required to build `gocv.io/x/gocv`)
- **FFmpeg** (recording encoding)
- **pkg-config**, C/C++ compiler (CGO)

#### YOLO model

Download a YOLOv8 ONNX model and place it in the `models/` directory:

```bash
mkdir -p models
pip install ultralytics   # or use another export method
python3 - <<'PY'
from ultralytics import YOLO
YOLO("yolov8n.pt").export(format="onnx")
PY
mv yolov8n.onnx models/
```

Alternatively, download a ready-made `yolov8n.onnx` from the [Ultralytics assets repository](https://github.com/ultralytics/assets/releases) and place it at `models/yolov8n.onnx`.

### Local setup

#### 1. Clone and fetch Go dependencies

```bash
git clone <repository-url> vms
cd vms
go mod download
```

#### 2. Install OpenCV (Ubuntu / Debian)

```bash
sudo apt update
sudo apt install -y \
  build-essential pkg-config \
  libopencv-dev \
  ffmpeg
```

For other distributions, see the [GoCV documentation](https://gocv.io/getting-started/linux/).

#### 3. PostgreSQL

Create the database and apply the schema:

```bash
sudo -u postgres psql <<'SQL'
CREATE DATABASE vms;
\c vms
SQL
psql -U postgres -d vms -f docker/init.sql
```

#### 4. `.env` configuration

Create a `.env` file in the **project root** (next to `go.mod`):

```env
DB_URL=postgres://postgres:123@localhost/postgres?sslmode=disable
DB_DRIVER=postgres

PORT=:8091
IP_ADDR=0.0.0.0

JWT_SECRET_KEY=ultramegasecretkey

YOLO_MODEL=./models/yolov8n.onnx
YOLO_SCORE_THRESHOLD=0.45
YOLO_NMS_THRESHOLD=0.5

RECORDINGS_DIR=../recordings

TIMEOUT_CAPTURE=3000000
```

| Variable | Description |
|----------|-------------|
| `DB_URL` | PostgreSQL connection string |
| `DB_DRIVER` | Database driver (`postgres`) |
| `IP_ADDR` | Listen address (`0.0.0.0` — all interfaces) |
| `PORT` | Port with leading colon, e.g. `:8091` |
| `JWT_SECRET_KEY` | Secret for signing JWT tokens |
| `YOLO_MODEL` | Path to ONNX model relative to `cmd/` |
| `YOLO_SCORE_THRESHOLD` | Detection confidence threshold (0–1) |
| `YOLO_NMS_THRESHOLD` | NMS threshold (0–1) |
| `RECORDINGS_DIR` | Recordings directory relative to `cmd/` |
| `TIMEOUT_CAPTURE` | RTSP timeout in microseconds (OpenCV FFmpeg) |

> **Important:** the application uses relative paths and must be started from the `cmd/` directory. The `.env` file is read as `../.env`, the frontend is served from `./frontend`, and recordings are stored in `../recordings`.

#### 5. Run

```bash
cd cmd
go run .
```

Open in your browser: **http://localhost:8091**

#### 6. Login

| Username | Password |
|----------|----------|
| `admin` | `admin123` |

This user is created by `docker/init.sql`. Change the password for production use.

### Docker setup

Build uses **official pre-built images of GoCV** with OpenCV and Go toolchain — link. [GoCV documentation for Docker](https://gocv.io/getting-started/docker/) and [gocv/opencv on Docker Hub](https://hub.docker.com/r/gocv/opencv).

Base image: `gocv/opencv:4.13.0` (compatible with `gocv v0.43.0` from `go.mod`). OpenCV **is not build** inside Dockerfile — pre-built image is used, as GoCV suggests. Build and runtime runs on the same base image (without copying libs into `golang:*-bookworm`).

#### Quick start

```bash
# 1. YOLO model (see section above)
mkdir -p models
# place yolov8n.onnx in models/

# 2. Build and run
docker compose up --build -d

# 3. Logs
docker compose logs -f vms
```

Web interface: **http://localhost:8091**

Stop:

```bash
docker compose down
```

PostgreSQL data and recordings are persisted in Docker volumes (`postgres_data`, `recordings`).

#### docker-compose services

| Service | Description | Port |
|---------|-------------|------|
| `db` | PostgreSQL 16 | 5432 (internal network) |
| `vms` | VMS application | 8091 |

#### Environment variables in Docker

Override via a `.env` file in the project root or shell export:

```bash
export JWT_SECRET_KEY=my-secret
docker compose up --build -d
```

Supported variables from `docker-compose.yml`: `JWT_SECRET_KEY`, `YOLO_SCORE_THRESHOLD`, `YOLO_NMS_THRESHOLD`, `TIMEOUT_CAPTURE`.

You can change OpenCV version on build (should be acceptable by GoCV):

```bash
docker compose build --build-arg OPENCV_VERSION=4.13.0
```

#### RTSP cameras on the local network

If cameras are only reachable from the host machine and not from the container, enable host networking in `docker-compose.yml`:

```yaml
vms:
  network_mode: host
  # and remove the ports section
```

With `network_mode: host`, the application is available at `http://<host-ip>:8091`.

#### Manual image build

```bash
docker build -t vms:latest .
docker run --rm -p 8091:8091 \
  -e DB_URL=postgres://postgres:postgres@db:5432/vms?sslmode=disable \
  -v "$(pwd)/models:/app/cmd/models:ro" \
  -v vms_recordings:/app/recordings \
  vms:latest
```

For `docker run`, a separate PostgreSQL container with the same DB schema is required.

### HTTP API

Base URL: `http://<host>:8091`

#### Authentication

```http
POST /api/login
Content-Type: application/json

{"login": "admin", "password": "admin123"}
```

Response:

```json
{
    "role": "admin",
    "token": "<JWT>"
}
```

All other `/api/*` routes require the header:

```http
Authorization: Bearer <JWT>
```

For `<img>` / `<video>` tags, the query parameter `?token=<JWT>` is also supported.

#### Cameras

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/cameras` | List cameras |
| `POST` | `/api/cameras` | Add a camera |
| `DELETE` | `/api/cameras/{id}` | Delete a camera |

Create camera request body:

```json
{
    "name": "Warehouse 1",
    "rtsplink": "rtsp://user:pass@192.168.1.100:554/stream"
}
```

#### Streaming

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/cameras/{id}/stream/start?detection=false` | Start MJPEG stream |
| `GET` | `/api/cameras/{id}/stream/stop` | Stop stream |
| `GET` | `/api/stream/{id}` | MJPEG stream (for `<img src>`) |

#### Recording

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/cameras/{id}/record/start?detection=false` | Start recording |
| `GET` | `/api/cameras/{id}/record/stop` | Stop recording |

Recordings are saved to:

```
recordings/{camera_id}/{DD-MM-YYYY}/{HH-MM-SS}.mp4
```

#### Archive

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/archive/{id}/{date}` | List files for a date (`DD-MM-YYYY`) |
| `GET` | `/api/recordings/{camera_id}/{date}/{file}.mp4` | Download / playback |

### Web interface

After login, two tabs are available:

1. **Monitoring** — camera selection, live stream, start/stop recording, archive search
2. **Camera configuration** — add and remove RTSP sources

### Database schema

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL
);

CREATE TABLE cameras (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rtsp_link TEXT NOT NULL
);
```

Full initialization script: [`docker/init.sql`](docker/init.sql).

### Troubleshooting

| Issue | Solution |
|-------|----------|
| `panic: open ../.env: no such file` | Run from `cmd/`; `.env` must be in the project root |
| Database connection error | Check `DB_URL`, ensure PostgreSQL is running and schema is applied |
| YOLO model fails to load | Verify the file exists at the path specified in `YOLO_MODEL` |
| No frames from camera | Check RTSP URL (VLC), increase `TIMEOUT_CAPTURE`, for Docker use `network_mode: host` |
| FFmpeg not found | Install `ffmpeg` (`apt install ffmpeg`) or use the Docker image |
| gocv build error | Install OpenCV 4.x and `pkg-config`, ensure `CGO_ENABLED=1` |