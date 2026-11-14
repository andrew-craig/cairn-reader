import Foundation
import Synchronization

/// Answers requests from a per-session handler, so tests running in parallel do not share one.
/// `StubURLProtocol.session(handler)` returns a session whose requests go to `handler`.
nonisolated final class StubURLProtocol: URLProtocol, @unchecked Sendable {
    typealias Handler = @Sendable (URLRequest) throws -> (Int, Data)

    private static let handlers = Mutex<[String: Handler]>([:])
    private static let idHeader = "X-Stub-ID"

    static func session(_ handler: @escaping Handler) -> URLSession {
        let id = UUID().uuidString
        handlers.withLock { $0[id] = handler }
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [StubURLProtocol.self]
        configuration.httpAdditionalHeaders = [idHeader: id]
        return URLSession(configuration: configuration)
    }

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        let id = request.value(forHTTPHeaderField: Self.idHeader) ?? ""
        guard let handler = Self.handlers.withLock({ $0[id] }) else {
            client?.urlProtocol(self, didFailWithError: URLError(.unknown))
            return
        }
        do {
            var request = request
            // URLSession moves the body into a stream before it reaches a protocol.
            if request.httpBody == nil, let stream = request.httpBodyStream {
                request.httpBody = Self.read(stream)
            }
            let (status, data) = try handler(request)
            let response = HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: nil, headerFields: nil)!
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}

    private static func read(_ stream: InputStream) -> Data {
        stream.open()
        defer { stream.close() }
        var data = Data()
        var buffer = [UInt8](repeating: 0, count: 4096)
        while stream.hasBytesAvailable {
            let count = stream.read(&buffer, maxLength: buffer.count)
            guard count > 0 else { break }
            data.append(buffer, count: count)
        }
        return data
    }
}
