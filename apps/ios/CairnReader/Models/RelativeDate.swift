import Foundation

extension Date {
    /// Day-granular label for a past date: "Today", "Yesterday", "3 days ago", "2 weeks ago",
    /// "Last month". Uses the same buckets as the RN app's `formatDate` (weeks under 30 days,
    /// 30-day months, 365-day years) with the system's wording. Dates slightly in the future
    /// (clock skew) read as "Today".
    nonisolated func relativeDayLabel(now: Date = .now, locale: Locale = .current) -> String {
        let days = Swift.max(0, Int(now.timeIntervalSince(self) / 86_400))
        var components = DateComponents()
        switch days {
        case ..<7: components.day = -days
        case ..<30: components.weekOfMonth = -(days / 7)
        case ..<365: components.month = -(days / 30)
        default: components.year = -(days / 365)
        }
        let formatter = RelativeDateTimeFormatter()
        formatter.locale = locale
        formatter.dateTimeStyle = .named
        formatter.unitsStyle = .full
        formatter.formattingContext = .beginningOfSentence
        return formatter.localizedString(from: components)
    }
}
