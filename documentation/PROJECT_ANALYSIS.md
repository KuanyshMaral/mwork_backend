# MWork Backend - Полный анализ проекта

## Оглавление
1. [Обзор проекта](#обзор-проекта)
2. [Архитектура](#архитектура)
3. [Бизнес-логика](#бизнес-логика)
4. [Модели данных](#модели-данных)
5. [API Endpoints](#api-endpoints)
6. [Интеграции](#интеграции)

---

## Обзор проекта

**MWork** - это платформа для кастингов и поиска моделей, которая соединяет работодателей (агентства, фотостудии) с моделями.

### Основные возможности:
- Регистрация и аутентификация пользователей (модели и работодатели)
- Создание и управление кастингами
- Система откликов на кастинги
- Портфолио моделей
- Система отзывов и рейтингов
- Чат и диалоги между пользователями
- Система подписок и платежей (Robokassa)
- Система уведомлений
- Поиск и матчинг моделей с кастингами
- Аналитика и статистика
- Административная панель

### Технологический стек:
- **Backend**: Go (Golang)
- **Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL
- **Authentication**: JWT (Access + Refresh tokens)
- **Storage**: Поддержка локального хранилища и S3-совместимых хранилищ
- **Payments**: Robokassa
- **WebSocket**: Для real-time чата

---

## Архитектура

### Структура проекта

\`\`\`
mwork_backend/
├── cmd/web/                    # Точка входа приложения
│   └── main.go
├── internal/
│   ├── app/                    # Инициализация приложения
│   │   └── app.go
│   ├── auth/                   # JWT токены
│   ├── config/                 # Конфигурация
│   ├── email/                  # Email провайдеры
│   ├── handlers/               # HTTP обработчики
│   ├── logger/                 # Логирование
│   ├── middleware/             # Middleware (auth, CORS, logging)
│   ├── models/                 # Модели данных (GORM)
│   ├── repositories/           # Слой доступа к данным
│   ├── routes/                 # Регистрация маршрутов
│   ├── services/               # Бизнес-логика
│   ├── storage/                # Файловое хранилище
│   └── validator/              # Валидация данных
├── ws/                         # WebSocket для чата
├── database/migrations/        # SQL миграции
└── postman/collections/        # Postman коллекция API
\`\`\`

### Архитектурные паттерны

**Clean Architecture / Layered Architecture:**

1. **Handlers Layer** (HTTP) - Обработка HTTP запросов
2. **Services Layer** - Бизнес-логика
3. **Repository Layer** - Доступ к данным
4. **Models Layer** - Структуры данных

**Dependency Injection:**
- Все зависимости инжектируются через конструкторы
- Используются интерфейсы для абстракции

**Repository Pattern:**
- Абстракция работы с базой данных
- Все операции с БД через репозитории

---

## Бизнес-логика

### 1. Система пользователей

#### Роли пользователей:
- **model** - Модели (актеры, фотомодели)
- **employer** - Работодатели (агентства, студии)
- **admin** - Администраторы платформы

#### Статусы пользователей:
- **pending** - Ожидает верификации email
- **active** - Активный пользователь
- **suspended** - Приостановлен
- **banned** - Заблокирован

#### Процесс регистрации:

\`\`\`
1. Пользователь отправляет данные регистрации
2. Система создает User с status=pending
3. Система создает профиль (ModelProfile или EmployerProfile)
4. Система назначает бесплатную подписку (Free Plan)
5. Отправляется email с токеном верификации
6. После верификации status меняется на active
\`\`\`

#### Аутентификация:

**JWT Tokens:**
- **Access Token** - короткоживущий (настраивается), для доступа к API
- **Refresh Token** - долгоживущий (7 дней), для обновления access token

**Процесс логина:**
\`\`\`
1. Проверка email/password
2. Проверка статуса пользователя
3. Генерация Access Token
4. Создание Refresh Token в БД
5. Возврат обоих токенов клиенту
\`\`\`

**Обновление токена:**
\`\`\`
1. Клиент отправляет Refresh Token
2. Проверка валидности и срока действия
3. Генерация нового Access Token
4. Ротация Refresh Token (старый удаляется, создается новый)
\`\`\`

### 2. Система кастингов

#### Статусы кастинга:
- **draft** - Черновик
- **active** - Активный (опубликован)
- **closed** - Закрыт

#### Жизненный цикл кастинга:

\`\`\`
1. Работодатель создает кастинг (status=draft)
2. Заполняет требования (возраст, рост, категории и т.д.)
3. Публикует кастинг (status=active)
4. Модели откликаются на кастинг
5. Работодатель просматривает отклики
6. Работодатель принимает/отклоняет отклики
7. Кастинг закрывается (status=closed)
\`\`\`

#### Требования к моделям в кастинге:
- Город (обязательно)
- Пол (male/female/any)
- Возраст (min/max)
- Рост (min/max)
- Вес (min/max)
- Размер одежды
- Размер обуви
- Категории (photo, video, fashion и т.д.)
- Языки
- Уровень опыта
- Тип работы (one_time/permanent)

### 3. Система откликов

#### Статусы отклика:
- **pending** - Ожидает рассмотрения
- **viewed** - Просмотрен работодателем
- **accepted** - Принят
- **rejected** - Отклонен

#### Процесс отклика:

\`\`\`
1. Модель находит подходящий кастинг
2. Отправляет отклик с сообщением
3. Система проверяет лимиты подписки
4. Создается CastingResponse (status=pending)
5. Работодатель получает уведомление
6. Работодатель просматривает отклик (status=viewed)
7. Работодатель принимает решение (accepted/rejected)
8. Модель получает уведомление о решении
\`\`\`

### 4. Система матчинга

**Алгоритм подбора моделей для кастинга:**

Система использует взвешенную оценку совпадений по критериям:

\`\`\`go
Критерии совпадения:
- Город (вес: 2.0) - обязательное совпадение
- Возраст (вес: 1.5)
- Рост (вес: 1.0)
- Вес (вес: 0.8)
- Пол (вес: 2.0)
- Категории (вес: 1.5)
- Языки (вес: 1.0)
- Опыт (вес: 1.2)

Итоговый score = сумма (совпадение * вес) / сумма весов
\`\`\`

**Процесс матчинга:**
\`\`\`
1. Система анализирует требования кастинга
2. Ищет модели, соответствующие обязательным критериям
3. Вычисляет score для каждой модели
4. Сортирует по убыванию score
5. Возвращает топ-N моделей
\`\`\`

### 5. Система подписок

#### Типы планов:
- **Free** - Бесплатный (ограниченный функционал)
- **Basic** - Базовый
- **Pro** - Профессиональный
- **Premium** - Премиум

#### Лимиты подписок:

\`\`\`json
{
  "publications": 5,      // Количество публикаций кастингов
  "responses": 10,        // Количество откликов
  "messages": 100,        // Количество сообщений
  "portfolio_photos": 20  // Количество фото в портфолио
}
\`\`\`

#### Процесс подписки:

\`\`\`
1. Пользователь выбирает план
2. Создается PaymentTransaction (status=pending)
3. Генерируется ссылка на оплату Robokassa
4. Пользователь оплачивает
5. Robokassa отправляет callback
6. Система проверяет подпись
7. Обновляется статус транзакции (status=paid)
8. Создается/обновляется UserSubscription
9. Пользователь получает уведомление
\`\`\`

### 6. Система отзывов

#### Правила создания отзыва:
- Только работодатель может оставить отзыв модели
- Отзыв привязан к конкретному кастингу
- Один работодатель может оставить только один отзыв на модель по одному кастингу
- Рейтинг: 1-5 звезд

#### Статусы отзыва:
- **pending** - На модерации
- **approved** - Одобрен
- **rejected** - Отклонен

#### Влияние на рейтинг:

\`\`\`
Рейтинг модели = Средняя оценка всех approved отзывов
\`\`\`

### 7. Система чата

#### Типы диалогов:
- **Личные диалоги** - между двумя пользователями
- **Групповые диалоги** - между несколькими участниками

#### Функции чата:
- Отправка текстовых сообщений
- Прикрепление файлов
- Реакции на сообщения (эмодзи)
- Ответы на сообщения (reply)
- Редактирование сообщений
- Удаление сообщений
- Отметка прочитанных сообщений
- Real-time обновления через WebSocket

### 8. Система уведомлений

#### Типы уведомлений:
- **NEW_RESPONSE** - Новый отклик на кастинг
- **RESPONSE_ACCEPTED** - Отклик принят
- **RESPONSE_REJECTED** - Отклик отклонен
- **NEW_MESSAGE** - Новое сообщение
- **CASTING_MATCH** - Найден подходящий кастинг
- **NEW_REVIEW** - Новый отзыв
- **SUBSCRIPTION_EXPIRING** - Подписка истекает
- **ANNOUNCEMENT** - Объявление от администрации

#### Шаблоны уведомлений:
- Администратор может создавать шаблоны для email/push уведомлений
- Поддержка переменных в шаблонах ({{.UserName}}, {{.CastingTitle}})

### 9. Система поиска

#### Типы поиска:
- **Простой поиск** - по ключевым словам
- **Расширенный поиск** - с фильтрами
- **Унифицированный поиск** - по всем сущностям

#### Поиск кастингов:
\`\`\`
Фильтры:
- Город
- Категории
- Диапазон оплаты
- Возраст
- Пол
- Дата кастинга
- Тип работы
\`\`\`

#### Поиск моделей:
\`\`\`
Фильтры:
- Город
- Возраст
- Рост
- Вес
- Пол
- Категории
- Языки
- Опыт
- Рейтинг
\`\`\`

### 10. Система аналитики

#### Метрики платформы:
- Общее количество пользователей
- Активные пользователи (DAU/WAU/MAU)
- Количество кастингов (всего/активных/закрытых)
- Количество откликов
- Конверсия откликов
- Средний рейтинг моделей
- Доход (MRR/ARR)
- Retention Rate
- Churn Rate

#### Аналитика для работодателей:
- Количество созданных кастингов
- Количество откликов на кастинги
- Средний response rate
- Эффективность кастингов

#### Аналитика для моделей:
- Количество откликов
- Процент принятых откликов
- Просмотры профиля
- Рейтинг

---

## Модели данных

### User (Пользователь)
\`\`\`go
type User struct {
    ID                string
    Name              string
    Email             string (unique)
    PasswordHash      string
    Role              UserRole (model/employer/admin)
    Status            UserStatus (pending/active/suspended/banned)
    IsVerified        bool
    VerificationToken string
    ResetToken        string
    ResetTokenExp     *time.Time
    
    // Relations
    ModelProfile    *ModelProfile
    EmployerProfile *EmployerProfile
    Subscription    *UserSubscription
    RefreshTokens   []RefreshToken
}
\`\`\`

### ModelProfile (Профиль модели)
\`\`\`go
type ModelProfile struct {
    ID             string
    UserID         string
    Name           string
    Age            int
    Height         float64
    Weight         float64
    Gender         string
    Experience     int (years)
    HourlyRate     float64
    Description    string
    ClothingSize   string
    ShoeSize       string
    City           string
    Languages      JSON (["русский", "английский"])
    Categories     JSON (["fashion", "advertising"])
    BarterAccepted bool
    ProfileViews   int
    Rating         float64
    IsPublic       bool
    
    // Relations
    PortfolioItems []PortfolioItem
    Reviews        []Review
}
\`\`\`

### EmployerProfile (Профиль работодателя)
\`\`\`go
type EmployerProfile struct {
    ID            string
    UserID        string
    CompanyName   string
    ContactPerson string
    Phone         string
    Website       string
    City          string
    CompanyType   string
    Description   string
    IsVerified    bool
    Rating        float64
}
\`\`\`

### Casting (Кастинг)
\`\`\`go
type Casting struct {
    ID              string
    EmployerID      string
    Title           string
    Description     string
    PaymentMin      float64
    PaymentMax      float64
    CastingDate     *time.Time
    CastingTime     *string
    Address         *string
    City            string
    Categories      JSON
    Gender          string
    AgeMin          *int
    AgeMax          *int
    HeightMin       *float64
    HeightMax       *float64
    WeightMin       *float64
    WeightMax       *float64
    ClothingSize    *string
    ShoeSize        *string
    ExperienceLevel *string
    Languages       JSON
    JobType         string (one_time/permanent)
    Status          CastingStatus (draft/active/closed)
    Views           int
    
    // Relations
    Employer  EmployerProfile
    Responses []CastingResponse
}
\`\`\`

### CastingResponse (Отклик на кастинг)
\`\`\`go
type CastingResponse struct {
    ID        string
    CastingID string
    ModelID   string
    Message   *string
    Status    ResponseStatus (pending/viewed/accepted/rejected)
    
    // Relations
    Model   ModelProfile
    Casting Casting
}
\`\`\`

### Review (Отзыв)
\`\`\`go
type Review struct {
    ID         string
    ModelID    string
    EmployerID string
    CastingID  *string
    Rating     int (1-5)
    ReviewText string
    Status     string (pending/approved/rejected)
    CreatedAt  time.Time
    
    // Relations
    Model    ModelProfile
    Employer EmployerProfile
    Casting  *Casting
}
\`\`\`

### PortfolioItem (Элемент портфолио)
\`\`\`go
type PortfolioItem struct {
    ID          string
    ModelID     string
    UploadID    *string
    Title       string
    Description string
    OrderIndex  int
    
    // Relations
    Upload *Upload
    Model  ModelProfile
}
\`\`\`

### UserSubscription (Подписка пользователя)
\`\`\`go
type UserSubscription struct {
    ID           string
    UserID       string
    PlanID       string
    Status       SubscriptionStatus (active/expired/cancelled)
    InvID        string (unique)
    CurrentUsage JSON ({"publications": 2, "responses": 5})
    StartDate    time.Time
    EndDate      time.Time
    AutoRenew    bool
    CancelledAt  *time.Time
    
    // Relations
    Plan SubscriptionPlan
}
\`\`\`

### SubscriptionPlan (Тарифный план)
\`\`\`go
type SubscriptionPlan struct {
    ID            string
    Name          string
    Description   string
    Slug          string (unique)
    Price         float64
    Currency      string (default: KZT)
    Duration      string (monthly/yearly)
    BillingPeriod int
    Features      JSON ({"premium_support": true})
    Limits        JSON ({"publications": 5, "responses": 10})
    IsActive      bool
    PaymentStatus string
}
\`\`\`

### Notification (Уведомление)
\`\`\`go
type Notification struct {
    ID      string
    UserID  string
    Type    string
    Title   string
    Message string
    Data    JSON ({"casting_id": "...", "model_id": "..."})
    IsRead  bool
    ReadAt  *time.Time
}
\`\`\`

### Dialog (Диалог)
\`\`\`go
type Dialog struct {
    ID           string
    Title        string
    Participants []User
    Messages     []Message
    LastMessage  *Message
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
\`\`\`

### Message (Сообщение)
\`\`\`go
type Message struct {
    ID         string
    DialogID   string
    SenderID   string
    Text       string
    ReplyToID  *string
    Attachments JSON
    Reactions   JSON
    IsEdited   bool
    IsDeleted  bool
    ReadBy     []string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
\`\`\`

---

## API Endpoints

### Структура API

**Base URL:** `http://localhost:4000/api/v1`

**Аутентификация:** Bearer Token в заголовке `Authorization`

### 1. Authentication (`/auth`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/auth/register` | Регистрация нового пользователя | ❌ |
| POST | `/auth/login` | Вход в систему | ❌ |
| POST | `/auth/refresh` | Обновление access token | ✅ |
| POST | `/auth/logout` | Выход из системы | ✅ |
| POST | `/auth/verify-email` | Верификация email | ❌ |
| POST | `/auth/password-reset` | Запрос сброса пароля | ❌ |
| POST | `/auth/reset-password` | Сброс пароля | ❌ |

**Пример регистрации модели:**
\`\`\`json
POST /auth/register
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "password123",
  "role": "model",
  "city": "Astana"
}
\`\`\`

**Пример регистрации работодателя:**
\`\`\`json
POST /auth/register
{
  "company_name": "My Company LLC",
  "email": "employer@example.com",
  "password": "password123",
  "role": "employer",
  "city": "Astana"
}
\`\`\`

**Ответ при логине:**
\`\`\`json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a1b2c3d4e5f6...",
  "user": {
    "id": "uuid",
    "email": "john@example.com",
    "role": "model",
    "status": "active",
    "is_verified": true
  }
}
\`\`\`

### 2. User Profile (`/profile`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/profile` | Получить свой профиль | ✅ |
| PUT | `/profile` | Обновить свой профиль | ✅ |
| POST | `/profile/password/change` | Изменить пароль | ✅ |

### 3. Model/Employer Profiles (`/profiles`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/profiles/model` | Создать профиль модели | ✅ |
| POST | `/profiles/employer` | Создать профиль работодателя | ✅ |
| GET | `/profiles/:userId` | Получить профиль пользователя | ❌ |
| GET | `/profiles/models/search` | Поиск моделей | ❌ |
| PUT | `/profiles/me` | Обновить свой профиль | ✅ |
| PUT | `/profiles/me/visibility` | Изменить видимость профиля | ✅ |
| GET | `/profiles/me/stats` | Получить статистику профиля | ✅ |

### 4. Castings (`/castings`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/castings` | Поиск кастингов | ❌ |
| GET | `/castings/:castingId` | Получить кастинг | ❌ |
| GET | `/castings/active` | Получить активные кастинги | ❌ |
| GET | `/castings/city/:city` | Кастинги по городу | ❌ |
| POST | `/castings` | Создать кастинг | ✅ |
| GET | `/castings/my` | Мои кастинги | ✅ |
| PUT | `/castings/:castingId` | Обновить кастинг | ✅ |
| DELETE | `/castings/:castingId` | Удалить кастинг | ✅ |
| PUT | `/castings/:castingId/status` | Изменить статус кастинга | ✅ |
| GET | `/castings/:castingId/stats` | Статистика кастинга | ✅ |
| GET | `/castings/stats/my` | Моя статистика | ✅ |
| GET | `/castings/matching` | Подходящие кастинги (для моделей) | ✅ |
| GET | `/castings/:castingId/responses` | Отклики на кастинг | ✅ |

**Пример создания кастинга:**
\`\`\`json
POST /castings
{
  "title": "Fashion Photoshoot",
  "description": "Looking for models for summer campaign",
  "min_price": 100,
  "max_price": 500,
  "date": "2025-12-01",
  "time": "10:00",
  "location": "Studio A",
  "city": "Moscow",
  "categories": ["photo"],
  "gender_required": "female",
  "min_age": 20,
  "max_age": 30,
  "min_height": 170,
  "max_height": 180,
  "clothing_size_required": "M",
  "work_type": "one-time"
}
\`\`\`

### 5. Responses (`/responses`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/responses/castings/:castingId` | Откликнуться на кастинг | ✅ |
| GET | `/responses/my` | Мои отклики | ✅ |
| DELETE | `/responses/:responseId` | Отозвать отклик | ✅ |
| GET | `/responses/castings/:castingId/list` | Отклики на кастинг | ✅ |
| PUT | `/responses/:responseId/status` | Изменить статус отклика | ✅ |
| PUT | `/responses/:responseId/viewed` | Отметить как просмотренный | ✅ |
| GET | `/responses/castings/:castingId/stats` | Статистика откликов | ✅ |
| GET | `/responses/:responseId` | Получить отклик | ✅ |

### 6. Reviews (`/reviews`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/reviews` | Создать отзыв | ✅ |
| GET | `/reviews/:reviewId` | Получить отзыв | ❌ |
| GET | `/reviews/models/:modelId` | Отзывы модели | ❌ |
| GET | `/reviews/models/:modelId/stats` | Статистика рейтинга | ❌ |
| GET | `/reviews/models/:modelId/summary` | Сводка отзывов | ❌ |
| GET | `/reviews/my` | Мои отзывы | ✅ |
| PUT | `/reviews/:reviewId` | Обновить отзыв | ✅ |
| DELETE | `/reviews/:reviewId` | Удалить отзыв | ✅ |
| GET | `/reviews/can-create` | Проверить возможность создания отзыва | ✅ |

### 7. Portfolio (`/portfolio`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/portfolio` | Добавить работу в портфолио | ✅ |
| GET | `/portfolio/:itemId` | Получить работу | ❌ |
| GET | `/portfolio/model/:modelId` | Портфолио модели | ❌ |
| GET | `/portfolio/featured` | Избранные работы | ❌ |
| GET | `/portfolio/recent` | Недавние работы | ❌ |
| PUT | `/portfolio/:itemId` | Обновить работу | ✅ |
| DELETE | `/portfolio/:itemId` | Удалить работу | ✅ |
| PUT | `/portfolio/reorder` | Изменить порядок работ | ✅ |
| PUT | `/portfolio/:itemId/visibility` | Изменить видимость | ✅ |
| GET | `/portfolio/stats/:modelId` | Статистика портфолио | ✅ |

### 8. Chat & Messages (`/dialogs`, `/messages`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/dialogs` | Создать диалог | ✅ |
| GET | `/dialogs` | Мои диалоги | ✅ |
| GET | `/dialogs/:dialogId` | Получить диалог | ✅ |
| POST | `/messages` | Отправить сообщение | ✅ |
| GET | `/dialogs/:dialogId/messages` | Сообщения диалога | ✅ |
| PUT | `/messages/:messageId` | Редактировать сообщение | ✅ |
| DELETE | `/messages/:messageId` | Удалить сообщение | ✅ |
| POST | `/messages/:messageId/reactions` | Добавить реакцию | ✅ |
| POST | `/dialogs/:dialogId/read` | Отметить как прочитанное | ✅ |
| GET | `/dialogs/:dialogId/unread-count` | Количество непрочитанных | ✅ |

**WebSocket:** `ws://localhost:4000/ws` (требует авторизации)

### 9. Subscriptions & Payments (`/plans`, `/subscriptions`, `/payments`, `/robokassa`)

#### Plans (Public)
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/plans` | Список планов | ❌ |
| GET | `/plans/:planId` | Получить план | ❌ |

#### Subscriptions (User)
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/subscriptions/my` | Моя подписка | ✅ |
| GET | `/subscriptions/my/stats` | Статистика использования | ✅ |
| POST | `/subscriptions/subscribe` | Оформить подписку | ✅ |
| PUT | `/subscriptions/cancel` | Отменить подписку | ✅ |
| PUT | `/subscriptions/renew` | Продлить подписку | ✅ |
| GET | `/subscriptions/check-limit` | Проверить лимит | ✅ |
| POST | `/subscriptions/increment-usage` | Увеличить использование | ✅ |
| PUT | `/subscriptions/reset-usage` | Сбросить использование | ✅ |

#### Payments (User)
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/payments/create` | Создать платеж | ✅ |
| GET | `/payments/history` | История платежей | ✅ |
| GET | `/payments/:paymentId/status` | Статус платежа | ✅ |

#### Robokassa Integration
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/robokassa/init` | Инициировать оплату | ✅ |
| POST | `/robokassa/callback` | Callback от Robokassa | ❌ |
| GET | `/robokassa/check/:invId` | Проверить статус | ✅ |

### 10. Uploads (`/uploads`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/uploads` | Загрузить файл | ✅ |
| POST | `/uploads/multi` | Загрузить несколько файлов | ✅ |
| GET | `/uploads/:uploadId` | Получить файл | ✅ |
| GET | `/uploads/user/me` | Мои файлы | ✅ |
| GET | `/uploads/storage/usage` | Использование хранилища | ✅ |
| DELETE | `/uploads/:uploadId` | Удалить файл | ✅ |

### 11. Notifications (`/notifications`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/notifications/my` | Мои уведомления | ✅ |
| GET | `/notifications/unread-count` | Количество непрочитанных | ✅ |
| GET | `/notifications/stats` | Статистика уведомлений | ✅ |
| POST | `/notifications` | Создать уведомление | ✅ |
| GET | `/notifications/:notificationId` | Получить уведомление | ✅ |
| PUT | `/notifications/:notificationId/read` | Отметить как прочитанное | ✅ |
| PUT | `/notifications/read-all` | Отметить все как прочитанные | ✅ |
| PUT | `/notifications/read-multiple` | Отметить несколько | ✅ |
| DELETE | `/notifications/:notificationId` | Удалить уведомление | ✅ |
| DELETE | `/notifications` | Удалить все уведомления | ✅ |

### 12. Search (`/search`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/search/castings` | Поиск кастингов | ❌ |
| POST | `/search/castings/advanced` | Расширенный поиск кастингов | ❌ |
| GET | `/search/castings/suggestions` | Подсказки для кастингов | ❌ |
| POST | `/search/models` | Поиск моделей | ❌ |
| POST | `/search/models/advanced` | Расширенный поиск моделей | ❌ |
| GET | `/search/models/suggestions` | Подсказки для моделей | ❌ |
| POST | `/search/employers` | Поиск работодателей | ❌ |
| POST | `/search/unified` | Унифицированный поиск | ❌ |
| GET | `/search/autocomplete` | Автодополнение | ❌ |
| GET | `/search/popular` | Популярные запросы | ❌ |
| GET | `/search/trends` | Тренды поиска | ❌ |
| GET | `/search/history` | История поиска | ✅ |
| DELETE | `/search/history` | Очистить историю | ✅ |

### 13. Matching (`/matching`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/matching/castings/:castingId/models` | Подходящие модели для кастинга | ✅ |
| POST | `/matching/models/search` | Поиск моделей по критериям | ✅ |
| GET | `/matching/compatibility` | Совместимость модели и кастинга | ✅ |
| GET | `/matching/models/:modelId/similar` | Похожие модели | ❌ |
| GET | `/matching/weights` | Веса критериев матчинга | ❌ |
| GET | `/matching/castings/:castingId/stats` | Статистика матчинга кастинга | ✅ |
| GET | `/matching/models/:modelId/stats` | Статистика матчинга модели | ✅ |

### 14. Analytics (`/analytics`)

#### Platform Analytics
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/analytics/platform/overview` | Обзор платформы | ✅ Admin |
| GET | `/analytics/platform/growth` | Метрики роста | ✅ Admin |
| GET | `/analytics/platform/health` | Здоровье платформы | ✅ Admin |

#### User Analytics
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/analytics/users` | Аналитика пользователей | ✅ Admin |
| GET | `/analytics/users/acquisition` | Привлечение пользователей | ✅ Admin |
| GET | `/analytics/users/retention` | Удержание пользователей | ✅ Admin |
| GET | `/analytics/users/active/count` | Активные пользователи | ✅ Admin |

#### Casting Analytics
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/analytics/castings` | Аналитика кастингов | ✅ Admin |
| GET | `/analytics/castings/:employerId/performance` | Эффективность работодателя | ✅ Admin |

#### Financial Analytics
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/analytics/financial` | Финансовая аналитика | ✅ Admin |

#### Geographic Analytics
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/analytics/geographic` | Географическая аналитика | ✅ Admin |
| GET | `/analytics/geographic/cities` | Эффективность по городам | ✅ Admin |

#### Category Analytics
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/analytics/categories` | Аналитика категорий | ✅ Admin |
| GET | `/analytics/categories/popular` | Популярные категории | ✅ Admin |

#### Performance Analytics
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/analytics/performance` | Производительность API | ✅ Admin |
| GET | `/analytics/realtime` | Метрики в реальном времени | ✅ Admin |
| GET | `/analytics/system/health` | Здоровье системы | ✅ Admin |

#### Reports
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/analytics/reports/custom` | Создать кастомный отчет | ✅ Admin |
| GET | `/analytics/reports/predefined` | Готовые отчеты | ✅ Admin |

### 15. Admin (`/admin`)

#### Admin - Users
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/admin/users` | Список пользователей | ✅ Admin |
| PUT | `/admin/users/:userId/status` | Изменить статус пользователя | ✅ Admin |
| PUT | `/admin/users/:userId/verify-employer` | Верифицировать работодателя | ✅ Admin |
| GET | `/admin/users/stats/registration` | Статистика регистраций | ✅ Admin |
| DELETE | `/admin/users/:userId` | Удалить пользователя | ✅ Admin |

#### Admin - Castings
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/admin/castings/close-expired` | Закрыть истекшие кастинги | ✅ Admin |
| GET | `/admin/castings/stats/platform` | Статистика кастингов | ✅ Admin |
| GET | `/admin/castings/stats/matching` | Статистика матчинга | ✅ Admin |
| GET | `/admin/castings/distribution/city` | Распределение по городам | ✅ Admin |
| GET | `/admin/castings/count/active` | Количество активных | ✅ Admin |
| GET | `/admin/castings/categories/popular` | Популярные категории | ✅ Admin |

#### Admin - Reviews
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/admin/reviews/stats/platform` | Статистика отзывов | ✅ Admin |
| GET | `/admin/reviews/recent` | Недавние отзывы | ✅ Admin |

#### Admin - Uploads
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/uploads/admin/stats` | Статистика загрузок | ✅ Admin |

#### Admin - Matching
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| PUT | `/admin/matching/weights` | Обновить веса матчинга | ✅ Admin |
| GET | `/admin/matching/stats/platform` | Статистика матчинга | ✅ Admin |
| POST | `/admin/matching/recalculate` | Пересчитать все матчи | ✅ Admin |
| GET | `/admin/matching/logs` | Логи матчинга | ✅ Admin |
| POST | `/admin/matching/batch` | Массовый матчинг | ✅ Admin |

#### Admin - Notifications
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/admin/notifications/templates` | Шаблоны уведомлений | ✅ Admin |
| POST | `/admin/notifications/templates` | Создать шаблон | ✅ Admin |
| GET | `/admin/notifications/templates/:templateId` | Получить шаблон | ✅ Admin |
| GET | `/admin/notifications/templates/type/:type` | Шаблон по типу | ✅ Admin |
| PUT | `/admin/notifications/templates/:templateId` | Обновить шаблон | ✅ Admin |
| DELETE | `/admin/notifications/templates/:templateId` | Удалить шаблон | ✅ Admin |
| GET | `/admin/notifications` | Все уведомления | ✅ Admin |
| GET | `/admin/notifications/stats/platform` | Статистика уведомлений | ✅ Admin |
| POST | `/admin/notifications/bulk-send` | Массовая рассылка | ✅ Admin |
| DELETE | `/admin/notifications/cleanup` | Очистить старые | ✅ Admin |

#### Admin - Subscriptions
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/admin/plans` | Создать план | ✅ Admin |
| PUT | `/admin/plans/:planId` | Обновить план | ✅ Admin |
| DELETE | `/admin/plans/:planId` | Удалить план | ✅ Admin |
| GET | `/admin/subscriptions/stats/platform` | Статистика подписок | ✅ Admin |
| GET | `/admin/subscriptions/stats/revenue` | Статистика доходов | ✅ Admin |
| GET | `/admin/subscriptions/expiring` | Истекающие подписки | ✅ Admin |
| GET | `/admin/subscriptions/expired` | Истекшие подписки | ✅ Admin |
| POST | `/admin/subscriptions/process-expired` | Обработать истекшие | ✅ Admin |

#### Admin - Search
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/admin/search/analytics` | Аналитика поиска | ✅ Admin |
| POST | `/admin/search/reindex` | Переиндексировать данные | ✅ Admin |

#### Admin - Chat
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/admin/dialogs` | Все диалоги | ✅ Admin |
| GET | `/admin/stats` | Статистика чата | ✅ Admin |
| POST | `/admin/clean` | Очистить старые сообщения | ✅ Admin |
| DELETE | `/admin/users/:userId/messages` | Удалить сообщения пользователя | ✅ Admin |

#### Admin - Analytics
| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| GET | `/analytics/admin/dashboard` | Дашборд администратора | ✅ Admin |

---

## Интеграции

### 1. Robokassa (Платежная система)

**Конфигурация:**
\`\`\`env
ROBOKASSA_MERCHANT_LOGIN=your_login
ROBOKASSA_PASSWORD_1=your_password_1
ROBOKASSA_PASSWORD_2=your_password_2
ROBOKASSA_TEST_MODE=true
\`\`\`

**Процесс оплаты:**
\`\`\`
1. Клиент: POST /robokassa/init {plan_id}
2. Сервер: Создает PaymentTransaction
3. Сервер: Генерирует подпись (MD5)
4. Сервер: Возвращает URL для редиректа
5. Клиент: Редирект на Robokassa
6. Пользователь: Оплачивает
7. Robokassa: POST /robokassa/callback
8. Сервер: Проверяет подпись
9. Сервер: Обновляет транзакцию и подписку
10. Сервер: Отправляет уведомление пользователю
\`\`\`

**Формула подписи:**
\`\`\`
MD5(MerchantLogin:OutSum:InvId:Password1)
\`\`\`

### 2. Email Provider

**Поддерживаемые провайдеры:**
- Mock (для разработки)
- SMTP (настраивается)

**Типы email:**
- Верификация email
- Сброс пароля
- Уведомления о событиях

**Шаблоны:**
- Поддержка переменных: `{{.UserName}}`, `{{.ResetURL}}`
- HTML и текстовые версии

### 3. Storage (Файловое хранилище)

**Поддерживаемые типы:**
- **Local** - локальное хранилище
- **S3** - AWS S3 или совместимые (MinIO, DigitalOcean Spaces)

**Конфигурация:**
\`\`\`env
STORAGE_TYPE=local
STORAGE_BASE_PATH=./uploads
STORAGE_BASE_URL=http://localhost:4000/uploads

# Для S3:
STORAGE_TYPE=s3
STORAGE_BUCKET=mwork-uploads
STORAGE_REGION=us-east-1
STORAGE_ACCESS_KEY=your_access_key
STORAGE_SECRET_KEY=your_secret_key
STORAGE_ENDPOINT=https://s3.amazonaws.com
STORAGE_USE_SSL=true
STORAGE_PUBLIC_READ=true
\`\`\`

**Модули загрузки:**
- `profile` - аватары и фото профилей
- `portfolio` - работы портфолио
- `casting` - изображения кастингов
- `chat` - вложения в сообщениях

**Ограничения:**
- Максимальный размер файла: настраивается
- Поддерживаемые форматы: изображения (jpg, png, gif), видео (mp4), документы (pdf)

### 4. WebSocket (Real-time чат)

**Подключение:**
\`\`\`javascript
const ws = new WebSocket('ws://localhost:4000/ws');
ws.onopen = () => {
  // Отправить токен авторизации
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'your_access_token'
  }));
};
\`\`\`

**Типы сообщений:**
- `new_message` - новое сообщение
- `message_edited` - сообщение отредактировано
- `message_deleted` - сообщение удалено
- `message_read` - сообщение прочитано
- `typing` - пользователь печатает
- `online` - пользователь онлайн
- `offline` - пользователь оффлайн

---

## Заключение

Это полная документация проекта MWork Backend. Для более детальной информации о конкретных эндпоинтах и их параметрах, обратитесь к Postman коллекции или к документации по API маршрутам.
