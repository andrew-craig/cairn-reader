import SwiftUI
import UIKit
import WebKit

/// The reader's web view (decision 2): shows the page from `ReaderTemplate` with no JavaScript,
/// never navigates away (a link tap goes to `onLink`), restores the saved scroll fraction once
/// the content is tall enough to reach it, and reports the scroll geometry from the scroll view.
///
/// A `WKWebView` rather than SwiftUI's `WebView`: restoring needs the scroll view's content size
/// as it grows (images load late), the moment the user starts dragging, and a synchronous offset
/// write, which the scroll view gives directly.
struct ArticleWebView: UIViewRepresentable {
    let html: String
    /// The fraction to restore when a page loads (read at each load, so a reload restores where the
    /// reader was); 0 restores nothing.
    let restoreFraction: @MainActor () -> Double
    let onScroll: @MainActor (ReaderModel.Scroll) -> Void
    let onLink: @MainActor (URL) -> Void

    /// Whether a saved `fraction` can be reached with the content and viewport measured so far:
    /// the target `fraction * contentHeight` is at most the largest offset,
    /// `contentHeight - viewport`. Never before the viewport is measured.
    nonisolated static func isScrollTargetReachable(contentHeight: Double, viewport: Double, fraction: Double) -> Bool {
        viewport > 0 && contentHeight - viewport >= fraction * contentHeight
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeUIView(context: Context) -> WKWebView {
        let configuration = WKWebViewConfiguration()
        // `cleaned_html` is sanitised by the server; no script runs all the same.
        configuration.defaultWebpagePreferences.allowsContentJavaScript = false
        configuration.setURLSchemeHandler(ReaderResources(), forURLScheme: ReaderResources.scheme)
        let webView = WKWebView(frame: .zero, configuration: configuration)
        webView.navigationDelegate = context.coordinator
        webView.scrollView.delegate = context.coordinator
        webView.allowsLinkPreview = false
        webView.isOpaque = false
        webView.backgroundColor = UIColor(resource: .background)
        context.coordinator.attach(webView)
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        context.coordinator.parent = self
        context.coordinator.load(html)
    }

    final class Coordinator: NSObject, WKNavigationDelegate, UIScrollViewDelegate {
        var parent: ArticleWebView
        private weak var webView: WKWebView?
        private var loadedHTML: String?
        private var restoreFraction = 0.0
        /// The saved position has been restored (or there was none), or the user took over.
        private var hasRestored = true
        private var contentSizeObservation: NSKeyValueObservation?

        init(_ parent: ArticleWebView) {
            self.parent = parent
        }

        func attach(_ webView: WKWebView) {
            self.webView = webView
            contentSizeObservation = webView.scrollView.observe(\.contentSize) { [weak self] _, _ in
                MainActor.assumeIsolated { self?.contentSizeChanged() }
            }
        }

        /// Loads `html` unless it is already shown, and starts restoring the parent's fraction.
        func load(_ html: String) {
            guard let webView, html != loadedHTML else { return }
            loadedHTML = html
            restoreFraction = parent.restoreFraction()
            hasRestored = restoreFraction <= 0
            webView.loadHTMLString(html, baseURL: ReaderResources.baseURL)
        }

        // MARK: Scrolling

        /// Moves towards the saved position on every growth of the content, until it can be
        /// reached (then it is restored) or the user drags.
        private func contentSizeChanged() {
            guard let scrollView = webView?.scrollView else { return }
            if !hasRestored {
                let scroll = Self.scroll(of: scrollView, isRestoring: true)
                if scroll.viewport > 0, scroll.contentHeight > 0 {
                    if ArticleWebView.isScrollTargetReachable(
                        contentHeight: scroll.contentHeight, viewport: scroll.viewport, fraction: restoreFraction)
                    {
                        hasRestored = true
                    }
                    let target = min(restoreFraction * scroll.contentHeight, max(scroll.contentHeight - scroll.viewport, 0))
                    // Reports through `scrollViewDidScroll` when the offset changes.
                    scrollView.contentOffset.y = CGFloat(target) - scrollView.adjustedContentInset.top
                }
            }
            report(scrollView)
        }

        func scrollViewDidScroll(_ scrollView: UIScrollView) {
            report(scrollView)
        }

        /// Once the user drags, the restore stops: they are never pulled away from where they went.
        func scrollViewWillBeginDragging(_ scrollView: UIScrollView) {
            hasRestored = true
        }

        private func report(_ scrollView: UIScrollView) {
            let scroll = Self.scroll(of: scrollView, isRestoring: !hasRestored)
            guard scroll.viewport > 0, scroll.contentHeight > 0 else { return }
            parent.onScroll(scroll)
        }

        /// The geometry measured from the top of the content, inside the bars' insets.
        private static func scroll(of scrollView: UIScrollView, isRestoring: Bool) -> ReaderModel.Scroll {
            let insets = scrollView.adjustedContentInset
            return ReaderModel.Scroll(
                offset: scrollView.contentOffset.y + insets.top,
                viewport: scrollView.bounds.height - insets.top - insets.bottom,
                contentHeight: scrollView.contentSize.height,
                isRestoring: isRestoring)
        }

        // MARK: Navigation

        /// Only the page itself (and its own anchors) loads; any other link goes to `onLink`, and
        /// nothing else navigates.
        func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction) async
            -> WKNavigationActionPolicy
        {
            let url = navigationAction.request.url
            let isPage = url?.scheme == ReaderResources.scheme
            if navigationAction.navigationType == .linkActivated, !isPage, let url {
                parent.onLink(url)
                return .cancel
            }
            return isPage && navigationAction.targetFrame?.isMainFrame == true ? .allow : .cancel
        }

        /// The content process was killed (e.g. memory pressure): load the page again.
        func webViewWebContentProcessDidTerminate(_ webView: WKWebView) {
            guard let html = loadedHTML else { return }
            loadedHTML = nil
            load(html)
        }
    }
}

/// Serves the reader's stylesheet and fonts from the app bundle under `cairn-reader://reader/`,
/// the page's base URL, so the page needs no file access and no network for them.
final class ReaderResources: NSObject, WKURLSchemeHandler {
    static let scheme = "cairn-reader"
    static let baseURL = URL(string: "cairn-reader://reader/")!
    private static let mimeTypes = ["css": "text/css", "ttf": "font/ttf"]

    func webView(_ webView: WKWebView, start urlSchemeTask: any WKURLSchemeTask) {
        guard let url = urlSchemeTask.request.url, let mimeType = Self.mimeTypes[url.pathExtension],
            let file = Bundle.main.url(forResource: url.lastPathComponent, withExtension: nil),
            let data = try? Data(contentsOf: file)
        else {
            urlSchemeTask.didFailWithError(URLError(.fileDoesNotExist))
            return
        }
        urlSchemeTask.didReceive(
            URLResponse(url: url, mimeType: mimeType, expectedContentLength: data.count, textEncodingName: nil))
        urlSchemeTask.didReceive(data)
        urlSchemeTask.didFinish()
    }

    func webView(_ webView: WKWebView, stop urlSchemeTask: any WKURLSchemeTask) {}
}
