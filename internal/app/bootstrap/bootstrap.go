package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"org-structure-api/internal/config"
	"org-structure-api/internal/database"
	"org-structure-api/internal/handlers"
	"org-structure-api/internal/repository"
	"org-structure-api/internal/services"
)

type App struct {
	Router *http.ServeMux
}

func Initialize() *App {
	applicationConfig := config.Load()

	databaseConnection, connectionError := database.Connect(applicationConfig.DatabaseURL())
	if connectionError != nil {
		log.Fatalf("failed to connect to database: %v", connectionError)
	}

	departmentRepository := repository.NewDepartmentRepo(databaseConnection)
	employeeRepository := repository.NewEmployeeRepo(databaseConnection)

	departmentService := services.NewDepartmentService(departmentRepository)
	employeeService := services.NewEmployeeService(employeeRepository, departmentRepository)

	requestHandler := handlers.New(departmentService, employeeService, databaseConnection)
	requestMux := http.NewServeMux()
	requestHandler.Register(requestMux)

	return &App{Router: requestMux}
}

func Run() {
	applicationInstance := Initialize()
	applicationInstance.Start(config.Load().ServerPort)
}

func (appInstance *App) Start(port string) {
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: appInstance.Router,
	}

	go func() {
		log.Printf("Server is listening addr=:%s", port)
		if serverError := httpServer.ListenAndServe(); serverError != nil && !errors.Is(serverError, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", serverError)
		}
	}()

	shutdownSignalChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownSignalChannel, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignalChannel
	log.Println("Shutdown signal received")

	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if shutdownError := httpServer.Shutdown(shutdownContext); shutdownError != nil {
		log.Printf("server shutdown error: %v", shutdownError)
	}

	log.Println("Server exited")
}
