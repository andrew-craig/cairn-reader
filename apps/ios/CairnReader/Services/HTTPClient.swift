import Foundation

/// One request against the configured server.
nonisolated struct HTTPRequest: Sendable {
    enum Method: String, Sendable {
        case get = "GET", post = "POST", put = "PUT", patch = "PATCH", delete = "DELETE"
    }

    var method: Method
    /// Server-relative, e.g. `/api/v1/content/user/{id}`.
    var path: String
    var query: [URLQueryItem]
    /// Sent as JSON.
    var body: (any Encodable & Sendable)?
    /// Extra headers, e.g. `Authorization`.
    var headers: [String: String]
    /// Idle timeout in seconds (`URLRequest.timeoutInterval`): the request fails with
    /// `NetworkError.timeout` when no data arrives for this long. Not a hard deadline. nil uses the
    /// session's.
    var timeout: TimeInterval?

    init(
        _ method: Method = .get, _ path: String, query: [URLQueryItem] = [],
        body: (any Encodable & Sendable)? = nil, headers: [String: String] = [:], timeout: TimeInterval? = nil
    ) {
        self.method = method
        self.path = path
        self.query = query
        self.body = body
        self.headers = headers
        self.timeout = timeout
    }
}

/// The `{ data, meta }` wrapper around most API responses.
nonisolated struct Envelope<Value: Decodable & Sendable>: Decodable, Sendable {
    let data: Value
}

/// The `{ data, pagination, meta }` wrapper around list and search responses.
nonisolated struct Page<Item: Decodable & Sendable>: Decodable, Sendable {
    struct Pagination: Decodable, Sendable {
        /// Cursor for the next page; absent on the last page.
        let cursor: String?
        let hasMore: Bool
        let limit: Int
    }

    let data: [Item]
    let pagination: Pagination
}

/// Performs requests against the configured server. Transport failures (and success bodies that
/// cannot be decoded) throw `NetworkError`; a non-2xx answer throws `HttpError`.
nonisolated struct HTTPClient: Sendable {
    let server: ServerConfig
    let session: URLSession

    init(server: ServerConfig = ServerConfig(), session: URLSession = .shared) {
        self.server = server
        self.session = session
    }

    /// Sends `request` and decodes the response body as `Response`.
    func send<Response: Decodable>(_ request: HTTPRequest, as _: Response.Type = Response.self) async throws
        -> Response
    {
        let data = try await perform(request)
        do {
            return try Self.decoder.decode(Response.self, from: data)
        } catch {
            throw NetworkError.undecodableBody
        }
    }

    /// Sends `request` and ignores the response body.
    func send(_ request: HTTPRequest) async throws {
        _ = try await perform(request)
    }

    private func perform(_ request: HTTPRequest) async throws -> Data {
        let urlRequest = try makeURLRequest(request)
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: urlRequest)
        } catch let error as URLError {
            switch error.code {
            case .cancelled: throw CancellationError()
            case .timedOut: throw NetworkError.timeout
            default: throw NetworkError.unreachable
            }
        }
        guard let http = response as? HTTPURLResponse else { throw NetworkError.undecodableBody }
        guard (200..<300).contains(http.statusCode) else {
            let body = try? Self.decoder.decode(ErrorBody.self, from: data)
            throw HttpError(status: http.statusCode, message: body?.message ?? body?.error)
        }
        return data
    }

    private func makeURLRequest(_ request: HTTPRequest) throws -> URLRequest {
        var url = server.url.appending(path: request.path)
        if !request.query.isEmpty {
            // Encodes everything but unreserved characters ourselves: Foundation leaves `+` raw
            // (base64 cursors and searches contain it) and Go reads it as a space. Covered by
            // HTTPClientTests.
            var components = URLComponents(url: url, resolvingAgainstBaseURL: false)!
            components.percentEncodedQueryItems = request.query.map {
                URLQueryItem(name: Self.encodeQuery($0.name), value: $0.value.map(Self.encodeQuery))
            }
            url = components.url!
        }
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = request.method.rawValue
        if let timeout = request.timeout {
            urlRequest.timeoutInterval = timeout
        }
        urlRequest.setValue("application/json", forHTTPHeaderField: "Accept")
        if let body = request.body {
            urlRequest.httpBody = try Self.encoder.encode(body)
            urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        for (field, value) in request.headers {
            urlRequest.setValue(value, forHTTPHeaderField: field)
        }
        return urlRequest
    }

    private struct ErrorBody: Decodable {
        let error: String?
        let message: String?
    }

    /// RFC 3986 unreserved characters; everything else in a query name or value is percent-encoded.
    private static let queryUnreserved = CharacterSet(
        charactersIn: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~")

    private static func encodeQuery(_ text: String) -> String {
        text.addingPercentEncoding(withAllowedCharacters: queryUnreserved)!
    }

    static let decoder: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let text = try container.decode(String.self)
            // Go writes RFC 3339 with fractional seconds only when they are non-zero.
            if let date = (try? Date(text, strategy: Self.iso8601WithFractionalSeconds))
                ?? (try? Date(text, strategy: .iso8601))
            {
                return date
            }
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Invalid date \(text)")
        }
        return decoder
    }()

    private static let iso8601WithFractionalSeconds = Date.ISO8601FormatStyle(includingFractionalSeconds: true)

    static let encoder: JSONEncoder = {
        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        return encoder
    }()
}
