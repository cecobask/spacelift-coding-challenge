package gateway

import (
	"errors"
	"fmt"
	"github.com/cecobask/spacelift-coding-challenge/internal/storage"
	"github.com/go-chi/chi/v5"
	"io"
	"mime"
	"net/http"
)

type Handler struct {
	minio *storage.Minio
}

const formFileKey = "file"

func NewHandler(minio *storage.Minio) *Handler {
	return &Handler{
		minio: minio,
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	object, stat, err := h.minio.GetObject(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return &handlerError{
				Inner:      err,
				Message:    fmt.Sprintf("object with id %s not found", id),
				StatusCode: http.StatusNotFound,
			}
		}
		return &handlerError{
			Inner:      err,
			Message:    fmt.Sprintf("could not get object with id %s", id),
			StatusCode: http.StatusInternalServerError,
		}
	}
	fileExtensions, err := mime.ExtensionsByType(stat.Metadata["Content-Type"][0])
	if err != nil {
		return &handlerError{
			Inner:      err,
			Message:    fmt.Sprintf("could not get file extension for object with id %s", id),
			StatusCode: http.StatusInternalServerError,
		}
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s%s", id, fileExtensions[0]))
	w.Header().Set("Content-Type", stat.ContentType)
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(object); err != nil {
		return &handlerError{
			Inner:      err,
			Message:    "could not write object to response",
			StatusCode: http.StatusInternalServerError,
		}
	}
	return nil
}

func (h *Handler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) error {
	file, _, err := r.FormFile(formFileKey)
	if err != nil {
		return &handlerError{
			Inner:      err,
			Message:    fmt.Sprintf("could not get file from form with key %s", formFileKey),
			StatusCode: http.StatusBadRequest,
		}
	}
	defer file.Close()
	fileHeader := make([]byte, 512)
	if _, err = file.Read(fileHeader); err != nil {
		return &handlerError{
			Inner:      err,
			Message:    "could not read file",
			StatusCode: http.StatusInternalServerError,
		}
	}
	if _, err = file.Seek(0, 0); err != nil {
		return &handlerError{
			Inner:      err,
			Message:    "could not seek file",
			StatusCode: http.StatusInternalServerError,
		}
	}
	id := chi.URLParam(r, "id")
	data, err := io.ReadAll(file)
	if err != nil {
		return &handlerError{
			Inner:      err,
			Message:    "could not read file",
			StatusCode: http.StatusInternalServerError,
		}
	}
	if err = h.minio.PutObject(r.Context(), id, data, http.DetectContentType(fileHeader)); err != nil {
		return &handlerError{
			Inner:      err,
			Message:    fmt.Sprintf("could not create or update object with id %s", id),
			StatusCode: http.StatusInternalServerError,
		}
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

type handlerError struct {
	Inner      error
	Message    string
	StatusCode int
}

func (e *handlerError) Error() string {
	return e.Message
}

func (e *handlerError) Unwrap() error {
	return e.Inner
}

type handlerFunc func(http.ResponseWriter, *http.Request) error

func (f handlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := f(w, r); err != nil {
		var handlerErr *handlerError
		if errors.As(err, &handlerErr) {
			http.Error(w, handlerErr.Message, handlerErr.StatusCode)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
