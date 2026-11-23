-- Пользователи сервиса на время MVP: собственного IdP у Заказчика ещё нет,
-- поэтому токены выпускаются по номеру телефона.
--
-- Номер хранится нормализованным (см. entity.NormalizePhone): только цифры
-- с ведущим плюсом. Формат продублирован CHECK-ом, потому что нормализация
-- в сервисе защищает лишь один путь записи, а таблица переживёт и посев,
-- и импорт, и ручной SQL.
--
-- Уникальность по паре (phone, role), а не по телефону: один и тот же человек
-- может входить в разных ролях — это разные учётные записи
-- с разными идентификаторами.
CREATE TABLE IF NOT EXISTS users
(
    id         uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    phone      text        NOT NULL,
    role       text        NOT NULL,
    -- Имя сотрудника для показа. Необязательное и заполняется им самим:
    -- вход идёт по номеру, форм регистрации нет, и в момент создания записи
    -- имени взять неоткуда. Пока оно пусто, человека называет телефон.
    name       text,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT users_role_allowed CHECK (role IN ('manager', 'admin')),

    -- Пустая строка и NULL не должны значить разное: «имя не задано» —
    -- одно состояние, и хранится оно одним значением.
    CONSTRAINT users_name_not_blank CHECK (name IS NULL OR btrim(name) <> ''),
    CONSTRAINT users_name_length CHECK (name IS NULL OR char_length(name) <= 100),

    -- Тот же формат, что и у leads.phone: цифры с ведущим плюсом, 10..15 знаков
    -- (верхняя граница — предел E.164). Единый предикат в обеих таблицах нужен
    -- потому, что один и тот же человек приходит и как заявка, и как учётная
    -- запись, и склеивать его по номеру можно только при одинаковом написании.
    CONSTRAINT users_phone_format CHECK (phone ~ '^\+[0-9]{10,15}$'),

    CONSTRAINT users_phone_role_unique UNIQUE (phone, role)
);
