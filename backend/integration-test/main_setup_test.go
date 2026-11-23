//go:build integration

// Файл, откуда можно управлять запуском тестов.
// Чтобы пропустить тесты, можно расскоментировать строку t.Skip().
// В будущем можно добавить флаги для управления запуском тестов.

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	test_init "github.com/eskermese/machinery-catalog/backend/integration-test/test-init"
)

// HTTPTestSuite - набор тестов для проверки HTTP REST API.
//
// Методы набора НЕ помечаются t.Parallel(): testify хранит один *testing.T на
// набор и подменяет его перед каждым методом. Параллельный подтест продолжает
// работу уже после подмены, поэтому его ассерты приписываются чужому тесту,
// а часть падений может потеряться. Конкурентность внутри теста создается
// явно через runConcurrently.
type HTTPTestSuite struct {
	// Позволяет разделять тесты на разные методы.
	suite.Suite

	ctx            context.Context // Общий контекст для всех тестов.
	httpHealthPath string          // Путь для проверки доступности.
	httpBasePath   string          // Базовый путь.
}

// TestHTTPTestSuite запускает тесты.
func TestHTTPTestSuite(t *testing.T) {
	// t.Skip()

	t.Parallel()

	suite.Run(t, new(HTTPTestSuite))
}

// SetupSuite - инициализация тестового окружения.
func (s *HTTPTestSuite) SetupSuite() {
	testLog.Info("Запуск HTTP тестов...")

	host := fmt.Sprintf("%s:%d", test_init.Cfg.ServerHostname, test_init.Cfg.HTTPListenAddr)

	s.ctx = context.Background()
	s.httpHealthPath = fmt.Sprintf("http://%s/version", host)
	s.httpBasePath = fmt.Sprintf("http://%s/api", host)

	if err := s.httpHealthCheck(); err != nil {
		s.T().Errorf("Ошибка проверки сервера: %s", err)
		return
	}

	if err := s.seedFXRate(); err != nil {
		s.T().Errorf("Ошибка посева курса: %s", err)
		return
	}
}

// Валютная пара набора: юань — валюта прайса, тенге — валюта пересчёта.
const (
	seedFXBase  = "CNY"
	seedFXQuote = "KZT"
)

// seedFXRateValue курс CNY/KZT для набора.
//
// Фиксированный, а не сегодняшний рыночный: тесты сравнивают суммы, и курс,
// меняющийся от прогона к прогону, сделал бы ожидания невыразимыми.
const seedFXRateValue = "72.500000"

// seedFXRateSource источник посеянного курса. Отдельное значение, а не имя
// реального провайдера: строка входит в UNIQUE (source, пара, дата), и посев
// не должен конфликтовать с курсом, загруженным фоновым воркером.
const seedFXRateSource = "integration-seed"

// seedFXRate кладёт курс CNY/KZT на сегодня.
//
// Прямой вставкой, а не через API: эндпоинта загрузки курса нет, и заводить
// его ради теста значило бы подгонять приложение под тест. Наценка не сеется
// намеренно — её отсутствие штатно и означает «продаём по курсу источника»
// (fxMarkupRepository.GetEffective возвращает (nil, nil)), а лишняя строка
// в fx_markups сделала бы ожидаемые суммы зависимыми от неё.
//
// ON CONFLICT DO NOTHING делает посев идемпотентным: контейнер базы между
// прогонами переиспользуется, и вторая вставка того же дня иначе упала бы
// на уникальности источника за дату.
//
// Условие WHERE в ON CONFLICT обязательно и повторяет условие индекса:
// уникальность частичная — ручной ввод из неё исключён, чтобы человек мог
// исправить опечатку, — а вывести частичный индекс без его условия Postgres
// не может.
func (s *HTTPTestSuite) seedFXRate() error {
	return test_init.Exec(s.ctx, `
		INSERT INTO fx_rates (base_currency, quote_currency, source, rate, rate_date)
		VALUES ($1, $2, $3, $4::numeric, CURRENT_DATE)
		ON CONFLICT (source, base_currency, quote_currency, rate_date)
			WHERE source <> 'manual' DO NOTHING`,
		seedFXBase, seedFXQuote, seedFXRateSource, seedFXRateValue)
}

// withoutFXRate выполняет действие при отсутствующем курсе и возвращает его
// обратно.
//
// Курс сеет подготовка набора, и оставить базу без него значило бы уронить
// соседние сценарии — выпуск предложения и пересчёт витрины. Возврат идёт
// через defer: падение проверки внутри не должно портить остальной прогон.
//
// Курс снимается из ДВУХ мест — базы и кэша, — и это два независимых
// обращения к двум серверам. Раньше сценарий начинался сразу после них,
// то есть в предположении, что оба удались; сценарий падал примерно раз
// на четыре прогона с непонятным «пришла цена в тенге». Теперь состояние
// не предполагается, а дожидается: пока ключ кэша не исчезнет, сценарий
// не начинается, а если не исчезает — тест падает с прямым объяснением,
// а не расхождением в ассерте.
func (s *HTTPTestSuite) withoutFXRate(action func()) {
	s.T().Helper()

	require.NoError(s.T(), test_init.Exec(s.ctx, `DELETE FROM fx_rates`),
		"снятие курса на время сценария")

	// Курс снят из базы — проверяем это, а не предполагаем: если строка
	// осталась, дальше падал бы ассерт сценария, и причину пришлось бы искать
	// в кэше, которого она не касается.
	var rates int
	require.NoError(s.T(), test_init.QueryRow(s.ctx, `SELECT count(*) FROM fx_rates`).Scan(&rates),
		"подсчёт курсов после снятия")
	require.Zero(s.T(), rates, "курс остался в базе")

	s.awaitFXCacheDropped()

	defer func() {
		require.NoError(s.T(), s.seedFXRate(), "возврат курса после сценария")
		s.dropFXCache()
	}()

	action()

	// Состояние проверяется и ПОСЛЕ сценария: если курс успел вернуться
	// во время запроса, падать должен этот ассерт с прямым объяснением,
	// а не проверка цены внутри сценария.
	var afterRates int
	require.NoError(s.T(), test_init.QueryRow(s.ctx, `SELECT count(*) FROM fx_rates`).Scan(&afterRates),
		"подсчёт курсов после сценария")
	require.Zero(s.T(), afterRates, "курс вернулся в базу во время сценария")

	exists, err := test_init.Cache.Exists(s.ctx, fxAppliedRateCacheKey)
	require.NoError(s.T(), err, "проверка кэша курса после сценария")
	require.False(s.T(), exists, "курс вернулся в кэш во время сценария")
}

// fxCacheDropAttempts число попыток убедиться, что кэш курса пуст.
// Двух секунд с запасом хватает: удаление ключа — одна команда, и повторы
// нужны лишь на случай записи, начатой до удаления.
const fxCacheDropAttempts = 100

// fxCacheDropDelay пауза между попытками.
const fxCacheDropDelay = 20 * time.Millisecond

// awaitFXCacheDropped удаляет кэш применённого курса и дожидается, пока ключа
// действительно не станет.
//
// Повтор, а не однократное удаление: запрос, начатый до снятия курса, мог
// прочитать курс из базы и записать его в кэш уже ПОСЛЕ нашего удаления.
// Одно удаление такую запись не переживает, а сценарий после неё проверяет
// не то состояние, ради которого написан.
func (s *HTTPTestSuite) awaitFXCacheDropped() {
	s.T().Helper()

	for attempt := 0; attempt < fxCacheDropAttempts; attempt++ {
		require.NoError(s.T(), test_init.Cache.Delete(s.ctx, fxAppliedRateCacheKey),
			"сброс кэша курса")

		exists, err := test_init.Cache.Exists(s.ctx, fxAppliedRateCacheKey)
		require.NoError(s.T(), err, "проверка кэша курса")

		if !exists {
			return
		}

		time.Sleep(fxCacheDropDelay)
	}

	require.FailNow(s.T(), "кэш курса не удалось снять: ключ "+fxAppliedRateCacheKey+" появляется снова")
}

// dropFXCache сбрасывает кэш применённого курса.
//
// Без ожидания: используется при ВОЗВРАТЕ курса, где важно лишь то, что
// следующий читатель пойдёт в базу, а не то, каким окажется кэш сейчас.
func (s *HTTPTestSuite) dropFXCache() {
	s.T().Helper()

	for _, key := range []string{fxAppliedRateCacheKey, fxAppliedUSDRateCacheKey} {
		if err := test_init.Cache.Delete(s.ctx, key); err != nil {
			s.T().Logf("сброс кэша курса %s: %s", key, err)
		}
	}
}

// Ключи кэша применённого курса. Совпадают с теми, что строит
// internal/storage/dragonfly.
//
// Пар две, потому что и цен на витрине две: тенге и доллар считаются по своим
// курсам и кэшируются порознь. Сбросить одну и решить, что кэш пуст, значит
// оставить второй сценарий читать курс, которого в базе уже нет.
const (
	fxAppliedRateCacheKey    = "fx:applied:" + seedFXBase + ":" + seedFXQuote
	fxAppliedUSDRateCacheKey = "fx:applied:" + seedFXBase + ":USD"
)
