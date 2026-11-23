package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/internal/auth"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/form"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository/mock_repo"
)

const (
	testPhone       = "+77001112233"                         // Телефон в тестах входа
	testUserID      = "8f1a2b3c-4d5e-4f60-8a7b-9c0d1e2f3a4b" // Идентификатор пользователя в тестах
	testJWTSecret   = "test-secret"                          // Секрет подписи в тестах
	testOtherUserID = "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d" // Второй идентификатор в тестах
)

// testIssueTime момент выпуска токена в тестах.
var testIssueTime = time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)

// newTestAuthService собирает сервис входа на моке репозитория и настоящих токенах.
func newTestAuthService(t *testing.T) (AuthService, *mock_repo.MockUserRepository) {
	t.Helper()

	userRepo := mock_repo.NewMockUserRepository(gomock.NewController(t))

	return NewAuthService(userRepo, auth.NewTokenService(testJWTSecret, auth.TokenTTL), true), userRepo
}

// TestAuthService_IssueToken_ManagerEntersFreely фиксирует свободный вход менеджера:
// учётная запись создаётся при первом входе, повторный вход возвращает ту же.
func TestAuthService_IssueToken_ManagerEntersFreely(t *testing.T) {
	t.Parallel()

	srv, userRepo := newTestAuthService(t)

	userRepo.EXPECT().FindOrCreate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, user *entity.User) (*entity.User, error) {
			assert.Equal(t, testPhone, user.Phone, "номер нормализован до репозитория")
			assert.Equal(t, entity.ActorTypeManager, user.Role, "роль берётся из маршрута")

			user.ID = testUserID

			return user, nil
		})

	issued, err := srv.IssueToken(context.Background(), form.AuthTokenIssue{
		Phone: "+7 (700) 111-22-33", Role: entity.ActorTypeManager,
	}, testIssueTime)

	require.NoError(t, err)
	assert.Equal(t, testUserID, issued.User.ID, "идентификатор нужен клиенту: иначе его негде взять")
	assert.Equal(t, entity.ActorTypeManager.String(), issued.User.Role)
	assert.NotEmpty(t, issued.Token)
}

// TestAuthService_IssueToken_SamePhoneSameUser фиксирует, что повторный вход
// возвращает тот же идентификатор: иначе пользователь потерял бы всё, что было
// привязано к первой учётной записи.
func TestAuthService_IssueToken_SamePhoneSameUser(t *testing.T) {
	t.Parallel()

	srv, userRepo := newTestAuthService(t)

	userRepo.EXPECT().FindOrCreate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, user *entity.User) (*entity.User, error) {
			user.ID = testUserID

			return user, nil
		}).Times(2)

	first, err := srv.IssueToken(context.Background(),
		form.AuthTokenIssue{Phone: testPhone, Role: entity.ActorTypeManager}, testIssueTime)
	require.NoError(t, err)

	// Второй вход тем же номером в другом формате записи.
	second, err := srv.IssueToken(context.Background(),
		form.AuthTokenIssue{Phone: "8 700 111 22 33", Role: entity.ActorTypeManager}, testIssueTime)
	require.NoError(t, err)

	assert.Equal(t, first.User.ID, second.User.ID)
}

// TestAuthService_IssueToken_RolesAreSeparateAccounts фиксирует, что один телефон
// в разных ролях — это разные учётные записи с разными идентификаторами.
func TestAuthService_IssueToken_RolesAreSeparateAccounts(t *testing.T) {
	t.Parallel()

	srv, userRepo := newTestAuthService(t)

	userRepo.EXPECT().FindOrCreate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, user *entity.User) (*entity.User, error) {
			user.ID = testUserID

			return user, nil
		})
	userRepo.EXPECT().GetByPhoneRole(gomock.Any(), testPhone, entity.ActorTypeAdmin).
		Return(&entity.User{ID: testOtherUserID, Phone: testPhone, Role: entity.ActorTypeAdmin}, nil)

	asManager, err := srv.IssueToken(context.Background(),
		form.AuthTokenIssue{Phone: testPhone, Role: entity.ActorTypeManager}, testIssueTime)
	require.NoError(t, err)

	asAdmin, err := srv.IssueToken(context.Background(),
		form.AuthTokenIssue{Phone: testPhone, Role: entity.ActorTypeAdmin}, testIssueTime)
	require.NoError(t, err)

	assert.NotEqual(t, asManager.User.ID, asAdmin.User.ID)
}

// TestAuthService_IssueToken_Admin фиксирует, что администратор входит только
// если заведён заранее: самостоятельная регистрация означала бы, что права
// раздаёт тот, кто их запрашивает.
func TestAuthService_IssueToken_Admin(t *testing.T) {
	t.Parallel()

	t.Run("не заведён — 403", func(t *testing.T) {
		t.Parallel()

		srv, userRepo := newTestAuthService(t)

		userRepo.EXPECT().GetByPhoneRole(gomock.Any(), testPhone, entity.ActorTypeAdmin).
			Return(nil, entity.ErrUserNotFound)
		// Учётная запись администратора не должна создаваться входом.
		userRepo.EXPECT().FindOrCreate(gomock.Any(), gomock.Any()).Times(0)

		_, err := srv.IssueToken(context.Background(),
			form.AuthTokenIssue{Phone: testPhone, Role: entity.ActorTypeAdmin}, testIssueTime)

		require.ErrorIs(t, err, entity.ErrRoleForbidden)
	})

	t.Run("заведён — токен", func(t *testing.T) {
		t.Parallel()

		srv, userRepo := newTestAuthService(t)

		userRepo.EXPECT().GetByPhoneRole(gomock.Any(), testPhone, entity.ActorTypeAdmin).
			Return(&entity.User{ID: testUserID, Phone: testPhone, Role: entity.ActorTypeAdmin}, nil)

		issued, err := srv.IssueToken(context.Background(),
			form.AuthTokenIssue{Phone: testPhone, Role: entity.ActorTypeAdmin}, testIssueTime)

		require.NoError(t, err)
		assert.Equal(t, entity.ActorTypeAdmin.String(), issued.User.Role)
		assert.NotEmpty(t, issued.Token)
	})
}

// TestAuthService_IssueToken_InvalidPhone фиксирует отказ по непригодному номеру.
func TestAuthService_IssueToken_InvalidPhone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		phone string
	}{
		{name: "пустой", phone: ""},
		{name: "слишком короткий", phone: "+7700"},
		{name: "без цифр", phone: "не телефон"},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			srv, userRepo := newTestAuthService(t)

			// До хранилища запрос доходить не должен.
			userRepo.EXPECT().FindOrCreate(gomock.Any(), gomock.Any()).Times(0)
			userRepo.EXPECT().GetByPhoneRole(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, err := srv.IssueToken(context.Background(),
				form.AuthTokenIssue{Phone: s.phone, Role: entity.ActorTypeManager}, testIssueTime)

			require.Error(t, err)
		})
	}
}

// TestAuthService_EnsureAdmins фиксирует идемпотентную посадку администраторов
// и то, что мусорные номера пропускаются, не роняя старт приложения.
func TestAuthService_EnsureAdmins(t *testing.T) {
	t.Parallel()

	srv, userRepo := newTestAuthService(t)

	seeded := make([]string, 0, 2)

	userRepo.EXPECT().EnsureExists(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, user *entity.User) error {
			assert.Equal(t, entity.ActorTypeAdmin, user.Role)

			seeded = append(seeded, user.Phone)

			return nil
		}).Times(2)

	err := srv.EnsureAdmins(context.Background(),
		[]string{"+7 700 111 22 33", "мусор", "87002223344"}, testIssueTime)

	require.NoError(t, err)
	// Второй номер записан с ведущей восьмёркой: для КЗ и РФ это тот же абонент,
	// что и с семёркой, поэтому нормализация приводит его к +7.
	assert.Equal(t, []string{"+77001112233", "+77002223344"}, seeded, "номера нормализованы")
}

// TestAuthService_IssueToken_SecretMissing фиксирует поведение без секрета подписи:
// в local сервис поднимается, но выпуск токена завершается ошибкой.
func TestAuthService_IssueToken_SecretMissing(t *testing.T) {
	t.Parallel()

	userRepo := mock_repo.NewMockUserRepository(gomock.NewController(t))
	userRepo.EXPECT().FindOrCreate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, user *entity.User) (*entity.User, error) {
			user.ID = testUserID

			return user, nil
		})

	srv := NewAuthService(userRepo, auth.NewTokenService("", auth.TokenTTL), true)

	_, err := srv.IssueToken(context.Background(),
		form.AuthTokenIssue{Phone: testPhone, Role: entity.ActorTypeManager}, testIssueTime)

	require.ErrorIs(t, err, auth.ErrSecretMissing)
}

// TestAuthService_CurrentUser фиксирует чтение своей записи из хранилища.
//
// Из хранилища, а не из токена: токен несёт лишь идентификатор и роль,
// а показать нужно и номер — и, что важнее, запись могла быть удалена
// после выпуска токена.
func TestAuthService_CurrentUser(t *testing.T) {
	t.Parallel()

	t.Run("запись найдена", func(t *testing.T) {
		t.Parallel()

		srv, users := newTestAuthService(t)

		user := &entity.User{ID: "u-1", Phone: "+77001112233", Role: entity.ActorTypeManager}
		users.EXPECT().GetByID(gomock.Any(), "u-1").Return(user, nil)

		got, err := srv.CurrentUser(context.Background(), "u-1")

		require.NoError(t, err)
		assert.Equal(t, user, got)
	})

	t.Run("запись удалена после выпуска токена", func(t *testing.T) {
		t.Parallel()

		srv, users := newTestAuthService(t)

		users.EXPECT().GetByID(gomock.Any(), "u-1").Return(nil, entity.ErrUserNotFound)

		_, err := srv.CurrentUser(context.Background(), "u-1")

		require.ErrorIs(t, err, entity.ErrUserNotFound)
	})

	t.Run("пустой идентификатор", func(t *testing.T) {
		t.Parallel()

		srv, users := newTestAuthService(t)

		users.EXPECT().GetByID(gomock.Any(), gomock.Any()).Times(0)

		_, err := srv.CurrentUser(context.Background(), "")

		require.ErrorIs(t, err, entity.ErrAuthRequired,
			"пустой актор означает несработавшую аутентификацию, а не анонимного пользователя")
	})
}

// TestAuthService_ListStaff фиксирует список сотрудников и отбор по роли.
func TestAuthService_ListStaff(t *testing.T) {
	t.Parallel()

	t.Run("все сотрудники", func(t *testing.T) {
		t.Parallel()

		srv, users := newTestAuthService(t)

		staff := entity.Users{
			{ID: "u-1", Phone: "+77001112233", Role: entity.ActorTypeAdmin},
			{ID: "u-2", Phone: "+77002223344", Role: entity.ActorTypeManager},
		}
		users.EXPECT().List(gomock.Any(), entity.ActorType("")).Return(staff, nil)

		got, err := srv.ListStaff(context.Background(), "")

		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("отбор по роли", func(t *testing.T) {
		t.Parallel()

		srv, users := newTestAuthService(t)

		users.EXPECT().List(gomock.Any(), entity.ActorTypeManager).Return(entity.Users{}, nil)

		_, err := srv.ListStaff(context.Background(), entity.ActorTypeManager)

		require.NoError(t, err)
	})

	t.Run("роль вне перечня входа", func(t *testing.T) {
		t.Parallel()

		srv, users := newTestAuthService(t)

		users.EXPECT().List(gomock.Any(), gomock.Any()).Times(0)

		// system — роль самого сервиса, учётных записей под ней не бывает.
		_, err := srv.ListStaff(context.Background(), entity.ActorTypeSystem)

		require.ErrorIs(t, err, entity.ErrRoleInvalid)
	})

	t.Run("отказ хранилища", func(t *testing.T) {
		t.Parallel()

		srv, users := newTestAuthService(t)

		users.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("база недоступна"))

		_, err := srv.ListStaff(context.Background(), "")

		require.Error(t, err)
	})
}

// TestAuthService_RenameSelf фиксирует смену собственного имени.
func TestAuthService_RenameSelf(t *testing.T) {
	t.Parallel()

	t.Run("имя сохраняется обрезанным", func(t *testing.T) {
		t.Parallel()

		srv, userRepo := newTestAuthService(t)

		userRepo.EXPECT().GetByID(gomock.Any(), "user-1").
			Return(&entity.User{ID: "user-1", Phone: "+77001112233", Role: entity.ActorTypeManager}, nil)
		userRepo.EXPECT().UpdateName(gomock.Any(), "user-1", entity.Pointer("Иван Петров")).Return(nil)

		user, err := srv.RenameSelf(context.Background(), "user-1", "  Иван Петров  ")
		require.NoError(t, err)

		require.NotNil(t, user.Name)
		assert.Equal(t, "Иван Петров", *user.Name)
	})

	t.Run("пустая строка очищает имя", func(t *testing.T) {
		t.Parallel()

		srv, userRepo := newTestAuthService(t)

		userRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&entity.User{
			ID: "user-1", Phone: "+77001112233", Role: entity.ActorTypeManager,
			Name: entity.Pointer("Иван"),
		}, nil)
		userRepo.EXPECT().UpdateName(gomock.Any(), "user-1", (*string)(nil)).Return(nil)

		user, err := srv.RenameSelf(context.Background(), "user-1", "")
		require.NoError(t, err)

		assert.Nil(t, user.Name)
	})

	// Запись читается до записи: идентификатор приходит из токена, а запись
	// могли удалить уже после его выпуска.
	t.Run("удалённая запись — отказ, а не обновление нуля строк", func(t *testing.T) {
		t.Parallel()

		srv, userRepo := newTestAuthService(t)

		userRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(nil, entity.ErrUserNotFound)
		userRepo.EXPECT().UpdateName(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		_, err := srv.RenameSelf(context.Background(), "user-1", "Иван")
		require.ErrorIs(t, err, entity.ErrUserNotFound)
	})

	t.Run("отказ хранилища не теряется", func(t *testing.T) {
		t.Parallel()

		srv, userRepo := newTestAuthService(t)

		userRepo.EXPECT().GetByID(gomock.Any(), "user-1").
			Return(&entity.User{ID: "user-1", Phone: "+77001112233", Role: entity.ActorTypeManager}, nil)
		userRepo.EXPECT().UpdateName(gomock.Any(), "user-1", gomock.Any()).
			Return(entity.ErrDatabaseCommon)

		_, err := srv.RenameSelf(context.Background(), "user-1", "Иван")
		require.ErrorIs(t, err, entity.ErrDatabaseCommon)
	})

	// Слишком длинное имя до хранилища не доходит.
	t.Run("отказ домена не доходит до хранилища", func(t *testing.T) {
		t.Parallel()

		srv, userRepo := newTestAuthService(t)

		userRepo.EXPECT().GetByID(gomock.Any(), "user-1").
			Return(&entity.User{ID: "user-1", Phone: "+77001112233", Role: entity.ActorTypeManager}, nil)
		userRepo.EXPECT().UpdateName(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		_, err := srv.RenameSelf(context.Background(), "user-1", strings.Repeat("я", 101))
		require.ErrorIs(t, err, entity.ErrUserNameInvalid)
	})
}

// TestAuthService_IssueToken_SelfRegistrationOff фиксирует поведение сервиса,
// открытого наружу: входит только заранее заведённый сотрудник.
//
// До появления настройки менеджером становился ЛЮБОЙ номер — учётная запись
// создавалась первым входом. Внутри сети это было допустимо, снаружи нет:
// менеджер видит заявки с именами и телефонами клиентов.
//
// Незаведённый номер и опечатка снаружи неотличимы, как и у администратора:
// по ответу сервиса не должно получаться подбирать номера сотрудников.
func TestAuthService_IssueToken_SelfRegistrationOff(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	userRepo := mock_repo.NewMockUserRepository(ctrl)
	srv := NewAuthService(userRepo, auth.NewTokenService(testJWTSecret, auth.TokenTTL), false)

	const phone = "+77001234567"

	userRepo.EXPECT().
		GetByPhoneRole(gomock.Any(), phone, entity.ActorTypeManager).
		Return(nil, entity.ErrUserNotFound)

	_, err := srv.IssueToken(
		context.Background(),
		form.AuthTokenIssue{Phone: phone, Role: entity.ActorTypeManager},
		time.Now().UTC(),
	)

	require.ErrorIs(t, err, entity.ErrRoleForbidden)
}

// TestAuthService_IssueToken_SelfRegistrationOffKnown — заведённый сотрудник
// входит и при закрытой регистрации.
func TestAuthService_IssueToken_SelfRegistrationOffKnown(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	userRepo := mock_repo.NewMockUserRepository(ctrl)
	srv := NewAuthService(userRepo, auth.NewTokenService(testJWTSecret, auth.TokenTTL), false)

	const phone = "+77001234567"
	known := entity.NewUser(phone, entity.ActorTypeManager, time.Now().UTC())
	known.ID = "11111111-1111-1111-1111-111111111111"

	userRepo.EXPECT().
		GetByPhoneRole(gomock.Any(), phone, entity.ActorTypeManager).
		Return(known, nil)

	issued, err := srv.IssueToken(
		context.Background(),
		form.AuthTokenIssue{Phone: phone, Role: entity.ActorTypeManager},
		time.Now().UTC(),
	)

	require.NoError(t, err)
	require.NotEmpty(t, issued.Token)
}

// TestAuthService_IssueToken_SelfRegistrationOffFailure — отказ хранилища
// при закрытой регистрации отдаётся наружу как есть.
//
// Отдельно от «сотрудник не заведён»: там отказ означает «нельзя», здесь —
// «не смогли проверить». Свести их в одну ветку значило бы отвечать «нельзя»
// на недоступную базу и прятать поломку за отказом в доступе.
func TestAuthService_IssueToken_SelfRegistrationOffFailure(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	userRepo := mock_repo.NewMockUserRepository(ctrl)
	srv := NewAuthService(userRepo, auth.NewTokenService(testJWTSecret, auth.TokenTTL), false)

	userRepo.EXPECT().
		GetByPhoneRole(gomock.Any(), gomock.Any(), entity.ActorTypeManager).
		Return(nil, errors.New("база недоступна"))

	_, err := srv.IssueToken(
		context.Background(),
		form.AuthTokenIssue{Phone: "+77001234567", Role: entity.ActorTypeManager},
		time.Now().UTC(),
	)

	require.Error(t, err)
	require.NotErrorIs(t, err, entity.ErrRoleForbidden)
}
