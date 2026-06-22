# VMS — Video Management System

Система управления видеонаблюдением на Go с веб-интерфейсом. Поддерживает подключение IP-камер по RTSP, live-трансляцию (MJPEG), запись видео в архив, просмотр записей и опциональное обнаружение людей на кадре (YOLOv8 + GoCV) с уведомлением.

### Возможности

- JWT-авторизация пользователей
- CRUD камер (имя + RTSP URL) с хранением в PostgreSQL
- Live-трансляция с камеры в браузере (MJPEG)
- Запись RTSP-потока в MP4 (сегменты по 60 секунд, новая папка каждый день)
- Архив записей с воспроизведением в браузере
- Детекция людей (YOLOv8 ONNX) при трансляции и записи
- Отправка уведомлений об обнаружении по Email

### Стек

| Компонент | Технология |
|-----------|------------|
| Backend | Go 1.26+, Gorilla Mux |
| База данных | PostgreSQL |
| Видео | OpenCV (gocv), FFmpeg |
| Детекция | YOLOv8 (ONNX) |
| Frontend | HTML / CSS / JavaScript |

### Структура проекта

```
vms/
├── api/                  # HTTP API, бизнес-логика
│   ├── archive/          # Список файлов архива
│   ├── auth/             # JWT-авторизация
│   ├── cameras/          # Работа с камерами в БД
│   ├── capture/          # Захват RTSP-потока
│   ├── database/         # Инициализация БД
│   ├── detection/        # ИИ-детекция
│   ├── dto/              # DTO-объекты
│   ├── handlers/         # HTTP-хендлеры
│   ├── notification/     # Email-уведомления
│   ├── record/           # Запись через FFmpeg
│   ├── server/           # Маршруты HTTP-сервера
│   ├── stream/           # MJPEG-трансляция
│   └── utils/            # Вспомогательные методы
├── cmd/
│   └── main.go           # Точка входа
│  
├── frontend/             # Веб-интерфейс
├── yolo-model/           # Каталог для YOLO-модели
├── recordings/           # Архив записей (создаётся автоматически)
├── init.sql              # Схема БД и пользователь по умолчанию
└── .env                  # Конфигурация
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

Либо используйте готовый файл `yolov8n.onnx` из папки `yolo-model` проекта.

### Локальный запуск

#### 1. Клонирование и зависимости Go

```bash
git clone <url-репозитория> vms
cd vms
go mod download
```

#### 2. Установка GoCV (OpenCV)

Для установки OpenCV используйте официальную документацию GoCV (см. [документацию GoCV](https://gocv.io/getting-started/)).

#### 3. PostgreSQL

Создайте базу и примените схему (файл `init.sql`):

```bash
sudo -u postgres psql <<'SQL'
CREATE DATABASE vms;
\c vms
SQL
psql -U postgres -d vms -f init.sql
```

#### 4. Конфигурация `.env`

Создайте файл `.env` в **корне проекта**:

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

SMTP_SERVER=smtp.gmail.com
SMTP_USERNAME=a@mail.co
SMTP_PASSWORD=123
NOTIFY_TO=a@mail.co
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
| `SMTP_SERVER` | Адрес SMTP-сервера для отправки уведомлений |
| `SMTP_USERNAME` | Имя пользователя для авторизации SMTP |
| `SMTP_PASSWORD` | Пароль для авторизации SMTP |
| `NOTIFY_TO` | Адрес почты, куда будут отправляться уведомления |

> **Важно:** приложение использует относительные пути и должно запускаться из каталога `cmd/`. Файл `.env` читается как `../.env`, frontend — из `../frontend`, записи — в `../recordings`.

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

Пользователь создаётся скриптом `init.sql`. Для продакшена смените пароль.

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

Полный скрипт инициализации: [`init.sql`](init.sql).

### Устранение неполадок

| Проблема | Решение |
|----------|---------|
| `panic: open ../.env: no such file` | Запускайте из `cmd/`, `.env` должен быть в корне проекта |
| Ошибка подключения к БД | Проверьте `DB_URL`, что PostgreSQL запущен и схема создана |
| Не загружается YOLO | Убедитесь, что файл существует по пути из `YOLO_MODEL` |
| Нет кадров с камеры | Проверьте RTSP URL (VLC), увеличьте `TIMEOUT_CAPTURE` |
| FFmpeg не найден | Установите `ffmpeg` (`apt install ffmpeg`) |
| Ошибка сборки gocv | Установите OpenCV 4.x и `pkg-config`, проверьте `CGO_ENABLED=1` |

