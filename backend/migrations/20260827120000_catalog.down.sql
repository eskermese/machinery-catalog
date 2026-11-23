-- Откат каталога. Порядок обратный созданию: сначала таблицы, которые ссылаются,
-- затем те, на которые ссылаются. DROP TABLE снимает и индексы, и комментарии,
-- поэтому отдельных DROP INDEX здесь нет.
DROP TABLE IF EXISTS residual_value_curves;
DROP TABLE IF EXISTS machine_images;
DROP TABLE IF EXISTS machine_attribute_values;
DROP TABLE IF EXISTS machines;
DROP TABLE IF EXISTS attribute_options;
DROP TABLE IF EXISTS attributes;
DROP TABLE IF EXISTS categories;

-- Триггер уходит вместе с таблицей, функция — нет.
DROP FUNCTION IF EXISTS categories_reject_cycle();
