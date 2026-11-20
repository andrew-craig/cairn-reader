import SwiftUI
import UIKit
import UIKit.UIGestureRecognizerSubclass
import WebKit

/// The reader's web view (decision 2): shows the page from `ReaderTemplate` with no JavaScript,
/// never navigates away (a link tap goes to `onLink`, with where it was touched), restores the saved scroll fraction as the
/// content grows until the user drags, and reports the scroll geometry from the scroll view.
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
    /// A link was tapped at a point in the web view's own coordinates.
    let onLink: @MainActor (URL, CGPoint) -> Void

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
        webView.addGestureRecognizer(context.coordinator.touches)
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
        /// The page and its images have loaded (`didFinish`).
        private var hasLoaded = false
        /// The user has dragged since the page started loading: the restore ends, and readings
        /// count from here.
        private var hasDragged = false
        private var contentSizeObservation: NSKeyValueObservation?
        /// Where the last touch went down, for the link it activates.
        let touches = TouchDownRecognizer()

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
            hasLoaded = false
            hasDragged = false
            webView.loadHTMLString(html, baseURL: ReaderResources.baseURL)
        }

        // MARK: Scrolling

        /// Moves to the saved position on every growth of the content until the user drags. The
        /// fraction was measured on the finished layout, which can reach the scroll view after
        /// `didFinish` and after a first layout that could already reach the target (text before
        /// its images), so only the last growth puts it in the right place.
        private func contentSizeChanged() {
            guard let scrollView = webView?.scrollView else { return }
            if !hasDragged, restoreFraction > 0 {
                let scroll = self.scroll(of: scrollView)
                if scroll.viewport > 0, scroll.contentHeight > 0 {
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
            hasDragged = true
        }

        private func report(_ scrollView: UIScrollView) {
            let scroll = self.scroll(of: scrollView)
            guard scroll.viewport > 0, scroll.contentHeight > 0 else { return }
            parent.onScroll(scroll)
        }

        /// The geometry measured from the top of the content, inside the bars' insets.
        private func scroll(of scrollView: UIScrollView) -> ReaderModel.Scroll {
            let insets = scrollView.adjustedContentInset
            return ReaderModel.Scroll(
                offset: scrollView.contentOffset.y + insets.top,
                viewport: scrollView.bounds.height - insets.top - insets.bottom,
                contentHeight: scrollView.contentSize.height,
                hasLoaded: hasLoaded,
                hasDragged: hasDragged)
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
                parent.onLink(url, touches.location)
                return .cancel
            }
            return isPage && navigationAction.targetFrame?.isMainFrame == true ? .allow : .cancel
        }

        /// The page and its images have loaded. Their layout may not have reached the scroll view
        /// yet (the content size can still be the viewport's, which reads as the end of the
        /// article, bug_f546), so this neither ends the restore nor confirms a reading by itself:
        /// readings count once the user has also dragged.
        func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
            hasLoaded = true
            report(webView.scrollView)
        }

        /// The content process was killed (e.g. memory pressure): load the page again.
        func webViewWebContentProcessDidTerminate(_ webView: WKWebView) {
            guard let html = loadedHTML else { return }
            loadedHTML = nil
            load(html)
        }
    }
}

/// Records where each touch goes down in its view and never recognises, so the web view's own
/// gestures (link taps, scrolling) are untouched. Touch-down always comes before the link
/// activation it leads to, which reaches the navigation delegate only after the finger lifts and a
/// round trip to the web content process, so the location is that tap's.
final class TouchDownRecognizer: UIGestureRecognizer {
    private(set) var location = CGPoint.zero

    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent) {
        if let touch = touches.first { location = touch.location(in: view) }
        state = .failed
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
