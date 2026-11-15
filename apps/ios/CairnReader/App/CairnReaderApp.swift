import SwiftUI

/// The composition root: builds the services once and hands them to the views through the
/// environment. Nothing below this is a global or is constructed inside a view.
@main
struct CairnReaderApp: App {
    @Environment(\.scenePhase) private var scenePhase
    @State private var sessionModel: SessionModel
    @State private var connectivity = Connectivity()
    @State private var loginModel: LoginModel
    private let systemService: SystemService

    init() {
        Typography.applyBarAppearance()
        let server = ServerConfig()
        let client = HTTPClient(server: server)
        let auth = AuthService(client: client)
        systemService = SystemService(client: client)
        let sessionModel = SessionModel(auth: auth)
        _sessionModel = State(initialValue: sessionModel)
        _loginModel = State(
            initialValue: LoginModel(auth: auth, server: server) { await sessionModel.login() })
    }

    var body: some Scene {
        WindowGroup {
            RootView(loginModel: loginModel)
                .environment(sessionModel)
                .environment(connectivity)
                .environment(\.systemService, systemService)
                .font(Typography.body)
                .task { await sessionModel.start() }
        }
        .onChange(of: scenePhase) { _, phase in
            AppDatabase.setSuspended(phase == .background)
        }
    }
}

/// Progress while the saved session is checked, Login when signed out, the tabs when signed in.
/// The offline banner sits above all three.
struct RootView: View {
    /// What the root shows for the session's state.
    enum Screen {
        case loading
        case signedOut
        case signedIn(UserSession)

        init(_ model: SessionModel) {
            if model.isLoading {
                self = .loading
            } else if let session = model.session {
                self = .signedIn(session)
            } else {
                self = .signedOut
            }
        }
    }

    @Environment(SessionModel.self) private var sessionModel
    @Environment(Connectivity.self) private var connectivity
    let loginModel: LoginModel

    var body: some View {
        Group {
            switch Screen(sessionModel) {
            case .loading:
                ProgressView()
                    .controlSize(.large)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            case .signedOut:
                LoginView(model: loginModel)
            case let .signedIn(session):
                // The id ends navigation state and screen models with the account (decision 10).
                MainTabs(session: session)
                    .id(session.user.id)
            }
        }
        .safeAreaInset(edge: .top, spacing: 0) {
            if !connectivity.isOnline {
                OfflineBanner()
            }
        }
        .animation(.default, value: connectivity.isOnline)
    }
}
