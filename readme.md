# Проект по курсу Язык программирования Go (Bank Service API) НИЯУ МИФИ

Учебный REST API проекит банковского сервиса.  

## Описание проекта

Проект реализует регистрацию пользователей, JWT-аутентификацию, управление счетами, переводы, банковские карты, кредиты, график платежей, SMTP-уведомления, интеграцию с ЦБ РФ и финансовую аналитику.

## Стек технологий

- Go 1.23+
- PostgreSQL 17
- gorilla/mux
- lib/pq
- golang-jwt/jwt/v5
- bcrypt
- pgcrypto
- HMAC-SHA256
- logrus
- gomail.v2
- beevik/etree

## Возможности API

Реализовано:

- регистрация пользователей;
- аутентификация через JWT;
- создание банковских счетов;
- просмотр счетов пользователя;
- пополнение счета;
- списание со счета;
- переводы между счетами;
- история транзакций;
- выпуск виртуальных карт;
- генерация номера карты по алгоритму Луна;
- шифрование номера и срока карты через PostgreSQL pgcrypto;
- хеширование CVV через bcrypt;
- HMAC для проверки целостности номера карты;
- оплата картой;
- оформление кредита;
- расчет аннуитетных платежей;
- генерация графика платежей;
- автоматическое списание кредитных платежей через шедулер;
- начисление штрафа 10% при просрочке;
- получение ключевой ставки ЦБ РФ через SOAP;
- SMTP-уведомления;
- аналитика доходов и расходов;
- прогноз баланса на N дней.

## Архитектура проекта

Проект построен по слоистой архитектуре:

```text
handler -> service -> repository -> PostgreSQL
```

Структура проекта:

```text
bank-service/
├── cmd/
│   └── app/
│       └── main.go
├── internal/
│   ├── apperror/
│   ├── config/
│   ├── db/
│   ├── dto/
│   ├── handler/
│   ├── integration/
│   │   ├── cbr/
│   │   └── smtp/
│   ├── middleware/
│   ├── models/
│   ├── repository/
│   ├── response/
│   ├── router/
│   ├── scheduler/
│   ├── security/
│   └── service/
├── migrations/
│   └── schema.sql
├── .env.example
├── .gitignore
├── go.mod
└── README.md
```

## Переменные окружения

Создайте файл `.env` в корне проекта.

Пример:

```env
SERVER_PORT=8080

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=bank_service
DB_SSLMODE=disable

JWT_SECRET=change_me
JWT_TTL_HOURS=24

HMAC_SECRET=change_me
PGP_SECRET=change_me

CBR_URL=https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx
CBR_RATE_MARGIN=5
CBR_LOOKBACK_DAYS=365

SMTP_ENABLED=false
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASS=change_me
SMTP_FROM=noreply@example.com

CREDIT_PAYMENT_SCHEDULER_ENABLED=true
CREDIT_PAYMENT_SCHEDULER_INTERVAL_HOURS=12
```

Важно: реальные секреты, пароли и `.env` нельзя коммитить в Git.

## Установка зависимостей

```bash
go mod tidy
```

Основные зависимости:

```bash
go get github.com/gorilla/mux
go get github.com/lib/pq
go get github.com/golang-jwt/jwt/v5
go get github.com/sirupsen/logrus
go get github.com/joho/godotenv
go get golang.org/x/crypto/bcrypt
go get github.com/beevik/etree
go get github.com/go-mail/mail/v2
```

## Настройка PostgreSQL

Создайте базу данных:

```bash
createdb bank_service
```

Или через `psql`:

```sql
CREATE DATABASE bank_service;
```

При запуске приложения схема из файла `migrations/schema.sql` применяется автоматически.

## Запуск проекта

```bash
go run ./cmd/app
```

После успешного запуска в логах должно быть:

```text
connected to PostgreSQL
database schema initialized
server started on port 8080
```

## Проверка health-check

```bash
curl -X GET http://localhost:8080/health
```

Пример ответа:

```json
{
  "status": "ok",
  "timestamp": "2026-05-18T12:00:00Z"
}
```

## API endpoints

### Публичные маршруты

| Метод | Endpoint | Описание |
|---|---|---|
| GET | `/health` | Проверка работоспособности |
| POST | `/register` | Регистрация |
| POST | `/login` | Авторизация |

### Защищенные маршруты

Требуют заголовок:

```http
Authorization: Bearer <access_token>
```

| Метод | Endpoint | Описание |
|---|---|---|
| POST | `/accounts` | Создать счет |
| GET | `/accounts` | Получить счета пользователя |
| POST | `/accounts/{accountId}/deposit` | Пополнить счет |
| POST | `/accounts/{accountId}/withdraw` | Списать со счета |
| POST | `/transfer` | Перевод между счетами |
| GET | `/transactions` | Все транзакции пользователя |
| GET | `/accounts/{accountId}/transactions` | Транзакции счета |
| POST | `/cards` | Выпустить карту |
| GET | `/cards` | Получить карты пользователя |
| GET | `/cards/{cardId}` | Получить карту |
| POST | `/cards/{cardId}/pay` | Оплата картой |
| POST | `/credits` | Оформить кредит |
| GET | `/credits` | Получить кредиты пользователя |
| GET | `/credits/{creditId}` | Получить кредит |
| GET | `/credits/{creditId}/schedule` | Получить график платежей |
| GET | `/analytics?month=YYYY-MM` | Аналитика за месяц |
| GET | `/accounts/{accountId}/predict?days=N` | Прогноз баланса |

## Примеры запросов

### Регистрация

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "kirill@example.com",
    "username": "kirill",
    "password": "password123"
  }'
```

### Логин

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "kirill@example.com",
    "password": "password123"
  }'
```

Пример ответа:

```json
{
  "access_token": "jwt_token",
  "token_type": "Bearer",
  "expires_at": "2026-05-19T12:00:00Z"
}
```

### Создание счета

```bash
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ТВОЙ_TOKEN" \
  -d '{}'
```

### Пополнение счета

```bash
curl -X POST http://localhost:8080/accounts/1/deposit \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ТВОЙ_TOKEN" \
  -d '{
    "amount": 10000
  }'
```

### Списание со счета

```bash
curl -X POST http://localhost:8080/accounts/1/withdraw \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ТВОЙ_TOKEN" \
  -d '{
    "amount": 1000
  }'
```

### Перевод между счетами

```bash
curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ТВОЙ_TOKEN" \
  -d '{
    "from_account_id": 1,
    "to_account_id": 2,
    "amount": 2500
  }'
```

### Выпуск карты

```bash
curl -X POST http://localhost:8080/cards \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ТВОЙ_TOKEN" \
  -d '{
    "account_id": 1
  }'
```

При создании карты CVV возвращается только один раз.

### Оплата картой

```bash
curl -X POST http://localhost:8080/cards/1/pay \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ТВОЙ_TOKEN" \
  -d '{
    "amount": 1500,
    "description": "Online payment"
  }'
```

### Оформление кредита

```bash
curl -X POST http://localhost:8080/credits \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ТВОЙ_TOKEN" \
  -d '{
    "account_id": 1,
    "amount": 100000,
    "term_months": 12
  }'
```

### График платежей

```bash
curl -X GET http://localhost:8080/credits/1/schedule \
  -H "Authorization: Bearer ТВОЙ_TOKEN"
```

### Аналитика

```bash
curl -X GET "http://localhost:8080/analytics?month=2026-05" \
  -H "Authorization: Bearer ТВОЙ_TOKEN"
```

Пример ответа:

```json
{
  "month": "2026-05",
  "income": 110000,
  "expenses": 9239.54,
  "net": 100760.46,
  "transactions_count": 2,
  "deposits": 0,
  "withdrawals": 0,
  "transfers_in": 0,
  "transfers_out": 0,
  "card_payments": 0,
  "credit_issues": 110000,
  "credit_payments": 9239.54,
  "penalties": 0,
  "credit_load": 10163.49
}
```

### Прогноз баланса

```bash
curl -X GET "http://localhost:8080/accounts/1/predict?days=30" \
  -H "Authorization: Bearer ТВОЙ_TOKEN"
```

Пример ответа:

```json
{
  "account_id": 1,
  "days": 30,
  "current_balance": 90760.46,
  "planned_payments": 9239.54,
  "predicted_balance": 81520.92,
  "currency": "RUB"
}
```

## Безопасность

В проекте реализованы следующие меры безопасности:

- пароли пользователей хешируются через bcrypt;
- CVV карты хранится только в виде bcrypt-хеша;
- номер карты и срок действия шифруются через `pgcrypto`;
- HMAC-SHA256 используется для проверки целостности номера карты;
- JWT используется для аутентификации пользователей;
- защищенные маршруты доступны только с валидным токеном;
- проверяется принадлежность счетов, карт и кредитов текущему пользователю;
- SQL-запросы параметризованы;
- денежные операции выполняются в транзакциях БД;
- полный номер карты не возвращается в списке карт;
- CVV возвращается только при создании карты.

## Интеграция с ЦБ РФ

При оформлении кредита приложение получает ключевую ставку ЦБ РФ через SOAP API:

```text
https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx
```

Итоговая ставка кредита рассчитывается так:

```text
ставка кредита = ключевая ставка ЦБ + маржа банка
```

Маржа задается через переменную:

```env
CBR_RATE_MARGIN=5
```

## SMTP-уведомления

Приложение может отправлять email-уведомления после:

- оплаты картой;
- оформления кредита.

Для разработки SMTP можно отключить:

```env
SMTP_ENABLED=false
```

Для реальной отправки нужно указать SMTP-настройки:

```env
SMTP_ENABLED=true
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASS=app_password
SMTP_FROM=noreply@example.com
```

## Шедулер кредитных платежей

Шедулер запускается вместе с приложением.

Он:

- ищет платежи со статусом `PLANNED` или `OVERDUE`;
- проверяет дату платежа;
- если денег достаточно — списывает платеж и ставит статус `PAID`;
- если денег недостаточно — начисляет штраф 10% и ставит статус `OVERDUE`;
- создает транзакции `CREDIT_PAYMENT` и `PENALTY`;
- закрывает кредит, если все платежи оплачены.

Настройки:

```env
CREDIT_PAYMENT_SCHEDULER_ENABLED=true
CREDIT_PAYMENT_SCHEDULER_INTERVAL_HOURS=12
```

## Таблицы базы данных

Основные таблицы:

```text
users
accounts
cards
transactions
credits
payment_schedules
```

## Типы транзакций

```text
DEPOSIT
WITHDRAW
TRANSFER
CARD_PAYMENT
CREDIT_ISSUE
CREDIT_PAYMENT
PENALTY
```

## Статусы транзакций

```text
SUCCESS
FAILED
PENDING
```

## Статусы кредита

```text
ACTIVE
CLOSED
OVERDUE
```

## Статусы платежей

```text
PLANNED
PAID
OVERDUE
```

## Ограничения

- поддерживается только валюта RUB;
- максимальный период прогноза баланса — 365 дней;
- JWT действует 24 часа;
- CVV не возвращается повторно после создания карты;
- пользователь не может управлять чужими счетами, картами и кредитами.

## Формат ошибок

Все ошибки возвращаются в едином JSON-формате:

```json
{
  "error": "message"
}
```

Примеры:

```json
{
  "error": "authorization header is required"
}
```

```json
{
  "error": "insufficient funds"
}
```

```json
{
  "error": "access denied to account"
}
```

## Проверка проекта

Основной сценарий проверки:

1. Зарегистрировать пользователя.
2. Выполнить логин.
3. Получить JWT.
4. Создать два счета.
5. Пополнить первый счет.
6. Выполнить списание.
7. Выполнить перевод между счетами.
8. Проверить историю транзакций.
9. Выпустить карту.
10. Оплатить картой.
11. Оформить кредит.
12. Проверить график платежей.
13. Проверить аналитику.
14. Проверить прогноз баланса.
15. Проверить защиту от доступа к чужим ресурсам.

