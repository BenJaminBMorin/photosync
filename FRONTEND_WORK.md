# PhotoSync Password Authentication - Frontend Implementation Guide

## Backend Status: ✅ COMPLETE & TESTED

All server-side implementation is complete and committed:
- ✅ Database schema with password support
- ✅ 8 new API endpoints (login, password reset, API key refresh)
- ✅ Security features (bcrypt hashing, rate limiting, email enumeration prevention)
- ✅ All services, handlers, and routes wired
- ✅ Server compiles and runs successfully
- ✅ Commit: `1e78f48`

---

## iOS Frontend Implementation

### Phase 1: API Models and Service Methods

#### File: `ios/PhotoSync/Models/APIModels.swift`

Add these models after existing models:

```swift
// Login Request/Response
struct LoginRequest: Codable {
    let email: String
    let password: String
    let deviceName: String
    let platform: String  // "iOS"
    let fcmToken: String
}

struct LoginResponse: Codable {
    let success: Bool
    let user: UserResponse
    let device: DeviceResponse
    let apiKey: String  // One-time returned, must be stored securely
}

// Password Reset Models
struct InitiateEmailResetRequest: Codable {
    let email: String
}

struct VerifyCodeRequest: Codable {
    let email: String
    let code: String
    let newPassword: String
}

struct InitiatePhoneResetRequest: Codable {
    let email: String
    let newPassword: String
}

struct PhoneResetStatusResponse: Codable {
    let status: String  // "pending", "approved", "denied", "expired"
    let expiresAt: Date
    let errorMessage: String?
}

struct CompletePhoneResetRequest: Codable {
    let requestId: String
}

// For refresh API key
struct RefreshAPIKeyRequest: Codable {
    let password: String
}

struct RefreshAPIKeyResponse: Codable {
    let apiKey: String
}
```

---

#### File: `ios/PhotoSync/Services/APIService.swift`

Add these methods to APIService (inside the actor):

```swift
// MARK: - Mobile Authentication

/// Login with email and password, get API key
func login(
    email: String,
    password: String,
    deviceName: String,
    fcmToken: String
) async throws -> LoginResponse {
    let url = try buildURL(path: "/api/mobile/auth/login")
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    
    let body = LoginRequest(
        email: email,
        password: password,
        deviceName: deviceName,
        platform: "iOS",
        fcmToken: fcmToken
    )
    request.httpBody = try JSONEncoder().encode(body)
    
    let (data, response) = try await session.data(for: request)
    try validateResponse(response)
    
    return try JSONDecoder().decode(LoginResponse.self, from: data)
}

/// Refresh API key after verifying password
func refreshAPIKey(password: String) async throws -> String {
    let url = try buildURL(path: "/api/mobile/auth/refresh-key")
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    addAPIKeyHeader(to: &request)
    
    let body = RefreshAPIKeyRequest(password: password)
    request.httpBody = try JSONEncoder().encode(body)
    
    let (data, response) = try await session.data(for: request)
    try validateResponse(response)
    
    let result = try JSONDecoder().decode(RefreshAPIKeyResponse.self, from: data)
    return result.apiKey
}

// MARK: - Password Reset - Email

/// Initiate email password reset (always returns success for security)
func initiateEmailPasswordReset(email: String) async throws {
    let url = try buildURL(path: "/api/mobile/auth/reset/email/initiate")
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    
    let body = InitiateEmailResetRequest(email: email)
    request.httpBody = try JSONEncoder().encode(body)
    
    let (_, response) = try await session.data(for: request)
    try validateResponse(response)
}

/// Verify reset code and set new password
func verifyPasswordResetCode(
    email: String,
    code: String,
    newPassword: String
) async throws {
    let url = try buildURL(path: "/api/mobile/auth/reset/email/verify")
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    
    let body = VerifyCodeRequest(
        email: email,
        code: code,
        newPassword: newPassword
    )
    request.httpBody = try JSONEncoder().encode(body)
    
    let (_, response) = try await session.data(for: request)
    try validateResponse(response)
}

// MARK: - Password Reset - Phone 2FA

/// Initiate phone-based password reset (sends FCM push to user's devices)
func initiatePhonePasswordReset(
    email: String,
    newPassword: String
) async throws -> String {
    let url = try buildURL(path: "/api/mobile/auth/reset/phone/initiate")
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    
    let body = InitiatePhoneResetRequest(
        email: email,
        newPassword: newPassword
    )
    request.httpBody = try JSONEncoder().encode(body)
    
    let (data, response) = try await session.data(for: request)
    try validateResponse(response)
    
    // Parse response to get request ID
    struct InitiatePhoneResetResponse: Codable {
        let requestId: String
        let expiresAt: Date
    }
    let result = try JSONDecoder().decode(InitiatePhoneResetResponse.self, from: data)
    return result.requestId
}

/// Check status of phone-based password reset approval
func checkPhoneResetStatus(requestId: String) async throws -> PhoneResetStatusResponse {
    let url = try buildURL(path: "/api/mobile/auth/reset/phone/status/\(requestId)")
    var request = URLRequest(url: url)
    request.httpMethod = "GET"
    
    let (data, response) = try await session.data(for: request)
    try validateResponse(response)
    
    return try JSONDecoder().decode(PhoneResetStatusResponse.self, from: data)
}

/// Complete phone-based password reset after device approval
func completePhoneReset(requestId: String) async throws {
    let url = try buildURL(path: "/api/mobile/auth/reset/phone/complete/\(requestId)")
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    
    let body = CompletePhoneResetRequest(requestId: requestId)
    request.httpBody = try JSONEncoder().encode(body)
    
    let (_, response) = try await session.data(for: request)
    try validateResponse(response)
}
```

---

### Phase 2: UI Views

#### File: `ios/PhotoSync/Views/LoginView.swift` (NEW)

```swift
import SwiftUI

struct LoginView: View {
    @Environment(\.dismiss) var dismiss
    @State private var email = ""
    @State private var password = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    
    var isValidForm: Bool {
        !email.isEmpty && !password.isEmpty && email.contains("@")
    }
    
    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                // Header
                VStack(spacing: 8) {
                    Image(systemName: "lock.fill")
                        .font(.system(size: 48))
                        .foregroundColor(.blue)
                    
                    Text("Login with Password")
                        .font(.title2)
                        .fontWeight(.bold)
                    
                    Text("Enter your email and password")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                .padding(.top, 32)
                
                // Form
                VStack(spacing: 16) {
                    // Email
                    VStack(alignment: .leading, spacing: 4) {
                        Label("Email", systemImage: "envelope")
                            .font(.subheadline)
                            .fontWeight(.medium)
                        
                        TextField("your@email.com", text: $email)
                            .textContentType(.emailAddress)
                            .keyboardType(.emailAddress)
                            .autocapitalization(.none)
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(8)
                    }
                    
                    // Password
                    VStack(alignment: .leading, spacing: 4) {
                        Label("Password", systemImage: "lock")
                            .font(.subheadline)
                            .fontWeight(.medium)
                        
                        SecureField("Enter password", text: $password)
                            .textContentType(.password)
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(8)
                    }
                }
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(12)
                
                // Error Message
                if showError {
                    HStack {
                        Image(systemName: "exclamationmark.circle.fill")
                            .foregroundColor(.red)
                        Text(errorMessage)
                            .font(.subheadline)
                        Spacer()
                    }
                    .padding()
                    .background(Color.red.opacity(0.1))
                    .cornerRadius(8)
                }
                
                // Login Button
                Button(action: login) {
                    if isLoading {
                        ProgressView()
                            .progressViewStyle(.circular)
                    } else {
                        Text("Login")
                            .fontWeight(.semibold)
                    }
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(isValidForm ? Color.blue : Color.gray)
                .foregroundColor(.white)
                .cornerRadius(8)
                .disabled(!isValidForm || isLoading)
                
                // Forgot Password
                NavigationLink("Forgot Password?", destination: PasswordResetView())
                    .font(.subheadline)
                    .foregroundColor(.blue)
                
                Spacer()
                
                // Or use API Key
                Divider()
                NavigationLink("Use API Key Instead", destination: APIKeySetupView())
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
            .padding()
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
    }
    
    private func login() {
        isLoading = true
        errorMessage = ""
        showError = false
        
        Task {
            do {
                let fcmToken = await NotificationService.shared.getFCMToken()
                let response = try await APIService.shared.login(
                    email: email,
                    password: password,
                    deviceName: UIDevice.current.name,
                    fcmToken: fcmToken ?? "unknown"
                )
                
                // Store API key securely
                await MainActor.run {
                    AppSettings.apiKey = response.apiKey
                    AppSettings.serverURL = AppSettings.normalizedServerURL
                    
                    // Clear form and dismiss
                    email = ""
                    password = ""
                    isLoading = false
                    
                    dismiss()
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
            }
        }
    }
}

#Preview {
    LoginView()
}
```

---

#### File: `ios/PhotoSync/Views/PasswordResetView.swift` (NEW)

```swift
import SwiftUI

struct PasswordResetView: View {
    @Environment(\.dismiss) var dismiss
    @State private var selectedMethod: ResetMethod = .email
    
    enum ResetMethod {
        case email
        case phone
    }
    
    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                // Header
                VStack(spacing: 8) {
                    Image(systemName: "key.fill")
                        .font(.system(size: 48))
                        .foregroundColor(.orange)
                    
                    Text("Reset Password")
                        .font(.title2)
                        .fontWeight(.bold)
                    
                    Text("Choose a reset method")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                .padding(.top, 32)
                
                // Method Selector
                Picker("Reset Method", selection: $selectedMethod) {
                    Text("Email with Code").tag(ResetMethod.email)
                    Text("Phone 2FA").tag(ResetMethod.phone)
                }
                .pickerStyle(.segmented)
                .padding()
                
                // Method-specific content
                Group {
                    if selectedMethod == .email {
                        NavigationLink(destination: EmailPasswordResetView()) {
                            VStack(alignment: .leading, spacing: 12) {
                                HStack {
                                    Image(systemName: "envelope.fill")
                                        .foregroundColor(.blue)
                                    VStack(alignment: .leading) {
                                        Text("Email Reset")
                                            .fontWeight(.semibold)
                                        Text("Get a code sent to your email")
                                            .font(.subheadline)
                                            .foregroundColor(.secondary)
                                    }
                                    Spacer()
                                    Image(systemName: "chevron.right")
                                        .foregroundColor(.gray)
                                }
                            }
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(8)
                        }
                    } else {
                        NavigationLink(destination: PhonePasswordResetView()) {
                            VStack(alignment: .leading, spacing: 12) {
                                HStack {
                                    Image(systemName: "iphone")
                                        .foregroundColor(.blue)
                                    VStack(alignment: .leading) {
                                        Text("Phone 2FA")
                                            .fontWeight(.semibold)
                                        Text("Approve on another device")
                                            .font(.subheadline)
                                            .foregroundColor(.secondary)
                                    }
                                    Spacer()
                                    Image(systemName: "chevron.right")
                                        .foregroundColor(.gray)
                                }
                            }
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(8)
                        }
                    }
                }
                
                Spacer()
            }
            .padding()
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
    }
}

#Preview {
    PasswordResetView()
}
```

---

#### File: `ios/PhotoSync/Views/EmailPasswordResetView.swift` (NEW)

```swift
import SwiftUI

struct EmailPasswordResetView: View {
    @Environment(\.dismiss) var dismiss
    @State private var currentStep: Step = .enterEmail
    @State private var email = ""
    @State private var code = ""
    @State private var newPassword = ""
    @State private var confirmPassword = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    
    enum Step {
        case enterEmail
        case enterCode
        case enterNewPassword
        case success
    }
    
    var body: some View {
        VStack(spacing: 24) {
            // Progress indicator
            HStack(spacing: 8) {
                ForEach([Step.enterEmail, .enterCode, .enterNewPassword, .success], id: \.self) { step in
                    Circle()
                        .fill(isStepCompleted(step) ? Color.blue : Color.gray.opacity(0.3))
                        .frame(width: 8, height: 8)
                }
            }
            .padding()
            
            VStack(spacing: 16) {
                if currentStep == .enterEmail {
                    emailStep
                } else if currentStep == .enterCode {
                    codeStep
                } else if currentStep == .enterNewPassword {
                    passwordStep
                } else {
                    successStep
                }
            }
            
            Spacer()
        }
        .padding()
        .navigationBarBackButtonHidden(currentStep != .enterEmail)
    }
    
    private var emailStep: some View {
        VStack(spacing: 16) {
            Text("Enter your email address")
                .font(.headline)
            
            TextField("your@email.com", text: $email)
                .textContentType(.emailAddress)
                .keyboardType(.emailAddress)
                .autocapitalization(.none)
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)
            
            if showError {
                HStack {
                    Image(systemName: "exclamationmark.circle.fill")
                        .foregroundColor(.red)
                    Text(errorMessage)
                        .font(.subheadline)
                }
                .padding()
                .background(Color.red.opacity(0.1))
                .cornerRadius(8)
            }
            
            Button(action: sendResetCode) {
                if isLoading {
                    ProgressView()
                } else {
                    Text("Send Code")
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(email.contains("@") ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(8)
            .disabled(!email.contains("@") || isLoading)
        }
    }
    
    private var codeStep: some View {
        VStack(spacing: 16) {
            Text("Enter the code sent to \(email)")
                .font(.headline)
            
            Text("Check your email for a 6-digit code")
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            TextField("000000", text: $code)
                .keyboardType(.numberPad)
                .frame(height: 50)
                .multilineTextAlignment(.center)
                .font(.title)
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)
            
            if showError {
                HStack {
                    Image(systemName: "exclamationmark.circle.fill")
                        .foregroundColor(.red)
                    Text(errorMessage)
                        .font(.subheadline)
                }
                .padding()
                .background(Color.red.opacity(0.1))
                .cornerRadius(8)
            }
            
            Button(action: verifyCode) {
                if isLoading {
                    ProgressView()
                } else {
                    Text("Verify Code")
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(code.count == 6 ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(8)
            .disabled(code.count != 6 || isLoading)
        }
    }
    
    private var passwordStep: some View {
        VStack(spacing: 16) {
            Text("Create new password")
                .font(.headline)
            
            Text("Password must be at least 8 characters")
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            SecureField("New password", text: $newPassword)
                .textContentType(.newPassword)
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)
            
            SecureField("Confirm password", text: $confirmPassword)
                .textContentType(.newPassword)
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(8)
            
            if showError {
                HStack {
                    Image(systemName: "exclamationmark.circle.fill")
                        .foregroundColor(.red)
                    Text(errorMessage)
                        .font(.subheadline)
                }
                .padding()
                .background(Color.red.opacity(0.1))
                .cornerRadius(8)
            }
            
            Button(action: resetPassword) {
                if isLoading {
                    ProgressView()
                } else {
                    Text("Reset Password")
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(isPasswordValid ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(8)
            .disabled(!isPasswordValid || isLoading)
        }
    }
    
    private var successStep: some View {
        VStack(spacing: 24) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 64))
                .foregroundColor(.green)
            
            VStack(spacing: 8) {
                Text("Password Reset!")
                    .font(.title2)
                    .fontWeight(.bold)
                
                Text("Your password has been successfully reset. You can now login with your new password.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }
            
            Button(action: { dismiss() }) {
                Text("Done")
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.blue)
                    .foregroundColor(.white)
                    .cornerRadius(8)
            }
        }
    }
    
    private var isPasswordValid: Bool {
        newPassword.count >= 8 && newPassword == confirmPassword
    }
    
    private func isStepCompleted(_ step: Step) -> Bool {
        switch step {
        case .enterEmail:
            return !email.isEmpty
        case .enterCode:
            return !code.isEmpty
        case .enterNewPassword:
            return isPasswordValid
        case .success:
            return currentStep == .success
        }
    }
    
    private func sendResetCode() {
        isLoading = true
        errorMessage = ""
        showError = false
        
        Task {
            do {
                try await APIService.shared.initiateEmailPasswordReset(email: email)
                await MainActor.run {
                    currentStep = .enterCode
                    isLoading = false
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
            }
        }
    }
    
    private func verifyCode() {
        isLoading = true
        errorMessage = ""
        showError = false
        
        Task {
            do {
                // Don't actually verify here, just move to next step
                // Verification happens when user sets password
                await MainActor.run {
                    currentStep = .enterNewPassword
                    isLoading = false
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
            }
        }
    }
    
    private func resetPassword() {
        isLoading = true
        errorMessage = ""
        showError = false
        
        Task {
            do {
                try await APIService.shared.verifyPasswordResetCode(
                    email: email,
                    code: code,
                    newPassword: newPassword
                )
                await MainActor.run {
                    currentStep = .success
                    isLoading = false
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
            }
        }
    }
}

#Preview {
    EmailPasswordResetView()
}
```

---

#### File: `ios/PhotoSync/Views/PhonePasswordResetView.swift` (NEW)

```swift
import SwiftUI

struct PhonePasswordResetView: View {
    @Environment(\.dismiss) var dismiss
    @State private var email = ""
    @State private var newPassword = ""
    @State private var confirmPassword = ""
    @State private var requestId = ""
    @State private var currentStep: Step = .enterPassword
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    @State private var statusCheckTimer: Timer?
    @State private var approvalStatus = ""
    
    enum Step {
        case enterPassword
        case waiting
        case approved
    }
    
    var isPasswordValid: Bool {
        newPassword.count >= 8 && newPassword == confirmPassword
    }
    
    var body: some View {
        VStack(spacing: 24) {
            if currentStep == .enterPassword {
                enterPasswordStep
            } else if currentStep == .waiting {
                waitingForApprovalStep
            } else {
                approvedStep
            }
            
            Spacer()
        }
        .padding()
        .navigationBarBackButtonHidden(currentStep != .enterPassword)
        .onDisappear {
            statusCheckTimer?.invalidate()
        }
    }
    
    private var enterPasswordStep: some View {
        VStack(spacing: 16) {
            Image(systemName: "lock.fill")
                .font(.system(size: 48))
                .foregroundColor(.blue)
            
            Text("Create new password")
                .font(.headline)
            
            Text("This will require approval from another device")
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            VStack(spacing: 12) {
                SecureField("New password", text: $newPassword)
                    .textContentType(.newPassword)
                    .padding()
                    .background(Color(.systemGray6))
                    .cornerRadius(8)
                
                SecureField("Confirm password", text: $confirmPassword)
                    .textContentType(.newPassword)
                    .padding()
                    .background(Color(.systemGray6))
                    .cornerRadius(8)
            }
            
            if showError {
                HStack {
                    Image(systemName: "exclamationmark.circle.fill")
                        .foregroundColor(.red)
                    Text(errorMessage)
                        .font(.subheadline)
                }
                .padding()
                .background(Color.red.opacity(0.1))
                .cornerRadius(8)
            }
            
            Button(action: requestPhoneReset) {
                if isLoading {
                    ProgressView()
                } else {
                    Text("Send Approval Request")
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(isPasswordValid ? Color.blue : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(8)
            .disabled(!isPasswordValid || isLoading)
        }
    }
    
    private var waitingForApprovalStep: some View {
        VStack(spacing: 24) {
            // Animated waiting indicator
            VStack(spacing: 16) {
                Image(systemName: "iphone.radiowaves.left.and.right")
                    .font(.system(size: 48))
                    .foregroundColor(.blue)
                    .scaleEffect(1.0)
                    .animation(.easeInOut(duration: 1).repeatForever(), value: UUID())
                    .onAppear {
                        // Trigger animation
                    }
                
                Text("Waiting for Approval")
                    .font(.headline)
                
                Text("Check your other device for an approval request. It will expire in 60 seconds.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }
            .padding()
            .background(Color.blue.opacity(0.1))
            .cornerRadius(12)
            
            // Status message
            HStack {
                Image(systemName: "info.circle.fill")
                    .foregroundColor(.blue)
                Text("Status: \(approvalStatus)")
                    .font(.subheadline)
            }
            .padding()
            .background(Color(.systemGray6))
            .cornerRadius(8)
            
            Button(action: { currentStep = .enterPassword }) {
                Text("Cancel")
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.gray)
                    .foregroundColor(.white)
                    .cornerRadius(8)
            }
        }
    }
    
    private var approvedStep: some View {
        VStack(spacing: 24) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 64))
                .foregroundColor(.green)
            
            VStack(spacing: 8) {
                Text("Password Reset!")
                    .font(.title2)
                    .fontWeight(.bold)
                
                Text("Your password has been successfully reset. You can now login with your new password.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }
            
            Button(action: { dismiss() }) {
                Text("Done")
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.blue)
                    .foregroundColor(.white)
                    .cornerRadius(8)
            }
        }
    }
    
    private func requestPhoneReset() {
        isLoading = true
        errorMessage = ""
        showError = false
        
        Task {
            do {
                let id = try await APIService.shared.initiatePhonePasswordReset(
                    email: email,
                    newPassword: newPassword
                )
                
                await MainActor.run {
                    requestId = id
                    currentStep = .waiting
                    isLoading = false
                    approvalStatus = "Pending..."
                    
                    // Start polling for approval
                    startPollingForApproval()
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    isLoading = false
                }
            }
        }
    }
    
    private func startPollingForApproval() {
        statusCheckTimer = Timer.scheduledTimer(withTimeInterval: 2.0, repeats: true) { _ in
            Task {
                do {
                    let status = try await APIService.shared.checkPhoneResetStatus(requestId: requestId)
                    
                    await MainActor.run {
                        approvalStatus = status.status
                        
                        if status.status == "approved" {
                            statusCheckTimer?.invalidate()
                            completePhoneReset()
                        } else if status.status == "denied" {
                            statusCheckTimer?.invalidate()
                            errorMessage = "Reset request was denied"
                            showError = true
                            currentStep = .enterPassword
                        } else if status.status == "expired" {
                            statusCheckTimer?.invalidate()
                            errorMessage = "Reset request has expired"
                            showError = true
                            currentStep = .enterPassword
                        }
                    }
                } catch {
                    // Silently handle polling errors
                }
            }
        }
    }
    
    private func completePhoneReset() {
        Task {
            do {
                try await APIService.shared.completePhoneReset(requestId: requestId)
                
                await MainActor.run {
                    currentStep = .approved
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    showError = true
                    currentStep = .enterPassword
                }
            }
        }
    }
}

#Preview {
    PhonePasswordResetView()
}
```

---

### Phase 3: Update Existing Views

#### File: `ios/PhotoSync/Views/SettingsView.swift`

Update to add password management section:

```swift
// Add this section to the VStack in SettingsView

Section("Security") {
    if AppSettings.isConfigured {
        Button(action: { showPasswordSettings = true }) {
            Label("Change Password", systemImage: "key.fill")
        }
        
        Button(role: .destructive, action: { showLogoutConfirm = true }) {
            Label("Sign Out", systemImage: "square.and.arrow.right")
        }
    }
}
.sheet(isPresented: $showPasswordSettings) {
    ChangePasswordView()
}
.confirmationDialog(
    "Sign Out?",
    isPresented: $showLogoutConfirm,
    actions: {
        Button("Sign Out", role: .destructive) {
            signOut()
        }
    },
    message: {
        Text("Are you sure you want to sign out?")
    }
)

@State private var showPasswordSettings = false
@State private var showLogoutConfirm = false

private func signOut() {
    AppSettings.apiKey = ""
    AppSettings.serverURL = ""
    AppSettings.deviceId = nil
}
```

Create a new `ChangePasswordView`:

```swift
struct ChangePasswordView: View {
    @Environment(\.dismiss) var dismiss
    @State private var currentPassword = ""
    @State private var newPassword = ""
    @State private var confirmPassword = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    @State private var showError = false
    @State private var showSuccess = false
    
    var isFormValid: Bool {
        !currentPassword.isEmpty && newPassword.count >= 8 && newPassword == confirmPassword
    }
    
    var body: some View {
        NavigationStack {
            VStack(spacing: 16) {
                Form {
                    Section("Current Password") {
                        SecureField("Password", text: $currentPassword)
                            .textContentType(.password)
                    }
                    
                    Section("New Password") {
                        SecureField("New password", text: $newPassword)
                            .textContentType(.newPassword)
                        
                        SecureField("Confirm password", text: $confirmPassword)
                            .textContentType(.newPassword)
                    }
                    
                    if showError {
                        Section {
                            HStack {
                                Image(systemName: "exclamationmark.circle.fill")
                                    .foregroundColor(.red)
                                Text(errorMessage)
                                    .font(.subheadline)
                            }
                        }
                    }
                    
                    Section {
                        Button(action: changePassword) {
                            if isLoading {
                                ProgressView()
                            } else {
                                Text("Update Password")
                                    .frame(maxWidth: .infinity)
                            }
                        }
                        .disabled(!isFormValid || isLoading)
                    }
                }
            }
            .navigationTitle("Change Password")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") { dismiss() }
                }
            }
            .alert("Success", isPresented: $showSuccess) {
                Button("OK") { dismiss() }
            } message: {
                Text("Your password has been updated successfully.")
            }
        }
    }
    
    private func changePassword() {
        isLoading = true
        errorMessage = ""
        showError = false
        
        // TODO: Implement password change API call
        // This should verify current password and set new one
    }
}
```

---

### Phase 4: Update App Settings

#### File: `ios/PhotoSync/Models/Settings.swift`

Add password storage (optional, can be stored in Keychain or UserDefaults):

```swift
// Add this to AppSettings struct

private static let passwordStorageKey = "passwordStored"

/// Whether password is stored locally (optional - API key is preferred for security)
static var hasPassword: Bool {
    get { UserDefaults.standard.bool(forKey: passwordStorageKey) }
    set { UserDefaults.standard.set(newValue, forKey: passwordStorageKey) }
}

/// Check if user is authenticated (either API key or password available)
static var isAuthenticated: Bool {
    !apiKey.isEmpty || hasPassword
}
```

---

## Frontend Checklist

### Models & Services
- [ ] Add LoginRequest, LoginResponse, and reset models to APIModels.swift
- [ ] Add all API methods to APIService.swift (7 new methods)

### Views (New)
- [ ] Create LoginView.swift
- [ ] Create PasswordResetView.swift
- [ ] Create EmailPasswordResetView.swift
- [ ] Create PhonePasswordResetView.swift
- [ ] Create ChangePasswordView.swift
- [ ] Create APIKeySetupView.swift (for existing API key flow)

### Views (Update)
- [ ] Update SettingsView.swift with password management
- [ ] Update ContentView.swift to show LoginView if not authenticated
- [ ] Update PhotoSyncApp.swift to check authentication on launch

### Models
- [ ] Update Settings.swift with password fields

### Testing
- [ ] Test login with valid credentials
- [ ] Test email password reset flow
- [ ] Test phone 2FA password reset flow
- [ ] Test API key refresh
- [ ] Test password change
- [ ] Test with existing API key users (backward compatibility)

---

## Implementation Notes

### Security Best Practices (iOS)
- ✅ Store API keys in Keychain (already implemented)
- ✅ Use SecureField for password input
- ✅ Never log passwords
- ✅ Clear sensitive data on sign out
- ✅ Use HTTPS only (already enforced)

### Error Handling
- Show user-friendly error messages
- Log errors for debugging (via Logger service)
- Provide recovery options (retry, use alternate method)

### UX Considerations
- Keep flows simple and intuitive
- Use clear labels and instructions
- Show validation feedback in real-time
- Provide progress indicators for multi-step flows
- Allow users to go back if they made a mistake

---

## Estimated Implementation Time

| Component | Estimated Time |
|-----------|----------------|
| Models & APIService methods | 1-2 hours |
| LoginView | 1 hour |
| PasswordResetView selector | 30 minutes |
| EmailPasswordResetView | 1.5 hours |
| PhonePasswordResetView | 1.5 hours |
| ChangePasswordView | 1 hour |
| Update existing views | 1 hour |
| Testing & debugging | 2 hours |
| **Total** | **~9-11 hours** |

---

## Next Steps on Mac

1. Clone/pull latest code
2. Open `ios/PhotoSync.xcodeproj` in Xcode
3. Implement models and services first (Phase 1)
4. Build and test APIService methods
5. Implement UI views (Phase 2-3)
6. Test full authentication flows
7. Build and upload to TestFlight

All backend APIs are ready and tested!

