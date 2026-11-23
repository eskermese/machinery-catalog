-- Каталог спецтехники: дерево категорий, метаданные характеристик, карточки
-- машин, значения характеристик и фотографии.
--
-- Шесть таблиц в одной миграции намеренно: они образуют один связный граф
-- (categories → attributes → attribute_options, categories → machines →
-- machine_attribute_values / machine_images) и по отдельности бессмысленны.
-- Откат такого блока должен быть атомарным.

-- Триграммы нужны запасному пути поиска: опечатки («эскаватор») и склонения,
-- которых не берёт снежковый стеммер («катки» против «каток»). Расширение
-- входит в стандартную поставку образа postgres:17-alpine — проверено
-- по pg_available_extensions, ставить в образ нечего.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ---------------------------------------------------------------------------
-- categories
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS categories
(
    id         uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    parent_id  uuid REFERENCES categories (id) ON DELETE RESTRICT,
    slug       text        NOT NULL,
    name_ru    text        NOT NULL,
    name_kk    text        NOT NULL,
    name_en    text        NOT NULL,
    position   integer     NOT NULL DEFAULT 0,
    is_active  boolean     NOT NULL DEFAULT true,
    image_path text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT categories_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT categories_parent_not_self CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT categories_position_non_negative CHECK (position >= 0)
);

-- Слаг уникален глобально, а не в пределах родителя: адрес категории строится
-- как /catalog/<slug> без пути предков, поэтому перенос ветки дерева не должен
-- ломать внешние ссылки. Индекс обслуживает и открытие категории по URL.
CREATE UNIQUE INDEX IF NOT EXISTS categories_slug_uidx ON categories (slug);

-- Единственный индекс на таблицу: и меню сайта, и админский список читают
-- потомков узла в порядке отображения. Отдельный частичный индекс под
-- is_active не заводится — дерево категорий это десятки строк, отбор активных
-- дешевле дополнительной структуры.
CREATE INDEX IF NOT EXISTS categories_parent_position_idx ON categories (parent_id, position);

-- Защита от цикла в дереве: A → B → A.
--
-- CHECK категорию сам себе родителем не пропустит, но цепочку длиннее одного
-- звена он увидеть не может — предикат CHECK не ходит по другим строкам.
-- Рекурсивный обход вверх по parent_id — единственный способ поймать цикл
-- в момент записи.
--
-- Триггер, а не проверка в сервисе, выбран по цене ошибки, а не по надёжности:
-- цикл проявляется не отказом на записи, а зависанием рекурсивного обхода
-- дерева на построении меню каталога — то есть на публичном пути и с задержкой.
-- При этом запись в parent_id редкая (десятки категорий, единичные правки),
-- поэтому стоимость проверки в рантайме нулевая, а путей записи больше одного:
-- админка, будущий импорт каталога и ручной SQL при разборе инцидента —
-- ровно та ситуация, где ошибиться проще всего, а проверки в сервисе нет.
CREATE OR REPLACE FUNCTION categories_reject_cycle() RETURNS trigger AS
$$
DECLARE
    ancestor uuid := NEW.parent_id;
    hops     int  := 0;
BEGIN
    WHILE ancestor IS NOT NULL
        LOOP
            IF ancestor = NEW.id THEN
                RAISE EXCEPTION 'цикл в дереве категорий: % не может быть собственным предком', NEW.id
                    USING ERRCODE = 'check_violation';
            END IF;

            -- Страховка от зависания самого триггера, если цикл всё же попал
            -- в данные в обход этой проверки: обход обязан завершиться.
            hops := hops + 1;

            IF hops > 64 THEN
                RAISE EXCEPTION 'глубина дерева категорий превысила допустимую'
                    USING ERRCODE = 'check_violation';
            END IF;

            SELECT parent_id INTO ancestor FROM categories WHERE id = ancestor;
        END LOOP;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- UPDATE OF parent_id: переименование или скрытие категории обхода не требует.
-- WHEN (NEW.parent_id IS NOT NULL): у корня предков нет и искать нечего.
CREATE OR REPLACE TRIGGER categories_reject_cycle_trg
    BEFORE INSERT OR UPDATE OF parent_id
    ON categories
    FOR EACH ROW
    WHEN (NEW.parent_id IS NOT NULL)
EXECUTE FUNCTION categories_reject_cycle();

COMMENT ON TABLE categories IS 'Дерево категорий спецтехники: экскаваторы, погрузчики, автокраны и т.д. Отсутствие циклов обеспечивает триггер categories_reject_cycle_trg: CHECK ловит только A → A, а цепочку A → B → A увидеть не может';
COMMENT ON COLUMN categories.parent_id IS 'Родительский узел; NULL — корень. RESTRICT: категорию с потомками нельзя удалить незаметно';
COMMENT ON COLUMN categories.slug IS 'Идентификатор в URL, уникален по всему дереву — ссылка не зависит от положения узла';
COMMENT ON COLUMN categories.position IS 'Ручной порядок вывода; сортировка по имени не подходит — маркетинг задаёт порядок сам';
COMMENT ON COLUMN categories.is_active IS 'Скрытие ветки с сайта без удаления: у категории остаются машины и внешние ссылки';
COMMENT ON COLUMN categories.image_path IS 'Обложка раздела, выбранная вручную: путь в хранилище, не URL. NULL — обложку подставляет сервис по первой опубликованной машине поддерева, поэтому у непустого раздела она есть и без действий администратора';

-- ---------------------------------------------------------------------------
-- attributes
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS attributes
(
    id            uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    category_id   uuid        NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    code          text        NOT NULL,
    name_ru       text        NOT NULL,
    name_kk       text        NOT NULL,
    name_en       text        NOT NULL,
    data_type     text        NOT NULL,
    unit          text,
    is_filterable boolean     NOT NULL DEFAULT false,
    position      integer     NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT attributes_data_type_allowed CHECK (data_type IN ('number', 'string', 'bool', 'enum')),

    -- Код уникален в пределах категории, а не глобально: bucket_volume
    -- у экскаватора и у погрузчика — разные характеристики с разным смыслом
    -- и разными единицами, сводить их в одну запись нельзя.
    CONSTRAINT attributes_category_code_unique UNIQUE (category_id, code),
    CONSTRAINT attributes_code_format CHECK (code ~ '^[a-z0-9]+(_[a-z0-9]+)*$'),

    -- Мишень составного внешнего ключа из machine_attribute_values.
    -- Сам по себе избыточен (id уже первичный ключ), но без него нельзя
    -- сослаться на пару (id, data_type) и тем самым связать тип характеристики
    -- с заполненной колонкой значения.
    CONSTRAINT attributes_id_data_type_unique UNIQUE (id, data_type),

    -- Единица измерения осмысленна только у числовой характеристики:
    -- «тип привода: гусеничный, кВт» — ошибка ввода, а не редкий случай.
    CONSTRAINT attributes_unit_number_only CHECK (unit IS NULL OR data_type = 'number'),
    CONSTRAINT attributes_position_non_negative CHECK (position >= 0)
);

-- Отрисовка блока характеристик и панели фильтров читает все атрибуты
-- категории в порядке вывода — один запрос, один индекс. UNIQUE (category_id,
-- code) для него не годится: он упорядочен по коду, а не по position.
CREATE INDEX IF NOT EXISTS attributes_category_position_idx ON attributes (category_id, position);

COMMENT ON TABLE attributes IS 'Метаданные характеристик, привязанные к категории: у экскаватора объём ковша, у крана вылет стрелы';
COMMENT ON COLUMN attributes.category_id IS 'CASCADE: набор характеристик принадлежит категории и без неё не существует';
COMMENT ON COLUMN attributes.code IS 'Машинный ключ характеристики для API и фильтров; человеку показываются name_*';
COMMENT ON COLUMN attributes.data_type IS 'Определяет, какая из колонок значений заполняется в machine_attribute_values; связь обеспечена составным внешним ключом, а не соглашением';
COMMENT ON COLUMN attributes.unit IS 'Единица измерения для отображения (м³, т, кВт); в значении не хранится — оно всегда в этой единице';
COMMENT ON COLUMN attributes.is_filterable IS 'Выносить ли характеристику в панель фильтров: показывать все подряд бессмысленно';

-- ---------------------------------------------------------------------------
-- attribute_options
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS attribute_options
(
    id           uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    attribute_id uuid        NOT NULL REFERENCES attributes (id) ON DELETE CASCADE,
    code         text        NOT NULL,
    name_ru      text        NOT NULL,
    name_kk      text        NOT NULL,
    name_en      text        NOT NULL,
    position     integer     NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT attribute_options_attribute_code_unique UNIQUE (attribute_id, code),
    CONSTRAINT attribute_options_code_format CHECK (code ~ '^[a-z0-9]+(_[a-z0-9]+)*$'),
    CONSTRAINT attribute_options_position_non_negative CHECK (position >= 0)
);

-- Отдельного индекса нет: единственный запрос — «варианты этого атрибута»,
-- и его обслуживает UNIQUE (attribute_id, code). Вариантов у характеристики
-- единицы, сортировка по position идёт в памяти.

COMMENT ON TABLE attribute_options IS 'Допустимые варианты для характеристик с data_type = enum';
COMMENT ON COLUMN attribute_options.code IS 'Значение, которое попадает в machine_attribute_values.text_value';

-- ---------------------------------------------------------------------------
-- machines
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS machines
(
    id              uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    category_id     uuid        NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
    brand           text        NOT NULL,
    model           text        NOT NULL,
    slug            text        NOT NULL,
    name_ru         text        NOT NULL,
    name_kk         text        NOT NULL,
    name_en         text        NOT NULL,
    description_ru  text,
    description_kk  text,
    description_en  text,
    price_usd_cents bigint      NOT NULL,
    is_price_public boolean     NOT NULL DEFAULT true,
    availability    text        NOT NULL,
    delivery_weeks  integer,
    production_year integer,
    condition       text        NOT NULL,

    -- Величины для расчёта стоимости владения.
    --
    -- Колонками, а не характеристиками EAV, и это не удобство, а надёжность.
    -- Расчёт обязан находить их по имени у любой машины любого раздела,
    -- а код характеристики уникален В ПРЕДЕЛАХ категории: «fuel_consumption»
    -- пришлось бы объявить 44 раза, и опечатка в одном разделе молча выключила
    -- бы расчёт для всей его ветки. Хуже того, потомок может переобъявить код
    -- предка, а единица измерения у характеристики — свободный текст: ничто
    -- не мешает одному разделу писать л/мч, а другому л/100 км. Расчёт
    -- стоимости владения по чужой единице — ровно та ошибка, которую нельзя
    -- допускать: он выглядит как расчёт, а является выдумкой.
    --
    -- Все три допускают NULL, и это состояние первого класса: «не заведено»
    -- обязано доходить до клиента как «нет данных», а не как ноль.
    fuel_consumption_lph   numeric(8, 2),
    service_interval_hours integer,
    service_cost_usd_cents bigint,

    is_active       boolean     NOT NULL DEFAULT false,
    published_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),

    -- Текст, по которому ищут. Генерируемая колонка, а не выражение в индексе:
    -- по ней строятся ДВА индекса (полнотекстовый и триграммный), и повторять
    -- склейку пяти колонок в каждом значило бы завести два места, которые
    -- разойдутся при добавлении шестого поля.
    --
    -- Ровно те пять колонок, по которым поиск шёл и раньше. Описание сюда
    -- не входит намеренно: карточка с «экскаватор» в описании — это не
    -- экскаватор, и по такому совпадению выдача засоряется.
    search_text     text        NOT NULL GENERATED ALWAYS AS (
        name_ru || ' ' || name_kk || ' ' || name_en || ' ' || brand || ' ' || model
    ) STORED,

    CONSTRAINT machines_price_non_negative CHECK (price_usd_cents >= 0),
    CONSTRAINT machines_availability_allowed CHECK (availability IN ('in_stock', 'on_order')),
    CONSTRAINT machines_condition_allowed CHECK (condition IN ('new', 'used')),
    CONSTRAINT machines_delivery_weeks_positive CHECK (delivery_weeks IS NULL OR delivery_weeks > 0),
    CONSTRAINT machines_production_year_sane CHECK (production_year IS NULL OR production_year BETWEEN 1950 AND 2100),
    CONSTRAINT machines_fuel_consumption_positive CHECK (fuel_consumption_lph IS NULL OR fuel_consumption_lph > 0),
    CONSTRAINT machines_service_interval_positive CHECK (service_interval_hours IS NULL OR service_interval_hours > 0),
    CONSTRAINT machines_service_cost_non_negative CHECK (service_cost_usd_cents IS NULL OR service_cost_usd_cents >= 0),
    CONSTRAINT machines_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),

    -- Опубликованная карточка обязана иметь дату публикации: витрина
    -- сортируется по published_at, и NULL в ней означал бы машину,
    -- которая видна, но не встаёт в ленту.
    CONSTRAINT machines_active_requires_published_at CHECK (NOT is_active OR published_at IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS machines_slug_uidx ON machines (slug);

-- Админский список машин категории: все статусы, свежие сверху.
-- Он же обслуживает проверку ON DELETE RESTRICT при удалении категории —
-- частичные индексы ниже для этого не годятся.
CREATE INDEX IF NOT EXISTS machines_category_created_idx ON machines (category_id, created_at DESC);

-- Публичная витрина категории. Частичный, потому что каталог не показывает
-- неактивные карточки ни при каком фильтре.
--
-- Третьей колонкой идёт id, хотя сам по себе он в предикате не участвует.
-- Причина — keyset-пагинация: курсор сравнивает пару (published_at, id),
-- и без id в индексе база не может пройти выдачу подряд с нужной позиции,
-- а вынуждена отсортировать отобранное заново. Порядок сортировки колонки
-- обязан совпадать с ORDER BY запроса, иначе индекс для него бесполезен.
CREATE INDEX IF NOT EXISTS machines_public_catalog_idx
    ON machines (category_id, published_at DESC, id DESC)
    WHERE is_active;

-- Та же витрина в сортировке по цене — самый частый пользовательский порядок,
-- переиспользовать индекс по published_at для него нельзя.
--
-- Ключ сортировки — не сама колонка, а выражение: у карточки со скрытой ценой
-- оно даёт NULL. Скрытая цена не ноль и не бесконечность, её просто нет,
-- и место такой карточке в конце выдачи в ОБЕ стороны сортировки.
--
-- Отсюда два индекса, а не один. Btree читается в обе стороны, но обратный
-- проход переворачивает и размещение NULL: индекс с ASC NULLS LAST даёт назад
-- DESC NULLS FIRST, то есть не тот порядок, который нужен. Одним индексом
-- «NULLS LAST в обе стороны» невыразимо, а без второго убывающая сортировка
-- потеряла бы готовый порядок и читала бы категорию целиком с сортировкой.
CREATE INDEX IF NOT EXISTS machines_public_price_idx
    ON machines (category_id, (CASE WHEN is_price_public THEN price_usd_cents END), id)
    WHERE is_active;

CREATE INDEX IF NOT EXISTS machines_public_price_desc_idx
    ON machines (category_id,
                 (CASE WHEN is_price_public THEN price_usd_cents END) DESC NULLS LAST,
                 id DESC)
    WHERE is_active;

-- Полнотекстовый поиск: основной путь. GIN, а не GiST, — выбор по нагрузке:
-- каталог читают в разы чаще, чем правят, а GIN даёт заметно более быстрый
-- поиск ценой более дорогой вставки. Конфигурация russian зашита в индекс
-- и обязана совпадать с той, что стоит в запросе, иначе индекс не применится.
CREATE INDEX IF NOT EXISTS machines_search_fts_idx
    ON machines USING gin (to_tsvector('russian', search_text));

-- Триграммы: запасной путь. Тот же индекс обслуживает и похожесть
-- (word_similarity для опечаток), и подстроку (ILIKE '%…%' для «370»),
-- поэтому запасной этап целиком укладывается в один проход по нему.
CREATE INDEX IF NOT EXISTS machines_search_trgm_idx
    ON machines USING gin (search_text gin_trgm_ops);

COMMENT ON COLUMN machines.search_text IS 'Склейка названий, бренда и модели: по ней идёт полнотекстовый и триграммный поиск';

COMMENT ON TABLE machines IS 'Карточка единицы техники в каталоге';
COMMENT ON COLUMN machines.category_id IS 'RESTRICT: удаление категории с карточками должно быть осознанным переносом, а не каскадом';
COMMENT ON COLUMN machines.price_usd_cents IS 'Цена в центах USD. Прайс ведётся в долларах, тенговая цена считается по курсу на момент показа';
COMMENT ON COLUMN machines.is_price_public IS 'Скрыть цену за формой запроса: часть позиций продаётся «по запросу»';
COMMENT ON COLUMN machines.availability IS 'in_stock — на складе, on_order — под заказ; определяет, показывать ли срок поставки';
COMMENT ON COLUMN machines.delivery_weeks IS 'Срок поставки в неделях. NULL допустим и для on_order: срок не всегда известен на момент публикации';
COMMENT ON COLUMN machines.is_active IS 'Видимость карточки в публичном каталоге';
COMMENT ON COLUMN machines.published_at IS 'Момент первой публикации; задаёт порядок ленты и не сбрасывается при снятии с публикации';
COMMENT ON COLUMN machines.fuel_consumption_lph IS 'Расход топлива в литрах на моточас. NULL — не заведён: строка «топливо» в расчёте владения обязана сказать «нет данных», а не показать ноль';
COMMENT ON COLUMN machines.service_interval_hours IS 'Моточасов между плановыми ТО. NULL — не заведён';
COMMENT ON COLUMN machines.service_cost_usd_cents IS 'Стоимость одного планового ТО в центах USD, как и цена машины. NULL — не заведена';

-- ---------------------------------------------------------------------------
-- residual_value_curves
-- ---------------------------------------------------------------------------

-- Кривая падения остаточной стоимости техники, по разделу.
--
-- Отдельной таблицей, а не колонкой карточки, потому что это не свойство
-- машины: одна и та же модель через три года стоит по-разному в разные годы
-- и на разных рынках. В карточке такое значение устарело бы молча — цена
-- в ней меняется рукой менеджера, а доля остаточной стоимости не меняется
-- никогда, потому что о ней никто не вспомнит.
--
-- КРИВАЯ ЦЕЛИКОМ, А НЕ СТРОКА НА ГОД. Строка на год означала бы тридцать
-- чисел на раздел и больше тысячи на каталог — столько руками не заводят,
-- таблица осталась бы наполовину пустой, и расчёт прыгал бы между годами
-- без объяснимой причины. Кривая задаётся тремя числами, монотонна
-- по построению (перепутать пятый год с четвёртым нечем) и продолжается
-- на любой горизонт: спросили не пять лет, а десять — новых данных не нужно.
--
-- Три параметра, потому что двух мало: первый год техника теряет заметно
-- больше последующих, а чистая геометрия без порога на длинном горизонте
-- уводит остаточную стоимость почти в ноль, чего с техникой не происходит.
--
-- Ключ — раздел: экскаватор и автокран теряют в цене по-разному, а внутри
-- раздела кривая одна. Пустая таблица — законное состояние: пока кривые
-- не заведены, расчёт честно сообщает, что остаточной стоимости он не знает.
CREATE TABLE IF NOT EXISTS residual_value_curves
(
    category_id         uuid        NOT NULL,
    first_year_drop_bps integer     NOT NULL,
    annual_drop_bps     integer     NOT NULL,
    floor_bps           integer     NOT NULL,
    updated_at          timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT residual_value_curves_pkey PRIMARY KEY (category_id),
    CONSTRAINT residual_value_curves_category_fkey
        FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE CASCADE,
    CONSTRAINT residual_value_curves_first_year_sane CHECK (first_year_drop_bps BETWEEN 0 AND 10000),
    CONSTRAINT residual_value_curves_annual_sane CHECK (annual_drop_bps BETWEEN 0 AND 10000),
    CONSTRAINT residual_value_curves_floor_sane CHECK (floor_bps BETWEEN 0 AND 10000),
    -- Порог выше того, что остаётся после первого года, сделал бы кривую
    -- возрастающей: машина «дорожала» бы со второго года.
    CONSTRAINT residual_value_curves_floor_below_first
        CHECK (floor_bps <= 10000 - first_year_drop_bps)
);

COMMENT ON TABLE residual_value_curves IS 'Кривая остаточной стоимости раздела: доля цены, которую техника сохраняет к концу N-го года';
COMMENT ON COLUMN residual_value_curves.first_year_drop_bps IS 'Сколько техника теряет за первый год, в базисных пунктах: 2000 = 20%';
COMMENT ON COLUMN residual_value_curves.annual_drop_bps IS 'Сколько теряется за каждый следующий год ОТ ОСТАТКА, в базисных пунктах: 1200 = 12% в год';
COMMENT ON COLUMN residual_value_curves.floor_bps IS 'Ниже этой доли стоимость не опускается: геометрия без порога уводит цену техники почти в ноль, чего не происходит';

-- ---------------------------------------------------------------------------
-- machine_attribute_values
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS machine_attribute_values
(
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    machine_id   uuid NOT NULL REFERENCES machines (id) ON DELETE CASCADE,
    attribute_id uuid NOT NULL,
    data_type    text NOT NULL,
    num_value    numeric(14, 3),
    text_value   text,
    bool_value   boolean,

    CONSTRAINT machine_attribute_values_machine_attribute_unique UNIQUE (machine_id, attribute_id),

    -- Ровно одна колонка значения заполнена. Без этого строка с двумя
    -- значениями читается по-разному в зависимости от того, какой тип
    -- запросил вызывающий код.
    CONSTRAINT machine_attribute_values_exactly_one_value CHECK (
        (num_value IS NOT NULL)::int
            + (text_value IS NOT NULL)::int
            + (bool_value IS NOT NULL)::int = 1
        ),

    -- Заполненная колонка обязана соответствовать типу характеристики.
    -- Вместе с внешним ключом ниже это делает рассогласование невозможным
    -- на уровне БД, а не только в коде сервиса.
    CONSTRAINT machine_attribute_values_type_matches_column CHECK (
        (data_type = 'number' AND num_value IS NOT NULL)
            OR (data_type = 'string' AND text_value IS NOT NULL)
            OR (data_type = 'enum' AND text_value IS NOT NULL)
            OR (data_type = 'bool' AND bool_value IS NOT NULL)
        ),

    -- Составной внешний ключ на (id, data_type) характеристики.
    --
    -- ON UPDATE намеренно не CASCADE, а NO ACTION: смена типа у характеристики,
    -- у которой уже есть заполненные значения, обязана отказать. Каскад молча
    -- переписал бы data_type в строках значений, оставив число в num_value
    -- при новом типе 'string' — и следующая же проверка CHECK сломалась бы
    -- на строке, которую никто не трогал.
    CONSTRAINT machine_attribute_values_attribute_typed_fkey
        FOREIGN KEY (attribute_id, data_type) REFERENCES attributes (id, data_type)
            ON DELETE CASCADE
);

-- Диапазонные фильтры каталога («мощность от и до») — единственная причина,
-- по которой характеристики хранятся EAV, а не в jsonb. Индекс частичный:
-- строки со строковыми и булевыми значениями в диапазонный поиск не попадают.
CREATE INDEX IF NOT EXISTS machine_attribute_values_num_filter_idx
    ON machine_attribute_values (attribute_id, num_value)
    WHERE num_value IS NOT NULL;

-- Фильтр по варианту enum и по строковому значению («тип привода = гусеничный»).
CREATE INDEX IF NOT EXISTS machine_attribute_values_text_filter_idx
    ON machine_attribute_values (attribute_id, text_value)
    WHERE text_value IS NOT NULL;

COMMENT ON TABLE machine_attribute_values IS
    'Значения характеристик машины (EAV). Выбран EAV, а не jsonb: каталогу нужны диапазонные фильтры по числовым характеристикам, и на объёме в сотни машин join по индексу дешевле разбора jsonb в каждой строке. Осознанно не закрыто на уровне БД: значение характеристики типа enum лежит в text_value как код варианта и внешнего ключа на attribute_options не имеет — колонок значений ровно три по форме таблицы, и ссылку на справочник вариантов из них не выразить. Существование кода варианта проверяет сервис';
COMMENT ON COLUMN machine_attribute_values.data_type IS
    'Тип характеристики, денормализованный из attributes. Хранится не ради чтения, а ради ссылочной целостности: только имея тип в самой строке значения, можно составным внешним ключом на attributes (id, data_type) связать тип с заполненной колонкой. Пишется тем же значением, что и в attributes; расхождение отвергнет внешний ключ';
COMMENT ON COLUMN machine_attribute_values.num_value IS 'Значение числовой характеристики в единице, объявленной в attributes.unit';
COMMENT ON COLUMN machine_attribute_values.text_value IS 'Строковое значение; для data_type = enum сюда пишется attribute_options.code';
COMMENT ON COLUMN machine_attribute_values.bool_value IS 'Значение флаговой характеристики';

-- ---------------------------------------------------------------------------
-- machine_images
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS machine_images
(
    id         uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    machine_id uuid        NOT NULL REFERENCES machines (id) ON DELETE CASCADE,
    path       text        NOT NULL,
    position   integer     NOT NULL DEFAULT 0,
    is_primary boolean     NOT NULL DEFAULT false,
    alt_ru     text,
    alt_kk     text,
    alt_en     text,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT machine_images_position_non_negative CHECK (position >= 0)
);

-- Не больше одного главного фото на машину. Частичный уникальный индекс —
-- единственный способ выразить это без триггера: обычный UNIQUE по
-- (machine_id, is_primary) запретил бы и вторую неглавную фотографию.
CREATE UNIQUE INDEX IF NOT EXISTS machine_images_primary_uidx
    ON machine_images (machine_id)
    WHERE is_primary;

-- Галерея карточки: все фото машины в порядке показа.
CREATE INDEX IF NOT EXISTS machine_images_gallery_idx ON machine_images (machine_id, position);

COMMENT ON TABLE machine_images IS 'Фотографии машины; главная используется в списках каталога, остальные — в галерее карточки';
COMMENT ON COLUMN machine_images.path IS 'Путь в файловом хранилище, не URL: базовый адрес выдачи задаётся конфигурацией и меняется независимо';
COMMENT ON COLUMN machine_images.is_primary IS 'Главное фото. Индекс machine_images_primary_uidx гарантирует не больше одного на машину; «хотя бы одно» проверяет сервис';
