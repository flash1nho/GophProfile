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

## 📄 License

MIT
