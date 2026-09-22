package errorcode

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const Domain = "taskmanager"

const DefaultLanguage = "en"

//go:embed error_codes.json
var catalogJSON []byte

const (
	CommonInvalidArgument    = "COMMON_INVALID_ARGUMENT"
	CommonNotFound           = "COMMON_NOT_FOUND"
	CommonAlreadyExists      = "COMMON_ALREADY_EXISTS"
	CommonFailedPrecondition = "COMMON_FAILED_PRECONDITION"
	CommonInternal           = "COMMON_INTERNAL"
	CommonUnavailable        = "COMMON_UNAVAILABLE"

	IdentityEmailRequired       = "IDENTITY_EMAIL_REQUIRED"
	IdentityDisplayNameRequired = "IDENTITY_DISPLAY_NAME_REQUIRED"
	IdentityProfileIDRequired   = "IDENTITY_PROFILE_ID_REQUIRED"
	IdentityProfileNotFound     = "IDENTITY_PROFILE_NOT_FOUND"
	IdentityEmailAlreadyExists  = "IDENTITY_EMAIL_ALREADY_EXISTS"

	IdentityAuthProviderRequired       = "IDENTITY_AUTH_PROVIDER_REQUIRED"
	IdentityAuthProviderUserIDRequired = "IDENTITY_AUTH_PROVIDER_USER_ID_REQUIRED"
	IdentityAuthProviderUnsupported    = "IDENTITY_AUTH_PROVIDER_UNSUPPORTED"
	IdentityAuthProfileInactive        = "IDENTITY_AUTH_PROFILE_INACTIVE"
	IdentityAuthStateInvalid           = "IDENTITY_AUTH_STATE_INVALID"
	IdentityAuthCodeRequired           = "IDENTITY_AUTH_CODE_REQUIRED"
	IdentityAuthExchangeFailed         = "IDENTITY_AUTH_EXCHANGE_FAILED"
	IdentityAuthTokenMissing           = "IDENTITY_AUTH_TOKEN_MISSING"
	IdentityAuthTokenInvalid           = "IDENTITY_AUTH_TOKEN_INVALID"
	IdentityAuthNotConfigured          = "IDENTITY_AUTH_NOT_CONFIGURED"

	TaskProfileIDRequired        = "TASK_PROFILE_ID_REQUIRED"
	TaskWorkspaceIDRequired      = "TASK_WORKSPACE_ID_REQUIRED"
	TaskTitleRequired            = "TASK_TITLE_REQUIRED"
	TaskIDRequired               = "TASK_ID_REQUIRED"
	TaskIDAndStatusIDRequired    = "TASK_ID_AND_STATUS_ID_REQUIRED"
	TaskNotFound                 = "TASK_NOT_FOUND"
	TaskParentNotInProfile       = "TASK_PARENT_NOT_IN_PROFILE"
	TaskHasSubtasks              = "TASK_HAS_SUBTASKS"
	TaskStatusSame               = "TASK_STATUS_SAME"
	TaskStatusNotInProfile       = "TASK_STATUS_NOT_IN_PROFILE"
	TaskStatusProfileMismatch    = "TASK_STATUS_PROFILE_MISMATCH"
	TaskNoStatusAvailable        = "TASK_NO_STATUS_AVAILABLE"
	TaskTransitionNotAllowed     = "TASK_TRANSITION_NOT_ALLOWED"
	TaskCycleDetected            = "TASK_CYCLE_DETECTED"
	TaskStatusNameRequired       = "TASK_STATUS_NAME_REQUIRED"
	TaskStatusSlugRequired       = "TASK_STATUS_SLUG_REQUIRED"
	TaskStatusSlugAlreadyExists  = "TASK_STATUS_SLUG_ALREADY_EXISTS"
	TaskStatusIDRequired         = "TASK_STATUS_ID_REQUIRED"
	TaskStatusNotFound           = "TASK_STATUS_NOT_FOUND"
	TaskStatusInUse              = "TASK_STATUS_IN_USE"
	TaskTransitionFieldsRequired = "TASK_TRANSITION_FIELDS_REQUIRED"
	TaskTransitionSameStatus     = "TASK_TRANSITION_SAME_STATUS"
	TaskTransitionIDRequired     = "TASK_TRANSITION_ID_REQUIRED"
	TaskTransitionNotFound       = "TASK_TRANSITION_NOT_FOUND"
	TaskTransitionAlreadyExists  = "TASK_TRANSITION_ALREADY_EXISTS"

	WorkspaceOwnerProfileIDRequired = "WORKSPACE_OWNER_PROFILE_ID_REQUIRED"
	WorkspaceNameRequired           = "WORKSPACE_NAME_REQUIRED"
	WorkspaceSlugRequired           = "WORKSPACE_SLUG_REQUIRED"
	WorkspaceIDRequired             = "WORKSPACE_ID_REQUIRED"
	WorkspaceNotFound               = "WORKSPACE_NOT_FOUND"
	WorkspaceSlugAlreadyExists      = "WORKSPACE_SLUG_ALREADY_EXISTS"
)

type LocalizedMessage struct {
	VI string `json:"vi"`
	EN string `json:"en"`
}

type Entry struct {
	Owner      string           `json:"owner"`
	GRPCCode   string           `json:"grpcCode"`
	HTTPStatus int              `json:"httpStatus"`
	Message    LocalizedMessage `json:"message"`
}

type catalog struct {
	Version     string           `json:"version"`
	Domain      string           `json:"domain"`
	DefaultCode string           `json:"defaultCode"`
	Codes       map[string]Entry `json:"codes"`
}

var loaded = mustLoad()

func mustLoad() catalog {
	var c catalog
	if err := json.Unmarshal(catalogJSON, &c); err != nil {
		panic(fmt.Sprintf("errorcode: error_codes.json không hợp lệ: %v", err))
	}
	if len(c.Codes) == 0 {
		panic("errorcode: error_codes.json không có mã lỗi nào")
	}
	return c
}

func Catalog() map[string]Entry { return loaded.Codes }

func Version() string { return loaded.Version }

func DefaultCode() string { return loaded.DefaultCode }

func Lookup(code string) (Entry, bool) {
	entry, ok := loaded.Codes[code]
	return entry, ok
}

func Message(code, lang string) (string, bool) {
	entry, ok := Lookup(code)
	if !ok {
		return "", false
	}
	if primary := strings.ToLower(firstLanguage(lang)); primary == "vi" {
		if entry.Message.VI != "" {
			return entry.Message.VI, true
		}
		if entry.Message.EN != "" {
			return entry.Message.EN, true
		}
		return "", false
	}
	if entry.Message.EN != "" {
		return entry.Message.EN, true
	}
	if entry.Message.VI != "" {
		return entry.Message.VI, true
	}
	return "", false
}

func firstLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return DefaultLanguage
	}
	return strings.SplitN(lang, "-", 2)[0]
}

func DefaultMessage(code string) string {
	if message, ok := Message(code, DefaultLanguage); ok {
		return message
	}
	if message, ok := Message(DefaultCode(), DefaultLanguage); ok {
		return message
	}
	return DefaultCode()
}

func Error(code string) error {
	st := status.New(grpcCode(code), DefaultMessage(code))
	detailed, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: code,
		Domain: Domain,
	})
	if err != nil {
		return st.Err()
	}
	return detailed.Err()
}

func grpcCode(code string) codes.Code {
	entry, ok := Lookup(code)
	if !ok {
		return codes.Internal
	}
	if parsed, ok := grpcCodeNames[entry.GRPCCode]; ok {
		return parsed
	}
	return codes.Internal
}

var grpcCodeNames = map[string]codes.Code{
	"InvalidArgument":    codes.InvalidArgument,
	"NotFound":           codes.NotFound,
	"AlreadyExists":      codes.AlreadyExists,
	"FailedPrecondition": codes.FailedPrecondition,
	"Aborted":            codes.Aborted,
	"PermissionDenied":   codes.PermissionDenied,
	"Unauthenticated":    codes.Unauthenticated,
	"ResourceExhausted":  codes.ResourceExhausted,
	"Unavailable":        codes.Unavailable,
	"Internal":           codes.Internal,
}
