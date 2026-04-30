package seed

import (
	"time"

	"gitlab.com/logtrace/logtrace"
)

// EventSeedData returns 10 event records with unique action names.
func EventSeedData() []logtrace.Event {
	now := time.Now().UTC()
	return []logtrace.Event{
		{
			ActionName: "user.login", Type: "authentication",
			Username: "alice morgan", UserID: "usr_001",
			HTTPMethod: "POST", HTTPStatus: "200", HTTPEndpoint: "/auth/login",
			ClientIP: "192.168.1.10", ClientUserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			GeoIPLocation: "New York, US",
			Metadata:      logtrace.Metadata{"browser": "Chrome", "os": "Windows 10"},
			CreatedAt:     now,
		},
		{
			ActionName: "user.logout", Type: "authentication",
			Username: "bob hartley", UserID: "usr_002",
			HTTPMethod: "POST", HTTPStatus: "200", HTTPEndpoint: "/auth/logout",
			ClientIP: "10.0.0.5", ClientUserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			GeoIPLocation: "San Francisco, US",
			Metadata:      logtrace.Metadata{"browser": "Safari", "os": "macOS"},
			CreatedAt:     now,
		},
		{
			ActionName: "payment.initiated", Type: "transaction",
			Username: "carol nguyen", UserID: "usr_003",
			HTTPMethod: "POST", HTTPStatus: "201", HTTPEndpoint: "/payments",
			ClientIP: "172.16.0.20", ClientUserAgent: "axios/1.4.0",
			GeoIPLocation: "London, GB",
			Metadata:      logtrace.Metadata{"amount": "99.99", "currency": "USD"},
			CreatedAt:     now,
		},
		{
			ActionName: "profile.updated", Type: "account",
			Username: "david okafor", UserID: "usr_004",
			HTTPMethod: "PATCH", HTTPStatus: "200", HTTPEndpoint: "/users/profile",
			ClientIP: "203.0.113.45", ClientUserAgent: "okhttp/4.9.3",
			GeoIPLocation: "Lagos, NG",
			Metadata:      logtrace.Metadata{"fields_changed": "email,phone"},
			CreatedAt:     now,
		},
		{
			ActionName: "document.uploaded", Type: "file",
			Username: "emily svensson", UserID: "usr_005",
			HTTPMethod: "POST", HTTPStatus: "201", HTTPEndpoint: "/documents/upload",
			ClientIP: "198.51.100.33", ClientUserAgent: "Mozilla/5.0 (X11; Linux x86_64)",
			GeoIPLocation: "Stockholm, SE",
			Metadata:      logtrace.Metadata{"file_type": "pdf", "size_kb": "1024"},
			CreatedAt:     now,
		},
		{
			ActionName: "api_key.created", Type: "developer",
			Username: "frank delacroix", UserID: "usr_006",
			HTTPMethod: "POST", HTTPStatus: "201", HTTPEndpoint: "/developers/keys",
			ClientIP: "192.0.2.88", ClientUserAgent: "PostmanRuntime/7.32.0",
			GeoIPLocation: "Paris, FR",
			Metadata:      logtrace.Metadata{"key_prefix": "ltk_live"},
			CreatedAt:     now,
		},
		{
			ActionName: "password.reset", Type: "security",
			Username: "grace kimani", UserID: "usr_007",
			HTTPMethod: "POST", HTTPStatus: "200", HTTPEndpoint: "/auth/password/reset",
			ClientIP: "100.64.0.12", ClientUserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0)",
			GeoIPLocation: "Nairobi, KE",
			Metadata:      logtrace.Metadata{"method": "email_link"},
			CreatedAt:     now,
		},
		{
			ActionName: "report.exported", Type: "data",
			Username: "henry walker", UserID: "usr_008",
			HTTPMethod: "GET", HTTPStatus: "200", HTTPEndpoint: "/reports/export",
			ClientIP: "192.168.10.50", ClientUserAgent: "python-requests/2.31.0",
			GeoIPLocation: "Sydney, AU",
			Metadata:      logtrace.Metadata{"format": "csv", "rows": "5000"},
			CreatedAt:     now,
		},
		{
			ActionName: "webhook.triggered", Type: "integration",
			Username: "isabella costa", UserID: "usr_009",
			HTTPMethod: "POST", HTTPStatus: "200", HTTPEndpoint: "/webhooks/trigger",
			ClientIP: "203.0.113.99", ClientUserAgent: "node-fetch/3.3.0",
			GeoIPLocation: "São Paulo, BR",
			Metadata:      logtrace.Metadata{"event_type": "payment.success"},
			CreatedAt:     now,
		},
		{
			ActionName: "organization.created", Type: "admin",
			Username: "james otieno", UserID: "usr_010",
			HTTPMethod: "POST", HTTPStatus: "201", HTTPEndpoint: "/organizations",
			ClientIP: "10.10.0.1", ClientUserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			GeoIPLocation: "Kampala, UG",
			Metadata:      logtrace.Metadata{"plan": "starter"},
			CreatedAt:     now,
		},
	}
}

// SessionSeedData returns 10 session records with unique user names.
func SessionSeedData() []logtrace.Session {
	now := time.Now().UTC()
	return []logtrace.Session{
		{
			UserID: "usr_001", UserName: "alice morgan",
			IPAddress: "192.168.1.10", Location: "New York, US",
			DeviceInfo: "Chrome 114 / Windows 10",
			Status:     "active", Token: "tok_alice_001",
			LoginAt:  now.Add(-2 * time.Hour),
			Metadata: logtrace.Metadata{"mfa": "true"},
		},
		{
			UserID: "usr_002", UserName: "bob hartley",
			IPAddress: "10.0.0.5", Location: "San Francisco, US",
			DeviceInfo: "Safari 16 / macOS Ventura",
			Status:     "active", Token: "tok_bob_002",
			LoginAt:  now.Add(-45 * time.Minute),
			Metadata: logtrace.Metadata{"mfa": "false"},
		},
		{
			UserID: "usr_003", UserName: "carol nguyen",
			IPAddress: "172.16.0.20", Location: "London, GB",
			DeviceInfo: "Firefox 115 / Ubuntu 22.04",
			Status:     "succesful", Token: "tok_carol_003",
			LoginAt:  now.Add(-5 * time.Hour),
			LogoutAt: now.Add(-4 * time.Hour),
			Metadata: logtrace.Metadata{"mfa": "true"},
		},
		{
			UserID: "usr_004", UserName: "david okafor",
			IPAddress: "203.0.113.45", Location: "Lagos, NG",
			DeviceInfo: "okhttp / Android 13",
			Status:     "expired", Token: "tok_david_004",
			LoginAt:  now.Add(-26 * time.Hour),
			LogoutAt: now.Add(-25 * time.Hour),
			Metadata: logtrace.Metadata{"mfa": "false"},
		},
		{
			UserID: "usr_005", UserName: "emily svensson",
			IPAddress: "198.51.100.33", Location: "Stockholm, SE",
			DeviceInfo: "Chrome 114 / Linux",
			Status:     "active", Token: "tok_emily_005",
			LoginAt:  now.Add(-30 * time.Minute),
			Metadata: logtrace.Metadata{"mfa": "true"},
		},
		{
			UserID: "usr_006", UserName: "frank delacroix",
			IPAddress: "192.0.2.88", Location: "Paris, FR",
			DeviceInfo: "Postman / macOS",
			Status:     "succesful", Token: "tok_frank_006",
			LoginAt:  now.Add(-3 * time.Hour),
			LogoutAt: now.Add(-2*time.Hour - 30*time.Minute),
			Metadata: logtrace.Metadata{"mfa": "true"},
		},
		{
			UserID: "usr_007", UserName: "grace kimani",
			IPAddress: "100.64.0.12", Location: "Nairobi, KE",
			DeviceInfo: "Mobile Safari / iOS 16",
			Status:     "active", Token: "tok_grace_007",
			LoginAt:  now.Add(-10 * time.Minute),
			Metadata: logtrace.Metadata{"mfa": "false"},
		},
		{
			UserID: "usr_008", UserName: "henry walker",
			IPAddress: "192.168.10.50", Location: "Sydney, AU",
			DeviceInfo: "python-requests / Linux",
			Status:     "failed", Token: "tok_henry_008",
			LoginAt:  now.Add(-1 * time.Hour),
			Metadata: logtrace.Metadata{"reason": "invalid_credentials"},
		},
		{
			UserID: "usr_009", UserName: "isabella costa",
			IPAddress: "203.0.113.99", Location: "São Paulo, BR",
			DeviceInfo: "node-fetch / Node.js 18",
			Status:     "active", Token: "tok_isabella_009",
			LoginAt:  now.Add(-15 * time.Minute),
			Metadata: logtrace.Metadata{"mfa": "true"},
		},
		{
			UserID: "usr_010", UserName: "james otieno",
			IPAddress: "10.10.0.1", Location: "Kampala, UG",
			DeviceInfo: "Chrome 114 / Windows 11",
			Status:     "active", Token: "tok_james_010",
			LoginAt:  now.Add(-5 * time.Minute),
			Metadata: logtrace.Metadata{"mfa": "false"},
		},
	}
}

// AuditLogSeedData returns 10 audit log records with unique actions.
func AuditLogSeedData() []logtrace.AuditLog {
	now := time.Now().UTC()
	return []logtrace.AuditLog{
		{
			Action: "user.created", UserID: "usr_001", UserName: "alice morgan",
			IPAddress: "192.168.1.10", RequestID: "req_al_001",
			Client: "Chrome 114", OperatingSystem: "Windows 10",
			Metadata:  logtrace.Metadata{"email": "alice.morgan@example.com"},
			Timestamp: now.Add(-2 * time.Hour),
		},
		{
			Action: "user.role_changed", UserID: "usr_002", UserName: "bob hartley",
			IPAddress: "10.0.0.5", RequestID: "req_bh_002",
			Client: "Safari 16", OperatingSystem: "macOS Ventura",
			Metadata:  logtrace.Metadata{"old_role": "member", "new_role": "admin"},
			Timestamp: now.Add(-90 * time.Minute),
		},
		{
			Action: "api_key.revoked", UserID: "usr_003", UserName: "carol nguyen",
			IPAddress: "172.16.0.20", RequestID: "req_cn_003",
			Client: "Firefox 115", OperatingSystem: "Ubuntu 22.04",
			Metadata:  logtrace.Metadata{"key_prefix": "ltk_test"},
			Timestamp: now.Add(-80 * time.Minute),
		},
		{
			Action: "billing.subscription_upgraded", UserID: "usr_004", UserName: "david okafor",
			IPAddress: "203.0.113.45", RequestID: "req_do_004",
			Client: "okhttp", OperatingSystem: "Android 13",
			Metadata:  logtrace.Metadata{"from_plan": "starter", "to_plan": "growth"},
			Timestamp: now.Add(-70 * time.Minute),
		},
		{
			Action: "organization.settings_updated", UserID: "usr_005", UserName: "emily svensson",
			IPAddress: "198.51.100.33", RequestID: "req_es_005",
			Client: "Chrome 114", OperatingSystem: "Linux",
			Metadata:  logtrace.Metadata{"fields": "name,timezone"},
			Timestamp: now.Add(-60 * time.Minute),
		},
		{
			Action: "invitation.sent", UserID: "usr_006", UserName: "frank delacroix",
			IPAddress: "192.0.2.88", RequestID: "req_fd_006",
			Client: "Postman", OperatingSystem: "macOS",
			Metadata:  logtrace.Metadata{"invited_email": "new.member@example.com"},
			Timestamp: now.Add(-50 * time.Minute),
		},
		{
			Action: "two_factor.enabled", UserID: "usr_007", UserName: "grace kimani",
			IPAddress: "100.64.0.12", RequestID: "req_gk_007",
			Client: "Mobile Safari", OperatingSystem: "iOS 16",
			Metadata:  logtrace.Metadata{"method": "totp"},
			Timestamp: now.Add(-40 * time.Minute),
		},
		{
			Action: "data.export_requested", UserID: "usr_008", UserName: "henry walker",
			IPAddress: "192.168.10.50", RequestID: "req_hw_008",
			Client: "python-requests", OperatingSystem: "Linux",
			Metadata:  logtrace.Metadata{"resource": "audit_logs", "format": "json"},
			Timestamp: now.Add(-30 * time.Minute),
		},
		{
			Action: "webhook.deleted", UserID: "usr_009", UserName: "isabella costa",
			IPAddress: "203.0.113.99", RequestID: "req_ic_009",
			Client: "node-fetch", OperatingSystem: "Node.js 18",
			Metadata:  logtrace.Metadata{"webhook_url": "https://hooks.example.com/old"},
			Timestamp: now.Add(-20 * time.Minute),
		},
		{
			Action: "session.force_logout", UserID: "usr_010", UserName: "james otieno",
			IPAddress: "10.10.0.1", RequestID: "req_jo_010",
			Client: "Chrome 114", OperatingSystem: "Windows 11",
			Metadata:  logtrace.Metadata{"reason": "suspicious_activity"},
			Timestamp: now.Add(-10 * time.Minute),
		},
	}
}
