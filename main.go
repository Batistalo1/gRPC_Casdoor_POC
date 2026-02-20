package main

import (
	"context"
	"log"
	"net"

	"grpc-casdoor-poc/config"
	authv1 "grpc-casdoor-poc/gen/go/proto/auth/v1"
	"grpc-casdoor-poc/internal/auth"
	"grpc-casdoor-poc/internal/casdoor"
	"grpc-casdoor-poc/internal/store"

	"github.com/coreos/go-oidc/v3/oidc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type SecureServer struct {
	authv1.UnimplementedSecureServiceServer
	casdoor   *casdoor.Client
	userStore *store.UserStore
	verifier  *oidc.IDTokenVerifier
}

func (s *SecureServer) Ping(ctx context.Context, req *authv1.PingRequest) (*authv1.PingResponse, error) {
	return &authv1.PingResponse{
		Message: "Pong: " + req.Message,
	}, nil
}

func (s *SecureServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	// Login via Casdoor to get the token
	token, err := s.casdoor.Login(req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Verify the token and extract claims
	idToken, err := s.verifier.Verify(ctx, token)
	if err != nil {
		return nil, status.Error(codes.Internal, "token verification failed")
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	idToken.Claims(&claims)

	// Upsert the user in DB based on Casdoor ID (claims.Sub)
	user, err := s.userStore.Upsert(ctx, claims.Sub, claims.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "db error")
	}

	// Return the access token and user info
	return &authv1.LoginResponse{
		AccessToken: token,
		UserId:      user.ID,
		Email:       user.Email,
	}, nil
}

func (s *SecureServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	// Register the user in Casdoor
	if err := s.casdoor.Register(req.Username, req.Email, req.Password); err != nil {
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}

	// log in to get the token and user info
	token, err := s.casdoor.Login(req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "login after register failed")
	}

	// Verify the token and extract claims
	idToken, _ := s.verifier.Verify(ctx, token)
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	idToken.Claims(&claims)

	// Upsert the user in DB based on Casdoor ID (claims.Sub)
	user, err := s.userStore.Upsert(ctx, claims.Sub, claims.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "db error")
	}

	// Return the user info
	return &authv1.RegisterResponse{
		UserId: user.ID,
		Email:  user.Email,
	}, nil
}

func main() {
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	// get OIDC provider of Casdoor and verifier
	provider, err := oidc.NewProvider(context.Background(), conf.CasdoorURL)
	if err != nil {
		log.Fatal(err)
	}

	// Configure the ID token verifier with the expected client ID
	verifier := provider.Verifier(&oidc.Config{
		ClientID: conf.ClientID,
	})

	// Initialize the Casdoor client
	casdoorClient := &casdoor.Client{
		BaseURL:      conf.CasdoorURL,
		ClientID:     conf.ClientID,
		ClientSecret: conf.ClientSecret,
		Organization: conf.Organization,
		AppName:      conf.AppName,
	}

	// Connect to the database
	userStore, err := store.NewUserStore(conf.DatabaseURL)
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	// Start gRPC server with authentication interceptor
	lis, err := net.Listen("tcp", conf.ServerPort)
	if err != nil {
		log.Fatal(err)
	}

	// Create gRPC server with the authentication interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.UnaryAuthInterceptor(verifier)),
	)

	// Register the server with all dependencies
	authv1.RegisterSecureServiceServer(grpcServer, &SecureServer{
		casdoor:   casdoorClient,
		userStore: userStore,
		verifier:  verifier,
	})

	reflection.Register(grpcServer)

	log.Println("gRPC server running on :50051")
	grpcServer.Serve(lis)
}
