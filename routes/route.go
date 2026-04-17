package routes

import (
	"net/http"
	"pureGo/controllers"
	"pureGo/middleware"
)

func SetUpRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /signup", controllers.Signup)
	mux.HandleFunc("POST /login", controllers.Login)
	mux.HandleFunc("GET /products", controllers.GetProducts)
	mux.HandleFunc("GET /products/search", controllers.SearchProducts)
	mux.HandleFunc("GET /products/paginated", controllers.GetProductsPaginated)
	mux.Handle("POST /products", middleware.AuthMiddleware(middleware.IdempotencyMiddleware(http.HandlerFunc(controllers.CreateProduct))))

	mux.HandleFunc("GET /products/{id}", controllers.GetProductByID)
	mux.Handle("PUT /products/{id}", middleware.AuthMiddleware(middleware.IdempotencyMiddleware(http.HandlerFunc(controllers.UpdateProduct))))
	mux.Handle("PUT /products/{id}/stock", middleware.AuthMiddleware(middleware.IdempotencyMiddleware(http.HandlerFunc(controllers.UpdateStockWithAudit))))
	mux.Handle("DELETE /products/{id}", middleware.AuthMiddleware(middleware.IdempotencyMiddleware(http.HandlerFunc(controllers.DeleteProduct))))

	mux.Handle("POST /stock/transfer", middleware.AuthMiddleware(middleware.IdempotencyMiddleware(http.HandlerFunc(controllers.TransferStockHandler))))

	mux.HandleFunc("GET /verify-email", controllers.VerifyEmail)
	mux.HandleFunc("POST /forgot-password", controllers.ForgotPassword)
	mux.HandleFunc("POST /reset-password", controllers.ResetPasswordSubmit)

	adminHandler := http.HandlerFunc(controllers.GetDashboard)
	warehouseAdminHandler := http.HandlerFunc(controllers.GetAdminDashboard)
	mux.Handle("GET /admin/dashboard",
		middleware.AuthMiddleware(
			middleware.RoleMiddleware("admin", adminHandler),
		),
	)
	mux.Handle("GET /admin/warehouse-dashboard",
		middleware.AuthMiddleware(
			middleware.RoleMiddleware("admin", warehouseAdminHandler),
		),
	)

	return middleware.Logger(mux)
}
