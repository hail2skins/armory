package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/controller"
)

// RegisterPaymentRoutes registers payment-related routes
func (s *Server) RegisterPaymentRoutes(r *gin.Engine, paymentController *controller.PaymentController) {
	// Pricing route
	r.GET("/pricing", paymentController.PricingHandler)

	// Checkout route
	r.POST("/checkout", paymentController.CreateCheckoutSession)

	// Payment success and cancel routes
	r.GET("/payment/success", paymentController.HandlePaymentSuccess)
	r.GET("/payment/cancel", paymentController.HandlePaymentCancellation)

	// Webhook route
	r.POST("/webhook", paymentController.HandleWebhook)

	// Payment history route (requires authentication)
	r.GET("/owner/payment-history", paymentController.ShowPaymentHistory)

	// Subscription cancellation routes
	r.GET("/subscription/cancel/confirm", paymentController.ShowCancelConfirmation)
	r.POST("/subscription/cancel", paymentController.CancelSubscription)
}
