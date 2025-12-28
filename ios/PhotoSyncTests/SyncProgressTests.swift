import XCTest
@testable import PhotoSync

final class SyncProgressTests: XCTestCase {

    func testProgressPercentWithNoTotal() {
        let progress = SyncProgress(total: 0, completed: 0, failed: 0, currentFileName: nil)

        XCTAssertEqual(progress.progressPercent, 0)
    }

    func testProgressPercentWithProgress() {
        let progress = SyncProgress(total: 100, completed: 50, failed: 0, currentFileName: nil)

        XCTAssertEqual(progress.progressPercent, 0.5)
    }

    func testProgressPercentComplete() {
        let progress = SyncProgress(total: 100, completed: 100, failed: 0, currentFileName: nil)

        XCTAssertEqual(progress.progressPercent, 1.0)
    }

    func testIsCompleteWhenAllSuccessful() {
        let progress = SyncProgress(total: 10, completed: 10, failed: 0, currentFileName: nil)

        XCTAssertTrue(progress.isComplete)
    }

    func testIsCompleteWhenSomeFailed() {
        let progress = SyncProgress(total: 10, completed: 7, failed: 3, currentFileName: nil)

        XCTAssertTrue(progress.isComplete)
    }

    func testIsCompleteWhenInProgress() {
        let progress = SyncProgress(total: 10, completed: 5, failed: 0, currentFileName: nil)

        XCTAssertFalse(progress.isComplete)
    }

    func testIsCancelledDefaultsFalse() {
        let progress = SyncProgress(total: 10, completed: 0, failed: 0, currentFileName: nil)

        XCTAssertFalse(progress.isCancelled)
    }

    func testCurrentFileNameIsStored() {
        let progress = SyncProgress(total: 10, completed: 5, failed: 0, currentFileName: "IMG_1234.jpg")

        XCTAssertEqual(progress.currentFileName, "IMG_1234.jpg")
    }
}
