package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"

	"github.com/eldarbr/go-s3/internal/myerrors"
	"github.com/golang-jwt/jwt"
)

var (
	ErrParsingToken = errors.New("couldn't parse the token")
	ErrWrongClaims  = errors.New("unknown claims type, cannot proceed")
)

type JWTService struct {
	publicKey *rsa.PublicKey
}

type ClaimUserRole struct {
	ServiceName string `json:"serviceName"`
	UserRole    string `json:"userRole"`
}

type AuthCustomClaims struct {
	UserIdentificator
	Roles []ClaimUserRole `json:"roles"`
}

type UserIdentificator struct {
	Username string `json:"username"`
	UserID   string `json:"userId"`
}

type ThisServiceUser struct {
	UserIdentificator
	UserRole string
}

type myCompletelaims struct {
	jwt.StandardClaims
	AuthCustomClaims
}

func NewJWTService(publicPath string) (*JWTService, error) {
	publicKeyBytes, err := os.ReadFile(publicPath)
	if err != nil {
		return nil, fmt.Errorf("NewJWTService public key read failed: %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("NewJWTService public key parse failed: %w", err)
	}

	return &JWTService{
		publicKey: publicKey,
	}, nil
}

func (jwtService *JWTService) ValidateToken(tokenString string) (*AuthCustomClaims, error) {
	if jwtService == nil {
		return nil, myerrors.ErrServiceNullPtr
	}

	var (
		claims        *myCompletelaims
		tokenClaimsOk bool
	)

	//nolint:exhaustruct // Only the type is what matters.
	token, err := jwt.ParseWithClaims(tokenString, &myCompletelaims{}, func(_ *jwt.Token) (interface{}, error) {
		return jwtService.publicKey, nil
	})
	if err != nil {
		return nil, ErrParsingToken
	} else if claims, tokenClaimsOk = token.Claims.(*myCompletelaims); !tokenClaimsOk {
		return nil, ErrWrongClaims
	}

	return &claims.AuthCustomClaims, nil
}

func (claims AuthCustomClaims) FirstMatch(serviceName string, requestedRoles []string) string {
	var theServiceRole string

	for _, role := range claims.Roles {
		if role.ServiceName == serviceName {
			theServiceRole = role.UserRole

			break
		}
	}

	for i := range requestedRoles {
		if requestedRoles[i] == "" {
			continue
		}

		if theServiceRole == requestedRoles[i] {
			return theServiceRole
		}
	}

	return ""
}
