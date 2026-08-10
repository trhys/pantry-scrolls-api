package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

func (cfg *ApiConfig) SendVerificationEmail(targetEmail string, token string) error {
	base := strings.TrimRight(cfg.React, "/")
	verificationURL := fmt.Sprintf("%s/verify/%s", base, token)

	sender := "no-reply@pantryscrolls.com"
	subject := "Verify your email address"

	htmlBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
			<h2 style="color: #333333;">Welcome!</h2>
			<p style="color: #555555; font-size: 16px; line-height: 1.5;">
				Please verify your email address to activate your account. This link will expire in 30 minutes.
			</p>
			<div style="margin: 30px 0; text-align: center;">
				<a href="%s" style="background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; font-weight: bold; border-radius: 4px; display: inline-block;">
					Verify Email Address
				</a>
			</div>
		</div>
	`, verificationURL)

	input := &ses.SendEmailInput{
		Source: &sender,
		Destination: &types.Destination{
			ToAddresses: []string{targetEmail},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: &subject,
			},
			Body: &types.Body{
				Html: &types.Content{
					Data: &htmlBody,
				},
			},
		},
	}

	_, err := cfg.SESClient.SendEmail(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to send email via AWS SES: %w", err)
	}

	slog.Info("Successfully sent verification email", "email_address", targetEmail)
	return nil
}

func (cfg *ApiConfig) SendPasswordReset(targetEmail string, token string) error {
	base := strings.TrimRight(cfg.React, "/")
	resetURL := fmt.Sprintf("%s/resetpassword/%s", base, token)

	sender := "no-reply@pantryscrolls.com"
	subject := "Password Reset Request"

	htmlBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
			<h2 style="color: #333333;">Welcome!</h2>
			<p style="color: #555555; font-size: 16px; line-height: 1.5;">
				Please click the link to reset your password. This link will expire in 30 minutes.
			</p>
			<div style="margin: 30px 0; text-align: center;">
				<a href="%s" style="background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; font-weight: bold; border-radius: 4px; display: inline-block;">
					Reset Password
				</a>
			</div>
		</div>
	`, resetURL)

	input := &ses.SendEmailInput{
		Source: &sender,
		Destination: &types.Destination{
			ToAddresses: []string{targetEmail},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: &subject,
			},
			Body: &types.Body{
				Html: &types.Content{
					Data: &htmlBody,
				},
			},
		},
	}

	_, err := cfg.SESClient.SendEmail(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to send email via AWS SES: %w", err)
	}

	slog.Info("Successfully sent password reset email", "email_address", targetEmail)
	return nil
}

func (cfg *ApiConfig) SendDeactivationEmail(targetEmail string, cancelToken string) error {
	base := strings.TrimRight(cfg.React, "/")
	cancelURL := fmt.Sprintf("%s/cancel-deactivation/%s", base, cancelToken)

	sender := "no-reply@pantryscrolls.com"
	subject := "Your account has been scheduled for deletion"

	htmlBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
			<h2 style="color: #333333;">Account Deactivation Notice</h2>
			<p style="color: #555555; font-size: 16px; line-height: 1.5;">
				Your Pantry Scrolls account has been scheduled for permanent deletion. Your account and all associated data will be removed in <strong>30 days</strong>.
			</p>
			<p style="color: #555555; font-size: 16px; line-height: 1.5;">
				If you did not request this or wish to keep your account, click the button below to cancel the deactivation. This link expires in 30 days.
			</p>
			<div style="margin: 30px 0; text-align: center;">
				<a href="%s" style="background-color: #dc3545; color: white; padding: 12px 24px; text-decoration: none; font-weight: bold; border-radius: 4px; display: inline-block;">
					Cancel Account Deletion
				</a>
			</div>
			<p style="color: #888888; font-size: 14px;">
				If you did not make this request, please cancel immediately using the link above, change your password, and contact support if you believe your account has been compromised.
			</p>
		</div>
	`, cancelURL)

	input := &ses.SendEmailInput{
		Source: &sender,
		Destination: &types.Destination{
			ToAddresses: []string{targetEmail},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: &subject,
			},
			Body: &types.Body{
				Html: &types.Content{
					Data: &htmlBody,
				},
			},
		},
	}

	_, err := cfg.SESClient.SendEmail(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to send email via AWS SES: %w", err)
	}

	slog.Info("Successfully sent deactivation email", "email_address", targetEmail)
	return nil
}
