package mail

import (
	"eraya/config"
	"eraya/domain"
	"fmt"
	"net/smtp"
	"strings"
)

type Mailer interface {
	SendOTP(to string, otp string) error
	SendOrderStatusUpdate(order *domain.Order, status string, estimatedDate string) error
}

type smtpMailer struct {
	config      config.SMTPConfig
	frontendURL string
}

func NewMailer(cnf config.SMTPConfig, frontendURL string) Mailer {
	if cnf.Host == "" {
		return &mockMailer{}
	}
	return &smtpMailer{config: cnf, frontendURL: frontendURL}
}

func (m *smtpMailer) SendOTP(to string, otp string) error {
	subject := "Subject: Eraya | Security Verification Code\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: auto; padding: 30px; border: 1px solid #f1f5f9; border-radius: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1 style="color: #0f172a; margin: 0; font-size: 24px; font-weight: 900; letter-spacing: -1px;">ERAYA</h1>
			</div>
			<h2 style="color: #1e293b; font-size: 20px; font-weight: 800; margin-bottom: 10px; text-align: center;">Security Verification</h2>
			<p style="color: #64748b; line-height: 1.6; text-align: center;">Hello,</p>
			<p style="color: #64748b; line-height: 1.6; text-align: center;">You are performing a sensitive action on your account. Please use the verification code below to authorize this request.</p>
			
			<div style="background: #f8fafc; padding: 30px; border-radius: 16px; margin: 25px 0; text-align: center; border: 2px dashed #e2e8f0;">
				<p style="margin: 0; font-size: 10px; font-weight: 900; color: #94a3b8; text-transform: uppercase; letter-spacing: 2px;">Verification Code</p>
				<p style="margin: 10px 0 0 0; font-size: 36px; font-weight: 900; color: #4338ca; letter-spacing: 8px;">%s</p>
			</div>

			<hr style="border: none; border-top: 1px solid #f1f5f9; margin: 30px 0;">
			<p style="font-size: 11px; color: #94a3b8; text-align: center;">&copy; 2026 Eraya. All rights reserved.</p>
		</div>
	`, otp)

	fromHeader := fmt.Sprintf("From: Eraya Support <%s>\n", m.config.User)
	msg := []byte(fromHeader + subject + mime + body)
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.User, m.config.Password, m.config.Host)

	return smtp.SendMail(addr, auth, m.config.User, []string{to}, msg)
}

func (m *smtpMailer) SendOrderStatusUpdate(order *domain.Order, status string, estimatedDate string) error {
	to := order.User.Email
	subject := fmt.Sprintf("Subject: Eraya | Order #%d Status Update: %s\n", order.ID, status)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	statusColor := "#0f172a" // Dark Slate
	statusDesc := "Your order status has been updated."

	switch status {
	case "Confirmed":
		statusColor = "#10b981" // Emerald
		statusDesc = "Good news! Your order has been accepted and is currently being prepared."
	case "Processing":
		statusColor = "#8b5cf6" // Purple
		statusDesc = "Your order is now being processed and carefully prepared for shipment."
	case "Shipped":
		statusColor = "#3b82f6" // Blue
		statusDesc = "Great news! Your package is now in transit and on its way to you."
	case "Delivered":
		statusColor = "#059669" // Green
		statusDesc = "Excellent! Your order has been successfully delivered. Enjoy your purchase!"
	case "Cancelled":
		statusColor = "#ef4444" // Red
		statusDesc = "We're sorry to inform you that your order has been cancelled."
	}

	estHtml := ""
	if estimatedDate != "" && status != "Delivered" && status != "Cancelled" {
		estHtml = fmt.Sprintf(`
			<div style="margin-top: 20px; padding: 15px; background: #f8fafc; border-radius: 12px; border: 1px solid #e2e8f0;">
				<p style="margin: 0; font-size: 10px; font-weight: 900; color: #64748b; text-transform: uppercase; letter-spacing: 1px;">Estimated Delivery</p>
				<p style="margin: 5px 0 0 0; font-size: 15px; font-weight: 800; color: #0f172a;">%s</p>
			</div>
		`, estimatedDate)
	}

	var itemRows []string
	for _, item := range order.Items {
		pName := "Product"
		if item.Product != nil {
			pName = item.Product.Name
		}
		row := fmt.Sprintf(`
			<div style="padding: 12px 0; border-bottom: 1px solid #f1f5f9;">
				<div style="display: flex; justify-content: space-between; align-items: center;">
					<div style="flex: 1;">
						<p style="margin: 0; font-size: 14px; font-weight: 800; color: #0f172a;">%s</p>
						<p style="margin: 4px 0 0 0; font-size: 12px; font-weight: 600; color: #64748b;">Quantity: %d</p>
					</div>
					<div style="text-align: right;">
						<p style="margin: 0; font-size: 14px; font-weight: 900; color: #0f172a;">৳%.0f</p>
					</div>
				</div>
			</div>
		`, pName, item.Quantity, item.PriceAtPurchase*float64(item.Quantity))
		itemRows = append(itemRows, row)
	}

	displayStatus := status
	if status == "Confirmed" {
		displayStatus = "Accepted"
	}

	trackingLink := m.frontendURL + "/profile"

	body := fmt.Sprintf(`
		<div style="font-family: 'Inter', -apple-system, system-ui, sans-serif; max-width: 600px; margin: auto; padding: 40px; border: 1px solid #f1f5f9; border-radius: 32px; background: white;">
			<div style="text-align: center; margin-bottom: 40px;">
				<h1 style="margin: 0; font-size: 24px; font-weight: 900; color: #0f172a; letter-spacing: -1px;">ERAYA</h1>
				<p style="margin: 5px 0 0 0; font-size: 10px; font-weight: 800; color: #94a3b8; text-transform: uppercase; letter-spacing: 3px;">Premium Artisanal Goods</p>
			</div>

			<div style="margin-bottom: 40px; text-align: center;">
				<div style="display: inline-block; padding: 6px 16px; background: %s; color: white; border-radius: 100px; font-size: 9px; font-weight: 900; text-transform: uppercase; letter-spacing: 2px; margin-bottom: 15px;">
					%s
				</div>
				<h2 style="margin: 0; font-size: 24px; font-weight: 900; color: #0f172a; letter-spacing: -0.5px;">Order Status Update</h2>
				<p style="margin: 15px 0 0 0; font-size: 14px; font-weight: 500; color: #64748b; line-height: 1.6;">Hello <strong>%s</strong>,<br>%s</p>
			</div>

			<div style="background: #f8fafc; border-radius: 24px; padding: 30px; border: 1px solid #f1f5f9;">
				<div style="margin-bottom: 25px; padding-bottom: 15px; border-bottom: 1px solid #e2e8f0;">
					<p style="margin: 0; font-size: 10px; font-weight: 900; color: #94a3b8; text-transform: uppercase; letter-spacing: 1.5px;">Order Number</p>
					<p style="margin: 5px 0 0 0; font-size: 20px; font-weight: 900; color: #0f172a;">#%d</p>
				</div>

				<div style="margin-bottom: 25px;">
					<p style="margin: 0; font-size: 10px; font-weight: 900; color: #94a3b8; text-transform: uppercase; letter-spacing: 1.5px;">Placed On</p>
					<p style="margin: 5px 0 0 0; font-size: 14px; font-weight: 800; color: #0f172a;">%s</p>
				</div>

				<div style="margin-bottom: 25px;">
					<p style="margin: 0 0 10px 0; font-size: 10px; font-weight: 900; color: #94a3b8; text-transform: uppercase; letter-spacing: 1.5px;">Items Purchased</p>
					%s
				</div>

				<div style="padding-top: 15px; border-top: 2px dashed #e2e8f0; display: flex; justify-content: space-between; align-items: center;">
					<p style="margin: 0; font-size: 10px; font-weight: 900; color: #94a3b8; text-transform: uppercase; letter-spacing: 1.5px;">Total Amount</p>
					<p style="margin: 0; font-size: 22px; font-weight: 900; color: %s; letter-spacing: -0.5px;">৳%.0f</p>
				</div>

				%s
			</div>

			<div style="text-align: center; margin-top: 30px;">
				<a href="%s" style="display: inline-block; padding: 16px 32px; background: #0f172a; color: white; text-decoration: none; border-radius: 16px; font-size: 12px; font-weight: 900; text-transform: uppercase; letter-spacing: 1px; transition: all 0.3s ease;">Live Tracking Details</a>
			</div>
			
			<p style="margin: 40px 0 0 0; font-size: 11px; color: #94a3b8; text-align: center; letter-spacing: 0.5px;">&copy; 2026 Eraya. All rights reserved.<br>Premium Artisanal Goods | Dhaka, Bangladesh</p>
		</div>
	`, statusColor, displayStatus, order.User.FullName, statusDesc, order.ID, order.CreatedAt.Format("02 Jan 2006"), strings.Join(itemRows, ""), statusColor, order.TotalPrice, estHtml, trackingLink)

	fromHeader := fmt.Sprintf("From: Eraya Support <%s>\n", m.config.User)
	msg := []byte(fromHeader + subject + mime + body)
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.User, m.config.Password, m.config.Host)

	return smtp.SendMail(addr, auth, m.config.User, []string{to}, msg)
}

type mockMailer struct{}

func (m *mockMailer) SendOTP(to string, otp string) error {
	fmt.Printf("\n--- MOCK OTP EMAIL SENT ---\nTo: %s\nOTP: %s\n---------------------------\n\n", to, otp)
	return nil
}

func (m *mockMailer) SendOrderStatusUpdate(order *domain.Order, status string, estimatedDate string) error {
	fmt.Printf("\n--- MOCK STATUS EMAIL SENT ---\nTo: %s\nOrder: #%d\nStatus: %s\nEst. Date: %s\n-----------------------------\n\n", order.User.Email, order.ID, status, estimatedDate)
	return nil
}
