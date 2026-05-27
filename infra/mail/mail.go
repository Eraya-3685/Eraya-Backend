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
		<div style="font-family: 'Plus Jakarta Sans', 'Inter', system-ui, sans-serif; max-width: 550px; margin: 40px auto; padding: 40px; border: 1px solid #e2e8f0; border-radius: 24px; background: #ffffff; box-shadow: 0 10px 30px rgba(0, 0, 0, 0.02);">
			<div style="text-align: center; margin-bottom: 35px;">
				<div style="display: inline-block; width: 42px; height: 42px; background: #e11d48; border-radius: 12px; line-height: 42px; text-align: center; color: #ffffff; font-weight: 800; font-size: 22px;">E</div>
				<h1 style="color: #0f172a; margin: 10px 0 0 0; font-size: 24px; font-weight: 800; letter-spacing: -0.03em;">Eraya</h1>
				<p style="margin: 4px 0 0 0; font-size: 9px; font-weight: 800; color: #94a3b8; letter-spacing: 2px; text-transform: uppercase;">Security Protocol</p>
			</div>

			<div style="text-align: center; margin-bottom: 30px;">
				<h2 style="color: #0f172a; font-size: 20px; font-weight: 800; margin-bottom: 10px;">Verify Your Account</h2>
				<p style="color: #475569; font-size: 14.5px; line-height: 1.6; margin: 0 auto; max-width: 420px;">
					Please use the 6-digit verification code below to authorize your secure login or action on Eraya.
				</p>
			</div>
			
			<div style="background: #f8fafc; padding: 32px 24px; border-radius: 16px; margin: 30px 0; text-align: center; border: 1px solid #e2e8f0;">
				<p style="margin: 0 0 12px 0; font-size: 10.5px; font-weight: 800; color: #94a3b8; letter-spacing: 1.5px; text-transform: uppercase;">One-Time Password (OTP)</p>
				<p style="margin: 0; font-size: 40px; font-weight: 800; color: #e11d48; letter-spacing: 8px; font-family: monospace;">%s</p>
				<p style="margin: 15px 0 0 0; font-size: 11.5px; color: #64748b; font-weight: 500;">This code is active for 10 minutes. Do not share it with anyone.</p>
			</div>

			<p style="font-size: 13px; color: #64748b; line-height: 1.6; text-align: center;">
				If you did not make this request, you can safely ignore this email or contact support.
			</p>

			<hr style="border: none; border-top: 1px solid #f1f5f9; margin: 35px 0;">
			<p style="font-size: 11px; color: #94a3b8; text-align: center; line-height: 1.5;">
				&copy; 2026 Eraya. All rights reserved.<br/>
				Dhaka, Bangladesh | Support: 09678-ERAYA
			</p>
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
	statusBg := "#f1f5f9"
	statusDesc := "Your order status has been updated."

	switch status {
	case "Confirmed":
		statusColor = "#059669" // Green/Emerald
		statusBg = "#ecfdf5"
		statusDesc = "Good news! Your order has been accepted and is currently being prepared."
	case "Processing":
		statusColor = "#7c3aed" // Violet/Purple
		statusBg = "#f5f3ff"
		statusDesc = "Your order is now being processed and carefully prepared for shipment."
	case "Shipped":
		statusColor = "#2563eb" // Blue
		statusBg = "#eff6ff"
		statusDesc = "Great news! Your package is now in transit and on its way to you."
	case "Delivered":
		statusColor = "#16a34a" // Green
		statusBg = "#f0fdf4"
		statusDesc = "Excellent! Your order has been successfully delivered. Enjoy your purchase!"
	case "Cancelled":
		statusColor = "#dc2626" // Red
		statusBg = "#fef2f2"
		statusDesc = "We're sorry to inform you that your order has been cancelled."
	}

	estHtml := ""
	if estimatedDate != "" && status != "Delivered" && status != "Cancelled" {
		estHtml = fmt.Sprintf(`
			<div style="margin-top: 25px; padding: 20px; background: #ffffff; border-radius: 12px; border: 1px solid #e2e8f0; text-align: left;">
				<p style="margin: 0; font-size: 10px; font-weight: 800; color: #94a3b8; letter-spacing: 1px; text-transform: uppercase;">Estimated Delivery Date</p>
				<p style="margin: 6px 0 0 0; font-size: 15px; font-weight: 800; color: #0f172a;">%s</p>
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
			<tr style="border-bottom: 1px solid #f1f5f9;">
				<td style="padding: 16px 0; text-align: left; vertical-align: middle;">
					<p style="margin: 0; font-size: 13.5px; font-weight: 700; color: #0f172a;">%s</p>
					<p style="margin: 4px 0 0 0; font-size: 11.5px; font-weight: 600; color: #64748b;">Qty: %d</p>
				</td>
				<td style="padding: 16px 0; text-align: right; vertical-align: middle; font-size: 13.5px; font-weight: 800; color: #0f172a;">
					৳%.0f
				</td>
			</tr>
		`, pName, item.Quantity, item.PriceAtPurchase*float64(item.Quantity))
		itemRows = append(itemRows, row)
	}

	displayStatus := status
	if status == "Confirmed" {
		displayStatus = "Accepted"
	}

	trackingLink := m.frontendURL + "/profile"

	body := fmt.Sprintf(`
		<div style="font-family: 'Plus Jakarta Sans', 'Inter', system-ui, sans-serif; max-width: 550px; margin: 40px auto; padding: 40px; border: 1px solid #e2e8f0; border-radius: 24px; background: #ffffff; box-shadow: 0 10px 30px rgba(0, 0, 0, 0.02);">
			<div style="text-align: center; margin-bottom: 35px;">
				<div style="display: inline-block; width: 42px; height: 42px; background: #e11d48; border-radius: 12px; line-height: 42px; text-align: center; color: #ffffff; font-weight: 800; font-size: 22px;">E</div>
				<h1 style="color: #0f172a; margin: 10px 0 0 0; font-size: 24px; font-weight: 800; letter-spacing: -0.03em;">Eraya</h1>
				<p style="margin: 4px 0 0 0; font-size: 9px; font-weight: 800; color: #94a3b8; letter-spacing: 2px; text-transform: uppercase;">Exclusive E-commerce Hub</p>
			</div>

			<div style="margin-bottom: 35px; text-align: center;">
				<div style="display: inline-block; padding: 6px 16px; background: %s; color: %s; border-radius: 100px; font-size: 10px; font-weight: 800; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 15px;">
					%s
				</div>
				<h2 style="margin: 0; font-size: 22px; font-weight: 800; color: #0f172a; letter-spacing: -0.02em;">Order Status Updated</h2>
				<p style="margin: 15px auto 0 auto; font-size: 14.5px; font-weight: 500; color: #475569; line-height: 1.6; max-width: 440px;">
					Hello <strong>%s</strong>,<br>%s
				</p>
			</div>

			<div style="background: #f8fafc; border-radius: 20px; padding: 30px; border: 1px solid #e2e8f0;">
				<table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
					<tr>
						<td style="padding: 0 0 15px 0; border-bottom: 1px solid #e2e8f0; text-align: left;">
							<p style="margin: 0; font-size: 10px; font-weight: 800; color: #94a3b8; letter-spacing: 1px; text-transform: uppercase;">Order Number</p>
							<p style="margin: 4px 0 0 0; font-size: 18px; font-weight: 800; color: #0f172a;">#%d</p>
						</td>
						<td style="padding: 0 0 15px 0; border-bottom: 1px solid #e2e8f0; text-align: right;">
							<p style="margin: 0; font-size: 10px; font-weight: 800; color: #94a3b8; letter-spacing: 1px; text-transform: uppercase;">Placed On</p>
							<p style="margin: 4px 0 0 0; font-size: 13.5px; font-weight: 700; color: #0f172a;">%s</p>
						</td>
					</tr>
				</table>

				<div style="margin-bottom: 25px; text-align: left;">
					<p style="margin: 0 0 10px 0; font-size: 10px; font-weight: 800; color: #94a3b8; letter-spacing: 1px; text-transform: uppercase;">Items Purchased</p>
					<table style="width: 100%; border-collapse: collapse;">
						%s
					</table>
				</div>

				<table style="width: 100%; border-collapse: collapse; border-top: 2px dashed #cbd5e1; padding-top: 15px; margin-top: 15px;">
					<tr>
						<td style="padding: 15px 0 0 0; text-align: left; font-size: 11px; font-weight: 800; color: #94a3b8; letter-spacing: 1.5px; text-transform: uppercase; vertical-align: middle;">
							Total Paid / Payable
						</td>
						<td style="padding: 15px 0 0 0; text-align: right; font-size: 24px; font-weight: 800; color: #e11d48; letter-spacing: -0.5px; vertical-align: middle;">
							৳%.0f
						</td>
					</tr>
				</table>

				%s
			</div>

			<div style="text-align: center; margin-top: 30px;">
				<a href="%s" style="display: inline-block; padding: 16px 32px; background: #0f172a; color: #ffffff; text-decoration: none; border-radius: 14px; font-size: 12px; font-weight: 800; letter-spacing: 1.5px; text-transform: uppercase; box-shadow: 0 8px 24px rgba(15, 23, 42, 0.15);">Live Tracking Details</a>
			</div>
			
			<hr style="border: none; border-top: 1px solid #f1f5f9; margin: 35px 0;">
			<p style="margin: 0; font-size: 11px; color: #94a3b8; text-align: center; line-height: 1.5;">
				&copy; 2026 Eraya. All rights reserved.<br/>
				Premium Artisanal Goods | Dhaka, Bangladesh | Support: 09678-ERAYA
			</p>
		</div>
	`, statusBg, statusColor, displayStatus, order.User.FullName, statusDesc, order.ID, order.CreatedAt.Format("02 Jan 2006"), strings.Join(itemRows, ""), order.TotalPrice, estHtml, trackingLink)

	fromHeader := fmt.Sprintf("From: Eraya Support <%s>\n", m.config.User)
	msg := []byte(fromHeader + subject + mime + body)
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.User, m.config.Password, m.config.Host)

	return smtp.SendMail(addr, auth, m.config.User, []string{to}, msg)
}

type mockMailer struct{}

func (m *mockMailer) SendOTP(to string, otp string) error {
	return nil
}

func (m *mockMailer) SendOrderStatusUpdate(order *domain.Order, status string, estimatedDate string) error {
	return nil
}
