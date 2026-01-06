import Foundation
import UIKit
import os.log

class Logger {
    static let shared = Logger()

    private let logFile: URL
    private let fileHandle: FileHandle?
    private let queue = DispatchQueue(label: "com.photosync.logger", qos: .utility)
    private let dateFormatter: DateFormatter

    private init() {
        // Create logs directory
        let logsDir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("Logs", isDirectory: true)

        try? FileManager.default.createDirectory(at: logsDir, withIntermediateDirectories: true)

        // Check if last session crashed
        let crashMarkerFile = logsDir.appendingPathComponent(".crash_marker")
        let didCrashLastTime = FileManager.default.fileExists(atPath: crashMarkerFile.path)

        // Create log file with timestamp
        let timestamp = ISO8601DateFormatter().string(from: Date())
        logFile = logsDir.appendingPathComponent("log_\(timestamp).txt")

        // Create file if it doesn't exist
        if !FileManager.default.fileExists(atPath: logFile.path) {
            FileManager.default.createFile(atPath: logFile.path, contents: nil)
        }

        // Open file handle
        fileHandle = try? FileHandle(forWritingTo: logFile)

        // Date formatter for log entries
        dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd HH:mm:ss.SSS"

        // Write initial log entry
        log("=== PhotoSync Log Started ===", level: .info)
        log("Device: \(UIDevice.current.model)", level: .info)
        log("iOS Version: \(UIDevice.current.systemVersion)", level: .info)
        log("App Version: \(Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "unknown")", level: .info)

        if didCrashLastTime {
            log("⚠️ PREVIOUS SESSION CRASHED - Check previous log file", level: .error)
            // Remove crash marker
            try? FileManager.default.removeItem(at: crashMarkerFile)
        }

        // Create crash marker for this session
        FileManager.default.createFile(atPath: crashMarkerFile.path, contents: nil)

        // Set up crash handler
        setupCrashHandler()

        // Clean up old logs (keep last 10)
        cleanupOldLogs(in: logsDir)
    }

    deinit {
        try? fileHandle?.close()
    }

    enum LogLevel: String {
        case debug = "DEBUG"
        case info = "INFO"
        case warning = "WARNING"
        case error = "ERROR"
        case fatal = "FATAL"
    }

    func log(_ message: String, level: LogLevel = .info, file: String = #file, function: String = #function, line: Int = #line) {
        let timestamp = dateFormatter.string(from: Date())
        let fileName = (file as NSString).lastPathComponent
        let logMessage = "[\(timestamp)] [\(level.rawValue)] [\(fileName):\(line)] \(function) - \(message)\n"

        // Print to console
        print(logMessage, terminator: "")

        // Write to file asynchronously
        queue.async { [weak self] in
            guard let self = self, let data = logMessage.data(using: .utf8) else { return }
            self.fileHandle?.seekToEndOfFile()
            self.fileHandle?.write(data)
        }

        // Also use OSLog for system integration
        let osLog = OSLog(subsystem: Bundle.main.bundleIdentifier ?? "PhotoSync", category: "app")
        switch level {
        case .debug:
            os_log(.debug, log: osLog, "%{public}@", message)
        case .info:
            os_log(.info, log: osLog, "%{public}@", message)
        case .warning:
            os_log(.default, log: osLog, "%{public}@", message)
        case .error, .fatal:
            os_log(.error, log: osLog, "%{public}@", message)
        }
    }

    func debug(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
        log(message, level: .debug, file: file, function: function, line: line)
    }

    func info(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
        log(message, level: .info, file: file, function: function, line: line)
    }

    func warning(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
        log(message, level: .warning, file: file, function: function, line: line)
    }

    func error(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
        log(message, level: .error, file: file, function: function, line: line)
    }

    func fatal(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
        log(message, level: .fatal, file: file, function: function, line: line)
        // Force flush before potential crash
        try? fileHandle?.synchronize()
    }

    private func setupCrashHandler() {
        NSSetUncaughtExceptionHandler { exception in
            Logger.shared.fatal("Uncaught exception: \(exception)")
            Logger.shared.fatal("Stack trace: \(exception.callStackSymbols.joined(separator: "\n"))")
        }

        signal(SIGABRT) { signal in
            Logger.shared.fatal("Received SIGABRT signal")
            Logger.shared.flushLogs()
        }

        signal(SIGILL) { signal in
            Logger.shared.fatal("Received SIGILL signal")
            Logger.shared.flushLogs()
        }

        signal(SIGSEGV) { signal in
            Logger.shared.fatal("Received SIGSEGV signal")
            Logger.shared.flushLogs()
        }

        signal(SIGFPE) { signal in
            Logger.shared.fatal("Received SIGFPE signal")
            Logger.shared.flushLogs()
        }

        signal(SIGBUS) { signal in
            Logger.shared.fatal("Received SIGBUS signal")
            Logger.shared.flushLogs()
        }

        signal(SIGPIPE) { signal in
            Logger.shared.fatal("Received SIGPIPE signal")
            Logger.shared.flushLogs()
        }
    }

    func flushLogs() {
        queue.sync {
            try? fileHandle?.synchronize()
        }
    }

    func getCurrentLogURL() -> URL {
        return logFile
    }

    func getAllLogURLs() -> [URL] {
        let logsDir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("Logs", isDirectory: true)

        guard let files = try? FileManager.default.contentsOfDirectory(
            at: logsDir,
            includingPropertiesForKeys: [.creationDateKey],
            options: .skipsHiddenFiles
        ) else {
            return []
        }

        return files.filter { $0.pathExtension == "txt" }
            .sorted { url1, url2 in
                let date1 = (try? url1.resourceValues(forKeys: [.creationDateKey]))?.creationDate ?? Date.distantPast
                let date2 = (try? url2.resourceValues(forKeys: [.creationDateKey]))?.creationDate ?? Date.distantPast
                return date1 > date2
            }
    }

    private func cleanupOldLogs(in directory: URL) {
        let allLogs = getAllLogURLs()
        let logsToDelete = allLogs.dropFirst(10)

        for logURL in logsToDelete {
            try? FileManager.default.removeItem(at: logURL)
            info("Deleted old log file: \(logURL.lastPathComponent)")
        }
    }

    func getLogContent(from url: URL) -> String? {
        return try? String(contentsOf: url, encoding: .utf8)
    }

    func didCrashLastSession() -> Bool {
        let logsDir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("Logs", isDirectory: true)
        let crashMarkerFile = logsDir.appendingPathComponent(".crash_marker")
        return FileManager.default.fileExists(atPath: crashMarkerFile.path)
    }

    func markSessionCleanExit() {
        let logsDir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("Logs", isDirectory: true)
        let crashMarkerFile = logsDir.appendingPathComponent(".crash_marker")
        try? FileManager.default.removeItem(at: crashMarkerFile)
        log("=== PhotoSync Clean Exit ===", level: .info)
    }

    func getPreviousSessionLog() -> URL? {
        let allLogs = getAllLogURLs()
        // Return second log (first is current session)
        return allLogs.count > 1 ? allLogs[1] : nil
    }
}

// Global convenience functions
func logDebug(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Logger.shared.debug(message, file: file, function: function, line: line)
}

func logInfo(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Logger.shared.info(message, file: file, function: function, line: line)
}

func logWarning(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Logger.shared.warning(message, file: file, function: function, line: line)
}

func logError(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Logger.shared.error(message, file: file, function: function, line: line)
}

func logFatal(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Logger.shared.fatal(message, file: file, function: function, line: line)
}
