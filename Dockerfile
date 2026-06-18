# OpenCV и Go toolchain — официальные pre-built образы GoCV:
# https://gocv.io/getting-started/docker/
# https://hub.docker.com/r/gocv/opencv
#
# Образ gocv/opencv:4.13.0 уже содержит совместимые OpenCV 4.13 и Go.
# Не смешивайте его с golang:*-bookworm — библиотеки из разных дистрибутивов
# несовместимы и приводят к ошибкам сборки/линковки CGO.

ARG OPENCV_VERSION=4.13.0
FROM gocv/opencv:${OPENCV_VERSION} AS builder

ENV CGO_ENABLED=1
ENV GOTOOLCHAIN=auto

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY api ./api
COPY cmd ./cmd

WORKDIR /src/cmd
RUN go build -o /vms .

FROM gocv/opencv:${OPENCV_VERSION}

RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app/cmd

COPY --from=builder /vms ./vms
COPY cmd/frontend ./frontend

RUN mkdir -p /app/recordings /app/cmd/models

# Приложение загружает ../.env относительно рабочей директории /app/cmd
RUN cat > /app/.env <<'EOF'
DB_DRIVER=postgres
DB_URL=postgres://postgres:postgres@db:5432/vms?sslmode=disable
IP_ADDR=0.0.0.0
PORT=:8091
JWT_SECRET_KEY=change-me-in-production
YOLO_MODEL=./models/yolov8n.onnx
YOLO_SCORE_THRESHOLD=0.45
YOLO_NMS_THRESHOLD=0.5
RECORDINGS_DIR=../recordings
TIMEOUT_CAPTURE=3000000
EOF

ENV IP_ADDR=0.0.0.0
ENV PORT=:8091
ENV DB_DRIVER=postgres
ENV DB_URL=postgres://postgres:postgres@db:5432/vms?sslmode=disable
ENV JWT_SECRET_KEY=change-me-in-production
ENV YOLO_MODEL=./models/yolov8n.onnx
ENV YOLO_SCORE_THRESHOLD=0.45
ENV YOLO_NMS_THRESHOLD=0.5
ENV RECORDINGS_DIR=../recordings
ENV TIMEOUT_CAPTURE=3000000

EXPOSE 8091

CMD ["./vms"]
