-- Курсы валют: чистый курс источника и отдельная история наценки.

-- ---------------------------------------------------------------------------
-- fx_rates
-- ---------------------------------------------------------------------------
--
-- Таблица только пополняется: строка описывает курс, который уже был показан
-- клиенту, и переписывать её задним числом нельзя — иначе цена в выданном КП
-- перестанет сходиться с историей. Запретить UPDATE/DELETE на уровне схемы без
-- триггера нельзя, поэтому правило держится соглашением и правами роли
-- приложения; здесь оно зафиксировано комментарием.

CREATE TABLE IF NOT EXISTS fx_rates
(
    id             uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    base_currency  text           NOT NULL,
    quote_currency text           NOT NULL,
    source         text           NOT NULL,
    rate           numeric(18, 6) NOT NULL,
    rate_date      date           NOT NULL,
    fetched_at     timestamptz    NOT NULL DEFAULT now(),

    -- Пара хранится двумя колонками, а не строкой «USD/KZT»: по отдельным
    -- колонкам строится индекс поиска последнего курса, разбирать строку
    -- в предикате пришлось бы на каждой строке.
    CONSTRAINT fx_rates_base_currency_format CHECK (base_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT fx_rates_quote_currency_format CHECK (quote_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT fx_rates_pair_distinct CHECK (base_currency <> quote_currency),

    CONSTRAINT fx_rates_rate_positive CHECK (rate > 0),

    -- Один источник даёт один курс пары в сутки. Повторная загрузка того же дня
    -- обязана конфликтовать, а не создавать второй курс: иначе «курс на дату»
    -- перестаёт быть однозначным.
    CONSTRAINT fx_rates_source_pair_date_unique UNIQUE (source, base_currency, quote_currency, rate_date)
);

-- «Актуальный курс USD/KZT» — запрос на каждый показ цены, поэтому он и получает
-- индекс. UNIQUE выше для него не годится: там ведущая колонка source, а курс
-- берётся по паре независимо от того, кто его поставил.
CREATE INDEX IF NOT EXISTS fx_rates_latest_idx
    ON fx_rates (base_currency, quote_currency, rate_date DESC);

COMMENT ON TABLE fx_rates IS
    'История курсов валют от внешних источников. Хранит только чистый курс: наценка сервиса живёт в fx_markups и к строке курса отношения не имеет. Append-only — строки не перезаписываются, потому что курс из выданного КП должен оставаться воспроизводимым';
COMMENT ON COLUMN fx_rates.base_currency IS 'Валюта, в которой ведётся прайс (USD)';
COMMENT ON COLUMN fx_rates.quote_currency IS 'Валюта, в которую пересчитывается цена (KZT)';
COMMENT ON COLUMN fx_rates.source IS 'Кто поставил курс: код внешнего источника либо ручной ввод. Часть ключа уникальности — курсы разных источников на одну дату сосуществуют';
COMMENT ON COLUMN fx_rates.rate IS 'Курс источника без наценки: сколько единиц quote_currency за одну base_currency. numeric(18,6): курс — не деньги, округление до минорной единицы здесь потеряло бы точность пересчёта';
COMMENT ON COLUMN fx_rates.rate_date IS 'Дата, на которую действует курс (date, а не timestamptz: источник публикует курс на сутки)';
COMMENT ON COLUMN fx_rates.fetched_at IS 'Момент получения курса сервисом; отличается от rate_date при загрузке задним числом';

-- ---------------------------------------------------------------------------
-- fx_markups
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS fx_markups
(
    id             uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    base_currency  text        NOT NULL,
    quote_currency text        NOT NULL,
    markup_bps     integer     NOT NULL,
    effective_from timestamptz NOT NULL,
    changed_by     uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT fx_markups_base_currency_format CHECK (base_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT fx_markups_quote_currency_format CHECK (quote_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT fx_markups_pair_distinct CHECK (base_currency <> quote_currency),
    CONSTRAINT fx_markups_markup_non_negative CHECK (markup_bps >= 0)
);

-- Единственный горячий запрос: «действующая наценка на пару в момент времени» —
-- строка с наибольшим effective_from, не превышающим этот момент. Порядок
-- колонок повторяет форму запроса: отбор по паре, затем поиск ближайшей
-- предшествующей даты, поэтому DESC на effective_from обязателен —
-- иначе планировщик добавит сортировку поверх скана.
CREATE INDEX IF NOT EXISTS fx_markups_effective_idx
    ON fx_markups (base_currency, quote_currency, effective_from DESC);

COMMENT ON TABLE fx_markups IS
    'История наценки к курсу источника. Отдельная таблица, а не колонка в fx_rates, по двум причинам. Первая: наценка — бизнес-настройка, её меняют по решению, а не по расписанию источника, и её календарь не совпадает с календарём котировок. Вторая: fx_rates append-only и защищена UNIQUE (source, пара, дата), поэтому в колонке наценку нельзя было бы изменить ни задним числом, ни сегодня, если курс на сегодня уже загружен. Здесь же смена наценки — обычная вставка строки, а прошлые значения остаются доказательством того, по какой наценке считалось выданное КП';
COMMENT ON COLUMN fx_markups.markup_bps IS
    'Наценка в базисных пунктах (100 bps = 1%). Целое, а не проценты дробью: наценка задаётся бизнесом в bps и участвует в арифметике без ошибки округления';
COMMENT ON COLUMN fx_markups.effective_from IS
    'Момент, с которого наценка действует. Действующей считается строка с наибольшим effective_from, не превышающим текущий момент. Значение в будущем допустимо: так наценка планируется заранее и вступает в силу сама';
COMMENT ON COLUMN fx_markups.changed_by IS
    'Кто изменил наценку; NULL — изменение не от человека (посев, миграция данных). SET NULL при удалении учётной записи: запись истории обязана пережить увольнение';
COMMENT ON COLUMN fx_markups.created_at IS
    'Момент внесения записи. Отличается от effective_from, когда наценку задают заранее или задним числом, и отвечает на вопрос «когда об этом узнала система»';
