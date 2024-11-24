package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

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
	UploadFile(ctx context.Context, request model.UploadFileRequest) error
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

func (apiHandler APIHandler) MiddlewareAuthorizeAnyClaim(requestedRoles []string, theServiceName string,
	next httprouter.Handle) httprouter.Handle {
	return func(respWriter http.ResponseWriter, request *http.Request, routerParams httprouter.Params) {
		claims, err := apiHandler.jwtService.ValidateToken(request.Header.Get("Authorization"))
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

		next(respWriter, request.WithContext(nextCtx), routerParams)
	}
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

	part, err := mpReader.NextPart()
	if err != nil {
		log.Println("nextpart", err.Error())
		writeJSONResponse(respWriter, model.ErrorResponse{Error: "bad request"}, http.StatusBadRequest)

		return
	}

	bucketID, err := strconv.Atoi(p.ByName("bucket"))
	if err != nil {
		log.Println("atoi", err.Error())
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

	err = apiHandler.business.UploadFile(rawRequest.Context(), model.UploadFileRequest{
		FileContent:   part,
		RequesterUUID: currentUserUUID,
		File: model.File{
			BucketID: int64(bucketID),
			Filename: part.FileName(),
			Access:   model.FileAccessPrivate,
		},
	})

	if err != nil {
		writeJSONResponse(respWriter, model.ErrorResponse{Error: err.Error()}, http.StatusInternalServerError)

		return
	}

	writeJSONResponse(respWriter, model.ErrorResponse{Error: "ok"}, http.StatusOK)
}

func (apiHandler APIHandler) GetFile(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	writeJSONResponse(w, model.ErrorResponse{Error: "GetFile"}, http.StatusOK)
}
