package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/eldarbr/go-s3/internal/auth"
	"github.com/eldarbr/go-s3/internal/model"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

type CacheImpl interface {
	GetAndIncrease(key string) int
}

type BusinessModule interface {
	CreateBucket(ctx context.Context, bucket *model.Bucket) error
	ListFiles(ctx context.Context, requesterUUID, bucketName string) ([]model.File, error)
	UploadFile(ctx context.Context, request model.UploadFileRequest) (*uuid.UUID, error)
	FetchFile(ctx context.Context, request model.FetchFileRequest) error
}

type APIHandler struct {
	jwtService *auth.JWTService
	cache      CacheImpl
	business   BusinessModule
	reqLimit   int
}

type ctxKey string

const (
	ctxKeyThisServiceUser ctxKey = "currentUserIdnetificator"
)

const (
	defaultRateLimiterIPSourceHeader = "X-Real-IP"
)

func writeJSONResponse(responseWriter http.ResponseWriter, response any, code int) {
	responseWriter.Header().Set("Content-Type", "application/json")

	resp, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		responseWriter.WriteHeader(http.StatusInternalServerError)
		responseWriter.Write([]byte("{\"error\": \"response marshal error\"}")) //nolint:errcheck // won't check.

		return
	}

	responseWriter.WriteHeader(code)
	responseWriter.Write(resp) //nolint:errcheck // won't check.
}

func NewAPIHandler(business BusinessModule, jwtService *auth.JWTService, cache CacheImpl, limit int) APIHandler {
	srv := APIHandler{
		jwtService: jwtService,
		cache:      cache,
		reqLimit:   limit,
		business:   business,
	}

	return srv
}

func (APIHandler) MethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	writeJSONResponse(w, model.ErrorResponse{Error: "method not allowed"}, http.StatusMethodNotAllowed)
}

func (APIHandler) NotFound(w http.ResponseWriter, _ *http.Request) {
	writeJSONResponse(w, model.ErrorResponse{Error: "not found"}, http.StatusNotFound)
}

func (apiHandler APIHandler) MiddlewareIPRateLimit(next httprouter.Handle) httprouter.Handle {
	return func(respWriter http.ResponseWriter, request *http.Request, routerParams httprouter.Params) {
		if apiHandler.cache == nil {
			log.Println("MiddlewareRateLimit uninitialized cache")
			writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

			return
		}

		ip := request.Header.Get(defaultRateLimiterIPSourceHeader)
		if ip != "" {
			ipRequests := apiHandler.cache.GetAndIncrease("ip:" + ip)
			if ipRequests > apiHandler.reqLimit {
				writeJSONResponse(respWriter, model.ErrorResponse{Error: "rate limited"}, http.StatusTooManyRequests)

				return
			}
		} else {
			log.Println("ratelimit Handler MiddlewareIPRateLimit didn't make a cache lookup - empty ip")
		}

		next(respWriter, request, routerParams)
	}
}

func (apiHandler APIHandler) MiddlewareAPIAuthorizeAnyClaim(requestedRoles []string, theServiceName string,
	next httprouter.Handle) httprouter.Handle {
	return func(respWriter http.ResponseWriter, request *http.Request, params httprouter.Params) {
		apiHandler.parseAuthToken(request.Header.Get("Authorization"), requestedRoles, theServiceName, next,
			respWriter, request, params)
	}
}

func (apiHandler APIHandler) MiddlewareFGWAuthorizeAnyClaim(requestedRoles []string, theServiceName string,
	next httprouter.Handle) httprouter.Handle {
	return func(respWriter http.ResponseWriter, request *http.Request, params httprouter.Params) {
		cookieToken, cookieErr := request.Cookie("tokenid")
		if cookieErr != nil {
			writeJSONResponse(respWriter, model.ErrorResponse{Error: "unauthorized"}, http.StatusUnauthorized)

			return
		}

		apiHandler.parseAuthToken(cookieToken.Value, requestedRoles, theServiceName, next, respWriter, request, params)
	}
}

func (apiHandler APIHandler) parseAuthToken(
	token string,
	requestedRoles []string,
	theServiceName string,
	next httprouter.Handle,
	respWriter http.ResponseWriter,
	request *http.Request,
	params httprouter.Params,
) {
	claims, err := apiHandler.jwtService.ValidateToken(token)
	if err != nil {
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "unauthorized"}, http.StatusUnauthorized)

		return
	}

	userRole := claims.FirstMatch(theServiceName, requestedRoles)

	if userRole == "" {
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "forbidden"}, http.StatusForbidden)

		return
	}

	nextCtx := context.WithValue(request.Context(), ctxKeyThisServiceUser, &auth.ThisServiceUser{
		UserRole: userRole,
		UserIdentificator: auth.UserIdentificator{
			Username: claims.Username,
			UserID:   claims.UserID,
		},
	})

	next(respWriter, request.WithContext(nextCtx), params)
}

func (apiHandler APIHandler) MiddlewareRateLimit(next httprouter.Handle) httprouter.Handle {
	return func(respWriter http.ResponseWriter, request *http.Request, routerParams httprouter.Params) {
		currentUser, ctxFetchOk := request.Context().Value(ctxKeyThisServiceUser).(*auth.ThisServiceUser)
		if apiHandler.cache == nil || !ctxFetchOk {
			log.Printf("MiddlewareRateLimit failed, cacheOk %v, ctxFechOk %v\n", apiHandler.cache != nil, ctxFetchOk)
			writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

			return
		}

		lookups := apiHandler.cache.GetAndIncrease("usr:" + currentUser.Username)
		if lookups > apiHandler.reqLimit {
			writeJSONResponse(respWriter, model.ErrorResponse{Error: "rate limited"}, http.StatusTooManyRequests)

			return
		}

		next(respWriter, request, routerParams)
	}
}

func (apiHandler APIHandler) CreateBucket(respWriter http.ResponseWriter, rawRequest *http.Request,
	_ httprouter.Params) {
	log.Printf("request CreateBucket received")

	var bucketRequest model.CreateBucketRequest

	// Decode the request body.
	err := json.NewDecoder(rawRequest.Body).Decode(&bucketRequest)
	if err != nil || !bucketRequest.Valid() {
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "bad request"}, http.StatusBadRequest)

		return
	}

	var (
		currentUserUUID uuid.UUID
		uuidParseError  error
	)

	currentUser, ctxFetchOk := rawRequest.Context().Value(ctxKeyThisServiceUser).(*auth.ThisServiceUser)

	if !ctxFetchOk {
		log.Println("bad ctx")
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

		return
	} else if currentUserUUID, uuidParseError = uuid.Parse(currentUser.UserID); uuidParseError != nil {
		log.Println("bad uuid")
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

		return
	}

	bucket := &model.Bucket{ //nolint:exhaustruct // the rest gets filled in the business.
		Name:         bucketRequest.Name,
		Availability: bucketRequest.Availability,
		OwnerID:      currentUserUUID,
	}

	err = apiHandler.business.CreateBucket(rawRequest.Context(), bucket)
	if err != nil {
		log.Println("Couldn't create a bucket", err.Error())
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "bad request"}, http.StatusBadRequest)

		return
	}

	writeJSONResponse(respWriter, model.CreateBucketResponse{
		SizeQuota: bucket.SizeQuota,
		Name:      bucket.Name,
	}, http.StatusOK)
}

func (apiHandler APIHandler) UploadFile(respWriter http.ResponseWriter, rawRequest *http.Request, p httprouter.Params) {
	mpReader, err := rawRequest.MultipartReader()
	if err != nil {
		log.Println("reader", err.Error())
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "bad request"}, http.StatusBadRequest)

		return
	}

	var (
		currentUserUUID uuid.UUID
		uuidParseError  error
	)

	currentUser, ctxFetchOk := rawRequest.Context().Value(ctxKeyThisServiceUser).(*auth.ThisServiceUser)

	if !ctxFetchOk {
		log.Println("bad ctx")
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

		return
	} else if currentUserUUID, uuidParseError = uuid.Parse(currentUser.UserID); uuidParseError != nil {
		log.Println("bad uuid")
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

		return
	}

	bucketName := p.ByName("bucketName")
	response := model.UploadFileResponse{Results: nil}

	for {
		part, partErr := mpReader.NextPart()
		if errors.Is(partErr, io.EOF) {
			break
		}

		if partErr != nil {
			log.Println("nextpart", partErr.Error())
			writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

			return
		}

		newFileUUID, saveErr := apiHandler.business.UploadFile(rawRequest.Context(), model.UploadFileRequest{
			FileContent:   part,
			RequesterUUID: currentUserUUID,
			BucketName:    bucketName,
			File: model.File{
				Filename: part.FileName(),
				Access:   model.FileAccessPrivate,
				MIME:     part.Header.Get("Content-Type"),
			},
		})

		newResult := model.UploadedFileInfo{
			FileName: part.FileName(),
		}

		if saveErr != nil {
			newResult.Result = model.UploadResultError
			newResult.Error = saveErr.Error()
		} else {
			newResult.Result = model.UploadResultOk
			newResult.IDstr = newFileUUID.String()
		}

		response.Results = append(response.Results, newResult)
	}

	writeJSONResponse(respWriter, response, http.StatusOK)
}

func (apiHandler APIHandler) GetFile(respWriter http.ResponseWriter, rawRequest *http.Request,
	params httprouter.Params) {
	var (
		currentUserUUID *uuid.UUID
		userToken       string
	)

	{ // prioritize Authorization header over the session token.
		userToken = rawRequest.Header.Get("Authorization")
		if userToken == "" {
			cookieToken, cookieErr := rawRequest.Cookie("tokenid")
			if cookieErr == nil {
				userToken = cookieToken.Value
			}
		}
	}

	if userToken != "" {
		claims, err := apiHandler.jwtService.ValidateToken(userToken)
		if err == nil {
			uuid, uuidParseError := uuid.Parse(claims.UserID)
			if uuidParseError != nil {
				log.Println("bad uuid")
				writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

				return
			}

			currentUserUUID = &uuid
		}
	}

	bucketName := params.ByName("bucketName")

	fileID, idParseErr := uuid.Parse(params.ByName("fileID"))
	if idParseErr != nil {
		log.Println("bad uuid")
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "bad request"}, http.StatusBadRequest)

		return
	}

	fetchReq := model.FetchFileRequest{
		BucketName:       bucketName,
		FileID:           fileID,
		RespWriter:       respWriter,
		RequestingUserID: currentUserUUID,
		RawRequest:       rawRequest,
	}

	err := apiHandler.business.FetchFile(rawRequest.Context(), fetchReq)
	if err != nil {
		log.Println(err.Error())
		writeJSONResponse(respWriter, model.ErrorResponse{Error: err.Error()}, http.StatusBadRequest)

		return
	}
}

func (apiHandler APIHandler) ListFiles(respWriter http.ResponseWriter, rawRequest *http.Request,
	params httprouter.Params) {
	log.Printf("request ListFiles received")

	currentUser, ctxFetchOk := rawRequest.Context().Value(ctxKeyThisServiceUser).(*auth.ThisServiceUser)
	if !ctxFetchOk {
		log.Println("bad ctx")
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "internal error"}, http.StatusInternalServerError)

		return
	}

	files, err := apiHandler.business.ListFiles(rawRequest.Context(), currentUser.UserID, params.ByName("bucketName"))
	if err != nil {
		log.Println("Couldn't list files in the bucket", params.ByName("bucketName"), err.Error())
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "bad request"}, http.StatusBadRequest)

		return
	}

	writeJSONResponse(respWriter, model.ListFilesResponse{
		Files: files,
	}, http.StatusOK)
}
