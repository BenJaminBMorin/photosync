import Foundation

/// Global logging helper functions for easy access throughout the app

/// Log an info message
func logInfo(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Task {
        await Logger.shared.info(message, file: file, function: function, line: line)
    }
}

/// Log a warning message
func logWarning(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Task {
        await Logger.shared.warning(message, file: file, function: function, line: line)
    }
}

/// Log an error message
func logError(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Task {
        await Logger.shared.error(message, file: file, function: function, line: line)
    }
}

/// Log a fatal error message
func logFatal(_ message: String, file: String = #file, function: String = #function, line: Int = #line) {
    Task {
        await Logger.shared.fatal(message, file: file, function: function, line: line)
    }
}
