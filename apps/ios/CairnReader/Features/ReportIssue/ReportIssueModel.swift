import Foundation
import Observation

/// One report of a badly extracted article (the React Native `ReportIssueModal`): the reasons, an
/// optional comment, and sending them. A model lives for one presentation of the sheet, so each
/// opening starts empty and `load()` prefills it from the user's earlier report.
@Observable
final class ReportIssueModel {
    static let commentLimit = 2000

    /// The reasons in the server's category order, with the React Native labels.
    static let reasons: [(category: ExtractionFeedbackCategory, label: String)] = [
        (.bodyMissingContent, "Part of the article is missing"),
        (.bodyExtraContent, "Contains things that aren't part of the article (ads, menus, related links)"),
        (.images, "Images are missing or broken"),
        (.formatting, "Formatting is wrong (lists, tables, code, layout)"),
        (.links, "Links are missing or broken"),
        (.title, "Title is wrong"),
        (.author, "Author is wrong"),
        (.description, "Summary is wrong"),
        (.leadImage, "Header image is wrong"),
        (.other, "Something else"),
    ]

    private(set) var selected: Set<ExtractionFeedbackCategory> = []
    /// Cut to `commentLimit` characters as it is set.
    var comment = "" {
        didSet {
            if comment.count > Self.commentLimit { comment = String(comment.prefix(Self.commentLimit)) }
        }
    }
    private(set) var isLoading = false
    private(set) var isSending = false
    /// The last send's failure, shown inline; cleared by a change of reasons or another send.
    private(set) var error: String?

    var canSend: Bool { !selected.isEmpty && !isSending && !isLoading }

    @ObservationIgnored private let read: ReadService
    @ObservationIgnored private let contentID: String

    init(read: ReadService, contentID: String) {
        self.read = read
        self.contentID = contentID
    }

    /// Prefills the reasons and comment from the user's earlier report, if any. A failed load
    /// leaves the form empty.
    func load() async {
        isLoading = true
        defer { isLoading = false }
        guard let existing = try? await read.getExtractionFeedback(contentID) else { return }
        selected = Set(existing.categories)
        comment = existing.comment ?? ""
    }

    func toggle(_ category: ExtractionFeedbackCategory) {
        if selected.contains(category) {
            selected.remove(category)
        } else {
            selected.insert(category)
        }
        error = nil
    }

    /// Sends the report; true when it was accepted. On failure the form is kept and `error` set.
    func send() async -> Bool {
        guard canSend else { return false }
        error = nil
        isSending = true
        defer { isSending = false }
        let trimmed = comment.trimmingCharacters(in: .whitespacesAndNewlines)
        let request = ExtractionFeedbackRequest(
            categories: Self.reasons.map(\.category).filter(selected.contains), comment: trimmed.isEmpty ? nil : trimmed)
        do {
            _ = try await read.submitExtractionFeedback(contentID, request)
            return true
        } catch {
            self.error = error.localizedDescription
            return false
        }
    }
}
