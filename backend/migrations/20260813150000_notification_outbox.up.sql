-- Исходящие уведомления (ТЗ 5). Событие пишется здесь, доставка появится вместе
-- с адаптером канала: воркера-отправщика в этой миграции нет намеренно.
--
-- processed_at остается NULL до момента успешной доставки, поэтому частичный
-- индекс покрывает единственный горячий запрос будущего отправщика — выборку
-- необработанных событий.
CREATE TABLE IF NOT EXISTS notification_outbox
(
    id             uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    event_type     text        NOT NULL,
    entity_type    text        NOT NULL,
    entity_id      uuid        NOT NULL,
    recipient_role text        NOT NULL,
    recipient_id   uuid        NOT NULL,
    payload        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at     timestamptz NOT NULL DEFAULT now(),
    processed_at   timestamptz,

    CONSTRAINT notification_outbox_recipient_role_allowed CHECK (
        recipient_role IN ('manager', 'admin', 'system')
        )
);

CREATE INDEX IF NOT EXISTS notification_outbox_unprocessed_idx
    ON notification_outbox (created_at)
    WHERE processed_at IS NULL;
