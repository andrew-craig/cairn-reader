import Foundation
import Testing
@testable import CairnReader

// Port of apps/shared/src/utils/readProgress.test.ts (`isPartiallyRead`), plus the row's details line.

struct ReadProgressTests {
    private func article(isRead: Bool = false, scrollPosition: Double? = nil) -> Article {
        var article = storedArticle("a1")
        article.isRead = isRead
        article.scrollPosition = scrollPosition
        return article
    }

    @Test func isFalseWithNoSavedProgress() {
        #expect(!article().isPartiallyRead)
        #expect(!article(scrollPosition: 0).isPartiallyRead)
    }

    @Test func isTrueWithProgressBetweenZeroAndOne() {
        #expect(article(scrollPosition: 0.4).isPartiallyRead)
        #expect(article(scrollPosition: 1).isPartiallyRead)
    }

    @Test func ignoresLegacyPixelOffsets() {
        #expect(!article(scrollPosition: 350).isPartiallyRead)
    }

    @Test func isFalseOnceTheArticleIsRead() {
        #expect(!article(isRead: true, scrollPosition: 0.5).isPartiallyRead)
    }
}

@MainActor
struct ArticleRowTests {
    private let now = Date(timeIntervalSince1970: 1_700_000_000)

    @Test func detailsJoinAuthorDateAndReadingTime() {
        var article = storedArticle("a1")
        article.author = "Jane Doe"
        article.publishedDate = now.addingTimeInterval(-3 * 86_400)
        article.readingTime = 5
        let date = article.publishedDate!.relativeDayLabel(now: now)
        #expect(ArticleRow.details(for: article, now: now) == "Jane Doe · \(date) · 5 min")
    }

    @Test func detailsFallBackToTheDomainAndLeaveOutWhatIsUnknown() {
        var article = storedArticle("a1")
        article.url = "https://www.example.com/post"
        #expect(ArticleRow.details(for: article, now: now) == "example.com")

        article.author = ""
        article.url = "not a url"
        #expect(ArticleRow.details(for: article, now: now) == "")
    }
}
