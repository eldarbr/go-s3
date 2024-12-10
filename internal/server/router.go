package server

import (
	"net/http"

	"github.com/eldarbr/go-s3/internal/model"
	"github.com/julienschmidt/httprouter"
)

const myOwnServiceName = "go-s3"

type APIHandlingModule interface {
	NotFound(w http.ResponseWriter, _ *http.Request)
	MiddlewareAPIAuthorizeAnyClaim(requestedRoles []string, theServiceName string,
		next httprouter.Handle) httprouter.Handle
	MiddlewareFGWAuthorizeAnyClaim(requestedRoles []string, theServiceName string,
		next httprouter.Handle) httprouter.Handle
	MiddlewareRateLimit(next httprouter.Handle) httprouter.Handle
	MiddlewareIPRateLimit(next httprouter.Handle) httprouter.Handle

	CreateBucket(w http.ResponseWriter, r *http.Request, p httprouter.Params)
	ListFiles(w http.ResponseWriter, r *http.Request, p httprouter.Params)
	EditFile(w http.ResponseWriter, r *http.Request, p httprouter.Params)
	DeleteFile(w http.ResponseWriter, r *http.Request, p httprouter.Params)
	UploadFile(w http.ResponseWriter, r *http.Request, p httprouter.Params)
	GetFile(w http.ResponseWriter, r *http.Request, p httprouter.Params)
}

func constructAdminOrRootMiddleware(apiHandler APIHandlingModule, final httprouter.Handle,
	authMiddle func(requestedRoles []string, theServiceName string, next httprouter.Handle) httprouter.Handle,
) httprouter.Handle {
	return apiHandler.MiddlewareIPRateLimit(authMiddle(
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
	handler.POST("/api/manage/buckets", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.CreateBucket, apiHandler.MiddlewareAPIAuthorizeAnyClaim))
	handler.POST("/fgw/manage/buckets", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.CreateBucket, apiHandler.MiddlewareFGWAuthorizeAnyClaim))

	// list files in a bucket.
	handler.GET("/api/manage/buckets/:bucketName/files", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.ListFiles, apiHandler.MiddlewareAPIAuthorizeAnyClaim))
	handler.GET("/fgw/manage/buckets/:bucketName/files", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.ListFiles, apiHandler.MiddlewareFGWAuthorizeAnyClaim))

	// edit a file.
	handler.PATCH("/api/manage/buckets/:bucketName/:fileID", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.EditFile, apiHandler.MiddlewareAPIAuthorizeAnyClaim))
	handler.PATCH("/fgw/manage/buckets/:bucketName/:fileID", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.EditFile, apiHandler.MiddlewareFGWAuthorizeAnyClaim))

	// delete a file.
	handler.DELETE("/api/manage/buckets/:bucketName/:fileID", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.DeleteFile, apiHandler.MiddlewareAPIAuthorizeAnyClaim))
	handler.DELETE("/fgw/manage/buckets/:bucketName/:fileID", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.DeleteFile, apiHandler.MiddlewareFGWAuthorizeAnyClaim))

	// upload a file.
	handler.POST("/api/buckets/:bucketName", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.UploadFile, apiHandler.MiddlewareAPIAuthorizeAnyClaim))
	handler.POST("/fgw/buckets/:bucketName", constructAdminOrRootMiddleware(
		apiHandler, apiHandler.UploadFile, apiHandler.MiddlewareFGWAuthorizeAnyClaim))

	// download a file.
	handler.GET("/buckets/:bucketName/:fileID", apiHandler.MiddlewareIPRateLimit(apiHandler.GetFile))

	return handler
}
