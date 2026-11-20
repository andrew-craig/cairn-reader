import Foundation
import Testing
import UIKit
@testable import CairnReader

/// Every token in design/tokens/colors.json resolves from the generated Colors.xcassets
/// with its light and dark value. The simulator can read the repository, so the test
/// reads the source palette instead of a hand-kept list of names.
@MainActor
struct DesignTokenTests {
    typealias Palette = [String: [String: String]]

    static let palette: Palette = {
        let url = URL(filePath: #filePath)
            .deletingLastPathComponent()
            .appending(path: "../../../design/tokens/colors.json")
            .standardized
        let json = try! JSONSerialization.jsonObject(with: Data(contentsOf: url)) as! [String: Any]
        return ["light", "dark"].reduce(into: [:]) { palette, mode in
            let tokens = json[mode] as! [String: [String: Any]]
            palette[mode] = tokens.mapValues { $0["$value"] as! String }
        }
    }()

    @Test(arguments: [UIUserInterfaceStyle.light, .dark])
    func everyTokenResolves(style: UIUserInterfaceStyle) throws {
        let tokens = try #require(Self.palette[style == .dark ? "dark" : "light"])
        #expect(!tokens.isEmpty)
        let traits = UITraitCollection(userInterfaceStyle: style)
        for (name, hex) in tokens {
            let color = try #require(UIColor(named: name, in: .main, compatibleWith: traits), "\(name) missing")
            #expect(Self.hex(color.resolvedColor(with: traits)) == hex.uppercased(), "\(name)")
        }
    }

    @Test func fontsLoad() {
        for font in BrandFont.allCases {
            #expect(UIFont(name: font.rawValue, size: 16) != nil, "\(font.rawValue) not bundled")
        }
    }

    /// '#RRGGBB', or '#RRGGBBAA' when not opaque, matching colors.json.
    static func hex(_ color: UIColor) -> String {
        var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
        color.getRed(&r, green: &g, blue: &b, alpha: &a)
        let bytes = [r, g, b, a].map { Int(($0 * 255).rounded()) }
        let rgb = String(format: "#%02X%02X%02X", bytes[0], bytes[1], bytes[2])
        return bytes[3] == 255 ? rgb : rgb + String(format: "%02X", bytes[3])
    }
}
