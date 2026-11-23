# Techline — каталог спецтехники

Каталог спецтехники для дилера с витриной на трёх языках и админкой. Цены ведутся в юанях и пересчитываются в тенге и доллары по курсу Нацбанка с наценкой. Есть калькулятор лизинга с графиком платежей, расчёт стоимости владения, сравнение машин, полнотекстовый поиск, заявки с сайта и коммерческие предложения по публичной ссылке. В админке вход по номеру телефона, роли менеджера и администратора.

> Коммерческий проект, обезличен для портфолио: «Techline / ТехЛайн», адрес и телефоны вымышленные, цены и лизинговые программы в демо-данных выдуманы.

| Часть | Стек | Папка |
|---|---|---|
| **Backend** (основная часть) | Go 1.26, PostgreSQL 17 (pgx/v5, golang-migrate), Dragonfly (Redis-совместимый кэш), chi, Swagger, Prometheus + Pyroscope | [`backend/`](backend/) |
| **Frontend** | Next.js 16 (App Router, React 19), TypeScript strict, Tailwind v4, shadcn/ui, типы API из OpenAPI | [`frontend/`](frontend/) |

![Админка: курсы валют и наценки](docs/screenshots/admin-fx-rates.png)

## Архитектура

- **Backend: гексагональная архитектура.** Слои `transport/http` и `worker` → `service` (сценарии) → `domain` (правила, без HTTP/SQL), исходящие порты `storage/repository` с реализациями `postgres` и `dragonfly`.
- **Деньги считаются точно.** Курсы хранятся в `shopspring/decimal`, суммы в целых минорных единицах. Калькулятор лизинга — чистый доменный пакет без зависимостей.
- **Фоновые воркеры.** Ежедневно подтягивают курс Нацбанка (устаревший курс не выдаётся за актуальный) и разбирают outbox-очередь уведомлений о заявках.
- **API описан в Swagger.** Фронт генерирует из `swagger.json` TypeScript-типы, сборка падает, если контракт разошёлся.
- **Frontend: витрина на серверных компонентах (SSR, i18n ru/kk/en) и админка.** Браузер не ходит на бэкенд напрямую: запросы идут через прокси Next.js (`/api/proxy/*`), который подставляет токен. Поэтому нет CORS, а адрес API не попадает в бандл.
- **Наблюдаемость:** метрики Prometheus и профилирование Pyroscope поднимаются в том же docker compose.
- **Тесты:** unit-тесты с `-race`, интеграционные HTTP-тесты против поднятого в docker compose стенда (`make integration`), golangci-lint в CI.

Подробнее: [`backend/README.md`](backend/README.md), [`frontend/PROJECT.md`](frontend/PROJECT.md) (что реализовано и почему).

## Запуск локально

Нужны Docker, Node.js 20+ и Go 1.26 (для миграций).

```bash
# 1. Backend: Postgres, Dragonfly, Prometheus, Pyroscope и сервис на :8000
cd backend
docker compose up --build -d postgres dragonfly pyroscope service

# 2. Миграции
make bin-deps   # один раз: migrate, swag, mockgen, golangci-lint
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/techline?sslmode=disable'
make migrate-up

# 3. Frontend на http://localhost:3000
cd ../frontend
cp .env.example .env.local
npm install
npm run dev
```

Админка открывается под номером `+7 700 000 00 01` (переменная `ADMIN_PHONES` в `docker-compose.yml`). Как наполнить каталог демо-данными, описано в [`frontend/dump/README.md`](frontend/dump/README.md). **Цены, лизинговые программы и остаточная стоимость в демо выдуманы** и не являются прайсом компании.

Переменные окружения перечислены в [`backend/.env.example`](backend/.env.example) и [`frontend/.env.example`](frontend/.env.example).

## Тесты

```bash
cd backend && make test && make lint
cd frontend && npm run typecheck && npm run lint
```
