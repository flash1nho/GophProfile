# 🧑‍💻 GophProfile

**GophProfile** — это микросервис на Go для управления аватарами пользователей.
Он предоставляет удобный **REST API** и простой **web-интерфейс** для загрузки, хранения и получения изображений профиля в различных форматах.

---

## ✨ Возможности

* 📤 Загрузка аватаров пользователей
* 🖼 Конвертация изображений (webp, png и др.)
* 📊 Получение метаданных аватаров
* 👤 Привязка аватаров к пользователям
* 📚 Получение списка аватаров пользователя
* 🌐 Web-интерфейс (загрузка + галерея)
* 🐳 Запуск через Docker

---

## ⚡ Быстрый старт

### Через Docker (рекомендуется)

```bash
docker-compose -f docker/docker-compose.yml up --build
```

После запуска сервис будет доступен по адресу:

👉 [http://localhost:8080](http://localhost:8080)

---

## 📡 API

### 📥 Получить аватар

```http
GET /api/v1/avatars/{avatar_id}
```

**Query параметры:**

| Параметр | Описание                                     |
| -------- | -------------------------------------------- |
| format   | Формат изображения (например: `webp`, `png`) |

---

### 📊 Получить метаданные аватара

```http
GET /api/v1/avatars/{avatar_id}/metadata
```

---

### 👤 Получить текущий аватар пользователя

```http
GET /api/v1/users/{user_id}/avatar
```

---

### 📚 Получить список аватаров пользователя

```http
GET /api/v1/users/{user_id}/avatars
```

---

## 🔧 Примеры использования

### Скачать аватар

```bash
curl "http://localhost:8080/api/v1/avatars/{avatar_id}?format=webp" \
  --output avatar.webp
```

---

### Получить метаданные

```bash
curl http://localhost:8080/api/v1/avatars/{avatar_id}/metadata
```

---

### Получить аватар пользователя

```bash
curl http://localhost:8080/api/v1/users/{user_id}/avatar \
  --output avatar.png
```

---

### Получить список аватаров

```bash
curl http://localhost:8080/api/v1/users/{user_id}/avatars
```

---

## 🌐 Web-интерфейс

| Функция  | URL                                                                                        |
| -------- | ------------------------------------------------------------------------------------------ |
| Загрузка | [http://localhost:8080/web/upload](http://localhost:8080/web/upload)                       |
| Галерея  | [http://localhost:8080/web/gallery/{user_id}](http://localhost:8080/web/gallery/{user_id}) |

---

## 🏗 Архитектура проекта

```
.
├── cmd/
│   ├── server/
│   └── worker/
├── internal/
│   ├── api/
│   ├── config/
│   ├── domain/
│   ├── dto/
│   ├── handlers/
│   ├── repository/
│   ├── services/
│   └── worker/
├── pkg/
├── web/
├── migrations/
├── docker/
```

---

## 🧪 Тестирование

```bash
go test ./...
```

---

## Kubernetes Deployment

### Prerequisites

Перед запуском убедитесь, что установлены:

* Docker
* Kubernetes cluster
* kubectl
* Helm 3+
* metrics-server
* NGINX Ingress Controller
* Prometheus Operator (для ServiceMonitor)

Проверка:

```bash
kubectl version --client
helm version
```

---

### Create Namespace

```bash
kubectl create namespace gophprofile
```

---

### Create Secrets

Создание Kubernetes secret:

```bash
kubectl create secret generic gophprofile \
  --namespace gophprofile \
  --from-literal=POSTGRES_DSN="postgres://user:password@postgres:5432/gophprofile?sslmode=disable" \
  --from-literal=JWT_SECRET="super-secret-key" \
  --from-literal=MINIO_ACCESS_KEY="minio" \
  --from-literal=MINIO_SECRET_KEY="minio123" \
  --from-literal=RABBITMQ_URL="amqp://guest:guest@rabbitmq:5672/"
```

Проверка:

```bash
kubectl get secrets -n gophprofile
```

---

## Deploy Application with Helm

### Development Environment

```bash
helm upgrade --install gophprofile ./helm/gophprofile \
  -n gophprofile \
  -f ./helm/gophprofile/values-dev.yaml
```

### Production Environment

```bash
helm upgrade --install gophprofile ./helm/gophprofile \
  -n gophprofile \
  -f ./helm/gophprofile/values-prod.yaml
```

---

## Verify Deployment

Проверка pod'ов:

```bash
kubectl get pods -n gophprofile
```

Проверка сервисов:

```bash
kubectl get svc -n gophprofile
```

Проверка ingress:

```bash
kubectl get ingress -n gophprofile
```

Проверка HPA:

```bash
kubectl get hpa -n gophprofile
```

Проверка ServiceMonitor:

```bash
kubectl get servicemonitor -n gophprofile
```

---

## Health Checks

### Liveness

```bash
curl http://localhost:8080/health/live
```

### Readiness

```bash
curl http://localhost:8080/health/ready
```

---

## Metrics

Prometheus metrics endpoint:

```bash
curl http://localhost:8080/metrics
```

---

## Port Forward

Для локального доступа:

```bash
kubectl port-forward svc/gophprofile 8080:80 -n gophprofile
```

После этого приложение будет доступно:

```text
http://localhost:8080
```

Swagger UI:

```text
http://localhost:8080/swagger/index.html
```

---

## Autoscaling

Проверка autoscaling:

```bash
kubectl describe hpa gophprofile -n gophprofile
```

Проверка metrics-server:

```bash
kubectl top pods -n gophprofile
```

---

## Monitoring

### Prometheus

Проверить targets:

```bash
kubectl port-forward svc/prometheus-operated 9090:9090 -n monitoring
```

Открыть:

```text
http://localhost:9090
```

### Grafana

```bash
kubectl port-forward svc/grafana 3000:80 -n monitoring
```

Открыть:

```text
http://localhost:3000
```

---

## Graceful Shutdown

Приложение поддерживает graceful shutdown:

* readiness probe переводится в failed state;
* Kubernetes перестаёт отправлять трафик;
* активные запросы завершаются корректно;
* background workers завершаются безопасно.

Проверка:

```bash
kubectl delete pod <pod-name> -n gophprofile
```

---

## Network Policies

Проверка network policies:

```bash
kubectl get networkpolicy -n gophprofile
```

---

## RBAC

Проверка service account:

```bash
kubectl get sa -n gophprofile
```

Проверка roles:

```bash
kubectl get roles -n gophprofile
kubectl get rolebindings -n gophprofile
```

---

## Architecture

### Kubernetes Architecture

```mermaid
flowchart LR

Client --> Ingress
Ingress --> Service
Service --> API

API --> PostgreSQL
API --> RabbitMQ
API --> MinIO

Prometheus --> API
Grafana --> Prometheus

Worker --> RabbitMQ
```

---

## Uninstall

Удаление приложения:

```bash
helm uninstall gophprofile -n gophprofile
```

Удаление namespace:

```bash
kubectl delete namespace gophprofile
```


---

## 📄 License

MIT
