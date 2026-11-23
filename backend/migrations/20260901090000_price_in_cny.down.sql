-- Откат валюты прайса: юань → доллар.
--
-- Симметричен up: тем же курсом 7.20 CNY за доллар. Цены демонстрационные,
-- и обратный пересчёт возвращает исходный порядок величин; расхождение
-- в один фынь на округлении здесь ничего не значит.

ALTER TABLE machines ADD COLUMN IF NOT EXISTS price_usd_cents bigint;

UPDATE machines SET price_usd_cents = round(price_cny_cents / 7.20);

ALTER TABLE machines ALTER COLUMN price_usd_cents SET NOT NULL;

DROP INDEX IF EXISTS machines_public_price_idx;
DROP INDEX IF EXISTS machines_public_price_desc_idx;

ALTER TABLE machines DROP CONSTRAINT IF EXISTS machines_price_non_negative;
ALTER TABLE machines DROP COLUMN IF EXISTS price_cny_cents;

ALTER TABLE machines ADD CONSTRAINT machines_price_non_negative CHECK (price_usd_cents >= 0);

CREATE INDEX IF NOT EXISTS machines_public_price_idx
    ON machines (category_id, (CASE WHEN is_price_public THEN price_usd_cents END), id)
    WHERE is_active;

CREATE INDEX IF NOT EXISTS machines_public_price_desc_idx
    ON machines (category_id,
                 (CASE WHEN is_price_public THEN price_usd_cents END) DESC NULLS LAST,
                 id DESC)
    WHERE is_active;

ALTER TABLE quotes ADD COLUMN IF NOT EXISTS price_usd_cents bigint;

UPDATE quotes SET price_usd_cents = round(price_cny_cents / 7.20);

ALTER TABLE quotes ALTER COLUMN price_usd_cents SET NOT NULL;

ALTER TABLE quotes DROP CONSTRAINT IF EXISTS quotes_price_non_negative;
ALTER TABLE quotes DROP COLUMN IF EXISTS price_cny_cents;

ALTER TABLE quotes ADD CONSTRAINT quotes_price_non_negative CHECK (price_usd_cents >= 0);

ALTER TABLE machines ADD COLUMN IF NOT EXISTS service_cost_usd_cents bigint;
UPDATE machines SET service_cost_usd_cents = round(service_cost_cny_cents / 7.20)
 WHERE service_cost_cny_cents IS NOT NULL;
ALTER TABLE machines DROP COLUMN IF EXISTS service_cost_cny_cents;

ALTER TABLE leasing_programs ADD COLUMN IF NOT EXISTS max_amount_usd_cents bigint;
UPDATE leasing_programs SET max_amount_usd_cents = round(max_amount_cny_cents / 7.20)
 WHERE max_amount_cny_cents IS NOT NULL;

ALTER TABLE leasing_programs DROP CONSTRAINT IF EXISTS leasing_programs_max_amount_positive;
ALTER TABLE leasing_programs DROP COLUMN IF EXISTS max_amount_cny_cents;
ALTER TABLE leasing_programs ADD CONSTRAINT leasing_programs_max_amount_positive
    CHECK (max_amount_usd_cents IS NULL OR max_amount_usd_cents > 0);
