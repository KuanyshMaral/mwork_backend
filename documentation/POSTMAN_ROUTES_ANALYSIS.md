# MWork Backend - Анализ Postman коллекции и связей между маршрутами

## Оглавление
1. [Структура коллекции](#структура-коллекции)
2. [Группы маршрутов](#группы-маршрутов)
3. [Связи между маршрутами](#связи-между-маршрутами)
4. [Типичные сценарии использования](#типичные-сценарии-использования)
5. [Зависимости и последовательности](#зависимости-и-последовательности)

---

## Структура коллекции

Postman коллекция содержит **15 основных групп** маршрутов:

1. **Authentication** - Аутентификация и регистрация
2. **User Profile** - Управление профилем пользователя
3. **Model Profiles** - Профили моделей и работодателей
4. **Castings** - Управление кастингами
5. **Responses** - Отклики на кастинги
6. **Reviews** - Отзывы и рейтинги
7. **Portfolio** - Портфолио моделей
8. **Chat & Messages** - Чат и сообщения
9. **Subscriptions & Payments** - Подписки и платежи
10. **Uploads** - Загрузка файлов
11. **Notifications** - Уведомления
12. **Search** - Поиск
13. **Matching** - Матчинг моделей и кастингов
14. **Analytics** - Аналитика
15. **Admin** - Административные функции

---

## Группы маршрутов

### 1. Authentication (8 маршрутов)

**Цель:** Управление аутентификацией пользователей

**Маршруты:**
- `POST /auth/register` - Регистрация (модель или работодатель)
- `POST /auth/login` - Вход в систему
- `POST /auth/refresh` - Обновление access token
- `POST /auth/logout` - Выход из системы
- `POST /auth/verify-email` - Верификация email
- `POST /auth/password-reset` - Запрос сброса пароля
- `POST /auth/reset-password` - Сброс пароля
- `POST /auth/password/change` - Изменение пароля

**Связи:**
- Регистрация → Верификация email → Логин
- Логин → Получение токенов → Доступ к защищенным маршрутам
- Refresh token → Обновление access token

**Переменные:**
- `{{access_token}}` - используется во всех защищенных маршрутах
- `{{user_id}}` - ID текущего пользователя

---

### 2. User Profile (3 маршрута)

**Цель:** Управление основным профилем пользователя

**Маршруты:**
- `GET /profile` - Получить свой профиль
- `PUT /profile` - Обновить свой профиль
- `POST /profile/password/change` - Изменить пароль

**Связи:**
- Требует аутентификации (access_token)
- Связан с Model/Employer Profiles

---

### 3. Model Profiles (8 маршрутов)

**Цель:** Управление профилями моделей и работодателей

**Маршруты:**
- `POST /profiles/model` - Создать профиль модели
- `POST /profiles/employer` - Создать профиль работодателя
- `GET /profiles/:userId` - Получить профиль пользователя
- `GET /profiles/models/search` - Поиск моделей
- `PUT /profiles/me` - Обновить свой профиль
- `PUT /profiles/me/visibility` - Изменить видимость профиля
- `GET /profiles/me/stats` - Получить статистику профиля

**Связи:**
- Создается автоматически при регистрации
- Используется в кастингах, откликах, отзывах
- Связан с Portfolio, Reviews

---

### 4. Castings (14 маршрутов)

**Цель:** Управление кастингами

**Маршруты:**
- `GET /castings` - Поиск кастингов (с фильтрами)
- `GET /castings/:castingId` - Получить кастинг
- `GET /castings/active` - Активные кастинги
- `GET /castings/city/:city` - Кастинги по городу
- `POST /castings` - Создать кастинг (только employer)
- `GET /castings/my` - Мои кастинги
- `PUT /castings/:castingId` - Обновить кастинг
- `DELETE /castings/:castingId` - Удалить кастинг
- `PUT /castings/:castingId/status` - Изменить статус
- `GET /castings/:castingId/stats` - Статистика кастинга
- `GET /castings/stats/my` - Моя статистика
- `GET /castings/matching` - Подходящие кастинги (для моделей)
- `GET /castings/:castingId/responses` - Отклики на кастинг

**Связи:**
- Создается работодателем
- Модели откликаются через Responses
- Используется в Matching для подбора моделей
- Связан с Reviews (после завершения)

**Жизненный цикл:**
\`\`\`
draft → active → closed
\`\`\`

---

### 5. Responses (8 маршрутов)

**Цель:** Управление откликами на кастинги

**Маршруты:**
- `POST /responses/castings/:castingId` - Откликнуться на кастинг
- `GET /responses/my` - Мои отклики
- `DELETE /responses/:responseId` - Отозвать отклик
- `GET /responses/castings/:castingId/list` - Отклики на кастинг (employer)
- `PUT /responses/:responseId/status` - Изменить статус (employer)
- `PUT /responses/:responseId/viewed` - Отметить как просмотренный
- `GET /responses/castings/:castingId/stats` - Статистика откликов
- `GET /responses/:responseId` - Получить отклик

**Связи:**
- Модель → Casting → Response
- Employer → Response → Accept/Reject
- Response → Notification (модели и работодателю)
- Response → Review (после завершения)

**Жизненный цикл:**
\`\`\`
pending → viewed → accepted/rejected
\`\`\`

---

### 6. Reviews (9 маршрутов)

**Цель:** Система отзывов и рейтингов

**Маршруты:**
- `POST /reviews` - Создать отзыв
- `GET /reviews/:reviewId` - Получить отзыв
- `GET /reviews/models/:modelId` - Отзывы модели
- `GET /reviews/models/:modelId/stats` - Статистика рейтинга
- `GET /reviews/models/:modelId/summary` - Сводка отзывов
- `GET /reviews/my` - Мои отзывы
- `PUT /reviews/:reviewId` - Обновить отзыв
- `DELETE /reviews/:reviewId` - Удалить отзыв
- `GET /reviews/can-create` - Проверить возможность создания

**Связи:**
- Employer → Model → Review (после завершения кастинга)
- Review → Model Rating (обновление рейтинга)
- Review → Notification (модели)

**Правила:**
- Только работодатель может оставить отзыв модели
- Один отзыв на одну модель по одному кастингу
- Рейтинг: 1-5 звезд

---

### 7. Portfolio (10 маршрутов)

**Цель:** Управление портфолио моделей

**Маршруты:**
- `POST /portfolio` - Добавить работу
- `GET /portfolio/:itemId` - Получить работу
- `GET /portfolio/model/:modelId` - Портфолио модели
- `GET /portfolio/featured` - Избранные работы
- `GET /portfolio/recent` - Недавние работы
- `PUT /portfolio/:itemId` - Обновить работу
- `DELETE /portfolio/:itemId` - Удалить работу
- `PUT /portfolio/reorder` - Изменить порядок
- `PUT /portfolio/:itemId/visibility` - Изменить видимость
- `GET /portfolio/stats/:modelId` - Статистика портфолио

**Связи:**
- Model → Portfolio Items
- Portfolio → Uploads (файлы)
- Portfolio → Model Profile (отображение)

---

### 8. Chat & Messages (10 маршрутов)

**Цель:** Система чата и сообщений

**Маршруты:**
- `POST /dialogs` - Создать диалог
- `GET /dialogs` - Мои диалоги
- `GET /dialogs/:dialogId` - Получить диалог
- `POST /messages` - Отправить сообщение
- `GET /dialogs/:dialogId/messages` - Сообщения диалога
- `PUT /messages/:messageId` - Редактировать сообщение
- `DELETE /messages/:messageId` - Удалить сообщение
- `POST /messages/:messageId/reactions` - Добавить реакцию
- `POST /dialogs/:dialogId/read` - Отметить как прочитанное
- `GET /dialogs/:dialogId/unread-count` - Количество непрочитанных

**WebSocket:** `ws://localhost:4000/ws`

**Связи:**
- Model ↔ Employer (диалог)
- Response → Dialog (начало общения)
- Message → Notification
- WebSocket → Real-time updates

---

### 9. Subscriptions & Payments (20 маршрутов)

**Цель:** Управление подписками и платежами

#### Plans (Public)
- `GET /plans` - Список планов
- `GET /plans/:planId` - Получить план

#### Subscriptions (User)
- `GET /subscriptions/my` - Моя подписка
- `GET /subscriptions/my/stats` - Статистика использования
- `POST /subscriptions/subscribe` - Оформить подписку
- `PUT /subscriptions/cancel` - Отменить подписку
- `PUT /subscriptions/renew` - Продлить подписку
- `GET /subscriptions/check-limit` - Проверить лимит
- `POST /subscriptions/increment-usage` - Увеличить использование
- `PUT /subscriptions/reset-usage` - Сбросить использование

#### Payments (User)
- `POST /payments/create` - Создать платеж
- `GET /payments/history` - История платежей
- `GET /payments/:paymentId/status` - Статус платежа

#### Robokassa Integration
- `POST /robokassa/init` - Инициировать оплату
- `POST /robokassa/callback` - Callback от Robokassa
- `GET /robokassa/check/:invId` - Проверить статус

**Связи:**
- User → Subscription → Plan
- Subscription → Limits (проверка при действиях)
- Payment → Robokassa → Callback → Subscription Update
- Subscription → Notification (истечение, обновление)

**Процесс оплаты:**
\`\`\`
1. User выбирает план
2. POST /robokassa/init
3. Redirect на Robokassa
4. Оплата
5. Robokassa → POST /robokassa/callback
6. Обновление подписки
7. Notification пользователю
\`\`\`

---

### 10. Uploads (6 маршрутов)

**Цель:** Управление загрузкой файлов

**Маршруты:**
- `POST /uploads` - Загрузить файл
- `POST /uploads/multi` - Загрузить несколько файлов
- `GET /uploads/:uploadId` - Получить файл
- `GET /uploads/user/me` - Мои файлы
- `GET /uploads/storage/usage` - Использование хранилища
- `DELETE /uploads/:uploadId` - Удалить файл

**Связи:**
- Portfolio → Uploads
- Profile → Uploads (аватар)
- Chat → Uploads (вложения)
- Subscription → Storage Limits

---

### 11. Notifications (10 маршрутов)

**Цель:** Система уведомлений

**Маршруты:**
- `GET /notifications/my` - Мои уведомления
- `GET /notifications/unread-count` - Количество непрочитанных
- `GET /notifications/stats` - Статистика
- `POST /notifications` - Создать уведомление
- `GET /notifications/:notificationId` - Получить уведомление
- `PUT /notifications/:notificationId/read` - Отметить как прочитанное
- `PUT /notifications/read-all` - Отметить все
- `PUT /notifications/read-multiple` - Отметить несколько
- `DELETE /notifications/:notificationId` - Удалить
- `DELETE /notifications` - Удалить все

**Связи:**
- Response → Notification (новый отклик, принятие/отклонение)
- Message → Notification (новое сообщение)
- Review → Notification (новый отзыв)
- Casting → Notification (матчинг)
- Subscription → Notification (истечение)

---

### 12. Search (12 маршрутов)

**Цель:** Поиск по платформе

**Маршруты:**
- `POST /search/castings` - Поиск кастингов
- `POST /search/castings/advanced` - Расширенный поиск кастингов
- `GET /search/castings/suggestions` - Подсказки
- `POST /search/models` - Поиск моделей
- `POST /search/models/advanced` - Расширенный поиск моделей
- `GET /search/models/suggestions` - Подсказки
- `POST /search/employers` - Поиск работодателей
- `POST /search/unified` - Унифицированный поиск
- `GET /search/autocomplete` - Автодополнение
- `GET /search/popular` - Популярные запросы
- `GET /search/trends` - Тренды поиска
- `GET /search/history` - История поиска
- `DELETE /search/history` - Очистить историю

**Связи:**
- Search → Castings/Models/Employers
- Search → Analytics (популярные запросы)

---

### 13. Matching (7 маршрутов)

**Цель:** Система подбора моделей и кастингов

**Маршруты:**
- `GET /matching/castings/:castingId/models` - Подходящие модели
- `POST /matching/models/search` - Поиск моделей по критериям
- `GET /matching/compatibility` - Совместимость
- `GET /matching/models/:modelId/similar` - Похожие модели
- `GET /matching/weights` - Веса критериев
- `GET /matching/castings/:castingId/stats` - Статистика матчинга кастинга
- `GET /matching/models/:modelId/stats` - Статистика матчинга модели

**Связи:**
- Casting → Matching → Models
- Model → Matching → Castings
- Matching → Notification (подходящий кастинг)

**Алгоритм:**
\`\`\`
1. Анализ требований кастинга
2. Поиск моделей по обязательным критериям
3. Вычисление score по весам
4. Сортировка по score
5. Возврат топ-N моделей
\`\`\`

---

### 14. Analytics (20 маршрутов)

**Цель:** Аналитика и статистика

#### Platform Analytics
- `GET /analytics/platform/overview` - Обзор платформы
- `GET /analytics/platform/growth` - Метрики роста
- `GET /analytics/platform/health` - Здоровье платформы

#### User Analytics
- `GET /analytics/users` - Аналитика пользователей
- `GET /analytics/users/acquisition` - Привлечение
- `GET /analytics/users/retention` - Удержание
- `GET /analytics/users/active/count` - Активные пользователи

#### Casting Analytics
- `GET /analytics/castings` - Аналитика кастингов
- `GET /analytics/castings/:employerId/performance` - Эффективность

#### Financial Analytics
- `GET /analytics/financial` - Финансовая аналитика

#### Geographic Analytics
- `GET /analytics/geographic` - Географическая аналитика
- `GET /analytics/geographic/cities` - По городам

#### Category Analytics
- `GET /analytics/categories` - Аналитика категорий
- `GET /analytics/categories/popular` - Популярные категории

#### Performance Analytics
- `GET /analytics/performance` - Производительность API
- `GET /analytics/realtime` - Real-time метрики
- `GET /analytics/system/health` - Здоровье системы

#### Reports
- `POST /analytics/reports/custom` - Кастомный отчет
- `GET /analytics/reports/predefined` - Готовые отчеты

**Связи:**
- Все сущности → Analytics
- Analytics → Admin Dashboard

---

### 15. Admin (50+ маршрутов)

**Цель:** Административные функции

#### Admin - Users
- `GET /admin/users` - Список пользователей
- `PUT /admin/users/:userId/status` - Изменить статус
- `PUT /admin/users/:userId/verify-employer` - Верифицировать
- `GET /admin/users/stats/registration` - Статистика регистраций
- `DELETE /admin/users/:userId` - Удалить пользователя

#### Admin - Castings
- `POST /admin/castings/close-expired` - Закрыть истекшие
- `GET /admin/castings/stats/platform` - Статистика
- `GET /admin/castings/stats/matching` - Статистика матчинга
- `GET /admin/castings/distribution/city` - Распределение
- `GET /admin/castings/count/active` - Активные
- `GET /admin/castings/categories/popular` - Популярные категории

#### Admin - Reviews
- `GET /admin/reviews/stats/platform` - Статистика
- `GET /admin/reviews/recent` - Недавние отзывы

#### Admin - Uploads
- `GET /uploads/admin/stats` - Статистика загрузок

#### Admin - Matching
- `PUT /admin/matching/weights` - Обновить веса
- `GET /admin/matching/stats/platform` - Статистика
- `POST /admin/matching/recalculate` - Пересчитать
- `GET /admin/matching/logs` - Логи
- `POST /admin/matching/batch` - Массовый матчинг

#### Admin - Notifications
- `GET /admin/notifications/templates` - Шаблоны
- `POST /admin/notifications/templates` - Создать шаблон
- `GET /admin/notifications/templates/:templateId` - Получить
- `PUT /admin/notifications/templates/:templateId` - Обновить
- `DELETE /admin/notifications/templates/:templateId` - Удалить
- `GET /admin/notifications` - Все уведомления
- `GET /admin/notifications/stats/platform` - Статистика
- `POST /admin/notifications/bulk-send` - Массовая рассылка
- `DELETE /admin/notifications/cleanup` - Очистить старые

#### Admin - Subscriptions
- `POST /admin/plans` - Создать план
- `PUT /admin/plans/:planId` - Обновить план
- `DELETE /admin/plans/:planId` - Удалить план
- `GET /admin/subscriptions/stats/platform` - Статистика
- `GET /admin/subscriptions/stats/revenue` - Доходы
- `GET /admin/subscriptions/expiring` - Истекающие
- `GET /admin/subscriptions/expired` - Истекшие
- `POST /admin/subscriptions/process-expired` - Обработать

#### Admin - Search
- `GET /admin/search/analytics` - Аналитика поиска
- `POST /admin/search/reindex` - Переиндексировать

#### Admin - Chat
- `GET /admin/dialogs` - Все диалоги
- `GET /admin/stats` - Статистика чата
- `POST /admin/clean` - Очистить старые
- `DELETE /admin/users/:userId/messages` - Удалить сообщения

#### Admin - Analytics
- `GET /analytics/admin/dashboard` - Дашборд

**Связи:**
- Admin → Все сущности (полный доступ)
- Admin → Analytics (мониторинг)
- Admin → Notifications (массовые рассылки)

---

## Связи между маршрутами

### Основные потоки данных

#### 1. Регистрация и аутентификация
\`\`\`
POST /auth/register
  ↓
POST /auth/verify-email
  ↓
POST /auth/login
  ↓
[access_token, refresh_token]
  ↓
Все защищенные маршруты
\`\`\`

#### 2. Создание профиля модели
\`\`\`
POST /auth/register (role=model)
  ↓
POST /profiles/model (автоматически)
  ↓
POST /portfolio (добавление работ)
  ↓
PUT /profiles/me/visibility (публикация)
\`\`\`

#### 3. Создание кастинга работодателем
\`\`\`
POST /auth/register (role=employer)
  ↓
POST /profiles/employer (автоматически)
  ↓
POST /castings (status=draft)
  ↓
PUT /castings/:id (заполнение)
  ↓
PUT /castings/:id/status (status=active)
  ↓
GET /matching/castings/:id/models (подбор моделей)
\`\`\`

#### 4. Отклик модели на кастинг
\`\`\`
GET /castings (поиск кастингов)
  ↓
GET /castings/:id (просмотр)
  ↓
POST /responses/castings/:id (отклик)
  ↓
[Notification работодателю]
  ↓
GET /responses/castings/:id/list (работодатель)
  ↓
PUT /responses/:id/status (accept/reject)
  ↓
[Notification модели]
\`\`\`

#### 5. Оставление отзыва
\`\`\`
PUT /responses/:id/status (accepted)
  ↓
[Работа завершена]
  ↓
GET /reviews/can-create (проверка)
  ↓
POST /reviews (создание отзыва)
  ↓
[Обновление рейтинга модели]
  ↓
[Notification модели]
\`\`\`

#### 6. Оформление подписки
\`\`\`
GET /plans (выбор плана)
  ↓
POST /robokassa/init (инициация оплаты)
  ↓
[Redirect на Robokassa]
  ↓
[Оплата]
  ↓
POST /robokassa/callback (от Robokassa)
  ↓
[Обновление подписки]
  ↓
[Notification пользователю]
\`\`\`

#### 7. Чат между пользователями
\`\`\`
POST /responses/castings/:id (отклик)
  ↓
POST /dialogs (создание диалога)
  ↓
WebSocket /ws (подключение)
  ↓
POST /messages (отправка сообщений)
  ↓
[Real-time обновления через WebSocket]
  ↓
POST /dialogs/:id/read (отметка прочитанных)
\`\`\`

---

## Типичные сценарии использования

### Сценарий 1: Модель ищет работу

\`\`\`
1. POST /auth/register (role=model)
2. POST /auth/verify-email
3. POST /auth/login
4. POST /profiles/model
5. POST /portfolio (добавление работ)
6. GET /castings (поиск кастингов)
7. GET /castings/matching (подходящие кастинги)
8. POST /responses/castings/:id (отклик)
9. GET /notifications/my (проверка уведомлений)
10. GET /dialogs (проверка сообщений)
\`\`\`

### Сценарий 2: Работодатель ищет модель

\`\`\`
1. POST /auth/register (role=employer)
2. POST /auth/verify-email
3. POST /auth/login
4. POST /profiles/employer
5. POST /castings (создание кастинга)
6. PUT /castings/:id/status (публикация)
7. GET /matching/castings/:id/models (подбор моделей)
8. GET /responses/castings/:id/list (просмотр откликов)
9. PUT /responses/:id/status (принятие/отклонение)
10. POST /reviews (оставление отзыва)
\`\`\`

### Сценарий 3: Администратор управляет платформой

\`\`\`
1. POST /auth/login (role=admin)
2. GET /analytics/admin/dashboard (обзор)
3. GET /admin/users (управление пользователями)
4. GET /admin/castings/stats/platform (статистика)
5. POST /admin/notifications/bulk-send (рассылка)
6. GET /admin/subscriptions/expiring (контроль подписок)
7. POST /admin/castings/close-expired (закрытие истекших)
\`\`\`

---

## Зависимости и последовательности

### Обязательные последовательности

1. **Регистрация → Верификация → Логин**
   - Нельзя войти без верификации email

2. **Логин → Access Token → Защищенные маршруты**
   - Все защищенные маршруты требуют токен

3. **Создание профиля → Действия**
   - Модель не может откликаться без профиля
   - Работодатель не может создавать кастинги без профиля

4. **Кастинг draft → active → Отклики**
   - Модели могут откликаться только на активные кастинги

5. **Отклик accepted → Отзыв**
   - Отзыв можно оставить только после принятого отклика

6. **Подписка → Лимиты → Действия**
   - Проверка лимитов перед созданием кастинга/отклика

### Необязательные последовательности

1. **Портфолио → Профиль**
   - Модель может работать без портфолио, но это снижает шансы

2. **Матчинг → Отклик**
   - Модель может откликаться без использования матчинга

3. **Чат → Отклик**
   - Чат может быть инициирован без отклика

---

## Заключение

Postman коллекция содержит **150+ маршрутов**, организованных в **15 логических групп**. Все маршруты тесно связаны между собой и образуют единую экосистему платформы MWork.

Основные связи:
- **Authentication** → Все защищенные маршруты
- **Profiles** → Castings, Responses, Reviews, Portfolio
- **Castings** → Responses, Matching, Reviews
- **Subscriptions** → Limits на все действия
- **Notifications** → Все события в системе
- **Analytics** → Все сущности для статистики
- **Admin** → Полный контроль над всеми сущностями

Для успешной работы с API необходимо понимать эти связи и следовать правильным последовательностям вызовов.
