package entity

import apperrors "github.com/eskermese/machinery-catalog/backend/pkg/apperrors"

// CodePrefix краткое название сервиса.
const CodePrefix = "SNT_"

// sharedInternalError возвращает дефолтную internal-ошибку shared-libs с префиксом сервиса.
func sharedInternalError() error {
	apperrors.SetGlobalCodePrefixOnce(CodePrefix)

	return apperrors.ErrInternal
}

// sharedFormDecodeError возвращает дефолтную form-decode-ошибку shared-libs с префиксом сервиса.
func sharedFormDecodeError() error {
	apperrors.SetGlobalCodePrefixOnce(CodePrefix)

	return apperrors.ErrFormDecode
}

var (
	// ErrInternal описывает внутреннюю ошибку сервиса.
	ErrInternal = sharedInternalError()

	// ErrFormDecode описывает ошибку декодирования входного запроса.
	ErrFormDecode = sharedFormDecodeError()

	// ---- System / Infra ----.
	ErrNilPointer     = apperrors.NewDomainError(apperrors.InternalErrorType, CodePrefix, "INTERNAL", "значение не может быть nil", false)
	ErrDatabaseCommon = apperrors.NewDomainError(apperrors.InternalErrorType, CodePrefix, "INTERNAL", "database common", false)

	ErrVersionConflict = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "VERSION_CONFLICT", "конфликт версий или конфликт локов", false)

	ErrInvalidID = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "INVALID_ID", "неверный идентификатор объекта", false)

	// ---- Auth ----.
	ErrAuthRequired  = apperrors.NewDomainError(apperrors.UnauthorizedErrorType, CodePrefix, "AUTH_REQUIRED", "запрос не аутентифицирован", false)
	ErrRoleForbidden = apperrors.NewDomainError(apperrors.AccessDeniedErrorType, CodePrefix, "ROLE_FORBIDDEN", "роль не имеет доступа к этому разделу", false)
	// ErrRoleInvalid сообщает, что запрошена роль вне перечня.
	ErrRoleInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ROLE_INVALID", "неизвестная роль", false)
	// ErrTokenExpired отделён от ErrAuthRequired намеренно: клиенту нужно отличать
	// «перелогинься» от «ты вообще не аутентифицирован».
	ErrTokenExpired      = apperrors.NewDomainError(apperrors.UnauthorizedErrorType, CodePrefix, "TOKEN_EXPIRED", "срок действия токена истёк", false)
	ErrUserNotFound      = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "USER_NOT_FOUND", "пользователь не найден", false)
	ErrUserAlreadyExists = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "USER_ALREADY_EXISTS", "пользователь уже существует", false)
	ErrPhoneInvalid      = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "PHONE_INVALID", "номер телефона указан неверно", false)
	ErrUserNameInvalid   = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "USER_NAME_INVALID", "имя слишком длинное", false)

	// ---- Курсы валют ----.
	ErrCurrencyCodeInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "CURRENCY_CODE_INVALID", "код валюты должен состоять из трёх заглавных букв", false)
	ErrCurrencyPairInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "CURRENCY_PAIR_INVALID", "валюты пары должны различаться", false)
	ErrFXRateInvalid       = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "FX_RATE_INVALID", "курс валюты должен быть положительным", false)
	ErrFXSourceInvalid     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "FX_SOURCE_INVALID", "источник курса не указан", false)
	// ErrFXRateDateInvalid сообщает о нечитаемой дате курса.
	ErrFXRateDateInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "FX_RATE_DATE_INVALID", "дата курса должна быть в формате ГГГГ-ММ-ДД", false)
	// ErrFXPairInvalid сообщает, что валютная пара не указана или указана неверно.
	ErrFXPairInvalid   = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "FX_PAIR_INVALID", "валютная пара указана неверно", false)
	ErrFXMarkupInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "FX_MARKUP_INVALID", "наценка к курсу не может быть отрицательной", false)
	// ErrFXRateNotFound отделён от прочих отказов: отсутствие курса вообще — это
	// незаконченная настройка сервиса, а не отказ конкретной операции.
	ErrFXRateNotFound = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "FX_RATE_NOT_FOUND", "курс валюты не найден", false)
	// ErrFXRateAlreadyExists сообщает, что курс этого источника на эту дату уже загружен.
	ErrFXRateAlreadyExists = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "FX_RATE_ALREADY_EXISTS", "курс источника на эту дату уже загружен", false)
	// ErrFXPairNotSupported сообщает, что источник не котирует эту пару.
	//
	// Отдельно от «курс не найден»: там курса нет на дату, а здесь источник
	// такую пару не публикует вовсе, и повторять запрос завтра бессмысленно.
	ErrFXPairNotSupported = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "FX_PAIR_NOT_SUPPORTED", "источник не котирует эту пару валют", false)
	// ErrFXProviderNotConfigured сообщает, что источник курса не подключён.
	ErrFXProviderNotConfigured = apperrors.NewDomainError(apperrors.InternalErrorType, CodePrefix, "FX_PROVIDER_NOT_CONFIGURED", "источник курса не настроен", false)

	// ---- Каталог ----.
	ErrSlugInvalid     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "SLUG_INVALID", "слаг должен состоять из строчных букв, цифр и дефисов", false)
	ErrSlugAlreadyUsed = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "SLUG_ALREADY_USED", "слаг уже занят", false)
	ErrPositionInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "POSITION_INVALID", "порядок вывода не может быть отрицательным", false)

	// ---- Настройки ----.
	ErrSettingUnknown = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "SETTING_UNKNOWN", "неизвестная настройка", false)

	// ---- Категории ----.
	ErrCategoryNotFound     = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "CATEGORY_NOT_FOUND", "категория не найдена", false)
	ErrCategoryNameRequired = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "CATEGORY_NAME_REQUIRED", "русское название категории обязательно; казахское и английское могут быть пустыми, пока перевода нет", false)
	ErrCategoryHasChildren  = apperrors.NewDomainError(apperrors.StateMismatchErrorType, CodePrefix, "CATEGORY_HAS_CHILDREN", "категорию с потомками или карточками нельзя удалить", false)
	ErrCategoryCycle        = apperrors.NewDomainError(apperrors.StateMismatchErrorType, CodePrefix, "CATEGORY_CYCLE", "категория не может быть собственным предком", false)

	// ---- Характеристики ----.
	ErrAttributeNotFound     = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "ATTRIBUTE_NOT_FOUND", "характеристика не найдена", false)
	ErrAttributeCodeInvalid  = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_CODE_INVALID", "код характеристики пуст или не соответствует формату: строчные латинские буквы, цифры и подчёркивание", false)
	ErrAttributeNameRequired = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_NAME_REQUIRED", "русское название характеристики обязательно; казахское и английское могут быть пустыми, пока перевода нет", false)
	// ErrAttributeCodeAlreadyUsed сообщает, что код занят в этой категории.
	//
	// Отдельно от ErrAttributeCodeInvalid: «код написан не по формату» и «код
	// уже кем-то занят» — разные новости для того, кто заполняет форму.
	// Первое исправляют, переписав строку, второе — придумав другое имя,
	// и один код на двоих заставлял клиента гадать, что именно не так.
	ErrAttributeCodeAlreadyUsed = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "ATTRIBUTE_CODE_ALREADY_USED", "код характеристики уже занят в этой категории", false)
	ErrAttributeTypeInvalid     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_TYPE_INVALID", "неизвестный тип характеристики", false)
	ErrAttributeUnitNotAllowed  = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_UNIT_NOT_ALLOWED", "единица измерения допустима только у числовой характеристики", false)
	// ErrAttributeForeignCategory сообщает, что характеристика принадлежит другой категории.
	ErrAttributeForeignCategory = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_FOREIGN_CATEGORY", "характеристика принадлежит другой категории", false)
	// ErrAttributeValueTypeMismatch сообщает о несоответствии значения объявленному типу характеристики.
	ErrAttributeValueTypeMismatch = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_VALUE_TYPE_MISMATCH", "тип значения не совпадает с типом характеристики", false)
	// ErrAttributeTypeLocked сообщает, что тип нельзя сменить: значения заполнены.
	//
	// Раньше здесь возвращалась ErrAttributeValueTypeMismatch — ошибка про
	// заполнение значения, — и её текст описывал не ту ситуацию: никто ничего
	// не заполнял, а составной внешний ключ (attribute_id, data_type) защищал
	// уже сохранённые значения от того, что тип под ними поменяется.
	ErrAttributeTypeLocked = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "ATTRIBUTE_TYPE_LOCKED", "тип характеристики нельзя изменить: её значения уже заполнены в карточках", false)
	// ErrAttributeOptionUnknown сообщает о коде варианта, которого нет в справочнике.
	ErrAttributeOptionUnknown = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_OPTION_UNKNOWN", "код варианта отсутствует в справочнике характеристики", false)
	ErrAttributeValueInvalid  = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_VALUE_INVALID", "значение характеристики заполнено неверно", false)
	// ErrAttributeOptionCodeInvalid сообщает о варианте с кодом неверного формата.
	ErrAttributeOptionCodeInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_OPTION_CODE_INVALID", "код варианта пуст или не соответствует формату: строчные латинские буквы, цифры и подчёркивание", false)
	// ErrAttributeOptionNameRequired сообщает о варианте без русского названия.
	ErrAttributeOptionNameRequired = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_OPTION_NAME_REQUIRED", "русское название варианта обязательно; казахское и английское могут быть пустыми, пока перевода нет", false)
	// ErrAttributeOptionDuplicate сообщает о повторе кода среди вариантов характеристики.
	ErrAttributeOptionDuplicate = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "ATTRIBUTE_OPTION_DUPLICATE", "код варианта повторяется в наборе", false)

	// ---- Карточки техники ----.
	ErrMachineNotFound              = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "MACHINE_NOT_FOUND", "карточка техники не найдена", false)
	ErrMachineNameRequired          = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_NAME_REQUIRED", "русское название карточки обязательно; казахское и английское могут быть пустыми, пока перевода нет", false)
	ErrMachineBrandModelRequired    = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_BRAND_MODEL_REQUIRED", "бренд и модель обязательны", false)
	ErrMachinePriceInvalid          = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_PRICE_INVALID", "цена не может быть отрицательной", false)
	ErrMachineAvailabilityInvalid   = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_AVAILABILITY_INVALID", "неизвестное наличие техники", false)
	ErrMachineConditionInvalid      = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_CONDITION_INVALID", "неизвестное состояние техники", false)
	ErrMachineDeliveryWeeksInvalid  = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_DELIVERY_WEEKS_INVALID", "срок поставки должен быть положительным", false)
	ErrMachineProductionYearInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_PRODUCTION_YEAR_INVALID", "год выпуска вне допустимого диапазона", false)
	// ErrMachinePublishedAtRequired сообщает, что активная карточка не имеет даты публикации.
	ErrMachinePublishedAtRequired = apperrors.NewDomainError(apperrors.StateMismatchErrorType, CodePrefix, "MACHINE_PUBLISHED_AT_REQUIRED", "активная карточка обязана иметь дату публикации", false)
	// ErrMachineFuelConsumptionInvalid сообщает о нечитаемом расходе топлива.
	ErrMachineFuelConsumptionInvalid = apperrors.NewDomainError(
		apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_FUEL_CONSUMPTION_INVALID",
		"расход топлива должен быть положительным числом", false)
	ErrMachineImagePathRequired    = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_IMAGE_PATH_REQUIRED", "путь к фотографии не указан", false)
	ErrMachineImagePrimaryConflict = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MACHINE_IMAGE_PRIMARY_CONFLICT", "у машины может быть только одно главное фото", false)
	// ErrMachineHasQuotes сообщает, что по карточке выпущены предложения.
	//
	// Выпущенное КП — обязательство перед клиентом: в нём зафиксированы цена
	// и курс, а машина названа ссылкой. Удалить её нельзя (RESTRICT на
	// quotes.machine_id), и карточку в такой ситуации снимают с публикации.
	ErrMachineHasQuotes = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "MACHINE_HAS_QUOTES", "по карточке выпущены коммерческие предложения: её можно снять с публикации, но не удалить", false)

	// ---- Поиск по каталогу ----.
	ErrCatalogSortInvalid   = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "CATALOG_SORT_INVALID", "неизвестный порядок сортировки", false)
	ErrCatalogCursorInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "CATALOG_CURSOR_INVALID", "курсор пагинации испорчен", false)
	ErrCatalogLimitInvalid  = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "CATALOG_LIMIT_INVALID", "размер страницы вне допустимого диапазона", false)

	// ---- Коммерческие предложения ----.
	ErrQuoteNotFound            = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "QUOTE_NOT_FOUND", "коммерческое предложение не найдено", false)
	ErrQuoteTokenInvalid        = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_TOKEN_INVALID", "ссылка на предложение испорчена", false)
	ErrQuoteMachineRequired     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_MACHINE_REQUIRED", "предложение должно ссылаться на машину", false)
	ErrQuoteScheduleTypeInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_SCHEDULE_TYPE_INVALID", "неизвестный тип графика платежей", false)
	ErrQuoteAmountInvalid       = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_AMOUNT_INVALID", "денежная величина предложения указана неверно", false)
	ErrQuoteFXInvalid           = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_FX_INVALID", "курс предложения указан неверно", false)
	ErrQuoteDownPaymentInvalid  = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_DOWN_PAYMENT_INVALID", "первоначальный взнос вне допустимого диапазона", false)
	ErrQuoteTermInvalid         = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_TERM_INVALID", "срок или ставка предложения указаны неверно", false)
	ErrQuoteResidualInvalid     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_RESIDUAL_INVALID", "выкупная стоимость превышает сумму финансирования", false)
	ErrQuoteExpiresAtInvalid    = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "QUOTE_EXPIRES_AT_INVALID", "срок годности предложения должен быть в будущем", false)
	// ErrQuoteTokenGeneration сообщает, что система не смогла дать случайность
	// для публичного токена: выпускать ссылку в таком состоянии нельзя.
	ErrQuoteTokenGeneration = apperrors.NewDomainError(apperrors.InternalErrorType, CodePrefix, "QUOTE_TOKEN_GENERATION", "не удалось выпустить ссылку на предложение", true)

	// ErrTooManyRequests сообщает, что с адреса пришло слишком много обращений.
	ErrTooManyRequests = apperrors.NewDomainError(apperrors.TooManyRequestsErrorType, CodePrefix, "TOO_MANY_REQUESTS", "слишком много запросов, повторите через минуту", true)

	// ErrMachinePriceHidden сообщает, что цена карточки скрыта на витрине.
	// Считать по ней публичный график нельзя: форма «цена по запросу»
	// существует затем, чтобы цифры не было снаружи.
	ErrMachinePriceHidden = apperrors.NewDomainError(apperrors.StateMismatchErrorType, CodePrefix, "MACHINE_PRICE_HIDDEN", "цена этой машины предоставляется по запросу", false)

	// ---- Кривая остаточной стоимости ----.
	ErrResidualCurveNotFound            = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "RESIDUAL_CURVE_NOT_FOUND", "кривая остаточной стоимости для этого раздела не заведена", false)
	ErrResidualCurveCategoryRequired    = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "RESIDUAL_CURVE_CATEGORY_REQUIRED", "кривая остаточной стоимости заводится для раздела", false)
	ErrResidualCurveBpsOutOfRange       = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "RESIDUAL_CURVE_BPS_OUT_OF_RANGE", "доли кривой задаются в базисных пунктах от 0 до 10000", false)
	ErrResidualCurveFloorAboveFirstYear = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "RESIDUAL_CURVE_FLOOR_ABOVE_FIRST_YEAR", "порог не может быть выше того, что остаётся после первого года: кривая стала бы возрастающей", false)

	// ---- Границы программы лизинга ----.
	ErrLeasingProgramNotFound        = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "LEASING_PROGRAM_NOT_FOUND", "программа лизинга не найдена", false)
	ErrLeasingProgramInactive        = apperrors.NewDomainError(apperrors.StateMismatchErrorType, CodePrefix, "LEASING_PROGRAM_INACTIVE", "программа лизинга не действует", false)
	ErrLeasingProgramNameRequired    = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEASING_PROGRAM_NAME_REQUIRED", "русское название программы обязательно; казахское и английское могут быть пустыми, пока перевода нет", false)
	ErrLeasingProgramPositionInvalid = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEASING_PROGRAM_POSITION_INVALID", "порядок вывода не может быть отрицательным", false)
	// ErrLeasingProgramInUse сообщает, что по программе уже выпущены предложения.
	// Такую программу выключают флагом, а не удаляют: удаление стёрло бы условия,
	// на которых выдано действующее КП.
	ErrLeasingProgramInUse       = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "LEASING_PROGRAM_IN_USE", "по программе выпущены предложения: её можно выключить, но не удалить", false)
	ErrLeasingDownPaymentTooLow  = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEASING_DOWN_PAYMENT_TOO_LOW", "первоначальный взнос меньше минимального по программе", false)
	ErrLeasingTermOutOfRange     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEASING_TERM_OUT_OF_RANGE", "срок вне диапазона программы", false)
	ErrLeasingRateOutOfRange     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEASING_RATE_OUT_OF_RANGE", "ставка вне диапазона программы", false)
	ErrLeasingAmountAboveMaximum = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEASING_AMOUNT_ABOVE_MAXIMUM", "сумма сделки превышает максимум по программе", false)

	// ---- Заявки ----.
	ErrLeadNotFound       = apperrors.NewDomainError(apperrors.NotFoundErrorType, CodePrefix, "LEAD_NOT_FOUND", "заявка не найдена", false)
	ErrLeadNameRequired   = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEAD_NAME_REQUIRED", "имя обязательно", false)
	ErrLeadSourceRequired = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEAD_SOURCE_REQUIRED", "источник заявки не указан", false)
	// ErrLeadMessageTooLong сообщает о слишком длинном сообщении клиента.
	ErrLeadMessageTooLong = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEAD_MESSAGE_TOO_LONG", "сообщение слишком длинное", false)
	// ErrLeadCommentTooLong сообщает о слишком длинном комментарии менеджера.
	ErrLeadCommentTooLong         = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEAD_COMMENT_TOO_LONG", "комментарий слишком длинный", false)
	ErrLeadStatusInvalid          = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEAD_STATUS_INVALID", "неизвестный статус заявки", false)
	ErrLeadStatusTransitionDenied = apperrors.NewDomainError(apperrors.StateTransitionDeniedErrorType, CodePrefix, "LEAD_STATUS_TRANSITION_DENIED", "недопустимый переход статуса заявки", false)
	ErrLeadManagerRequired        = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEAD_MANAGER_REQUIRED", "ответственный менеджер не указан", false)
	ErrLeadQuoteRequiresMachine   = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "LEAD_QUOTE_REQUIRES_MACHINE", "заявка по предложению должна ссылаться на машину", false)
	// ErrLeadTooFrequent сообщает, что с одного номера пришло слишком много заявок подряд.
	ErrLeadTooFrequent = apperrors.NewDomainError(apperrors.ConflictErrorType, CodePrefix, "LEAD_TOO_FREQUENT", "заявка с этого номера уже принята, дождитесь звонка", false)

	// ---- Изображения ----.
	ErrMediaKindUnknown     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MEDIA_KIND_UNKNOWN", "неизвестный раздел хранилища изображений", false)
	ErrMediaFileMissing     = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MEDIA_FILE_MISSING", "файл не передан или пуст", false)
	ErrMediaTypeUnsupported = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MEDIA_TYPE_UNSUPPORTED", "поддерживаются только JPEG, PNG и WebP", false)
	// ErrMediaTooLarge отвечает на файл сверх предела. Тип InvalidArgument, а не
	// отдельная категория под 413: клиенту важен код ошибки, а разводить ради
	// одного случая новую категорию в общем механизме дороже, чем 400 с кодом.
	ErrMediaTooLarge = apperrors.NewDomainError(apperrors.InvalidArgumentErrorType, CodePrefix, "MEDIA_TOO_LARGE", "файл больше допустимого размера", false)
	// ErrMediaWriteFailed сообщает, что файл не удалось записать в хранилище.
	ErrMediaWriteFailed = apperrors.NewDomainError(apperrors.InternalErrorType, CodePrefix, "MEDIA_WRITE_FAILED", "не удалось сохранить файл", false)

	// ErrNotificationSenderNotConfigured сообщает, что канал доставки не подключён.
	ErrNotificationSenderNotConfigured = apperrors.NewDomainError(apperrors.InternalErrorType, CodePrefix, "NOTIFICATION_SENDER_NOT_CONFIGURED", "канал доставки уведомлений не настроен", false)
)
