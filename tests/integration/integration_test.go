package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mochaeng/payment-gateway/internal/app"
	"github.com/mochaeng/payment-gateway/internal/config"
	"github.com/mochaeng/payment-gateway/internal/constants"
	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valyala/fasthttp"
)

type IntegrationTestSuite struct {
	suite.Suite
	redisURL       string
	redisContainer testcontainers.Container
	ctx            context.Context
	app            *app.Application
	mockProcessors *MockProcessors
}

type MockProcessors struct {
	defaultServer    *httptest.Server
	fallbackServer   *httptest.Server
	defaultHealth    models.HealthResponse
	fallbackHealth   models.HealthResponse
	defaultPayments  []models.PaymentProcessorRequest
	fallbackPayments []models.PaymentProcessorRequest
}

func (suite *IntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	redisContainer, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.Require().NoError(err)
	suite.redisContainer = redisContainer

	host, err := redisContainer.Host(suite.ctx)
	suite.Require().NoError(err)

	port, err := redisContainer.MappedPort(suite.ctx, "6379")
	suite.Require().NoError(err)

	suite.redisURL = fmt.Sprintf("redis://%s:%s", host, port.Port())

	suite.setupMockProcessors()

	testConfig := &config.Config{
		Port:                "8080",
		RedisURL:            suite.redisURL,
		HealthCheckInterval: 1 * time.Second,
		RequestTimeout:      2 * time.Second,
		MaxQueueSize:        100,
		ProcessorThreshold:  300,
		Urls:                make(map[constants.PaymentMode]*config.ProcessorsConfig),
	}

	testConfig.Urls[constants.DefaultProcessorKey] = &config.ProcessorsConfig{
		BaseURL:    suite.mockProcessors.defaultServer.URL,
		PaymentURL: suite.mockProcessors.defaultServer.URL + "/payments",
		HealthURL:  suite.mockProcessors.defaultServer.URL + "/payments/service-health",
	}

	testConfig.Urls[constants.FallbackProcessorKey] = &config.ProcessorsConfig{
		BaseURL:    suite.mockProcessors.fallbackServer.URL,
		PaymentURL: suite.mockProcessors.fallbackServer.URL + "/payments",
		HealthURL:  suite.mockProcessors.fallbackServer.URL + "/payments/service-health",
	}

	app, err := app.NewApp(testConfig)
	suite.Require().NoError(err)
	suite.app = app

	// wait for health monitoring
	time.Sleep(2 * time.Second)
}

func (suite *IntegrationTestSuite) SetupTest() {
	suite.mockProcessors.defaultPayments = []models.PaymentProcessorRequest{}
	suite.mockProcessors.fallbackPayments = []models.PaymentProcessorRequest{}
	suite.mockProcessors.defaultHealth = models.HealthResponse{
		Failing:         false,
		MinResponseTime: 100,
	}
	suite.mockProcessors.fallbackHealth = models.HealthResponse{
		Failing:         false,
		MinResponseTime: 200,
	}
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.mockProcessors != nil {
		suite.mockProcessors.defaultServer.Close()
		suite.mockProcessors.fallbackServer.Close()
	}

	if suite.redisContainer != nil {
		suite.redisContainer.Terminate(suite.ctx)
	}
}

func (suite *IntegrationTestSuite) setupMockProcessors() {
	suite.mockProcessors = &MockProcessors{
		defaultHealth: models.HealthResponse{
			Failing:         false,
			MinResponseTime: 100,
		},
		fallbackHealth: models.HealthResponse{
			Failing:         false,
			MinResponseTime: 200,
		},
		defaultPayments:  []models.PaymentProcessorRequest{},
		fallbackPayments: []models.PaymentProcessorRequest{},
	}

	suite.mockProcessors.defaultServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/payments/service-health":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(suite.mockProcessors.defaultHealth)
		case "/payments":
			if r.Method == "POST" {
				var req models.PaymentProcessorRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				suite.mockProcessors.defaultPayments = append(suite.mockProcessors.defaultPayments, req)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(models.PaymentProcessorResponse{
					Message: "payment processed successfully",
				})
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	suite.mockProcessors.fallbackServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/payments/service-health":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(suite.mockProcessors.fallbackHealth)
		case "/payments":
			if r.Method == "POST" {
				var req models.PaymentProcessorRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				suite.mockProcessors.fallbackPayments = append(suite.mockProcessors.fallbackPayments, req)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(models.PaymentProcessorResponse{
					Message: "payment processed successfully",
				})
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func (suite *IntegrationTestSuite) TestPaymentProcessing_SinglePayment() {
	paymentReq := models.PaymentRequest{
		CorrelationID: "test-0001",
		Amount:        25.50,
	}

	reqBody, err := json.Marshal(paymentReq)
	suite.Require().NoError(err)

	var ctx fasthttp.RequestCtx
	ctx.Request.SetRequestURI("/payments")
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody(reqBody)

	server := suite.app.Mount()
	server.Handler(&ctx)

	suite.Equal(http.StatusOK, ctx.Response.StatusCode())

	var response map[string]string
	err = json.Unmarshal(ctx.Response.Body(), &response)
	suite.Require().NoError(err)
	suite.Equal("Payment processed", response["message"])

	time.Sleep(100 * time.Millisecond)

	suite.Len(suite.mockProcessors.defaultPayments, 1)
	suite.Len(suite.mockProcessors.fallbackPayments, 0)
	suite.Len(suite.mockProcessors.defaultPayments, 1)
	suite.Equal("test-0001", suite.mockProcessors.defaultPayments[0].CorrelationID)
	suite.Equal(25.50, suite.mockProcessors.defaultPayments[0].Amount)
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
