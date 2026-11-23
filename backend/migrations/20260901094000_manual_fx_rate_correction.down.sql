-- Откат: возвращаем ограничение на все источники.
--
-- Если к этому моменту у ручного источника накопилось несколько курсов
-- за одну дату, откат упадёт — и это правильно: молча выбросить
-- исправления, оставив опечатки, хуже явного отказа.

DROP INDEX IF EXISTS fx_rates_external_source_pair_date_uidx;

ALTER TABLE fx_rates ADD CONSTRAINT fx_rates_source_pair_date_unique
    UNIQUE (source, base_currency, quote_currency, rate_date);
