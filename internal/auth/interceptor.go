package auth

import (
	"context"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryAuthInterceptor(verifier *oidc.IDTokenVerifier) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		// Public routes — no token required
		if info.FullMethod == "/auth.v1.SecureService/Login" ||
			info.FullMethod == "/auth.v1.SecureService/Register" {
			return handler(ctx, req)
		}

		// Extract data from the "authorization" metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Get the "authorization" header value
		auth := md.Get("authorization")
		if len(auth) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}

		// Extract the token from the "Bearer <token>" format
		token := strings.TrimPrefix(auth[0], "Bearer ")

		if _, err := verifier.Verify(ctx, token); err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(ctx, req)
	}
}
