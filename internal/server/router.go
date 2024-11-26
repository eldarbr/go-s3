package server

import (
	"net/http"

	"github.com/eldarbr/go-s3/internal/model"
	"github.com/julienschmidt/httprouter"
)

const myOwnServiceName = "go-s3"

type APIHandlingModule interface {
	NotFound(w http.ResponseWriter, _ *http.Request)
	MiddlewareAuthorizeAnyClaim(requestedRoles []string, theServiceName string, next httprouter.Handle) httprouter.Handle
	MiddlewareRateLimit(next httprouter.Handle) httprouter.Handle
	MiddlewareIPRateLimit(next httprouter.Handle) httprouter.Handle

	CreateBucket(w http.ResponseWriter, r *http.Request, p httprouter.Params)
	UploadFile(w http.ResponseWriter, r *http.Request, p httprouter.Params)
	GetFile(w http.ResponseWriter, r *http.Request, p httprouter.Params)
}

func constructAdminOrRootMiddleware(apiHandler APIHandlingModule, final httprouter.Handle) httprouter.Handle {
	return apiHandler.MiddlewareIPRateLimit(apiHandler.MiddlewareAuthorizeAnyClaim(
		[]string{
			model.UserRoleTypeAdmin, model.UserRoleTypeRoot,
		},
		myOwnServiceName,
		apiHandler.MiddlewareRateLimit(final),
	))
}

func NewRouter(apiHandler APIHandlingModule) http.Handler {
	handler := httprouter.New()

	handler.HandleOPTIONS = false
	handler.RedirectTrailingSlash = false
	handler.HandleMethodNotAllowed = false
	handler.RedirectFixedPath = false

	handler.NotFound = http.HandlerFunc(apiHandler.NotFound)

	// create a bucket.
	handler.POST("/manage/buckets", constructAdminOrRootMiddleware(apiHandler, apiHandler.CreateBucket))

	// upload a file.
	handler.POST("/buckets/:bucket", constructAdminOrRootMiddleware(apiHandler, apiHandler.UploadFile))

	// download a file.
	handler.GET("/buckets/:bucket/:fileID", apiHandler.MiddlewareIPRateLimit(apiHandler.GetFile))

	return handler
}
