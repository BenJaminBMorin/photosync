package services

const recoveryEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 600px;
            margin: 40px auto;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .content {
            padding: 40px 30px;
        }
        .content p {
            margin: 0 0 20px 0;
            font-size: 16px;
            color: #4a5568;
        }
        .button-container {
            text-align: center;
            margin: 30px 0;
        }
        .button {
            display: inline-block;
            background: #667eea;
            color: white;
            padding: 14px 32px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 16px;
        }
        .button:hover {
            background: #5a67d8;
        }
        .warning {
            background: #fef3c7;
            border-left: 4px solid #f59e0b;
            padding: 16px;
            margin: 24px 0;
            border-radius: 4px;
        }
        .warning strong {
            color: #92400e;
            display: block;
            margin-bottom: 8px;
        }
        .warning p {
            color: #78350f;
            margin: 0;
            font-size: 14px;
        }
        .link-box {
            background: #f8fafc;
            padding: 16px;
            border-radius: 4px;
            margin: 20px 0;
            word-break: break-all;
            font-family: monospace;
            font-size: 12px;
            color: #64748b;
        }
        .footer {
            text-align: center;
            color: #94a3b8;
            font-size: 14px;
            padding: 20px 30px;
            border-top: 1px solid #e2e8f0;
        }
        .footer p {
            margin: 5px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 PhotoSync Recovery</h1>
        </div>
        <div class="content">
            <p>Hello <strong>{{.Name}}</strong>,</p>
            <p>You requested access recovery for your PhotoSync account. Click the button below to securely log into your account:</p>

            <div class="button-container">
                <a href="{{.RecoveryLink}}" class="button">Access Your Account</a>
            </div>

            <div class="warning">
                <strong>⚠️ Security Notice</strong>
                <p>This link expires in <strong>15 minutes</strong> and can only be used <strong>once</strong>. If you didn't request this, please ignore this email - your account remains secure.</p>
            </div>

            <p>If the button doesn't work, copy and paste this link into your browser:</p>
            <div class="link-box">{{.RecoveryLink}}</div>

            <p style="margin-top: 30px;">Having trouble? Contact your PhotoSync administrator for assistance.</p>
        </div>
        <div class="footer">
            <p>This is an automated message from PhotoSync</p>
            <p>Do not reply to this email</p>
        </div>
    </div>
</body>
</html>`

const testEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 600px;
            margin: 40px auto;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .content {
            padding: 40px 30px;
        }
        .content p {
            margin: 0 0 20px 0;
            font-size: 16px;
            color: #4a5568;
        }
        .success-box {
            background: #d1fae5;
            border-left: 4px solid #10b981;
            padding: 16px;
            margin: 24px 0;
            border-radius: 4px;
        }
        .success-box p {
            color: #065f46;
            margin: 0;
            font-size: 14px;
        }
        .footer {
            text-align: center;
            color: #94a3b8;
            font-size: 14px;
            padding: 20px 30px;
            border-top: 1px solid #e2e8f0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✅ Test Email</h1>
        </div>
        <div class="content">
            <p>Congratulations! Your SMTP configuration is working correctly.</p>

            <div class="success-box">
                <p>✓ Connection successful<br>
                ✓ Authentication passed<br>
                ✓ Email delivered</p>
            </div>

            <p>Your PhotoSync server is now configured to send emails for:</p>
            <ul>
                <li>Account recovery</li>
                <li>Security notifications</li>
                <li>System alerts</li>
            </ul>

            <p style="margin-top: 30px; color: #64748b; font-size: 14px;">
                Sent at: {{.Timestamp}}
            </p>
        </div>
        <div class="footer">
            <p>This is a test message from PhotoSync</p>
        </div>
    </div>
</body>
</html>`

type RecoveryEmailData struct {
	Name         string
	RecoveryLink string
}

type TestEmailData struct {
	Timestamp string
}
