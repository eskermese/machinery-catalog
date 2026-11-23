-- Заявки с сайта. Уведомление менеджеру идёт через notification_outbox,
-- отдельной таблицы очереди здесь нет.

CREATE TABLE IF NOT EXISTS leads
(
    id         uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    phone      text        NOT NULL,
    city       text,
    machine_id uuid REFERENCES machines (id) ON DELETE SET NULL,
    quote_id   uuid REFERENCES quotes (id) ON DELETE SET NULL,
    source     text        NOT NULL,
    status     text        NOT NULL DEFAULT 'new',
    message    text,
    comment    text,
    manager_id uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    -- Статус — конечный автомат, которым управляет сервис, поэтому перечень
    -- закрыт. Расширение набора — осознанная миграция, а не появление нового
    -- значения из опечатки в коде.
    CONSTRAINT leads_status_allowed CHECK (status IN ('new', 'in_progress', 'won', 'lost')),

    CONSTRAINT leads_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT leads_phone_format CHECK (phone ~ '^\+[0-9]{10,15}$'),
    CONSTRAINT leads_source_not_blank CHECK (length(btrim(source)) > 0),

    -- Сообщение клиента: пустое означает «не написал», и хранить его как
    -- пустую строку нельзя — иначе «не писал» и «отправил пробел» станут
    -- разными состояниями с одинаковым смыслом. Верхняя граница здесь,
    -- а не только в домене: поле приходит с публичной формы.
    CONSTRAINT leads_message_length
        CHECK (message IS NULL OR length(btrim(message)) BETWEEN 1 AND 2000),

    -- Заявка «по этому КП» без самой машины — противоречие: КП всегда о машине,
    -- и потерять эту связь можно только вместе с обеими ссылками.
    CONSTRAINT leads_quote_requires_machine CHECK (quote_id IS NULL OR machine_id IS NOT NULL)
);

-- Рабочая очередь: необработанные заявки, свежие сверху. Ведущий status,
-- потому что менеджер всегда смотрит один статус, а не всю таблицу.
CREATE INDEX IF NOT EXISTS leads_status_created_idx ON leads (status, created_at DESC);

-- «Мои заявки». Частичный: нераспределённые заявки берутся из очереди по
-- статусу, а не из этого индекса.
CREATE INDEX IF NOT EXISTS leads_manager_created_idx
    ON leads (manager_id, created_at DESC)
    WHERE manager_id IS NOT NULL;

-- Поиск заявки по номеру: клиент перезванивает сам, и менеджеру нужно поднять
-- его историю до начала разговора.
CREATE INDEX IF NOT EXISTS leads_phone_idx ON leads (phone);

COMMENT ON TABLE leads IS 'Заявки с сайта: обратный звонок, запрос цены, запрос КП';
COMMENT ON COLUMN leads.phone IS 'Номер в нормализованном виде (цифры с ведущим +), как и у учётных записей — иначе один клиент не склеивается по разным заявкам';
COMMENT ON COLUMN leads.machine_id IS 'Машина, из карточки которой пришла заявка; NULL — общий запрос. SET NULL: заявка переживает удаление карточки, терять контакт из-за чистки каталога нельзя';
COMMENT ON COLUMN leads.quote_id IS 'КП, по которому клиент обратился; NULL — заявка не из КП';
COMMENT ON COLUMN leads.message IS
    'Что написал КЛИЕНТ в форме: какая деталь нужна, к какой технике, что случилось. Отдельно от comment намеренно — тот принадлежит менеджеру, и одно поле на двоих означало бы, что правка менеджера затирает запрос клиента';
COMMENT ON COLUMN leads.source IS
    'Откуда пришла заявка: страница, форма или рекламная метка. Без CHECK намеренно — набор источников меняется вместе с маркетингом, и закрытый перечень требовал бы миграции на каждую кампанию';
COMMENT ON COLUMN leads.status IS 'Стадия обработки заявки менеджером';
COMMENT ON COLUMN leads.manager_id IS 'Ответственный менеджер; NULL — заявка ещё не распределена. SET NULL при удалении учётной записи: заявка остаётся, ответственный назначается заново';
COMMENT ON COLUMN leads.updated_at IS 'Обновляется сервисом при смене статуса или ответственного; триггера на таблице нет';
