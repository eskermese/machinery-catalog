-- Лизинг: программы партнёров и сформированные по ним коммерческие предложения.
--
-- Обе таблицы в одной миграции: quotes ссылается на leasing_programs и без неё
-- не поднимается, а откат должен снимать связку целиком.

-- ---------------------------------------------------------------------------
-- leasing_programs
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS leasing_programs
(
    id                       uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    name_ru                  text          NOT NULL,
    name_kk                  text          NOT NULL,
    name_en                  text          NOT NULL,
    logo_path                text,
    min_down_payment_bps     integer       NOT NULL,
    term_months_min          integer       NOT NULL,
    term_months_max          integer       NOT NULL,
    rate_min_bps             integer       NOT NULL,
    rate_max_bps             integer       NOT NULL,
    max_amount_usd_cents     bigint,
    is_active                boolean       NOT NULL DEFAULT true,
    position                 integer       NOT NULL DEFAULT 0,
    created_at               timestamptz   NOT NULL DEFAULT now(),
    updated_at               timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT leasing_programs_down_payment_range CHECK (min_down_payment_bps BETWEEN 0 AND 10000),

    -- Границы диапазонов проверяются парой, а не по отдельности: программа
    -- с term_min > term_max проходит поштучную проверку и ломает калькулятор
    -- пустым множеством допустимых сроков.
    CONSTRAINT leasing_programs_term_range CHECK (term_months_min > 0 AND term_months_max >= term_months_min),
    CONSTRAINT leasing_programs_rate_range CHECK (rate_min_bps >= 0 AND rate_max_bps >= rate_min_bps),

    CONSTRAINT leasing_programs_max_amount_positive CHECK (max_amount_usd_cents IS NULL OR max_amount_usd_cents > 0),
    CONSTRAINT leasing_programs_position_non_negative CHECK (position >= 0)
);

-- Индексов нет намеренно: таблица — единицы строк, калькулятор читает её
-- целиком одним запросом. Индекс здесь только увеличил бы стоимость записи.

COMMENT ON TABLE leasing_programs IS 'Программы лизинговых партнёров: границы, в которых калькулятор считает платёж';
COMMENT ON COLUMN leasing_programs.logo_path IS 'Путь к логотипу партнёра в файловом хранилище, не URL';
COMMENT ON COLUMN leasing_programs.min_down_payment_bps IS
    'Минимальный первоначальный взнос в базисных пунктах (15.00% = 1500). Те же единицы, что и quotes.down_payment_bps: сравнение выбранного взноса с границей программы идёт без приведения, а разные единицы на границе — класс ошибок на ровном месте';
COMMENT ON COLUMN leasing_programs.rate_min_bps IS
    'Нижняя граница ставки в базисных пунктах (100 bps = 1%). Целые bps, а не проценты дробью: ставка участвует в формуле платежа, и целое число исключает накопление ошибки округления';
COMMENT ON COLUMN leasing_programs.rate_max_bps IS 'Верхняя граница ставки в базисных пунктах';
COMMENT ON COLUMN leasing_programs.max_amount_usd_cents IS
    'Потолок суммы сделки в центах USD; NULL — без ограничения. Валюта та же, что у прайса, чтобы сравнение с ценой машины не требовало курса';

-- ---------------------------------------------------------------------------
-- quotes
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS quotes
(
    id                        uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    public_token              text           NOT NULL,
    machine_id                uuid           NOT NULL REFERENCES machines (id) ON DELETE RESTRICT,
    configuration             jsonb          NOT NULL DEFAULT '{}'::jsonb,

    -- Цена машины в валюте прайса и курс, по которому она переведена в тенге.
    price_usd_cents           bigint         NOT NULL,
    fx_rate                   numeric(18, 6) NOT NULL,
    fx_rate_date              date           NOT NULL,
    fx_markup_bps             integer        NOT NULL,

    -- Блок расчёта лизинга. Весь в тиынах: расчёт ведётся сразу в тенге,
    -- потому что клиент платит в тенге и график обязан сходиться в валюте
    -- платежа. Стоимость переводится в тиыны по fx_rate до расчёта, а не
    -- после: пересчёт готового графика округлял бы каждую строку второй раз,
    -- и сумма платежей перестала бы сходиться с суммой финансирования.
    asset_cost_kzt_cents      bigint         NOT NULL,
    leasing_program_id        uuid REFERENCES leasing_programs (id) ON DELETE RESTRICT,
    schedule_type             text           NOT NULL,
    down_payment_bps          integer        NOT NULL,
    term_months               integer        NOT NULL,
    rate_bps                  integer        NOT NULL,
    residual_value_kzt_cents  bigint         NOT NULL DEFAULT 0,
    commission_kzt_cents      bigint         NOT NULL DEFAULT 0,

    financed_amount_kzt_cents bigint         NOT NULL,
    monthly_payment_kzt_cents bigint         NOT NULL,
    last_payment_kzt_cents    bigint         NOT NULL,
    total_paid_kzt_cents      bigint         NOT NULL,
    overpayment_kzt_cents     bigint         NOT NULL,
    payment_schedule          jsonb          NOT NULL DEFAULT '[]'::jsonb,

    author_id                 uuid REFERENCES users (id) ON DELETE RESTRICT,
    expires_at                timestamptz    NOT NULL,
    first_opened_at           timestamptz,
    created_at                timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT quotes_price_non_negative CHECK (price_usd_cents >= 0),
    CONSTRAINT quotes_fx_rate_positive CHECK (fx_rate > 0),
    CONSTRAINT quotes_fx_markup_non_negative CHECK (fx_markup_bps >= 0),

    CONSTRAINT quotes_schedule_type_allowed CHECK (schedule_type IN ('annuity', 'differentiated')),
    CONSTRAINT quotes_down_payment_range CHECK (down_payment_bps BETWEEN 0 AND 10000),
    CONSTRAINT quotes_term_positive CHECK (term_months > 0),
    CONSTRAINT quotes_rate_non_negative CHECK (rate_bps >= 0),
    CONSTRAINT quotes_residual_non_negative CHECK (residual_value_kzt_cents >= 0),
    CONSTRAINT quotes_commission_non_negative CHECK (commission_kzt_cents >= 0),

    CONSTRAINT quotes_asset_cost_non_negative CHECK (asset_cost_kzt_cents >= 0),
    CONSTRAINT quotes_financed_non_negative CHECK (financed_amount_kzt_cents >= 0),

    -- Сумма финансирования — это стоимость за вычетом взноса, поэтому она
    -- не может превышать саму стоимость. Проверка ловит перепутанные местами
    -- значения: по отдельности обе величины остаются правдоподобными.
    CONSTRAINT quotes_financed_within_asset_cost CHECK (financed_amount_kzt_cents <= asset_cost_kzt_cents),
    CONSTRAINT quotes_monthly_payment_non_negative CHECK (monthly_payment_kzt_cents >= 0),
    CONSTRAINT quotes_last_payment_non_negative CHECK (last_payment_kzt_cents >= 0),
    CONSTRAINT quotes_total_paid_non_negative CHECK (total_paid_kzt_cents >= 0),
    CONSTRAINT quotes_overpayment_non_negative CHECK (overpayment_kzt_cents >= 0),

    -- Выкуп — часть суммы финансирования, а не добавка к ней: он не
    -- амортизируется в графике, но гасится последним платежом из того же тела.
    CONSTRAINT quotes_residual_within_financed CHECK (residual_value_kzt_cents <= financed_amount_kzt_cents),

    -- КП с истечением раньше выпуска — мёртвая ссылка в момент отправки.
    CONSTRAINT quotes_expires_after_created CHECK (expires_at > created_at),
    CONSTRAINT quotes_first_opened_after_created CHECK (first_opened_at IS NULL OR first_opened_at >= created_at),

    -- Токен раздаётся клиенту ссылкой, поэтому он должен быть длинным
    -- и непредсказуемым; нижняя граница длины отсекает случайную заглушку.
    CONSTRAINT quotes_public_token_length CHECK (length(public_token) >= 32)
);

-- Открытие КП клиентом по ссылке — единственный публичный запрос к таблице.
CREATE UNIQUE INDEX IF NOT EXISTS quotes_public_token_uidx ON quotes (public_token);

-- «Мои КП» в кабинете менеджера, свежие сверху. Частичный: КП без автора
-- (выпущенные автоматически) в этот список не попадают.
CREATE INDEX IF NOT EXISTS quotes_author_created_idx
    ON quotes (author_id, created_at DESC)
    WHERE author_id IS NOT NULL;

COMMENT ON TABLE quotes IS
    'Коммерческие предложения. Цена, курс и параметры расчёта фиксируются в момент выпуска: клиент и менеджер обязаны видеть одну и ту же цифру спустя недели после формирования. Блок расчёта хранится в тиынах целиком — расчёт лизинга ведётся сразу в валюте платежа';
COMMENT ON COLUMN quotes.public_token IS 'Секрет в ссылке на КП: доступ к предложению даётся без аутентификации, поэтому токен и есть право доступа';
COMMENT ON COLUMN quotes.machine_id IS 'RESTRICT: выпущенное КП — обязательство перед клиентом, машину под ним удалить нельзя';
COMMENT ON COLUMN quotes.configuration IS 'Выбранная комплектация и опции. jsonb, а не таблица: состав опций не участвует в фильтрах и читается только вместе с самим КП';
COMMENT ON COLUMN quotes.price_usd_cents IS 'Цена машины на момент выпуска в центах USD; изменение прайса не трогает уже выданные КП';

COMMENT ON COLUMN quotes.fx_rate IS
    'ПРИМЕНЁННЫЙ курс — уже с наценкой сервиса. Именно на него умножается цена при переводе в тиыны, и именно он показывается клиенту. Курс источника отдельно не хранится: он восстанавливается как fx_rate / (1 + fx_markup_bps / 10000). Значение копируется в КП, а не берётся ссылкой на fx_rates: КП должен переживать любую правку истории курсов';
COMMENT ON COLUMN quotes.fx_rate_date IS 'Дата курса источника, из которого выведен применённый курс, — то, что показывается клиенту рядом с тенговой ценой';
COMMENT ON COLUMN quotes.fx_markup_bps IS
    'Наценка, зашитая в fx_rate, в базисных пунктах. Хранится, чтобы применённый курс был разложим на составляющие: без неё нельзя ответить, сколько в цене от рынка, а сколько от наценки, и нельзя проверить расчёт задним числом';

COMMENT ON COLUMN quotes.leasing_program_id IS 'Выбранная программа; NULL — расчёт без лизинга. RESTRICT: программу, по которой есть КП, снимают флагом is_active, а не удалением';
COMMENT ON COLUMN quotes.schedule_type IS 'Тип графика, по которому считался платёж. Без него запись невоспроизводима: при одних и тех же сроке и ставке аннуитет и дифференцированный дают разные платежи';
COMMENT ON COLUMN quotes.down_payment_bps IS
    'Первоначальный взнос в базисных пунктах (20.00% = 2000). Целые bps, а не проценты дробью: домен считает в bps, и конверсия на границе была бы лишним классом ошибок. Точность та же — сотые доли процента. Сам взнос в тиынах не хранится: он однозначно выводится как стоимость в тиынах минус financed_amount_kzt_cents';
COMMENT ON COLUMN quotes.rate_bps IS 'Ставка, применённая в расчёте, в базисных пунктах: программа задаёт диапазон, в КП попадает конкретное значение';
COMMENT ON COLUMN quotes.residual_value_kzt_cents IS
    'Выкупная стоимость в тиынах. В графике не амортизируется: остаток долга до последнего платежа не опускается ниже неё, а гасится она целиком последним платежом. Ноль — выкуп не предусмотрен';
COMMENT ON COLUMN quotes.commission_kzt_cents IS
    'Разовая комиссия за организацию в тиынах. В тело финансирования не входит — проценты на неё не начисляются, — но входит в total_paid_kzt_cents и в overpayment_kzt_cents';

COMMENT ON COLUMN quotes.asset_cost_kzt_cents IS
    'Стоимость машины в тиынах на момент выпуска — то, с чем работал калькулятор. Хранится, а не выводится как round(price_usd_cents * fx_rate): правило округления живёт в коде сервиса, и после его смены старое КП показало бы другую сумму. Тот же довод, по которому хранится monthly_payment_kzt_cents';
COMMENT ON COLUMN quotes.financed_amount_kzt_cents IS 'Сумма финансирования в тиынах: стоимость машины в тиынах за вычетом первоначального взноса';
COMMENT ON COLUMN quotes.monthly_payment_kzt_cents IS
    'Регулярный ежемесячный платёж в тиынах: для аннуитета — постоянная сумма, для дифференцированного — первый платёж, самый большой. Хранится, а не вычисляется на лету: это результат формулы аннуитета, и после смены её реализации старое КП должно показывать прежнюю цифру';
COMMENT ON COLUMN quotes.last_payment_kzt_cents IS
    'Последний платёж в тиынах. Отличается от регулярного: он вбирает выкупную стоимость и накопленную разницу округления, а у дифференцированного графика ещё и заметно меньше первого';
COMMENT ON COLUMN quotes.total_paid_kzt_cents IS
    'Итого выплат в тиынах: сумма платежей графика плюс комиссия. Первоначальный взнос сюда НЕ входит — иначе переплата перестала бы быть переплатой; полные расходы клиента это взнос плюс это значение';
COMMENT ON COLUMN quotes.overpayment_kzt_cents IS 'Переплата в тиынах: всё сверх суммы финансирования, то есть проценты вместе с комиссией';
COMMENT ON COLUMN quotes.payment_schedule IS
    'Полный график: массив объектов {number, principal, interest, amount, balance} в тиынах. jsonb, а не отдельная таблица: запросов по отдельным строкам графика нет — он читается только целиком вместе с самим КП, и таблица дала бы join и лишние сущности без единого сценария, который их окупает';

COMMENT ON COLUMN quotes.author_id IS 'Менеджер, выпустивший КП; NULL — выпуск без участия человека';
COMMENT ON COLUMN quotes.expires_at IS 'Срок годности предложения: после него зафиксированный курс уже не отражает рынок';
COMMENT ON COLUMN quotes.first_opened_at IS 'Момент первого открытия ссылки клиентом; проставляется один раз и служит признаком, что КП дошло';
