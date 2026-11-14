import Foundation
import Testing
@testable import CairnReader

struct CairnReaderAppTests {
    @Test func bundleIdentifier() {
        #expect(Bundle.main.bundleIdentifier == "com.cairnapp.cairnreader")
    }
}
