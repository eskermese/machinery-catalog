package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/form"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/presenter"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// tokenIssuer выпускает токен доступа. Интерфейс объявлен здесь, а не в пакете
// реализации: сервису важен только факт выпуска, а замена собственных JWT на
// внешний IdP не должна затрагивать бизнес-логику входа.
type tokenIssuer interface {
	// Issue выпускает токен пользователя.
	Issue(userID string, role entity.ActorType, currentTime time.Time) (string, error)
	// IsConfigured сообщает, что выпуск токенов вообще возможен.
	IsConfigured() bool
}

// AuthService представляет интерфейс сервиса выпуска токенов.
type AuthService interface {
	// IssueToken выпускает токен по номеру телефона и роли маршрута.
	IssueToken(ctx context.Context, issueForm form.AuthTokenIssue, currentTime time.Time) (presenter.AuthToken, error)
	// EnsureAdmins создаёт учётные записи администраторов из конфигурации.
	EnsureAdmins(ctx context.Context, phones []string, currentTime time.Time) error
	// CurrentUser возвращает учётную запись действующего пользователя.
	//
	// Читает её из хранилища, а не собирает из токена: токен несёт лишь
	// идентификатор и роль, а показать нужно и номер — и, что важнее,
	// запись могла быть удалена после выпуска токена.
	CurrentUser(ctx context.Context, id string) (*entity.User, error)
	// ListStaff возвращает учётные записи сотрудников.
	ListStaff(ctx context.Context, role entity.ActorType) (entity.Users, error)
	// RenameSelf задаёт имя учётной записи, от имени которой идёт запрос.
	//
	// Только своё: имя ничего не решает в правах, а спрашивать его у кого-то
	// другого некому — вход идёт по номеру, форм регистрации нет, и знает
	// ответ сам человек.
	RenameSelf(ctx context.Context, id, name string) (*entity.User, error)
}

// authService реализация сервиса выпуска токенов.
type authService struct {
	userRepo repository.UserRepository // Репозиторий учётных записей
	tokens   tokenIssuer               // Выпуск токенов

	// selfRegistration разрешает менеджеру завести учётную запись входом.
	// Выключается там, где сервис доступен снаружи, — см. конфигурацию.
	selfRegistration bool
}

// NewAuthService создаёт сервис выпуска токенов.
func NewAuthService(
	userRepo repository.UserRepository,
	tokens tokenIssuer,
	selfRegistration bool,
) AuthService {
	return &authService{userRepo: userRepo, tokens: tokens, selfRegistration: selfRegistration}
}

// IssueToken выпускает токен по номеру телефона.
//
// Администратор входит, только если он заранее заведён: самостоятельная
// регистрация администратора означала бы, что права раздаёт тот, кто их
// запрашивает.
//
// Менеджер — по настройке. При включённой самостоятельной регистрации учётная
// запись создаётся первым входом: ручки «завести сотрудника» в API нет, и это
// единственный способ появиться в системе. При выключенной входит только тот,
// кто уже заведён, — иначе любой номер получал бы доступ к заявкам с именами
// и телефонами клиентов.
//
// Подтверждения владения номером нет ни в одном режиме — см. «Известные
// заглушки» в README.
func (s *authService) IssueToken(
	ctx context.Context,
	issueForm form.AuthTokenIssue,
	currentTime time.Time,
) (presenter.AuthToken, error) {
	if err := issueForm.Validate(); err != nil {
		return presenter.AuthToken{}, fmt.Errorf("валидация формы: %w", err)
	}

	user, err := s.resolveUser(ctx, issueForm, currentTime)
	if err != nil {
		return presenter.AuthToken{}, err
	}

	token, err := s.tokens.Issue(user.ID, user.Role, currentTime)
	if err != nil {
		return presenter.AuthToken{}, fmt.Errorf("выпуск токена: %w", err)
	}

	return presenter.NewAuthToken(token, user), nil
}

// resolveUser находит или создаёт учётную запись под роль маршрута.
func (s *authService) resolveUser(
	ctx context.Context,
	issueForm form.AuthTokenIssue,
	currentTime time.Time,
) (*entity.User, error) {
	if issueForm.Role == entity.ActorTypeAdmin {
		user, err := s.userRepo.GetByPhoneRole(ctx, issueForm.Phone, entity.ActorTypeAdmin)
		if err != nil {
			// Незаведённый администратор и опечатка в номере снаружи неотличимы:
			// подбирать номера администраторов по ответу сервиса не должно получаться.
			if errors.Is(err, entity.ErrUserNotFound) {
				return nil, entity.ErrRoleForbidden
			}

			return nil, fmt.Errorf("получение администратора: %w", err)
		}

		return user, nil
	}

	if !s.selfRegistration {
		user, err := s.userRepo.GetByPhoneRole(ctx, issueForm.Phone, issueForm.Role)
		if err != nil {
			// Незаведённый сотрудник и опечатка в номере снаружи неотличимы,
			// как и у администратора выше: подбирать номера по ответу сервиса
			// не должно получаться.
			if errors.Is(err, entity.ErrUserNotFound) {
				return nil, entity.ErrRoleForbidden
			}

			return nil, fmt.Errorf("получение сотрудника: %w", err)
		}

		return user, nil
	}

	user, err := s.userRepo.FindOrCreate(ctx, entity.NewUser(issueForm.Phone, issueForm.Role, currentTime))
	if err != nil {
		return nil, fmt.Errorf("получение пользователя: %w", err)
	}

	return user, nil
}

// EnsureAdmins создаёт учётные записи администраторов из конфигурации.
// Вызывается при старте приложения и идемпотентен: повторный запуск ничего не меняет.
func (s *authService) EnsureAdmins(ctx context.Context, phones []string, currentTime time.Time) error {
	for _, phone := range phones {
		normalized := entity.NormalizePhone(phone)

		if !entity.IsValidPhone(normalized) {
			logger.FromContext(ctx).WithFields(logger.Fields{
				"phone": phone,
			}).Warn("номер администратора пропущен: не похож на телефон")

			continue
		}

		admin := entity.NewUser(normalized, entity.ActorTypeAdmin, currentTime)
		if err := s.userRepo.EnsureExists(ctx, admin); err != nil {
			return fmt.Errorf("создание администратора: %w", err)
		}
	}

	return nil
}

// CurrentUser возвращает учётную запись действующего пользователя.
func (s *authService) CurrentUser(ctx context.Context, id string) (*entity.User, error) {
	if id == "" {
		return nil, entity.ErrAuthRequired
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("получение учётной записи: %w", err)
	}

	return user, nil
}

// ListStaff возвращает учётные записи сотрудников.
func (s *authService) ListStaff(ctx context.Context, role entity.ActorType) (entity.Users, error) {
	if role != "" && !entity.IsLoginRole(role) {
		return nil, entity.ErrRoleInvalid
	}

	users, err := s.userRepo.List(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("получение учётных записей: %w", err)
	}

	return users, nil
}

// RenameSelf задаёт имя своей учётной записи.
//
// Запись сначала читается, а не обновляется вслепую: идентификатор приходит
// из токена, а запись могла быть удалена уже после его выпуска — тогда
// правильный ответ «такой записи нет», а не молчаливое обновление нуля строк.
func (s *authService) RenameSelf(ctx context.Context, id, name string) (*entity.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("получение учётной записи: %w", err)
	}

	if nameErr := user.SetName(name); nameErr != nil {
		return nil, nameErr
	}

	if err = s.userRepo.UpdateName(ctx, user.ID, user.Name); err != nil {
		return nil, fmt.Errorf("сохранение имени: %w", err)
	}

	return user, nil
}
