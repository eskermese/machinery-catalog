-- Валюта прайса: доллар → юань.
--
-- Цена у поставщика номинирована в юанях, и она же — единственный источник
-- правды. Тенге и доллары ВЫВОДЯТСЯ из неё, каждый по своей паре и своей
-- наценке. Второго хранимого прайса нет намеренно: две цены разойдутся,
-- и никто не скажет, какая верная.
--
-- Миграция аддитивная: колонка добавляется, значения переносятся, старая
-- снимается. Так она переживает существующие данные.

-- ---------------------------------------------------------------------------
-- Курс пересчёта
-- ---------------------------------------------------------------------------
--
-- 7.20 CNY за доллар — порядок величины 2026 года. Точность здесь роли
-- не играет: цены каталога ДЕМОНСТРАЦИОННЫЕ (на сайте-источнике их нет,
-- см. dump/README.md), и переносятся они только чтобы сортировка и фильтр
-- по цене продолжали работать на тех же относительных величинах.
--
-- Для quotes значение то же и по другой причине: там цена уже зафиксирована
-- и пересчитывать её по-настоящему нечем — курса CNY на дату выпуска
-- не существовало. Важно лишь, чтобы выданное КП показывало ТЕ ЖЕ тенге,
-- что и раньше, а они хранятся отдельной колонкой и не трогаются вовсе.

-- ---------------------------------------------------------------------------
-- machines
-- ---------------------------------------------------------------------------

ALTER TABLE machines ADD COLUMN IF NOT EXISTS price_cny_cents bigint;

UPDATE machines SET price_cny_cents = round(price_usd_cents * 7.20);

ALTER TABLE machines ALTER COLUMN price_cny_cents SET NOT NULL;

-- Индексы сортировки по цене строятся по новой колонке. Старые снимаются
-- вместе с колонкой: DROP COLUMN уносит зависящие от неё индексы сам,
-- но полагаться на это нельзя — порядок важен, и явное снятие читается.
DROP INDEX IF EXISTS machines_public_price_idx;
DROP INDEX IF EXISTS machines_public_price_desc_idx;

ALTER TABLE machines DROP CONSTRAINT IF EXISTS machines_price_non_negative;
ALTER TABLE machines DROP COLUMN IF EXISTS price_usd_cents;

ALTER TABLE machines ADD CONSTRAINT machines_price_non_negative CHECK (price_cny_cents >= 0);

-- Те же два индекса, что были у долларовой цены, и по той же причине:
-- обратный проход переворачивает размещение NULL, поэтому убывающая
-- сортировка требует своего индекса.
CREATE INDEX IF NOT EXISTS machines_public_price_idx
    ON machines (category_id, (CASE WHEN is_price_public THEN price_cny_cents END), id)
    WHERE is_active;

CREATE INDEX IF NOT EXISTS machines_public_price_desc_idx
    ON machines (category_id,
                 (CASE WHEN is_price_public THEN price_cny_cents END) DESC NULLS LAST,
                 id DESC)
    WHERE is_active;

COMMENT ON COLUMN machines.price_cny_cents IS
    'Цена у поставщика в фынях (1/100 юаня). Единственный хранимый прайс: тенге и доллары выводятся из неё по своим парам и своим наценкам';

-- ---------------------------------------------------------------------------
-- quotes
-- ---------------------------------------------------------------------------
--
-- Зафиксированный курс (fx_rate, fx_rate_date, fx_markup_bps) НЕ трогаем:
-- он описывает пару, действовавшую на момент выпуска, и относится к прошлому.
-- Сумма в тиынах тоже остаётся как есть — именно она показывается клиенту,
-- и она обязана совпасть с тем, что он видел.

ALTER TABLE quotes ADD COLUMN IF NOT EXISTS price_cny_cents bigint;

UPDATE quotes SET price_cny_cents = round(price_usd_cents * 7.20);

ALTER TABLE quotes ALTER COLUMN price_cny_cents SET NOT NULL;

ALTER TABLE quotes DROP CONSTRAINT IF EXISTS quotes_price_non_negative;
ALTER TABLE quotes DROP COLUMN IF EXISTS price_usd_cents;

ALTER TABLE quotes ADD CONSTRAINT quotes_price_non_negative CHECK (price_cny_cents >= 0);

COMMENT ON COLUMN quotes.price_cny_cents IS
    'Цена машины на момент выпуска в фынях; изменение прайса не трогает уже выданные КП';

-- ---------------------------------------------------------------------------
-- Величины, выраженные В ВАЛЮТЕ ПРАЙСА
-- ---------------------------------------------------------------------------
--
-- Их две, и обе меняют валюту по той же причине, что и сама цена.
--
-- machines.service_cost_usd_cents — стоимость планового ТО. Задаётся
-- по паспорту машины, то есть у поставщика и в его валюте, а в тенге
-- переводится тем же курсом, что и цена.
--
-- leasing_programs.max_amount_usd_cents — потолок суммы сделки. Он
-- СРАВНИВАЕТСЯ С ЦЕНОЙ напрямую, без приведения курсом (см.
-- LeasingProgram.CheckBounds). Оставить его в долларах значило бы сравнивать
-- юани с долларами и молча пропускать сделки в семь раз дороже потолка.

ALTER TABLE machines ADD COLUMN IF NOT EXISTS service_cost_cny_cents bigint;
UPDATE machines SET service_cost_cny_cents = round(service_cost_usd_cents * 7.20)
 WHERE service_cost_usd_cents IS NOT NULL;
ALTER TABLE machines DROP COLUMN IF EXISTS service_cost_usd_cents;

COMMENT ON COLUMN machines.service_cost_cny_cents IS
    'Стоимость одного планового ТО в фынях; в тенге переводится тем же курсом, что и цена';

ALTER TABLE leasing_programs ADD COLUMN IF NOT EXISTS max_amount_cny_cents bigint;
UPDATE leasing_programs SET max_amount_cny_cents = round(max_amount_usd_cents * 7.20)
 WHERE max_amount_usd_cents IS NOT NULL;

ALTER TABLE leasing_programs DROP CONSTRAINT IF EXISTS leasing_programs_max_amount_positive;
ALTER TABLE leasing_programs DROP COLUMN IF EXISTS max_amount_usd_cents;
ALTER TABLE leasing_programs ADD CONSTRAINT leasing_programs_max_amount_positive
    CHECK (max_amount_cny_cents IS NULL OR max_amount_cny_cents > 0);

COMMENT ON COLUMN leasing_programs.max_amount_cny_cents IS
    'Потолок суммы сделки в фынях; сравнивается с ценой напрямую, поэтому обязан быть в валюте прайса';
