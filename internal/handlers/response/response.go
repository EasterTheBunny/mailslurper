package response

import (
	"io"
	"log"
	"net/http"
	"reflect"

	"github.com/easterthebunny/render"
)

const (
	// limit request body to 1 KiB extend this as necessary.
	maxBodyReadLimit int64 = 1024
)

//nolint:gochecknoinits
func init() {
	render.Respond = SetDefaultResponder()
	render.Decode = SetDefaultDecoder()
}

// HTTPNoContentResponse ...
func HTTPNoContentResponse() *APIResponse {
	return &APIResponse{
		HTTPStatusCode: http.StatusNoContent,
		StatusText:     ""}
}

// HTTPNewOKResponse ...
func HTTPNewOKResponse(data render.Renderer) *APIResponse {
	return &APIResponse{
		HTTPStatusCode: http.StatusOK,
		StatusText:     "",
		Data:           data}
}

// HTTPNewOKListResponse ...
func HTTPNewOKListResponse(data []render.Renderer) *APIListResponse {
	return &APIListResponse{
		HTTPStatusCode: http.StatusOK,
		StatusText:     "",
		Data:           data}
}

// HTTPBadRequest ...
func HTTPBadRequest(err error) *APIResponse {
	return &APIResponse{
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Bad Request",
		Error:          NewErrorResponseSet(err)}
}

// HTTPConflict ...
func HTTPConflict(err error) *APIResponse {
	return &APIResponse{
		HTTPStatusCode: http.StatusConflict,
		StatusText:     http.StatusText(http.StatusConflict),
		Error:          NewErrorResponseSet(err)}
}

// HTTPInternalServerError ...
func HTTPInternalServerError(err error) *APIResponse {
	return &APIResponse{
		HTTPStatusCode: http.StatusInternalServerError,
		StatusText:     "Internal server error",
		Error:          NewErrorResponseSet(err)}
}

// HTTPNotFound ...
func HTTPNotFound(err error) *APIResponse {
	return &APIResponse{
		HTTPStatusCode: http.StatusNotFound,
		StatusText:     "Not Found",
		Error:          NewErrorResponseSet(err)}
}

// HTTPUnauthorized ...
func HTTPUnauthorized(err error) *APIResponse {
	return &APIResponse{
		HTTPStatusCode: http.StatusUnauthorized,
		StatusText:     "Unauthorized",
		Error:          NewErrorResponseSet(err)}
}

// HTTPStatusError ...
func HTTPStatusError(stat int, err error) *APIResponse {
	return &APIResponse{
		HTTPStatusCode: stat,
		StatusText:     http.StatusText(stat),
		Error:          NewErrorResponseSet(err)}
}

// NewDataResponse ...
func NewDataResponse(data render.Renderer) *APIResponse {
	return &APIResponse{Data: data}
}

// NewErrResponse provides a shortcut to produce a response with a single error.
func NewErrResponse(err []*ErrResponse) *APIResponse {
	return &APIResponse{Error: &err}
}

// NewErrorResponseSet ...
func NewErrorResponseSet(err error) *[]*ErrResponse {
	if err == nil {
		return &[]*ErrResponse{}
	}

	response := &ErrResponse{Err: err, ErrorText: err.Error()}

	return &[]*ErrResponse{response}
}

// NewList ...
func NewList(list interface{}) []render.Renderer {
	var renderables []render.Renderer

	valueOf := reflect.ValueOf(list)

	if valueOf.Kind() != reflect.Slice {
		return renderables
	}

	for idx := range valueOf.Len() {
		itemValueOf := valueOf.Index(idx)

		if itemValueOf.Kind() != reflect.Ptr {
			return renderables
		}

		if itemValueOf.Type().Implements(rendererType) {
			if itemValueOf.IsNil() {
				return renderables
			}

			renderable, isInterface := itemValueOf.Interface().(render.Renderer)
			if !isInterface {
				panic("value does not implement render.Renderer interface")
			}

			renderables = append(renderables, renderable)
		}
	}

	return renderables
}

func NewImageResponse(stat int, data []byte, tp ImageType) *ImageResponse {
	return &ImageResponse{
		HTTPStatusCode: stat,
		StatusText:     http.StatusText(stat),
		Data:           data,
		Type:           tp,
	}
}

func NewTextResponse(stat int, data []byte) *TextResponse {
	return &TextResponse{
		HTTPStatusCode: stat,
		StatusText:     http.StatusText(stat),
		Data:           data,
	}
}

var (
	//nolint:gochecknoglobals
	rendererType = reflect.TypeOf(new(render.Renderer)).Elem()
)

// APIResponse is the base type for all structured responses from the server.
type APIResponse struct {
	HTTPStatusCode int             `json:"-"`
	StatusText     string          `json:"-"` // user-level status message
	Data           render.Renderer `json:"data"`
	Error          *[]*ErrResponse `json:"errors,omitempty"`
}

// Render implements the render.Renderer interface for use with chi-router.
func (ar *APIResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// APIListResponse is the base type for all structured responses from the server.
type APIListResponse struct {
	HTTPStatusCode int               `json:"-"`
	StatusText     string            `json:"-"` // user-level status message
	Data           []render.Renderer `json:"data"`
	Error          *[]*ErrResponse   `json:"errors,omitempty"`
}

// Render implements the render.Renderer interface for use with chi-router.
func (ar *APIListResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// ErrResponse is the base type for all api errors.
type ErrResponse struct {
	Err       error  `json:"-"`                // low-level runtime error
	ErrorText string `json:"detail,omitempty"` // application-level error message, for debugging
}

// Render implements the render.Renderer interface for use with chi-router.
func (e *ErrResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	if e.Err != nil {
		e.ErrorText = e.Err.Error()
	}

	return nil
}

type ImageType uint8

const (
	PNG ImageType = iota
	JPEG
)

type ImageResponse struct {
	HTTPStatusCode int
	StatusText     string
	Data           []byte
	Type           ImageType
}

// Render implements the render.Renderer interface for use with chi-router.
func (r *ImageResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

type TextResponse struct {
	HTTPStatusCode int
	StatusText     string
	Data           []byte
}

// Render implements the render.Renderer interface for use with chi-router.
func (r *TextResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// JSONResponse is the base type for all structured responses from the server.
type JSONResponse struct {
	HTTPStatusCode int
	StatusText     string
	Value          render.Renderer
}

// Render implements the render.Renderer interface for use with chi-router.
func (_ *JSONResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// HTMLResponse is the base type for all structured responses from the server.
type HTMLResponse struct {
	HTTPStatusCode int
	StatusText     string
	Value          string
}

// Render implements the render.Renderer interface for use with chi-router.
func (_ *HTMLResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// DataResponse represents an octet stream response.
type DataResponse struct {
	HTTPStatusCode int
	StatusText     string
	Data           []byte
}

// Render implements the render.Renderer interface for use with chi-router.
func (_ *DataResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// SetDefaultResponder ...
func SetDefaultResponder() func(http.ResponseWriter, *http.Request, any) {
	return func(writer http.ResponseWriter, request *http.Request, value any) {
		switch valueType := value.(type) {
		case *APIResponse, *APIListResponse:
			render.JSON(writer, request, valueType)
		case *ImageResponse:
			var mime string

			switch valueType.Type {
			case PNG:
				mime = "image/png"
			case JPEG:
				mime = "image/jpeg"
			}

			writer.Header().Set("Content-Type", mime)
			writer.WriteHeader(valueType.HTTPStatusCode)
			_, _ = writer.Write(valueType.Data)
		case *TextResponse:
			render.PlainText(writer, request, string(valueType.Data))
		case *JSONResponse:
			render.JSON(writer, request, valueType.Value)
		case *HTMLResponse:
			render.HTML(writer, request, valueType.Value)
		case *DataResponse:
			render.Data(writer, request, valueType.Data)
		default:
			panic("response body incorrectly formatted")
		}
	}
}

// SetDefaultDecoder ...
func SetDefaultDecoder() func(*http.Request, interface{}) error {
	return func(request *http.Request, value interface{}) error {
		var err error

		switch request.Header.Get("Content-Type") {
		case "application/json":
			err = render.DecodeJSON(io.LimitReader(request.Body, maxBodyReadLimit), value)
			// in this case, there is a decode error; probably a malformed or malicious
			// input. panic and log the incident
			if err != nil {
				panic(err)
			}
		default:
			err = ErrUnsupportedContentType
		}

		return err
	}
}

func RenderOrLog(writer http.ResponseWriter, request *http.Request, resp render.Renderer, logger *log.Logger) {
	if err := render.Render(writer, request, resp); err != nil {
		logger.Print(err.Error())
	}
}

func ValidContextsAndMethod(request *http.Request, method string, values ...any) error {
	for _, value := range values {
		if reflect.ValueOf(value).IsNil() {
			return ErrIncorrectRoute
		}
	}

	if request.Method != method {
		return ErrMethodNotAllowed
	}

	return nil
}
