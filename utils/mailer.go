package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

func getSenderEmail() string {
	fromEmail := os.Getenv("FROM_EMAIL")
	if fromEmail != "" {
		return fromEmail
	}

	// Fallback for setups that only configure SMTP_EMAIL.
	return os.Getenv("SMTP_EMAIL")
}

func SendVerification(toEmail, token string) error {
	log.Println("[Email] Using Gmail SMTP")

	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if from == "" || password == "" {
		return fmt.Errorf("missing SMTP config")
	}

	auth := smtp.PlainAuth("", from, password, host)

	verifyLink := fmt.Sprintf("http://localhost:8080/verify-email?token=%s", token)

	subject := "Subject: Verify your email\r\n"
	body := fmt.Sprintf(`
Hello,

Click the link below to verify your email:

%s
`, verifyLink)

	msg := []byte(subject + "\r\n" + body)

	addr := host + ":" + port

	err := smtp.SendMail(addr, auth, from, []string{toEmail}, msg)
	if err != nil {
		log.Println("[Email] SMTP error:", err)
		return err
	}

	log.Println("[Email] Email sent successfully via Gmail SMTP")
	return nil
}

func SendResetEmail(targetEmail, token string) error {
	log.Println("[Email] Starting SendResetEmail")

	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || host == "" || port == "" {
		return fmt.Errorf("missing SMTP config for reset email")
	}

	auth := smtp.PlainAuth("", from, password, host)

	subject := "Subject: Reset your password\r\n"

	resetLink := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", token)
	body := fmt.Sprintf(`
Hello,

You requested a password reset.

Use the link below to set a new password:
%s

This link is valid for 15 minutes.
If you did not request this, you can ignore this email.
`, resetLink)

	msg := []byte(subject + "\r\n" + body)
	addr := host + ":" + port

	err := smtp.SendMail(addr, auth, from, []string{targetEmail}, msg)
	if err != nil {
		log.Println("[Email] SMTP reset email error:", err)
		return err
	}

	log.Println("[Email] Reset email sent successfully via Gmail SMTP")
	return nil
}

func SendLowStockAlert(productName string, currentQuantity int) error {
	fromEmail := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	adminEmail := os.Getenv("ADMIN_EMAIL")

	if fromEmail == "" || password == "" || host == "" || port == "" || adminEmail == "" {
		return fmt.Errorf("low stock alert failed: missing environment configuration")
	}
	auth := smtp.PlainAuth("", fromEmail, password, host)

	subject := fmt.Sprintf("Subject: CRITICAL: Low Stock Alert - %s\r\n", productName)
	body := fmt.Sprintf(
		"Attention: The product '%s' has reached a low stock level. Current quantity: %d. Please reorder immediately.\r\n",
		productName, currentQuantity,
	)

	msg := []byte(subject + "\r\n" + body)
	addr := host + ":" + port

	err := smtp.SendMail(addr, auth, fromEmail, []string{adminEmail}, msg)
	if err != nil {
		log.Println("[Email] SMTP low stock alert error:", err)
		return err
	}

	log.Printf("Low stock alert sent for %s (Current: %d)", productName, currentQuantity)
	return nil
}
