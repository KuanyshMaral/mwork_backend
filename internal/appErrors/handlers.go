package appErrors

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse - стандартный ответ об ошибке
type ErrorResponse struct {
	Error *AppError `json:"error"`
}

// ErrorHandler - интерфейс для обработки ошибок
type ErrorHandler interface {
	HandleError(w http.ResponseWriter, r *http.Request, err error)
}

// HTTPErrorHandler - обработчик HTTP ошибок
type HTTPErrorHandler struct {
	Debug bool
}

func (h *HTTPErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *AppError

	switch e := err.(type) {
	case *AppError:
		appErr = e
	default:
		// Обернуть неизвестные ошибки
		appErr = InternalError(err)
		if !h.Debug {
			// В продакшене скрывать детали внутренних ошибок
			appErr = New(CodeInternalError, "Internal server error", http.StatusInternalServerError)
		}
	}

	// Логирование
	if appErr.HTTPCode >= 500 {
		log.Printf("Server error: %v", err)
	}

	// Отправка ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPCode)

	response := ErrorResponse{Error: appErr}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// Вспомогательная функция для быстрой обработки
func HandleHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	handler := &HTTPErrorHandler{Debug: true} // В проде установить false
	handler.HandleError(w, r, err)
}
