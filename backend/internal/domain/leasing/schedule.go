package leasing

import (
	"github.com/shopspring/decimal"
)

// annuitySchedule строит график с постоянным ежемесячным платежом.
//
// Остаток долга на старте включает выкупную стоимость и до последнего платежа
// не опускается ниже неё: выкуп не амортизируется, а гасится в конце. Поэтому
// регулярный платёж считается по приведённой стоимости за вычетом выкупа.
func annuitySchedule(financed, residual int64, rate decimal.Decimal, term int) []Payment {
	payment := annuityPayment(financed, residual, rate, term)

	// Часть платежа, уходящая в тело, — остаток постоянной суммы после процентов.
	// Она может оказаться нулевой или отрицательной при большой выкупной стоимости,
	// и это корректно: такой договор гасит только проценты, а тело уходит в выкуп.
	return buildSchedule(financed, rate, term, func(interest int64) int64 {
		return payment - interest
	})
}

// annuityPayment считает постоянный ежемесячный платёж.
//
// Формула приведённой стоимости с выкупным платежом в конце:
//
//	PV = P * (1 - (1+i)^-n) / i + R * (1+i)^-n
//
// откуда, домножив на f = (1+i)^n, чтобы обойтись одним делением:
//
//	P = i * (PV*f - R) / (f - 1)
//
// При нулевой ставке формула вырождается: и числитель, и знаменатель обращаются
// в ноль, поэтому нулевая ставка обрабатывается отдельной ветвью, а не
// подстановкой близкого к нулю значения. Экономически это тот же предел:
// без процентов тело за вычетом выкупа делится на срок поровну.
func annuityPayment(financed, residual int64, rate decimal.Decimal, term int) int64 {
	if rate.IsZero() {
		amortized := financed - residual

		return toCents(
			decimal.NewFromInt(amortized).DivRound(decimal.NewFromInt(int64(term)), internalPrecision),
		)
	}

	factor := compoundFactor(rate, term)
	one := decimal.NewFromInt(1)

	numerator := rate.Mul(
		decimal.NewFromInt(financed).Mul(factor).Sub(decimal.NewFromInt(residual)),
	)

	return toCents(numerator.DivRound(factor.Sub(one), internalPrecision))
}

// differentiatedSchedule строит график с равными долями основного долга.
//
// Делится только амортизируемая часть — сумма финансирования за вычетом
// выкупной стоимости; выкуп целиком уходит в последний платёж.
func differentiatedSchedule(financed, residual int64, rate decimal.Decimal, term int) []Payment {
	amortized := financed - residual

	regularPrincipal := toCents(
		decimal.NewFromInt(amortized).DivRound(decimal.NewFromInt(int64(term)), internalPrecision),
	)

	return buildSchedule(financed, rate, term, func(_ int64) int64 {
		return regularPrincipal
	})
}

// buildSchedule разворачивает график, применяя правило принципала к каждому платежу.
//
// Общая для обоих типов графика часть: начисление процентов на остаток и сборка
// строк. Последний платёж правило не использует — он всегда гасит весь остаток,
// то есть забирает и выкупную стоимость, и накопленную разницу округления.
// Благодаря этому сумма долей тела по графику точно равна сумме финансирования,
// а сумма платежей — сумме финансирования вместе с процентами.
func buildSchedule(
	financed int64,
	rate decimal.Decimal,
	term int,
	principalFor func(interest int64) int64,
) []Payment {
	schedule := make([]Payment, 0, term)
	balance := financed

	for number := 1; number <= term; number++ {
		interest := interestOn(balance, rate)

		principal := principalFor(interest)
		if number == term {
			principal = balance
		}

		balance -= principal

		schedule = append(schedule, Payment{
			Number:         number,
			PrincipalCents: principal,
			InterestCents:  interest,
			AmountCents:    principal + interest,
			BalanceCents:   balance,
		})
	}

	return schedule
}

// interestOn начисляет проценты на остаток долга за один период.
//
// Округление до минорной единицы происходит на каждом платеже, а не в конце:
// клиент видит в графике именно те суммы, которые списываются со счёта.
func interestOn(balance int64, rate decimal.Decimal) int64 {
	if rate.IsZero() || balance == 0 {
		return 0
	}

	return toCents(decimal.NewFromInt(balance).Mul(rate))
}

// compoundFactor возвращает (1 + i)^n.
//
// Промежуточный результат усекается до внутренней точности на каждом шаге:
// точное возведение в степень дало бы число длиной в тысячи знаков, полностью
// незначимых на фоне минорной единицы.
func compoundFactor(rate decimal.Decimal, term int) decimal.Decimal {
	base := decimal.NewFromInt(1).Add(rate)
	factor := decimal.NewFromInt(1)

	for range term {
		factor = factor.Mul(base).Round(internalPrecision)
	}

	return factor
}
