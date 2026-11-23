/**
 * @generated openapi-typescript
 *
 * Файл собран из снимка контракта openapi/swagger.json.
 * РУКАМИ НЕ ПРАВИТЬ: изменения затрёт следующий `npm run gen:api`,
 * а `npm run check:api` уронит сборку на расхождении.
 *
 * Обновить: npm run sync:spec && npm run gen:api
 */

/* eslint-disable */
export interface paths {
    "/v1/admin/catalog/attributes": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Создание характеристики
         * @description Создаёт характеристику вместе с набором вариантов enum
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.AttributeSave"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Attribute"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/attributes/{attributeID}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /**
         * Изменение характеристики
         * @description Заменяет характеристику вместе с набором вариантов enum. Смена типа у характеристики с заполненными значениями отвергается
         */
        put: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор характеристики */
                    attributeID: string;
                };
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.AttributeSave"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Attribute"];
                    };
                };
                /** @description Код ошибки: например, смена типа у заполненной характеристики */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Характеристика не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        post?: never;
        /**
         * Удаление характеристики
         * @description Необратимая операция: каскад схемы уносит варианты и заполненные значения всех карточек категории
         */
        delete: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор характеристики */
                    attributeID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description Характеристика удалена */
                204: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Характеристика не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/categories": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Категории (админка)
         * @description Полное дерево категорий, включая скрытые с сайта
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.CategoryList"];
                    };
                };
                /** @description Запрос не аутентифицирован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        /**
         * Создание категории
         * @description Занятый слаг возвращает 409
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.CategorySave"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Category"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Слаг уже занят */
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/categories/{categoryID}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /**
         * Изменение категории
         * @description Занятый слаг возвращает 409, попытка сделать категорию собственным предком — 400
         */
        put: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор категории */
                    categoryID: string;
                };
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.CategorySave"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Category"];
                    };
                };
                /** @description Код ошибки: например, цикл в дереве */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Категория не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Слаг уже занят */
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        post?: never;
        /**
         * Удаление категории
         * @description Необратимая операция: каскад схемы уносит характеристики категории и заполненные значения всех её карточек. Категория с потомками или карточками не удаляется
         */
        delete: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор категории */
                    categoryID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description Категория удалена */
                204: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
                /** @description У категории есть потомки или карточки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Категория не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/categories/{categoryID}/attributes": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Характеристики категории (админка)
         * @description Характеристики вместе с вариантами для типа enum
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор категории */
                    categoryID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.AttributeList"];
                    };
                };
                /** @description Запрос не аутентифицирован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/machines": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Создание карточки
         * @description Карточка, её характеристики и фотографии пишутся одной транзакцией. Занятый слаг возвращает 409
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.MachineSave"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Machine"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Слаг уже занят */
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/machines/search": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Поиск по каталогу (админка)
         * @description Поиск по всему каталогу, включая неопубликованные карточки. Цена отдаётся всегда, даже когда скрыта на витрине. Отбор isActive показывает только опубликованные или только черновики; query ищет подстроку в названии, бренде и модели
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.MachineSearch"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.MachinePage"];
                    };
                };
                /** @description Код ошибки: невалидный фильтр или испорченный курсор */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/machines/{machineID}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /**
         * Изменение карточки
         * @description Заменяет карточку целиком вместе с характеристиками и фотографиями — одной транзакцией. Занятый другим слаг возвращает 409
         */
        put: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки */
                    machineID: string;
                };
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.MachineSave"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Machine"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Слаг уже занят */
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        post?: never;
        /**
         * Удаление карточки
         * @description Необратимая операция: каскад схемы уносит значения характеристик и фотографии карточки, а файлы снимков удаляются, если на них не ссылается другая карточка. Карточка с выпущенным КП не удаляется — её снимают с публикации
         */
        delete: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки */
                    machineID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description Карточка удалена */
                204: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description По карточке выпущены коммерческие предложения */
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/machines/{machineID}/images": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Фотографии карточки
         * @description Фотографии в порядке показа
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки */
                    machineID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.MachineImageList"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        /**
         * Замена фотографий
         * @description Заменяет набор целиком: пустой набор очищает галерею. Главной может быть не более одной фотографии
         */
        put: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки */
                    machineID: string;
                };
                cookie?: never;
            };
            /** @description Набор фотографий */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.MachineImagesReplace"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.MachineImageList"];
                    };
                };
                /** @description Код ошибки: например, две главные фотографии */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/machines/{machineID}/images/{imageID}/primary": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Назначение главного фото
         * @description Снимает признак с остальных фотографий одним запросом: состояния с двумя главными не существует
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки */
                    machineID: string;
                    /** @description Идентификатор фотографии */
                    imageID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.MachineImageList"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка или фотография не найдены */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/machines/{machineID}/publish": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Публикация карточки
         * @description Делает карточку видимой на витрине и проставляет дату первой публикации. Повторная публикация дату не сбрасывает: она задаёт место в ленте
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки */
                    machineID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Machine"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/machines/{machineID}/unpublish": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Снятие карточки с публикации
         * @description Скрывает карточку с витрины, сохраняя дату первой публикации
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки */
                    machineID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Machine"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/machines/{slug}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Карточка техники (админка)
         * @description Любая карточка, включая неопубликованную, вместе с характеристиками и фотографиями
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки в URL */
                    slug: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Machine"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/residual-curves": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Кривые остаточной стоимости
         * @description Все заведённые кривые вместе с посчитанными по ним долями по годам. Разделы без своей кривой сюда не попадают — они наследуют кривую предка
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.ResidualValueCurveList"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/catalog/residual-curves/{categoryID}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Кривая остаточной стоимости раздела
         * @description Кривая, заведённая ИМЕННО для этого раздела. Вверх по дереву не поднимается: справочник показывает своё, а не унаследованное, иначе редактор принял бы кривую предка за собственную и завёл бы её копию
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор раздела */
                    categoryID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.ResidualValueCurve"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description У раздела нет своей кривой */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        /**
         * Сохранение кривой остаточной стоимости
         * @description Кривая задаётся тремя числами целиком, а не строкой на каждый год: тридцать чисел на раздел руками не заводят. Заводит кривую раздела или переписывает её — у раздела она ровно одна. Возвращает кривую вместе с посчитанными долями по годам
         */
        put: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор раздела */
                    categoryID: string;
                };
                cookie?: never;
            };
            /** @description Кривая остаточной стоимости */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.ResidualValueCurveSave"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.ResidualValueCurve"];
                    };
                };
                /** @description Код ошибки: доли вне границ или порог выше остатка первого года */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Раздел не найден */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        post?: never;
        /**
         * Удаление кривой остаточной стоимости
         * @description После удаления раздел наследует кривую предка, а если её нет ни у кого — расчёт владения снова честно скажет, что остаточной стоимости не знает
         */
        delete: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор раздела */
                    categoryID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description No Content */
                204: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description У раздела нет своей кривой */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/fx/markups": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * История наценок
         * @description Наценки пары, свежие первыми. Действующая — первая, у которой момент вступления не в будущем
         */
        get: {
            parameters: {
                query: {
                    /** @description Валюта, из которой считаем */
                    base: string;
                    /** @description Валюта, в которую считаем */
                    quote: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.FXMarkupList"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Не авторизован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Недостаточно прав */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        /**
         * Задать наценку
         * @description Наценка своя у каждой пары. История сохраняется: прежняя запись остаётся и объясняет цену в ранее выданных предложениях
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            /** @description Наценка */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.FXMarkupSave"];
                };
            };
            responses: {
                /** @description Created */
                201: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.FXMarkup"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Не авторизован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Недостаточно прав */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/fx/rates": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * История ручных курсов
         * @description Курсы, введённые человеком, свежие первыми. Загруженные воркером сюда не попадают: их история — это история источника, а не решений администратора
         */
        get: {
            parameters: {
                query: {
                    /** @description Валюта, из которой считаем */
                    base: string;
                    /** @description Валюта, в которую считаем */
                    quote: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.FXRateList"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Не авторизован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        /**
         * Ввести курс вручную
         * @description Ручной курс заменяет курс внешнего источника за ту же дату. Повторный ввод за ту же дату — исправление опечатки: создаётся новая запись, прежняя остаётся в истории
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            /** @description Курс */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.FXRateSave"];
                };
            };
            responses: {
                /** @description Created */
                201: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.FXRate"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Не авторизован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/fx/settings": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Настройки показа цены */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.PriceSettings"];
                    };
                };
                /** @description Не авторизован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Недостаточно прав */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
            };
        };
        /**
         * Изменить настройки показа цены
         * @description Доллар считается из юаня по паре CNY→USD со своей наценкой, а не хранится отдельно. Включение без заданного курса этой пары оставит цену в долларах пустой — состояние курса видно в ручке состояния
         */
        put: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            /** @description Настройки */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.PriceSettingsSave"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.PriceSettings"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Не авторизован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Недостаточно прав */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/fx/status": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Состояние курсов
         * @description Действующий курс каждой пары, что предложил внешний источник, что ввёл человек, признак устаревания. Пара второго отображения приходит всегда — по ней видно, готова ли витрина показывать цену в долларах
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.FXStatusList"];
                    };
                };
                /** @description Не авторизован */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/leads": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Заявки
         * @description Заявки с фильтром по статусу и ответственному менеджеру, свежие сверху. Keyset-пагинация: курсор следующей страницы приходит в ответе. Общее число считается только по запросу
         */
        get: {
            parameters: {
                query?: {
                    /** @description Статус: new, in_progress, won, lost */
                    status?: string;
                    /** @description Ответственный менеджер */
                    managerID?: string;
                    /** @description Размер страницы */
                    limit?: number;
                    /** @description Курсор следующей страницы из предыдущего ответа */
                    cursor?: string;
                    /** @description Посчитать общее число найденных заявок */
                    withTotal?: boolean;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.LeadList"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/leads/{leadID}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Заявка
         * @description Заявка по идентификатору
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор заявки */
                    leadID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Lead"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Заявка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        /**
         * Удаление заявки
         * @description Удаляет заявку вместе с контактом клиента, без следа. Существует ради требования человека убрать свои персональные данные — статусом такое не исполняется. Выпущенное по заявке предложение удаление переживает: обязательство перед клиентом не исчезает оттого, что прибрали заявку. Предложения, в отличие от заявок, не удаляются вовсе — они истекают
         */
        delete: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор заявки */
                    leadID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description No Content */
                204: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Заявка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/leads/{leadID}/assign": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Назначение ответственного
         * @description Назначает менеджера, ответственного за заявку. Не зависит от статуса: распределение — вопрос загрузки людей, а не стадии обработки
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор заявки */
                    leadID: string;
                };
                cookie?: never;
            };
            /** @description Ответственный менеджер */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.LeadAssign"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Lead"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Заявка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/leads/{leadID}/comment": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Комментарий менеджера к заявке
         * @description Заменяет заметку менеджера целиком; пустая строка её стирает. Сообщение клиента (message) этой ручкой не меняется — это разные поля, и правка менеджера не должна затирать запрос клиента
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор заявки */
                    leadID: string;
                };
                cookie?: never;
            };
            /** @description Комментарий менеджера */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.LeadComment"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Lead"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Заявка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/leads/{leadID}/status": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Смена статуса заявки
         * @description Переводит заявку по таблице переходов new → in_progress → won|lost. Недопустимый переход отвергается
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор заявки */
                    leadID: string;
                };
                cookie?: never;
            };
            /** @description Целевой статус */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.LeadStatusChange"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Lead"];
                    };
                };
                /** @description Недопустимый переход статуса */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Заявка не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/leasing-programs": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Программы лизинга (админка)
         * @description Все программы, включая недействующие: выключенную нужно видеть, чтобы включить обратно
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.LeasingProgramList"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        /**
         * Сохранение программы лизинга
         * @description Создаёт программу при пустом id и обновляет при заполненном. Согласованность границ между собой проверяет домен
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            /** @description Программа лизинга */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.LeasingProgramSave"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.LeasingProgram"];
                    };
                };
                /** @description Код ошибки: границы программы несогласованы или невалидный вход */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Программа не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/leasing-programs/{programID}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /**
         * Удаление программы лизинга
         * @description Программу, по которой уже выпущены предложения, удалить нельзя — её выключают флагом isActive
         */
        delete: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор программы */
                    programID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description No Content */
                204: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content?: never;
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Операция доступна только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Программа не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description По программе выпущены предложения */
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/media/categories": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Загрузка обложки раздела каталога
         * @description Принимает изображение JPEG, PNG или WebP не больше 5 МиБ. Тип определяется по содержимому файла, имя генерирует сервер. Путь из ответа кладётся в imagePath категории; без него обложку подставляет сервис по первой опубликованной машине поддерева
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["postV1AdminMediaCategories"];
            responses: {
                /** @description Created */
                201: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.UploadedFile"];
                    };
                };
                /** @description Код ошибки: файл не передан, больше предела или неподдерживаемого типа */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/media/machines": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Загрузка фотографии карточки
         * @description Принимает изображение JPEG, PNG или WebP не больше 5 МиБ. Тип определяется по содержимому файла, имя генерирует сервер. Возвращает путь и адрес в том же виде, что и фотография карточки
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["postV1AdminMediaCategories"];
            responses: {
                /** @description Created */
                201: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.UploadedFile"];
                    };
                };
                /** @description Код ошибки: файл не передан, больше предела или неподдерживаемого типа */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/media/programs": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Загрузка логотипа программы лизинга
         * @description Принимает изображение JPEG, PNG или WebP не больше 5 МиБ. Тип определяется по содержимому файла, имя генерирует сервер
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["postV1AdminMediaCategories"];
            responses: {
                /** @description Created */
                201: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.UploadedFile"];
                    };
                };
                /** @description Код ошибки: файл не передан, больше предела или неподдерживаемого типа */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/quotes": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Мои коммерческие предложения
         * @description Предложения, выпущенные текущим пользователем, свежие сверху
         */
        get: {
            parameters: {
                query?: {
                    /** @description Размер страницы */
                    limit?: number;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.QuoteList"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        /**
         * Выпуск коммерческого предложения
         * @description Читает опубликованную карточку, фиксирует курс, проверяет параметры против границ программы лизинга и считает график. Возвращает предложение вместе с токеном публичной ссылки
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            /** @description Параметры предложения */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.QuoteIssue"];
                };
            };
            responses: {
                /** @description Created */
                201: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.QuoteIssued"];
                    };
                };
                /** @description Код ошибки: параметры вне границ программы или невалидный вход */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка или программа не найдены */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/admin/quotes/{quoteID}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Коммерческое предложение (админка)
         * @description Предложение по внутреннему идентификатору. Отметку первого открытия не ставит
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор предложения */
                    quoteID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Quote"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Роль не имеет доступа к разделу */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Предложение не найдено */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/auth/admin/token": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Токен администратора
         * @description Выпускает токен только для заранее заведённого администратора (ADMIN_PHONES)
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.AuthTokenIssue"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.AuthToken"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Номер не заведён как администратор */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/auth/manager/token": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Токен менеджера
         * @description Выпускает токен по номеру телефона. Учётная запись создаётся при первом входе
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.AuthTokenIssue"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.AuthToken"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/auth/me": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Своя учётная запись
         * @description Учётная запись, от имени которой действует токен. Читается из хранилища, а не собирается из токена: запись могла быть удалена после его выпуска
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.AuthUser"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Учётная запись не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        /**
         * Смена своего имени
         * @description Задаёт имя для показа в админке. Только своё: имя не участвует в правах, а знает его сам сотрудник — вход идёт по номеру, форм регистрации нет. Пустая строка очищает имя
         */
        patch: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            /** @description Имя */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.AuthUserRename"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.AuthUser"];
                    };
                };
                /** @description Код ошибки: имя слишком длинное */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Учётная запись не найдена */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        trace?: never;
    };
    "/v1/auth/staff": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Сотрудники
         * @description Учётные записи сотрудников в порядке (роль, телефон). Только для администратора
         */
        get: {
            parameters: {
                query?: {
                    /** @description Роль: manager или admin */
                    role?: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.AuthUserList"];
                    };
                };
                /** @description Неизвестная роль */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Токен не передан, испорчен или истёк */
                401: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Список сотрудников доступен только администратору */
                403: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/catalog/categories": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Дерево категорий
         * @description Плоский список категорий с parent_id в порядке (родитель, position). Дерево клиент собирает сам одним проходом
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.CategoryList"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/catalog/categories/{categoryID}/attributes": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Характеристики категории
         * @description Набор характеристик для панели фильтров вместе с вариантами для типа enum. Включает характеристики, объявленные у родительских категорий: они действуют и в дочерних. Объявление в самой категории затеняет одноимённое у предка. У числовых приходят границы значений в разделе: из них строится подсказка под полем фильтра
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /**
                     * @description Идентификатор категории
                     * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
                     */
                    categoryID: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.AttributeList"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/catalog/machines/search": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Поиск по каталогу
         * @description Поиск опубликованных карточек с фильтрами, сортировкой и keyset-пагинацией. Черновики не возвращаются ни при каких параметрах, включая isActive — на витрине он игнорируется. Поиск query ищет подстроку в названии, бренде и модели. Цена в тенге приходит вместе с датой курса; при отсутствии курса обоих полей нет, а выдача работает как обычно
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["form.MachineSearch"];
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.MachinePage"];
                    };
                };
                /** @description Код ошибки: невалидный фильтр или испорченный курсор */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/catalog/machines/{slug}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Карточка техники
         * @description Опубликованная карточка вместе с характеристиками и фотографиями. Неопубликованная отдаётся как несуществующая. Цена в тенге приходит вместе с датой курса; при отсутствии курса обоих полей нет
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /**
                     * @description Идентификатор карточки в URL
                     * @example xcmg-xe215c
                     */
                    slug: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Machine"];
                    };
                };
                /** @description Карточка не найдена или не опубликована */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/leads": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Заявка с сайта
         * @description Принимает обратный звонок или запрос цены. Номер нормализуется. Слишком частые заявки с одного номера отклоняются с кодом 409
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            /** @description Заявка */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.LeadCreate"];
                };
            };
            responses: {
                /** @description Created */
                201: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.LeadAccepted"];
                    };
                };
                /** @description Код ошибки */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Заявка с этого номера уже принята */
                409: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/leasing-programs": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Программы лизинга
         * @description Действующие программы с границами, внутри которых считается платёж
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.LeasingProgramList"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/leasing/calculate": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Расчёт лизинга
         * @description Справочный расчёт по опубликованной карточке. Цену и курс сервер берёт сам: цену по слагу, курс — действующий. Ничего не сохраняется и не фиксируется. Машина со скрытой ценой отвергается. Частота обращений с одного адреса ограничена
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            /** @description Параметры расчёта */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.LeasingCalculate"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.LeasingCalculation"];
                    };
                };
                /** @description Код ошибки: параметры вне границ программы, цена скрыта или невалидный вход */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка, программа или курс не найдены */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Превышена частота обращений с адреса */
                429: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/machines/{slug}/tco": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Стоимость владения
         * @description Разложенная стоимость владения за срок. Расход топлива, интервал и стоимость ТО, остаточная стоимость берутся по карточке; чего не хватило — в поле missing
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Идентификатор карточки в URL */
                    slug: string;
                };
                cookie?: never;
            };
            /** @description Параметры расчёта */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.TCOCalculate"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.TCOResult"];
                    };
                };
                /** @description Код ошибки: цена скрыта или невалидный вход */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Карточка или курс не найдены */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Превышена частота обращений с адреса */
                429: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/quotes/{token}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Коммерческое предложение по ссылке
         * @description Предложение вместе с машиной, зафиксированным курсом, параметрами расчёта и полным графиком платежей. Первое открытие фиксируется. Истёкшее предложение возвращается с признаком isExpired
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Токен публичной ссылки */
                    token: string;
                };
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.Quote"];
                    };
                };
                /** @description Предложение не найдено */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/v1/quotes/{token}/recalculate": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Пересчёт предложения на других условиях
         * @description Считает график на новых условиях по зафиксированному в предложении курсу. Ничего не сохраняет и выпущенное предложение не меняет
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path: {
                    /** @description Публичный токен предложения */
                    token: string;
                };
                cookie?: never;
            };
            /** @description Новые условия */
            requestBody: {
                content: {
                    "application/json": components["schemas"]["form.QuoteRecalculate"];
                };
            };
            responses: {
                /** @description OK */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["presenter.LeasingCalculation"];
                    };
                };
                /** @description Код ошибки: параметры вне границ программы или невалидный вход */
                400: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Предложение не найдено */
                404: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Превышена частота обращений с адреса */
                429: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse400"];
                    };
                };
                /** @description Внутренняя ошибка сервера */
                500: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": components["schemas"]["swagger.HTTPResponse500"];
                    };
                };
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        "form.AttributeFilter": {
            /**
             * @description AttributeID характеристика, к которой применяется условие.
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            attributeID: string;
            /**
             * @description Codes допустимые коды варианта; несколько означают «любой из».
             * @example [
             *       "crawler"
             *     ]
             */
            codes?: string[];
            /**
             * @description Flag требуемое значение флаговой характеристики.
             * @example true
             */
            flag?: boolean;
            /**
             * @description NumberMax верхняя граница числовой характеристики включительно.
             * @example 250.5
             */
            numberMax?: string;
            /**
             * @description NumberMin нижняя граница числовой характеристики включительно.
             * @example 150
             */
            numberMin?: string;
        };
        "form.AttributeOptionSave": {
            /**
             * @description Code код варианта.
             * @example crawler
             */
            code: string;
            /**
             * @description NameEN название на английском.
             * @example Crawler
             */
            nameEN?: string;
            /**
             * @description NameKK название на казахском.
             * @example Шынжыр табанды
             */
            nameKK?: string;
            /**
             * @description NameRU название на русском.
             * @example Гусеничный
             */
            nameRU: string;
            /**
             * @description Position порядок вывода.
             * @example 1
             */
            position?: number;
        };
        "form.AttributeSave": {
            /**
             * @description CategoryID категория, которой принадлежит характеристика.
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            categoryID: string;
            /**
             * @description Code машинный ключ характеристики.
             * @example engine_power
             */
            code: string;
            /**
             * @description DataType тип значения: number, string, bool или enum.
             * @example number
             * @enum {string}
             */
            dataType: "number" | "string" | "bool" | "enum";
            /**
             * @description IsFilterable выносить ли характеристику в панель фильтров.
             * @example true
             */
            isFilterable?: boolean;
            /**
             * @description NameEN название на английском.
             * @example Engine power
             */
            nameEN?: string;
            /**
             * @description NameKK название на казахском.
             * @example Қуаты
             */
            nameKK?: string;
            /**
             * @description NameRU название на русском.
             * @example Мощность
             */
            nameRU: string;
            /** @description Options варианты для типа enum. */
            options?: components["schemas"]["form.AttributeOptionSave"][];
            /**
             * @description Position порядок вывода.
             * @example 1
             */
            position?: number;
            /**
             * @description Unit единица измерения; допустима только у числовой характеристики.
             * @example кВт
             */
            unit?: string;
        };
        "form.AuthTokenIssue": {
            /**
             * @description Номер телефона
             * @example +77001234567
             */
            phone: string;
        };
        "form.AuthUserRename": {
            /**
             * @description Name имя для показа; пустая строка очищает имя.
             * @example Иван Петров
             */
            name?: string;
        };
        "form.CategorySave": {
            /**
             * @description ImagePath обложка раздела: путь в хранилище из ответа
             *     POST /v1/admin/media/categories. Пусто — обложку подставит сервис
             *     по первой опубликованной машине поддерева.
             * @example categories/7c9e6679.jpg
             */
            imagePath?: string;
            /**
             * @description IsActive видимость ветки на сайте.
             * @example true
             */
            isActive?: boolean;
            /** @example Excavators */
            nameEN?: string;
            /**
             * @description NameKK и NameEN названия на казахском и английском.
             *
             *     НЕОБЯЗАТЕЛЬНЫ, в отличие от русского. Русский — источник правды:
             *     витрина показывает его, пока перевода нет. Требовать перевод здесь
             *     значило бы запретить заводить раздел до того, как его перевели,
             *     а заглушка вместо перевода хуже пустоты — витрина не отличит её
             *     от настоящего названия.
             * @example Экскаваторлар
             */
            nameKK?: string;
            /**
             * @description NameRU название на русском.
             * @example Экскаваторы
             */
            nameRU: string;
            /**
             * @description ParentID родительская категория; пусто — корень.
             * @example
             */
            parentID?: string;
            /**
             * @description Position порядок вывода.
             * @example 1
             */
            position?: number;
            /**
             * @description Slug идентификатор в URL.
             * @example excavators
             */
            slug: string;
        };
        "form.FXMarkupSave": {
            /**
             * @description BaseCurrency валюта, из которой считаем.
             * @example CNY
             */
            baseCurrency: string;
            /** @description EffectiveFrom момент вступления в силу; пусто — сейчас. */
            effectiveFrom?: string;
            /**
             * @description MarkupBps наценка в базисных пунктах: 200 это 2 %.
             *
             *     Базисные пункты, а не проценты дробью: целые участвуют в арифметике
             *     без ошибки округления, и наценка оказывается в той же единице,
             *     что ставка лизинга.
             * @example 200
             */
            markupBps?: number;
            /**
             * @description QuoteCurrency валюта, в которую считаем.
             * @example KZT
             */
            quoteCurrency: string;
        };
        "form.FXRateSave": {
            /**
             * @description BaseCurrency валюта, из которой считаем.
             * @example CNY
             */
            baseCurrency: string;
            /**
             * @description QuoteCurrency валюта, в которую считаем.
             * @example KZT
             */
            quoteCurrency: string;
            /**
             * @description Rate курс строкой: сколько единиц котируемой валюты за одну базовую.
             *
             *     Строкой, а не числом: JSON-число прошло бы через двоичную плавающую
             *     точку и вернулось бы с хвостом в младшем разряде — а из этого курса
             *     считается цена всего каталога.
             * @example 72.55
             */
            rate: string;
            /**
             * @description RateDate дата, на которую действует курс, в формате ГГГГ-ММ-ДД.
             *     Пусто — сегодня.
             * @example 2026-09-01
             */
            rateDate?: string;
        };
        "form.LeadAssign": {
            /**
             * @description ManagerID ответственный менеджер.
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            managerID: string;
        };
        "form.LeadComment": {
            /**
             * @description Comment заметка менеджера по заявке.
             * @example Перезвонить после обеда
             */
            comment?: string;
        };
        "form.LeadCreate": {
            /**
             * @description City город.
             * @example Алматы
             */
            city?: string;
            /**
             * @description MachineID машина, из карточки которой пришла заявка.
             * @example
             */
            machineID?: string;
            /**
             * @description Message что написал клиент: какая деталь нужна, к какой технике,
             *     что случилось. Необязательное.
             *
             *     Отдельно от комментария менеджера: то поле принадлежит ему, и одно
             *     на двоих означало бы, что его правка затирает запрос клиента.
             * @example Нужен масляный фильтр на XE215C
             */
            message?: string;
            /**
             * @description Name имя клиента.
             * @example Иван
             */
            name: string;
            /**
             * @description Phone номер телефона в любом написании: нормализуется сервисом.
             * @example +7 (700) 111-22-33
             */
            phone: string;
            /**
             * @description QuoteID коммерческое предложение, по которому обратились.
             * @example
             */
            quoteID?: string;
            /**
             * @description Source откуда пришла заявка: страница, форма или рекламная метка.
             * @example machine_card
             */
            source: string;
        };
        "form.LeadStatusChange": {
            /**
             * @description Status целевой статус.
             * @example in_progress
             * @enum {string}
             */
            status: "new" | "in_progress" | "won" | "lost";
        };
        "form.LeasingCalculate": {
            /**
             * @description CommissionKztCents разовая комиссия в тиынах.
             * @example 0
             */
            commissionKztCents?: number;
            /**
             * @description DownPaymentBps первоначальный взнос в базисных пунктах: 20.00% это 2000.
             * @example 2000
             */
            downPaymentBps?: number;
            /**
             * @description LeasingProgramID выбранная программа; пусто — расчёт по переданным параметрам.
             * @example
             */
            leasingProgramID?: string;
            /**
             * @description MachineSlug карточка, по цене которой считается платёж.
             * @example xcmg-xe215c
             */
            machineSlug: string;
            /**
             * @description RateBps годовая ставка в базисных пунктах: 18% это 1800.
             * @example 1800
             */
            rateBps?: number;
            /**
             * @description ResidualValueKztCents выкупная стоимость в тиынах.
             * @example 0
             */
            residualValueKztCents?: number;
            /**
             * @description ScheduleType тип графика: annuity или differentiated.
             * @example annuity
             * @enum {string}
             */
            scheduleType: "annuity" | "differentiated";
            /**
             * @description TermMonths срок в месяцах.
             * @example 36
             */
            termMonths: number;
        };
        "form.LeasingProgramSave": {
            /**
             * @description ID программы; пусто — создание новой.
             * @example
             */
            id?: string;
            /**
             * @description IsActive действует ли программа.
             * @example true
             */
            isActive?: boolean;
            /**
             * @description LogoPath путь к логотипу партнёра внутри хранилища.
             * @example
             */
            logoPath?: string;
            /**
             * @description MaxAmountCnyCents потолок суммы сделки в фынях (1/100 юаня); пусто — без потолка.
             * @example 50000000
             */
            maxAmountCnyCents?: number;
            /**
             * @description MinDownPaymentBps минимальный первоначальный взнос в базисных пунктах.
             * @example 1500
             */
            minDownPaymentBps?: number;
            /**
             * @description NameEN название на английском.
             * @example Halyk Leasing
             */
            nameEN: string;
            /**
             * @description NameKK название на казахском.
             * @example Halyk Leasing
             */
            nameKK: string;
            /**
             * @description NameRU название на русском.
             * @example Halyk Leasing
             */
            nameRU: string;
            /**
             * @description Position порядок вывода.
             * @example 1
             */
            position?: number;
            /**
             * @description RateMaxBps верхняя граница ставки в базисных пунктах.
             * @example 2400
             */
            rateMaxBps?: number;
            /**
             * @description RateMinBps нижняя граница ставки в базисных пунктах.
             * @example 1800
             */
            rateMinBps?: number;
            /**
             * @description TermMonthsMax верхняя граница срока в месяцах.
             * @example 60
             */
            termMonthsMax: number;
            /**
             * @description TermMonthsMin нижняя граница срока в месяцах.
             * @example 12
             */
            termMonthsMin: number;
        };
        "form.MachineAttributeValueSave": {
            /**
             * @description AttributeID характеристика.
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            attributeID: string;
            /**
             * @description Code код варианта для характеристики типа enum.
             * @example crawler
             */
            code?: string;
            /**
             * @description Flag значение флаговой характеристики.
             * @example true
             */
            flag?: boolean;
            /**
             * @description Number числовое значение строкой, чтобы не терять разряды на float64.
             * @example 155
             */
            number?: string;
            /**
             * @description Text строковое значение.
             * @example
             */
            text?: string;
        };
        "form.MachineImageSave": {
            /**
             * @description AltEN подпись на английском.
             * @example
             */
            altEN?: string;
            /**
             * @description AltKK подпись на казахском.
             * @example
             */
            altKK?: string;
            /**
             * @description AltRU подпись на русском.
             * @example
             */
            altRU?: string;
            /**
             * @description IsPrimary главное фото; допустимо не более одного на карточку.
             * @example true
             */
            isPrimary?: boolean;
            /**
             * @description Path путь в файловом хранилище.
             * @example machines/xcmg-xe215c-1.jpg
             */
            path: string;
            /**
             * @description Position порядок показа.
             * @example 0
             */
            position?: number;
        };
        "form.MachineImagesReplace": {
            /** @description Images новый набор фотографий; пустой набор очищает галерею. */
            images?: components["schemas"]["form.MachineImageSave"][];
        };
        "form.MachineSave": {
            /** @description AttributeValues значения характеристик. */
            attributeValues?: components["schemas"]["form.MachineAttributeValueSave"][];
            /**
             * @description Availability наличие: in_stock или on_order.
             * @example in_stock
             * @enum {string}
             */
            availability: "in_stock" | "on_order";
            /**
             * @description Brand бренд.
             * @example XCMG
             */
            brand: string;
            /**
             * @description CategoryID категория карточки.
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            categoryID: string;
            /**
             * @description Condition состояние: new или used.
             * @example new
             * @enum {string}
             */
            condition: "new" | "used";
            /**
             * @description DeliveryWeeks срок поставки в неделях.
             * @example 8
             */
            deliveryWeeks?: number;
            /**
             * @description DescriptionEN описание на английском.
             * @example
             */
            descriptionEN?: string;
            /**
             * @description DescriptionKK описание на казахском.
             * @example
             */
            descriptionKK?: string;
            /**
             * @description DescriptionRU описание на русском.
             * @example
             */
            descriptionRU?: string;
            /**
             * @description FuelConsumptionLph расход топлива в литрах на моточас.
             *
             *     Строкой, а не числом: расход паспортный и дробный («12.7»), и float64
             *     на границе контракта — лишний повод для расхождения в младшем разряде.
             *     Пусто — не заведён, и расчёт стоимости владения так и скажет.
             * @example 12.7
             */
            fuelConsumptionLph?: string;
            /** @description Images фотографии карточки. */
            images?: components["schemas"]["form.MachineImageSave"][];
            /**
             * @description IsPricePublic показывать ли цену на витрине.
             * @example true
             */
            isPricePublic?: boolean;
            /**
             * @description Model модель.
             * @example XE215C
             */
            model: string;
            /**
             * @description NameEN название на английском.
             * @example XCMG XE215C excavator
             */
            nameEN?: string;
            /**
             * @description NameKK название на казахском.
             * @example XCMG XE215C экскаваторы
             */
            nameKK?: string;
            /**
             * @description NameRU название на русском.
             * @example Экскаватор XCMG XE215C
             */
            nameRU: string;
            /**
             * @description PriceCNYCents цена в фынях (1/100 юаня).
             *
             *     Целое в центах, а не доллары строкой или числом: на границе не должно
             *     быть места для потери точности, а «89500.00» пришлось бы разбирать
             *     и округлять — то есть заводить ровно тот шаг, где точность и теряется.
             * @example 8950000
             */
            priceCnyCents?: number;
            /**
             * @description ProductionYear год выпуска.
             * @example 2024
             */
            productionYear?: number;
            /**
             * @description ServiceCostCnyCents стоимость одного планового ТО в фынях (1/100 юаня);
             *     пусто — не заведена.
             * @example 120000
             */
            serviceCostCnyCents?: number;
            /**
             * @description ServiceIntervalHours моточасов между плановыми ТО; пусто — не заведён.
             * @example 500
             */
            serviceIntervalHours?: number;
            /**
             * @description Slug идентификатор в URL.
             * @example xcmg-xe215c
             */
            slug: string;
        };
        "form.MachineSearch": {
            /** @description Attributes условия по характеристикам, произвольный набор. */
            attributes?: components["schemas"]["form.AttributeFilter"][];
            /**
             * @description Availability наличие: in_stock или on_order.
             * @example in_stock
             * @enum {string}
             */
            availability?: "in_stock" | "on_order";
            /**
             * @description CategoryID ограничивает выдачу категорией.
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            categoryID?: string;
            /**
             * @description Condition состояние: new или used.
             * @example new
             * @enum {string}
             */
            condition?: "new" | "used";
            /**
             * @description Cursor курсор следующей страницы, полученный в предыдущем ответе.
             * @example
             */
            cursor?: string;
            /**
             * @description IncludeSubcategories расширяет отбор на всё поддерево категории.
             * @example true
             */
            includeSubcategories?: boolean;
            /**
             * @description IsActive отбирает опубликованные (true) или черновики (false).
             *
             *     На публичной витрине игнорируется: там признак уже зафиксирован,
             *     и переопределить его параметром запроса нельзя.
             * @example false
             */
            isActive?: boolean;
            /**
             * @description Limit размер страницы. Значение сверх допустимого урезается до максимума.
             * @example 20
             */
            limit?: number;
            /**
             * @description PriceMaxCents верхняя граница цены в фынях (1/100 юаня) включительно.
             * @example 15000000
             */
            priceMaxCents?: number;
            /**
             * @description PriceMinCents нижняя граница цены в фынях (1/100 юаня) включительно.
             * @example 5000000
             */
            priceMinCents?: number;
            /**
             * @description ProductionYearMax верхняя граница года выпуска включительно.
             * @example 2026
             */
            productionYearMax?: number;
            /**
             * @description ProductionYearMin нижняя граница года выпуска включительно.
             * @example 2018
             */
            productionYearMin?: number;
            /**
             * @description Query подстрока для поиска по названию, бренду и модели.
             *
             *     Одно поле и для витрины, и для админки: ищутся одни и те же колонки,
             *     а видимость черновиков решает не оно.
             * @example экскаватор
             */
            query?: string;
            /**
             * @description Sort порядок сортировки: price_asc, price_desc или published_desc.
             * @example price_asc
             * @enum {string}
             */
            sort?: "price_asc" | "price_desc" | "published_desc";
            /**
             * @description WithTotal просить ли общее число найденных карточек.
             *
             *     Стоит второго запроса по тому же предикату, поэтому по умолчанию
             *     выключено: бесконечной прокрутке итог не нужен.
             * @example false
             */
            withTotal?: boolean;
        };
        "form.PriceSettingsSave": {
            /**
             * @description ShowUSDPrice показывать ли цену в долларах на витрине.
             * @example true
             */
            showUsdPrice?: boolean;
        };
        "form.QuoteIssue": {
            /**
             * @description CommissionKztCents разовая комиссия в тиынах.
             * @example 0
             */
            commissionKztCents?: number;
            /** @description Configuration выбранная комплектация и опции. */
            configuration?: {
                [key: string]: unknown;
            };
            /**
             * @description DownPaymentBps первоначальный взнос в базисных пунктах: 20.00% это 2000.
             * @example 2000
             */
            downPaymentBps?: number;
            /**
             * @description LeasingProgramID выбранная программа; пусто — расчёт по переданным параметрам.
             * @example
             */
            leasingProgramID?: string;
            /**
             * @description MachineSlug карточка, по которой выпускается предложение.
             * @example xcmg-xe215c
             */
            machineSlug: string;
            /**
             * @description RateBps годовая ставка в базисных пунктах: 18% это 1800.
             * @example 1800
             */
            rateBps?: number;
            /**
             * @description ResidualValueKztCents выкупная стоимость в тиынах.
             * @example 0
             */
            residualValueKztCents?: number;
            /**
             * @description ScheduleType тип графика: annuity или differentiated.
             * @example annuity
             * @enum {string}
             */
            scheduleType: "annuity" | "differentiated";
            /**
             * @description TermMonths срок в месяцах.
             * @example 36
             */
            termMonths: number;
            /**
             * @description ValidForDays срок годности предложения в днях; 0 — значение по умолчанию.
             * @example 14
             */
            validForDays?: number;
        };
        "form.QuoteRecalculate": {
            /**
             * @description DownPaymentBps первоначальный взнос в базисных пунктах.
             * @example 2000
             */
            downPaymentBps?: number;
            /**
             * @description RateBps годовая ставка в базисных пунктах.
             * @example 1800
             */
            rateBps?: number;
            /**
             * @description ScheduleType тип графика: annuity или differentiated.
             * @example annuity
             * @enum {string}
             */
            scheduleType: "annuity" | "differentiated";
            /**
             * @description TermMonths срок в месяцах.
             * @example 48
             */
            termMonths: number;
        };
        "form.ResidualValueCurveSave": {
            /**
             * @description AnnualDropBps сколько теряется за каждый следующий год ОТ ОСТАТКА.
             * @example 1200
             */
            annualDropBps?: number;
            /**
             * @description FirstYearDropBps сколько техника теряет за первый год, в базисных пунктах.
             *
             *     Отдельно от остальных лет: за первый год техника теряет заметно больше,
             *     и одна ставка на все годы либо завысила бы остаток свежей машины,
             *     либо занизила бы старой.
             * @example 2000
             */
            firstYearDropBps?: number;
            /**
             * @description FloorBps доля, ниже которой стоимость не опускается.
             *
             *     Чистая геометрия на длинном горизонте уводит цену почти в ноль, чего
             *     с техникой не происходит: у неё есть цена узлов и металла.
             * @example 2000
             */
            floorBps?: number;
        };
        "form.TCOCalculate": {
            /**
             * @description FuelPriceKztCents цена литра топлива в тиынах.
             * @example 29000
             */
            fuelPriceKztCents?: number;
            /**
             * @description HoursPerYear наработка в моточасах за год.
             * @example 1500
             */
            hoursPerYear: number;
            /**
             * @description OperatorHourlyKztCents оплата оператора за моточас в тиынах.
             * @example 250000
             */
            operatorHourlyKztCents?: number;
            /**
             * @description Years срок владения в годах.
             * @example 5
             */
            years: number;
        };
        "presenter.Attribute": {
            /**
             * @description Категория
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            categoryID: string;
            /**
             * @description Машинный ключ
             * @example engine_power
             */
            code: string;
            /**
             * @description Тип значения
             * @example number
             */
            dataType: string;
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Показывать в фильтрах
             * @example true
             */
            isFilterable: boolean;
            /**
             * @description Название на английском; пусто, пока нет перевода
             * @example Engine power
             */
            nameEN: string;
            /**
             * @description Название на казахском; пусто, пока нет перевода
             * @example Қуаты
             */
            nameKK: string;
            /**
             * @description Название на русском
             * @example Мощность
             */
            nameRU: string;
            /** @example 420 */
            numberMax?: string;
            /**
             * @description NumberMin и NumberMax границы значений в разделе; пусто — значений нет
             *     или характеристика не числовая. Заполняются только публичной ручкой:
             *     из них витрина строит подсказку под полем фильтра.
             * @example 55
             */
            numberMin?: string;
            /** @description Варианты для типа enum */
            options: components["schemas"]["presenter.AttributeOption"][];
            /**
             * @description Порядок вывода
             * @example 1
             */
            position: number;
            /**
             * @description Единица измерения
             * @example кВт
             */
            unit?: string;
        };
        "presenter.AttributeList": {
            /** @description Характеристики в порядке вывода */
            items: components["schemas"]["presenter.Attribute"][];
        };
        "presenter.AttributeOption": {
            /**
             * @description Код варианта
             * @example crawler
             */
            code: string;
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Название на английском; пусто, пока нет перевода
             * @example Crawler
             */
            nameEN: string;
            /**
             * @description Название на казахском; пусто, пока нет перевода
             * @example Шынжыр табанды
             */
            nameKK: string;
            /**
             * @description Название на русском
             * @example Гусеничный
             */
            nameRU: string;
            /**
             * @description Порядок вывода
             * @example 1
             */
            position: number;
        };
        "presenter.AuthToken": {
            /**
             * @description Токен доступа
             * @example eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
             */
            token: string;
            /** @description Учётная запись */
            user: components["schemas"]["presenter.AuthUser"];
        };
        "presenter.AuthUser": {
            /**
             * @description Идентификатор пользователя
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Name имя для показа; отсутствует, пока сотрудник его не задал.
             *
             *     Необязательное намеренно: подставлять вместо незаполненного имени
             *     заглушку значило бы выдумывать её за человека, а клиенту в этом случае
             *     есть что показать — телефон.
             * @example Иван Петров
             */
            name?: string;
            /**
             * @description Номер телефона
             * @example +77001234567
             */
            phone: string;
            /** @example manager */
            role: string;
        };
        "presenter.AuthUserList": {
            /** @description Сотрудники в порядке (роль, телефон) */
            items: components["schemas"]["presenter.AuthUser"][];
        };
        "presenter.Category": {
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description ImagePath обложка, выбранная вручную. Пусто — её не выбирали.
             *
             *     Нужна админке: без неё редактор не отличит «я выбрал этот снимок»
             *     от «сервис подставил, и так совпало».
             * @example categories/7c9e6679.jpg
             */
            imagePath?: string;
            /**
             * @description ImageURL адрес обложки раздела относительно корня API.
             *
             *     Заданная вручную, а если её нет — главное фото первой машины
             *     поддерева. Клиент не знает, откуда она взялась, и знать не должен:
             *     это одна картинка раздела. Пусто — в разделе нет ни одной
             *     опубликованной машины с фотографией.
             * @example /media/categories/7c9e6679.jpg
             */
            imageURL?: string;
            /**
             * @description Видимость на сайте
             * @example true
             */
            isActive: boolean;
            /**
             * @description Название на английском; пусто, пока нет перевода
             * @example Excavators
             */
            nameEN: string;
            /**
             * @description Название на казахском; пусто, пока нет перевода
             * @example Экскаваторлар
             */
            nameKK: string;
            /**
             * @description Название на русском
             * @example Экскаваторы
             */
            nameRU: string;
            /**
             * @description Родительская категория; пусто — корень
             * @example
             */
            parentID?: string;
            /**
             * @description Порядок вывода
             * @example 1
             */
            position: number;
            /**
             * @description Идентификатор в URL
             * @example excavators
             */
            slug: string;
        };
        "presenter.CategoryList": {
            /** @description Категории в порядке (родитель, position) */
            items: components["schemas"]["presenter.Category"][];
        };
        "presenter.FXMarkup": {
            /**
             * @description Валюта, из которой считаем
             * @example CNY
             */
            baseCurrency: string;
            /**
             * @description Кто задал; пусто — сервис
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            changedBy?: string;
            /**
             * @description С какого момента действует
             * @example 2026-09-01T00:00:00Z
             */
            effectiveFrom: string;
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Наценка в базисных пунктах
             * @example 200
             */
            markupBps: number;
            /**
             * @description Валюта, в которую считаем
             * @example KZT
             */
            quoteCurrency: string;
        };
        "presenter.FXMarkupList": {
            /** @description Наценки, свежие первыми */
            items: components["schemas"]["presenter.FXMarkup"][];
        };
        "presenter.FXPairStatus": {
            /**
             * @description Валюта, из которой считаем
             * @example CNY
             */
            baseCurrency: string;
            /**
             * @description EffectiveRate применённый курс с наценкой — тот, по которому считается
             *     цена. Пусто, если курса нет.
             * @example 74.001000
             */
            effectiveRate?: string;
            /**
             * @description EffectiveRateDate дата действующего курса.
             * @example 2026-09-01
             */
            effectiveRateDate?: string;
            /**
             * @description EffectiveSource кто поставил действующий курс.
             * @example manual
             */
            effectiveSource?: string;
            /**
             * @description HasRate есть ли по паре действующий курс. False означает, что цена
             *     в этой валюте на витрине не показывается вовсе.
             * @example true
             */
            hasRate: boolean;
            /** @description ManualRate последний курс, введённый человеком. */
            manualRate?: components["schemas"]["presenter.FXRate"];
            /**
             * @description MarkupBps действующая наценка на пару.
             * @example 200
             */
            markupBps?: number;
            /** @description ProviderRate последний курс внешнего источника — то, что он предлагает. */
            providerRate?: components["schemas"]["presenter.FXRate"];
            /**
             * @description Валюта, в которую считаем
             * @example KZT
             */
            quoteCurrency: string;
            /**
             * @description Stale курс есть, но старше допустимого возраста.
             * @example false
             */
            stale?: boolean;
        };
        "presenter.FXRate": {
            /**
             * @description Валюта, из которой считаем
             * @example CNY
             */
            baseCurrency: string;
            /**
             * @description Момент получения
             * @example 2026-09-01T09:00:00Z
             */
            fetchedAt: string;
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Валюта, в которую считаем
             * @example KZT
             */
            quoteCurrency: string;
            /**
             * @description Курс без наценки, строкой
             * @example 72.550000
             */
            rate: string;
            /**
             * @description Дата, на которую действует курс
             * @example 2026-09-01
             */
            rateDate: string;
            /**
             * @description Кто поставил курс
             * @example manual
             */
            source: string;
        };
        "presenter.FXRateList": {
            /** @description Курсы, свежие первыми */
            items: components["schemas"]["presenter.FXRate"][];
        };
        "presenter.FXStatusList": {
            /** @description Состояние по каждой паре */
            items: components["schemas"]["presenter.FXPairStatus"][];
            /**
             * @description ShowUsdPrice включён ли показ цены в долларах. Лежит здесь, а не
             *     отдельной ручкой: админка показывает состояние пары CNY→USD и этот
             *     флаг рядом — без него непонятно, важно ли отсутствие курса этой пары.
             * @example false
             */
            showUsdPrice?: boolean;
        };
        "presenter.Lead": {
            /**
             * @description AllowedTransitions статусы, в которые можно перевести заявку отсюда.
             *
             *     Приходит вместе с самой заявкой, а не отдельной справочной ручкой:
             *     перечень зависит от текущего статуса, и справочник заставил бы клиента
             *     держать у себя таблицу переходов и применять её самому — то есть
             *     хранить копию правила, расхождение с которой обнаружилось бы только
             *     отказом бэкенда.
             *
             *     Пустой список означает, что заявка завершена: won и lost терминальны.
             * @example [
             *       "in_progress"
             *     ]
             */
            allowedTransitions: string[];
            /**
             * @description Город
             * @example Алматы
             */
            city?: string;
            /**
             * @description Комментарий менеджера
             * @example
             */
            comment?: string;
            /**
             * @description Момент поступления
             * @example 2026-08-27T10:00:00Z
             */
            createdAt: string;
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Machine краткая ссылка на карточку: то, что нужно показать в списке
             *     и по чему перейти. Приходит вместе с machineID, когда машина указана.
             *
             *     Одного идентификатора мало: чтобы показать название, клиенту пришлось
             *     бы тянуть карточку на каждую строку списка или весь каталог целиком.
             */
            machine?: components["schemas"]["presenter.MachineRef"];
            /**
             * @description Машина
             * @example
             */
            machineID?: string;
            /**
             * @description Manager краткая ссылка на учётную запись ответственного.
             *
             *     Одного идентификатора мало: список сотрудников доступен только
             *     администратору, и менеджер, глядя на свою заявку, не смог бы
             *     разрешить его вовсе.
             */
            manager?: components["schemas"]["presenter.StaffRef"];
            /**
             * @description Ответственный менеджер
             * @example
             */
            managerID?: string;
            /**
             * @description Message то, что написал КЛИЕНТ в форме заявки.
             *
             *     Отдельно от comment намеренно: тот принадлежит менеджеру. В админке
             *     их нельзя показывать одним полем — правка менеджера затёрла бы
             *     единственную запись того, зачем человек обратился.
             * @example Нужен масляный фильтр на XE215C
             */
            message?: string;
            /**
             * @description Имя клиента
             * @example Иван
             */
            name: string;
            /**
             * @description Номер в нормализованном виде
             * @example +77001112233
             */
            phone: string;
            /**
             * @description Коммерческое предложение
             * @example
             */
            quoteID?: string;
            /**
             * @description Источник заявки
             * @example machine_card
             */
            source: string;
            /**
             * @description Стадия обработки
             * @example new
             */
            status: string;
            /**
             * @description Момент изменения
             * @example 2026-08-27T10:00:00Z
             */
            updatedAt: string;
        };
        "presenter.LeadAccepted": {
            /**
             * @description Идентификатор заявки
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Стадия обработки
             * @example new
             */
            status: string;
        };
        "presenter.LeadList": {
            /**
             * @description HasMore есть ли ещё заявки за этой страницей.
             * @example true
             */
            hasMore: boolean;
            /** @description Заявки, свежие сверху */
            items: components["schemas"]["presenter.Lead"][];
            /**
             * @description NextCursor курсор следующей страницы; пусто — страница последняя.
             *
             *     Непрозрачная строка: её содержимое — деталь реализации пагинации,
             *     и клиент обязан возвращать её как есть, а не собирать сам.
             * @example
             */
            nextCursor: string;
            /**
             * @description Total общее число найденных заявок.
             *
             *     Приходит только если клиент попросил его флагом withTotal: подсчёт —
             *     отдельный запрос по тому же предикату.
             * @example 137
             */
            total?: number;
        };
        "presenter.LeasingCalculation": {
            /** @description Calculation параметры и результат расчёта — те же, что и в предложении. */
            calculation: components["schemas"]["presenter.QuoteCalculation"];
            /** @description FX курс, по которому выполнен расчёт. */
            fx: components["schemas"]["presenter.LeasingCalculationFX"];
            /**
             * @description PriceCnyCents цена машины в фынях (1/100 юаня), взятая сервером по слагу.
             * @example 8950000
             */
            priceCnyCents: number;
        };
        "presenter.LeasingCalculationFX": {
            /**
             * @description Rate применённый курс — уже с наценкой. Строкой, чтобы не потерять
             *     младшие разряды на float64.
             * @example 532.786800
             */
            rate: string;
            /**
             * @description RateDate дата курса источника.
             *
             *     Приходит всегда: при отсутствии свежего курса расчёт выполнен
             *     по последнему известному, и без даты это неразличимо.
             * @example 2026-08-27T00:00:00Z
             */
            rateDate: string;
        };
        "presenter.LeasingProgram": {
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description IsActive действует ли программа.
             *
             *     В публичной выдаче всегда true — она отдаёт только действующие;
             *     в админской показывает выключенные, которые иначе выглядели бы
             *     потерянными.
             * @example true
             */
            isActive: boolean;
            /**
             * @description Путь к логотипу внутри хранилища
             * @example
             */
            logoPath?: string;
            /**
             * @description LogoURL адрес логотипа относительно корня API; пусто — логотипа нет.
             * @example /media/programs/halyk.png
             */
            logoURL?: string;
            /**
             * @description Потолок суммы сделки
             * @example 50000000
             */
            maxAmountCnyCents?: number;
            /**
             * @description Минимальный взнос в bps
             * @example 1500
             */
            minDownPaymentBps: number;
            /**
             * @description Название на английском; пусто, пока нет перевода
             * @example Halyk Leasing
             */
            nameEN: string;
            /**
             * @description Название на казахском; пусто, пока нет перевода
             * @example Halyk Leasing
             */
            nameKK: string;
            /**
             * @description Название на русском
             * @example Halyk Leasing
             */
            nameRU: string;
            /**
             * @description Порядок вывода
             * @example 1
             */
            position: number;
            /**
             * @description Верхняя граница ставки
             * @example 2400
             */
            rateMaxBps: number;
            /**
             * @description Нижняя граница ставки
             * @example 1800
             */
            rateMinBps: number;
            /**
             * @description Верхняя граница срока
             * @example 60
             */
            termMonthsMax: number;
            /**
             * @description Нижняя граница срока
             * @example 12
             */
            termMonthsMin: number;
        };
        "presenter.LeasingProgramList": {
            /** @description Действующие программы */
            items: components["schemas"]["presenter.LeasingProgram"][];
        };
        "presenter.Machine": {
            /** @description Значения характеристик */
            attributeValues?: components["schemas"]["presenter.MachineAttributeValue"][];
            /**
             * @description Наличие
             * @example in_stock
             */
            availability: string;
            /**
             * @description Бренд
             * @example XCMG
             */
            brand: string;
            /**
             * @description Категория
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            categoryID: string;
            /**
             * @description Состояние
             * @example new
             */
            condition: string;
            /**
             * @description Срок поставки в неделях
             * @example 8
             */
            deliveryWeeks?: number;
            /**
             * @description Описание на английском
             * @example
             */
            descriptionEN?: string;
            /**
             * @description Описание на казахском
             * @example
             */
            descriptionKK?: string;
            /**
             * @description Описание на русском
             * @example
             */
            descriptionRU?: string;
            /**
             * @description FuelConsumptionLph, ServiceIntervalHours и ServiceCostCnyCents —
             *     величины для расчёта стоимости владения.
             *
             *     Отсутствуют, если не заведены: пустое поле означает «нет данных»,
             *     и подменять его нулём нельзя — ноль здесь читался бы как «не расходует
             *     топлива». Расход строкой по той же причине, что и число характеристики:
             *     он дробный, и float64 на границе контракта теряет младший разряд.
             * @example 12.7
             */
            fuelConsumptionLph?: string;
            /**
             * @description FXRateDate дата курса, по которому пересчитана цена.
             *
             *     Приходит только вместе с priceKztCents и обязательна при ней: применённый
             *     курс при отсутствии свежего — это последний известный, и без даты
             *     вчерашний курс неотличим от сегодняшнего. Наценка наружу не выносится:
             *     она уже внутри курса, а её размер посетителю знать незачем.
             * @example 2026-08-27T00:00:00Z
             */
            fxRateDate?: string;
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /** @description Фотографии */
            images?: components["schemas"]["presenter.MachineImage"][];
            /**
             * @description Опубликована ли карточка
             * @example true
             */
            isActive: boolean;
            /**
             * @description IsPricePublic показывается ли цена на витрине.
             * @example true
             */
            isPricePublic: boolean;
            /**
             * @description Модель
             * @example XE215C
             */
            model: string;
            /**
             * @description Название на английском; пусто, пока нет перевода
             * @example XCMG XE215C excavator
             */
            nameEN: string;
            /**
             * @description Название на казахском; пусто, пока нет перевода
             * @example XCMG XE215C экскаваторы
             */
            nameKK: string;
            /**
             * @description Название на русском
             * @example Экскаватор XCMG XE215C
             */
            nameRU: string;
            /**
             * @description PriceCNYCents цена в фынях (1/100 юаня). Пусто, если цена скрыта за формой запроса.
             * @example 8950000
             */
            priceCnyCents?: number;
            /**
             * @description PriceKZTCents цена в тиынах по действующему курсу; отсутствует, когда
             *     курса нет или цена скрыта на витрине.
             *
             *     Пересчёт справочный: он не фиксируется и меняется вместе с курсом.
             *     Зафиксированную цену даёт только выпущенное коммерческое предложение.
             * @example 4675943000
             */
            priceKztCents?: number;
            /**
             * @description PriceUSDCents цена в центах — второе, НЕОБЯЗАТЕЛЬНОЕ отображение.
             *
             *     Приходит, только когда показ доллара включён настройкой и курс пары
             *     есть. Отображением всё и ограничивается: ни лизинг, ни стоимость
             *     владения, ни коммерческое предложение доллар не используют — они
             *     считаются от юаня и тенге.
             * @example 1243000
             */
            priceUsdCents?: number;
            /**
             * @description PrimaryImage главное фото карточки.
             *
             *     Существует ради списков: плитке каталога нужна одна картинка, и тянуть
             *     ради неё всю галерею — это десятки лишних строк на страницу. В выдаче
             *     поиска приходит только оно, images там пусто; на самой карточке приходят
             *     оба поля, и primaryImage там — та же фотография, что помечена isPrimary
             *     внутри images.
             *
             *     Необязательное: у карточки может не быть ни одной фотографии.
             */
            primaryImage?: components["schemas"]["presenter.MachineImage"];
            /**
             * @description Год выпуска
             * @example 2024
             */
            productionYear?: number;
            /**
             * @description Дата первой публикации
             * @example 2026-08-27T10:00:00Z
             */
            publishedAt?: string;
            /** @example 120000 */
            serviceCostCnyCents?: number;
            /** @example 500 */
            serviceIntervalHours?: number;
            /**
             * @description Идентификатор в URL
             * @example xcmg-xe215c
             */
            slug: string;
            /**
             * @description USDRateDate дата курса, по которому получена цена в долларах.
             *
             *     Своя, а не общая с тенге: пары разные, и курс одной бывает свежее
             *     курса другой.
             * @example 2026-08-27T00:00:00Z
             */
            usdRateDate?: string;
        };
        "presenter.MachineAttributeValue": {
            /**
             * @description Характеристика
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            attributeID: string;
            /**
             * @description Код варианта
             * @example crawler
             */
            code?: string;
            /**
             * @description Тип значения
             * @example number
             */
            dataType: string;
            /**
             * @description Значение флага
             * @example true
             */
            flag?: boolean;
            /**
             * @description Числовое значение
             * @example 155
             */
            number?: string;
            /**
             * @description Строковое значение
             * @example
             */
            text?: string;
        };
        "presenter.MachineImage": {
            /**
             * @description Подпись на английском
             * @example
             */
            altEN?: string;
            /**
             * @description Подпись на казахском
             * @example
             */
            altKK?: string;
            /**
             * @description Подпись на русском
             * @example
             */
            altRU?: string;
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Главное фото
             * @example true
             */
            isPrimary: boolean;
            /**
             * @description Путь внутри хранилища
             * @example machines/xcmg-xe215c-1.jpg
             */
            path: string;
            /**
             * @description Порядок показа
             * @example 0
             */
            position: number;
            /**
             * @description URL адрес фотографии относительно корня API: клиент дописывает его
             *     к тому же адресу, по которому обращается к каталогу.
             * @example /media/machines/xcmg-xe215c-1.jpg
             */
            url: string;
        };
        "presenter.MachineImageList": {
            /** @description Фотографии в порядке показа */
            items: components["schemas"]["presenter.MachineImage"][];
        };
        "presenter.MachinePage": {
            /**
             * @description HasMore есть ли ещё карточки за этой страницей.
             * @example true
             */
            hasMore: boolean;
            /** @description Items карточки страницы. */
            items: components["schemas"]["presenter.Machine"][];
            /**
             * @description NextCursor курсор следующей страницы; пусто — страница последняя.
             *
             *     Непрозрачная строка: её содержимое — деталь реализации пагинации,
             *     и клиент обязан возвращать её как есть, а не собирать сам.
             * @example
             */
            nextCursor?: string;
            /**
             * @description Total общее число найденных карточек.
             *
             *     Приходит только если клиент попросил его флагом withTotal: точный
             *     подсчёт при keyset-пагинации — отдельный запрос по тому же предикату,
             *     и он обязан просмотреть все подходящие строки.
             * @example 137
             */
            total?: number;
        };
        "presenter.MachineRef": {
            /**
             * @description CategorySlug слаг категории карточки.
             *
             *     Нужен для перехода: адрес карточки на витрине вложен в категорию
             *     (/catalog/{категория}/{машина}), и одного слага машины для ссылки мало.
             * @example excavators
             */
            categorySlug: string;
            /** @example 7c9e6679-7425-40de-944b-e07fc1f90ae7 */
            id: string;
            /** @example Экскаватор XCMG XE215C */
            nameRU: string;
            /** @example xcmg-xe215c */
            slug: string;
        };
        "presenter.PriceSettings": {
            /**
             * @description ShowUSDPrice показывать ли цену в долларах на витрине.
             * @example true
             */
            showUsdPrice?: boolean;
        };
        "presenter.Quote": {
            /** @description Параметры и результат расчёта */
            calculation: components["schemas"]["presenter.QuoteCalculation"];
            /** @description Выбранная комплектация */
            configuration?: {
                [key: string]: unknown;
            };
            /**
             * @description Момент выпуска
             * @example 2026-08-27T10:00:00Z
             */
            createdAt: string;
            /**
             * @description Срок годности
             * @example 2026-09-10T10:00:00Z
             */
            expiresAt: string;
            /** @description Момент первого открытия */
            firstOpenedAt?: string;
            /** @description Зафиксированный курс */
            fx: components["schemas"]["presenter.QuoteFX"];
            /**
             * @description Идентификатор
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            id: string;
            /**
             * @description Истекло ли предложение
             * @example false
             */
            isExpired: boolean;
            /**
             * @description Machine машина предложения целиком: с фотографиями, характеристиками
             *     и ценой по зафиксированному курсу. Приходит там, где предложение читают
             *     по одному, — на публичной странице и в карточке предложения.
             */
            machine?: components["schemas"]["presenter.Machine"];
            /**
             * @description MachineRef краткая ссылка на карточку: название и то, по чему перейти.
             *
             *     Приходит в списке вместо Machine. Читать полную карточку на каждую
             *     строку списка значило бы делать запрос на строку ради одного названия,
             *     а отдавать пустой объект «машины» — врать про её отсутствие.
             */
            machineRef?: components["schemas"]["presenter.MachineRef"];
            /**
             * @description Цена машины в фынях (1/100 юаня)
             * @example 8950000
             */
            priceCnyCents: number;
        };
        "presenter.QuoteCalculation": {
            /**
             * @description Стоимость машины в тиынах
             * @example 4675943000
             */
            assetCostKztCents: number;
            /**
             * @description Комиссия за организацию
             * @example 0
             */
            commissionKztCents: number;
            /**
             * @description Первоначальный взнос в bps
             * @example 2000
             */
            downPaymentBps: number;
            /**
             * @description Первоначальный взнос в тиынах
             * @example 935188600
             */
            downPaymentKztCents: number;
            /**
             * @description Сумма финансирования
             * @example 3740754400
             */
            financedAmountKztCents: number;
            /**
             * @description Последний платёж
             * @example 143214500
             */
            lastPaymentKztCents: number;
            /**
             * @description Программа лизинга
             * @example
             */
            leasingProgramID?: string;
            /**
             * @description LeasingProgramName название программы, по которой считали.
             *
             *     Одного идентификатора мало: публичный справочник отдаёт только
             *     действующие программы, и у предложения, выпущенного по с тех пор
             *     выключенной, название взять было бы негде.
             * @example Halyk Leasing
             */
            leasingProgramName?: string;
            /**
             * @description Регулярный платёж
             * @example 143214400
             */
            monthlyPaymentKztCents: number;
            /**
             * @description Переплата
             * @example 1414964100
             */
            overpaymentKztCents: number;
            /**
             * @description Ставка в bps
             * @example 1800
             */
            rateBps: number;
            /**
             * @description Выкупная стоимость
             * @example 0
             */
            residualValueKztCents: number;
            /**
             * @description Schedule полный график платежей.
             *
             *     Приходит там, где предложение читают по одному: на публичной странице
             *     и в карточке предложения. В списке его нет — см. NewQuoteList.
             */
            schedule?: components["schemas"]["presenter.QuotePayment"][];
            /**
             * @description Тип графика
             * @example annuity
             */
            scheduleType: string;
            /**
             * @description Срок в месяцах
             * @example 36
             */
            termMonths: number;
            /**
             * @description Проценты за срок
             * @example 1414964100
             */
            totalInterestKztCents: number;
            /**
             * @description Итого выплат
             * @example 5155718500
             */
            totalPaidKztCents: number;
        };
        "presenter.QuoteFX": {
            /**
             * @description MarkupBps наценка, зашитая в курс.
             * @example 200
             */
            markupBps: number;
            /**
             * @description Rate применённый курс — уже с наценкой. Строкой, чтобы не потерять
             *     младшие разряды на float64.
             * @example 532.786800
             */
            rate: string;
            /**
             * @description RateDate дата курса источника.
             * @example 2026-08-27T00:00:00Z
             */
            rateDate: string;
        };
        "presenter.QuoteIssued": {
            /**
             * @description PublicToken токен публичной ссылки.
             * @example IEkYYm4kQ0lLZ1hnR2dnUlZUR0FBQUFBQUFBQUFBQQ
             */
            publicToken: string;
            /** @description Выпущенное предложение */
            quote: components["schemas"]["presenter.Quote"];
        };
        "presenter.QuoteList": {
            /** @description Предложения, свежие сверху */
            items: components["schemas"]["presenter.Quote"][];
        };
        "presenter.QuotePayment": {
            /**
             * @description Сумма платежа
             * @example 14321440
             */
            amount: number;
            /**
             * @description Остаток долга после платежа
             * @example 366057040
             */
            balance: number;
            /**
             * @description Часть процентов
             * @example 6450000
             */
            interest: number;
            /**
             * @description Номер платежа
             * @example 1
             */
            number: number;
            /**
             * @description Часть основного долга
             * @example 7871440
             */
            principal: number;
        };
        "presenter.ResidualValueCurve": {
            /**
             * @description AnnualDropBps сколько теряется за каждый следующий год от остатка.
             * @example 1200
             */
            annualDropBps?: number;
            /**
             * @description CategoryID раздел, для которого заведена кривая.
             * @example 7c9e6679-7425-40de-944b-e07fc1f90ae7
             */
            categoryID?: string;
            /**
             * @description FirstYearDropBps сколько техника теряет за первый год.
             * @example 2000
             */
            firstYearDropBps?: number;
            /**
             * @description FloorBps доля, ниже которой стоимость не опускается.
             * @example 2000
             */
            floorBps?: number;
            /** @description Preview доли по годам, посчитанные по кривой. */
            preview?: components["schemas"]["presenter.ResidualValueYear"][];
        };
        "presenter.ResidualValueCurveList": {
            /** @description Items кривые, заведённые в справочнике. */
            items?: components["schemas"]["presenter.ResidualValueCurve"][];
        };
        "presenter.ResidualValueYear": {
            /**
             * @description RateBps доля цены, которую техника сохраняет к концу года.
             * @example 6195
             */
            rateBps?: number;
            /**
             * @description Year год владения.
             * @example 3
             */
            year?: number;
        };
        "presenter.StaffRef": {
            /** @example 7c9e6679-7425-40de-944b-e07fc1f90ae7 */
            id: string;
            /** @example Иван Петров */
            name?: string;
            /** @example +77001112233 */
            phone: string;
        };
        "presenter.TCOItem": {
            /**
             * @description AmountKztCents сумма статьи за срок. Отсутствует — статья не посчитана.
             * @example 12000000
             */
            amountKztCents?: number;
            /**
             * @description Missing величина, которой не хватило. Пусто — статья посчитана.
             *
             *     Список, а не одно значение: статья «обслуживание» требует и интервала,
             *     и стоимости, и не хватить может обоих сразу.
             * @example [
             *       "fuel_consumption"
             *     ]
             */
            missing?: string[];
        };
        "presenter.TCOResult": {
            /**
             * @description Complete посчитаны ли все статьи.
             *
             *     Отдельным полем, хотя выводится из missing: клиент решает по нему,
             *     показывать ли расчёт вообще, и заставлять его собирать это из пяти
             *     вложенных списков значило бы разложить одно решение по пяти местам.
             * @example false
             */
            complete: boolean;
            /**
             * @description CostPerHourKztCents стоимость моточаса.
             * @example 347700
             */
            costPerHourKztCents: number;
            /** @description Fuel топливо за срок. */
            fuel: components["schemas"]["presenter.TCOItem"];
            /** @description FX курс, которым цена машины приведена к тенге. */
            fx: components["schemas"]["presenter.QuoteFX"];
            /** @description Machine карточка, по которой считали. */
            machineRef?: components["schemas"]["presenter.MachineRef"];
            /**
             * @description Missing все недостающие величины одним списком.
             * @example [
             *       "fuel_consumption"
             *     ]
             */
            missing: string[];
            /** @description Operator оплата труда оператора за срок. */
            operator: components["schemas"]["presenter.TCOItem"];
            /** @description Purchase покупка машины: цена карточки, приведённая к тенге. */
            purchase: components["schemas"]["presenter.TCOItem"];
            /** @description ResidualValue остаточная стоимость на конец срока: вычитается из итога. */
            residualValue: components["schemas"]["presenter.TCOItem"];
            /** @description Service плановое обслуживание за срок. */
            service: components["schemas"]["presenter.TCOItem"];
            /**
             * @description ServicesCount число плановых обслуживаний за срок.
             * @example 12
             */
            servicesCount?: number;
            /**
             * @description TotalHours наработка за весь срок в моточасах.
             * @example 6000
             */
            totalHours: number;
            /**
             * @description TotalKztCents итог за срок: сумма посчитанных статей за вычетом
             *     остаточной стоимости. Непосчитанные статьи в него не входят — читать
             *     итог, не посмотрев на missing, нельзя.
             * @example 20860000
             */
            totalKztCents: number;
        };
        "presenter.UploadedFile": {
            /**
             * @description Путь внутри хранилища
             * @example machines/7c9e6679-7425-40de-944b-e07fc1f90ae7.jpg
             */
            path: string;
            /**
             * @description URL адрес файла относительно корня API, как и у фотографии карточки.
             * @example /media/machines/7c9e6679-7425-40de-944b-e07fc1f90ae7.jpg
             */
            url: string;
        };
        "swagger.HTTPResponse400": {
            /**
             * @description Стабильный код ошибки.
             * @example SNT_SLUG_ALREADY_USED
             */
            code: string;
            /**
             * @description Описание ошибки для человека.
             * @example слаг уже занят
             */
            message: string;
        };
        "swagger.HTTPResponse500": {
            /**
             * @description Стабильный код ошибки.
             * @example SNT_INTERNAL
             */
            code: string;
            /**
             * @description Описание ошибки для человека.
             * @example внутренняя ошибка сервиса
             */
            message: string;
        };
    };
    responses: never;
    parameters: never;
    requestBodies: {
        /** @description Категория */
        "form.CategorySave": {
            content: {
                "application/json": components["schemas"]["form.CategorySave"];
            };
        };
        /** @description Номер телефона */
        "form.AuthTokenIssue": {
            content: {
                "application/json": components["schemas"]["form.AuthTokenIssue"];
            };
        };
        /** @description Карточка */
        "form.MachineSave": {
            content: {
                "application/json": components["schemas"]["form.MachineSave"];
            };
        };
        /** @description Характеристика */
        "form.AttributeSave": {
            content: {
                "application/json": components["schemas"]["form.AttributeSave"];
            };
        };
        /** @description Условия поиска */
        "form.MachineSearch": {
            content: {
                "application/json": components["schemas"]["form.MachineSearch"];
            };
        };
        postV1AdminMediaCategories: {
            content: {
                "multipart/form-data": {
                    /**
                     * Format: binary
                     * @description Изображение
                     */
                    file: string;
                };
            };
        };
    };
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export type operations = Record<string, never>;
