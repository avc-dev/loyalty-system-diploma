# Gophermart - Накопительная система лояльности

Система лояльности "Гофермарт" позволяет пользователям регистрироваться, загружать номера заказов для расчета баллов лояльности и использовать накопленные баллы для оплаты новых заказов.

## Архитектура

Проект построен на принципах **Clean Architecture** с четким разделением на слои:

- **Domain Layer** - доменные модели, ошибки и интерфейсы
- **Repository Layer** - работа с PostgreSQL через pgx/v5
- **Service Layer** - бизнес-логика
- **Handler Layer** - HTTP handlers с chi router
- **Worker Pool** - фоновая обработка начислений

## Технологии

- **Go 1.25+**
- **PostgreSQL 15** - основная БД
- **pgx/v5** - драйвер для PostgreSQL
- **chi/v5** - HTTP роутер
- **zap** - структурированное логирование
- **JWT** - аутентификация
- **bcrypt** - хеширование паролей
- **golang-migrate** - миграции БД
- **mockery** - генерация моков для тестирования

## Установка и запуск

### Предварительные требования

- Go 1.25 или выше
- PostgreSQL 15
- Docker и Docker Compose (опционально)

### Запуск с Docker Compose

```bash
# Запуск PostgreSQL
docker-compose up -d

# Применение миграций
migrate -path migrations -database "postgres://gophermart_user:gophermart_pass@localhost:5432/gophermart?sslmode=disable" up

# Сборка и запуск приложения
go build -o gophermart cmd/gophermart/main.go
./gophermart -a :8080 -d "postgres://gophermart_user:gophermart_pass@localhost:5432/gophermart?sslmode=disable" -r "http://localhost:8081"
```

### Конфигурация

Конфигурация поддерживается через переменные окружения (приоритет) или флаги командной строки:

| Параметр | Env переменная | Флаг | Описание | По умолчанию |
|----------|---------------|------|----------|--------------|
| Адрес сервера | `RUN_ADDRESS` | `-a` | Адрес и порт запуска | `:8080` |
| URI БД | `DATABASE_URI` | `-d` | Строка подключения к PostgreSQL | - |
| Адрес accrual | `ACCRUAL_SYSTEM_ADDRESS` | `-r` | Адрес системы начислений | - |
| JWT Secret | `JWT_SECRET` | - | Секретный ключ для JWT | `default-secret...` |
| Уровень логов | `LOG_LEVEL` | - | Уровень логирования | `info` |

**Пример:**

```bash
export DATABASE_URI="postgres://user:pass@localhost:5432/gophermart?sslmode=disable"
export ACCRUAL_SYSTEM_ADDRESS="http://localhost:8081"
export RUN_ADDRESS=":8080"
export JWT_SECRET="my-super-secret-key"

./gophermart
```

## API Endpoints

### Аутентификация

#### POST /api/user/register
Регистрация нового пользователя

**Request:**
```json
{
  "login": "user123",
  "password": "password123"
}
```

**Response:** `200 OK`
- Header: `Authorization: Bearer <jwt_token>`

**Ошибки:**
- `400` - неверный формат запроса
- `409` - логин уже занят
- `500` - внутренняя ошибка сервера

#### POST /api/user/login
Аутентификация пользователя

**Request/Response:** аналогично регистрации

**Ошибки:**
- `400` - неверный формат запроса
- `401` - неверная пара логин/пароль
- `500` - внутренняя ошибка сервера

### Заказы

#### POST /api/user/orders
Загрузка номера заказа (требуется аутентификация)

**Request:**
```
Content-Type: text/plain

79927398713
```

**Response:**
- `200` - номер заказа уже был загружен этим пользователем
- `202` - новый номер заказа принят в обработку
- `400` - неверный формат запроса
- `401` - пользователь не аутентифицирован
- `409` - номер заказа уже был загружен другим пользователем
- `422` - неверный формат номера заказа (не прошел алгоритм Луна)
- `500` - внутренняя ошибка сервера

#### GET /api/user/orders
Получение списка загруженных заказов (требуется аутентификация)

**Response:** `200 OK`
```json
[
  {
    "number": "9278923470",
    "status": "PROCESSED",
    "accrual": 500,
    "uploaded_at": "2020-12-10T15:15:45+03:00"
  },
  {
    "number": "12345678903",
    "status": "PROCESSING",
    "uploaded_at": "2020-12-10T15:12:01+03:00"
  }
]
```

**Статусы:**
- `NEW` - заказ загружен, но не обработан
- `PROCESSING` - идет расчет вознаграждения
- `INVALID` - система отказала в расчете
- `PROCESSED` - расчет завершен

**Ошибки:**
- `204` - нет данных для ответа
- `401` - пользователь не авторизован
- `500` - внутренняя ошибка сервера

### Баланс

#### GET /api/user/balance
Получение текущего баланса (требуется аутентификация)

**Response:** `200 OK`
```json
{
  "current": 500.5,
  "withdrawn": 42
}
```

#### POST /api/user/balance/withdraw
Списание баллов (требуется аутентификация)

**Request:**
```json
{
  "order": "2377225624",
  "sum": 751
}
```

**Response:**
- `200` - успешная обработка запроса
- `401` - пользователь не авторизован
- `402` - недостаточно средств
- `422` - неверный номер заказа
- `500` - внутренняя ошибка сервера

#### GET /api/user/withdrawals
История списаний (требуется аутентификация)

**Response:** `200 OK`
```json
[
  {
    "order": "2377225624",
    "sum": 500,
    "processed_at": "2020-12-09T16:09:57+03:00"
  }
]
```

## Разработка

### Makefile команды

Проект включает Makefile с полезными командами для разработки:

```bash
make help              # Показать все доступные команды
make build             # Собрать приложение
make test              # Запустить все тесты
make test-coverage     # Запустить тесты с покрытием
make fmt               # Форматировать код
make vet               # Проверить код с go vet
make generate-mocks    # Сгенерировать все моки
make clean-mocks       # Удалить сгенерированные моки
make docker-up         # Запустить Docker сервисы
make docker-down       # Остановить Docker сервисы
make dev               # Полная настройка для разработки
```

### Запуск тестов

```bash
# Все тесты
go test ./...

# С покрытием
go test -cover ./...

# Детальное покрытие
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Unit тесты только (без интеграционных)
go test -short ./...

# Использование Makefile
make test
make test-coverage
make test-unit
```

### Тестирование с Mocks

Проект использует **mockery** для автоматической генерации моков интерфейсов:

```bash
# Установка mockery (один раз)
make install-mockery

# Генерация всех моков
make generate-mocks

# Очистка сгенерированных моков
make clean-mocks
```

#### Архитектура тестирования

- **Repository слой**: Тестируется с pgxmock для имитации PostgreSQL
- **Service слой**: Тестируется с generated mocks для зависимостей
- **Handler слой**: Интеграционные тесты с HTTP и generated mocks
- **Utils**: Unit тесты с table-driven подходом

#### Использование моков в тестах

```go
func TestMyService_DoSomething(t *testing.T) {
    // Создание мока с автоматической проверкой ожиданий
    mockRepo := mocks.NewMyRepositoryMock(t)

    // Настройка ожиданий
    mockRepo.EXPECT().GetData(ctx, "param").Return(data, nil).Once()

    service := NewMyService(mockRepo)
    result, err := service.DoSomething(ctx, "param")

    assert.NoError(t, err)
    assert.Equal(t, expected, result)
    // AssertExpectations вызывается автоматически при завершении теста
}
```

#### Структура моков

```
internal/mocks/
├── domain/
│   ├── UserRepository_mock.go
│   ├── OrderRepository_mock.go
│   ├── TransactionRepository_mock.go
│   ├── AuthService_mock.go
│   ├── OrderService_mock.go
│   ├── BalanceService_mock.go
│   └── AccrualClient_mock.go
└── utils/password/
    └── Hasher_mock.go
```

Mоки генерируются автоматически из интерфейсов в `internal/domain/interfaces.go` и автоматически синхронизируются при изменениях интерфейсов.

### Структура проекта

```
.
├── cmd/
│   └── gophermart/
│       └── main.go              # Точка входа
├── internal/
│   ├── app/
│   │   └── app.go               # Инициализация приложения
│   ├── config/
│   │   └── config.go            # Конфигурация
│   ├── domain/
│   │   ├── models.go            # Доменные модели
│   │   ├── errors.go            # Доменные ошибки
│   │   └── interfaces.go        # Интерфейсы
│   ├── handlers/
│   │   ├── auth.go              # Аутентификация
│   │   ├── orders.go            # Заказы
│   │   ├── balance.go           # Баланс
│   │   └── middleware.go        # Middleware
│   ├── mocks/                   # Автогенерированные моки
│   │   ├── domain/              # Моки доменных интерфейсов
│   │   └── utils/               # Моки утилитарных интерфейсов
│   ├── service/
│   │   ├── auth.go              # Сервис аутентификации
│   │   ├── orders.go            # Сервис заказов
│   │   ├── balance.go           # Сервис баланса
│   │   └── accrual_client.go    # Клиент accrual системы
│   ├── repository/
│   │   └── postgres/
│   │       ├── user.go          # Репозиторий пользователей
│   │       ├── order.go         # Репозиторий заказов
│   │       └── transaction.go   # Репозиторий транзакций
│   ├── worker/
│   │   └── pool.go              # Worker pool
│   └── utils/
│       ├── luhn/                # Алгоритм Луна
│       ├── jwt/                 # JWT утилиты
│       └── password/            # Хеширование паролей
├── migrations/                  # SQL миграции
├── docker-compose.yml           # Docker Compose конфигурация
├── Makefile                     # Команды для разработки
├── .mockery.yaml                # Конфигурация mockery
└── README.md
```

## Особенности реализации

### Безопасность многопоточности

- Списание средств реализовано через транзакции с `SELECT FOR UPDATE` для предотвращения race conditions
- Worker pool использует каналы для безопасной передачи данных между горутинами
- Graceful shutdown корректно завершает все горутины

### Обработка ошибок

Все ошибки оборачиваются с контекстом для удобной отладки:
```go
fmt.Errorf("service layer: operation failed: %w", err)
```

Sentinel errors не оборачиваются и возвращаются как есть для проверки через `errors.Is()`.

### Логирование

Используется структурированное логирование с zap:
- Request ID для трассировки запросов
- Логирование всех ошибок с полным контекстом
- Метрики производительности (время выполнения запросов)

### Worker Pool

- Фоновая обработка заказов с автоматическим опросом системы начислений
- Обработка rate limiting (429) с exponential backoff
- Graceful shutdown с корректным завершением всех задач

## Лицензия

MIT
